package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/home"
	"github.com/kirankandel/lovm/internal/selector"
	"github.com/kirankandel/lovm/internal/version"
)

func (a *App) uninstall(args []string) error {
	spec, err := oneSpec("uninstall", args)
	if err != nil {
		return err
	}
	installed, err := a.Home.Installed()
	if err != nil {
		return err
	}
	var matches []version.Build
	for _, b := range installed {
		if spec.Matches(b) {
			matches = append(matches, b)
		}
	}
	switch len(matches) {
	case 0:
		return errs.NotInstalled(spec.String(), "")
	case 1:
	default:
		return errs.Usage(fmt.Sprintf("%s matches %d installed versions; name one, e.g. lovm uninstall %s", spec, len(matches), matches[0]))
	}
	if err := os.RemoveAll(a.Home.VersionDir(matches[0])); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Removed LibreOffice %s\n", matches[0])
	return nil
}

func (a *App) ls(args []string) error {
	if len(args) != 0 {
		return errs.Usage("ls takes no arguments")
	}
	installed, err := a.Home.Installed()
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		fmt.Fprintln(a.Out, "No versions installed. Try: lovm install latest")
		return nil
	}
	active, sel, activeErr := selector.Active(a.Home, a.Getenv, a.Cwd)
	if activeErr != nil && !noActiveVersion(activeErr) {
		return activeErr
	}
	for _, b := range installed {
		if activeErr == nil && b == active {
			fmt.Fprintf(a.Out, "* %s  (%s from %s)\n", b, sel.Spec, sel.Origin)
		} else {
			fmt.Fprintf(a.Out, "  %s\n", b)
		}
	}
	return nil
}

// noActiveVersion reports errors that just mean nothing is active, which ls
// shows as the absence of a marker.
func noActiveVersion(err error) bool {
	var e *errs.Error
	return errors.As(err, &e) && (e.Code == errs.CodeNotSelected || e.Code == errs.CodeNotInstalled)
}

func (a *App) use(args []string) error {
	spec, err := oneSpec("use", args)
	if err != nil {
		return err
	}
	path := filepath.Join(a.Cwd, selector.RCFile)
	if err := os.WriteFile(path, []byte(spec.String()+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Pinned %s in %s\n", spec, path)
	return a.noteIfNotInstalled(spec)
}

func (a *App) setDefault(args []string) error {
	spec, err := oneSpec("default", args)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(a.Home.Root, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(a.Home.DefaultFile(), []byte(spec.String()+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Default set to %s\n", spec)
	return a.noteIfNotInstalled(spec)
}

func (a *App) noteIfNotInstalled(spec version.Spec) error {
	installed, err := a.Home.Installed()
	if err != nil {
		return err
	}
	if _, ok := selector.PickInstalled(spec, installed); !ok {
		fmt.Fprintf(a.Err, "Note: %s is not installed yet; run: lovm install %s\n", spec, spec)
	}
	return nil
}

func (a *App) current(args []string) error {
	if len(args) != 0 {
		return errs.Usage("current takes no arguments")
	}
	b, sel, err := selector.Active(a.Home, a.Getenv, a.Cwd)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "%s (%s from %s)\n", b, sel.Spec, sel.Origin)
	return nil
}

func (a *App) which(args []string) error {
	var b version.Build
	switch len(args) {
	case 0:
		active, _, err := selector.Active(a.Home, a.Getenv, a.Cwd)
		if err != nil {
			return err
		}
		b = active
	case 1:
		spec, err := oneSpec("which", args)
		if err != nil {
			return err
		}
		if b, err = a.installedBuild(spec); err != nil {
			return err
		}
	default:
		return errs.Usage("usage: lovm which [version]")
	}
	soffice, err := home.SofficePath(a.Home.InstallDir(b), a.Platform.OS)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.Out, soffice)
	return nil
}

// exec runs a command with LOVM_VERSION set and the shim first on PATH, so
// `soffice` inside the command still gets its per-version profile.
func (a *App) exec(ctx context.Context, args []string) error {
	if len(args) < 3 || args[1] != "--" {
		return errs.Usage("usage: lovm exec <version> -- <command> [args...]")
	}
	spec, err := oneSpec("exec", args[:1])
	if err != nil {
		return err
	}
	b, err := a.installedBuild(spec)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, args[2], args[3:]...)
	cmd.Env = append(os.Environ(),
		"LOVM_VERSION="+b.String(),
		"PATH="+a.Home.BinDir()+string(os.PathListSeparator)+a.Getenv("PATH"),
	)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = a.In, a.Out, a.Err
	err = cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() > 0 {
		return errs.ChildExit(exitErr.ExitCode())
	}
	return err
}

func (a *App) installedBuild(spec version.Spec) (version.Build, error) {
	installed, err := a.Home.Installed()
	if err != nil {
		return version.Build{}, err
	}
	b, ok := selector.PickInstalled(spec, installed)
	if !ok {
		return version.Build{}, errs.NotInstalled(spec.String(), "")
	}
	return b, nil
}

func (a *App) cache(args []string) error {
	if len(args) != 1 || args[0] != "clear" {
		return errs.Usage("usage: lovm cache clear")
	}
	if err := os.RemoveAll(a.Home.CacheDir()); err != nil {
		return err
	}
	fmt.Fprintln(a.Out, "Cache cleared")
	return nil
}
