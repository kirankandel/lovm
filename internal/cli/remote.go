package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/kirankandel/lovm/internal/archive"
	"github.com/kirankandel/lovm/internal/catalog"
	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/install"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/resolve"
	"github.com/kirankandel/lovm/internal/version"
)

// maxParallelLookups bounds concurrent requests to the archive server.
const maxParallelLookups = 8

func (a *App) install(ctx context.Context, args []string) error {
	spec, err := oneSpec("install", args)
	if err != nil {
		return err
	}
	view, err := a.openArchive(ctx, false)
	if err != nil {
		return err
	}
	if spec, err = view.Catalog.Expand(spec); err != nil {
		return err
	}
	res, err := a.resolver(view).Resolve(ctx, spec)
	a.saveLookups(view)
	if err != nil {
		return err
	}
	if _, err := os.Stat(a.Home.VersionDir(res.Build)); err == nil {
		fmt.Fprintf(a.Out, "LibreOffice %s is already installed\n", res.Build)
		return a.ensureShim()
	}
	if res.Rosetta {
		fmt.Fprintf(a.Err, "Note: LibreOffice %s has no Apple Silicon build; installing the x86_64 build, which runs under Rosetta 2\n", res.Build)
	}
	fmt.Fprintf(a.Out, "Installing LibreOffice %s for %s\n", res.Build, res.Platform)
	if err := install.New(a.Home, a.Err).Install(ctx, res); err != nil {
		return err
	}
	if err := a.ensureShim(); err != nil {
		return err
	}
	fmt.Fprintf(a.Out, "Installed LibreOffice %s\n", res.Build)
	a.printPathHint()
	return nil
}

func (a *App) lsRemote(ctx context.Context, args []string) error {
	opts, err := parseLsRemoteArgs(args)
	if err != nil {
		return err
	}
	view, err := a.openArchive(ctx, opts.refresh)
	if err != nil {
		return err
	}
	spec, err := view.Catalog.Expand(opts.spec)
	if err != nil {
		return err
	}
	var builds []version.Build
	for _, b := range view.AllBuilds() {
		if spec.Matches(b) && (opts.all || view.IsRelease(b)) {
			builds = append(builds, b)
		}
	}
	results, err := lookupAll(ctx, a.resolver(view), builds)
	a.saveLookups(view)
	if err != nil {
		return err
	}
	installed, err := a.Home.Installed()
	if err != nil {
		return err
	}
	for i, b := range builds {
		var notes []string
		if channel := view.Channel(b); channel != "" {
			notes = append(notes, channel)
		}
		if !view.IsRelease(b) {
			notes = append(notes, "RC")
		}
		switch {
		case !results[i].found && !opts.all:
			continue
		case !results[i].found:
			notes = append(notes, "no "+a.Platform.String()+" build")
		case results[i].res.Rosetta:
			notes = append(notes, "Rosetta")
		}
		if slices.Contains(installed, b) {
			notes = append(notes, "installed")
		}
		line := b.String()
		if len(notes) > 0 {
			line += "  (" + strings.Join(notes, ", ") + ")"
		}
		fmt.Fprintln(a.Out, line)
	}
	return nil
}

type lsRemoteOptions struct {
	spec         version.Spec
	all, refresh bool
}

// parseLsRemoteArgs is hand-rolled because the flag package stops at the
// first positional argument, which would break "lovm ls-remote 24.8 --all".
func parseLsRemoteArgs(args []string) (lsRemoteOptions, error) {
	var opts lsRemoteOptions
	prefix := ""
	for _, arg := range args {
		switch {
		case arg == "--all":
			opts.all = true
		case arg == "--refresh":
			opts.refresh = true
		case strings.HasPrefix(arg, "-"):
			return opts, errs.Usage(fmt.Sprintf("unknown flag %q", arg))
		case prefix != "":
			return opts, errs.Usage("ls-remote takes at most one version prefix")
		default:
			prefix = arg
		}
	}
	if prefix == "" {
		prefix = "latest"
	}
	spec, err := version.ParseSpec(prefix)
	if err != nil {
		return opts, err
	}
	if spec.BelowFloor() {
		return opts, errs.BelowFloor(spec.String())
	}
	opts.spec = spec
	return opts, nil
}

type lookupResult struct {
	res   resolve.Result
	found bool
}

// lookupAll finds installers for many builds in parallel, keeping input order.
// The first ls-remote probes each build once; later runs are served from cache.
func lookupAll(ctx context.Context, r resolve.Resolver, builds []version.Build) ([]lookupResult, error) {
	results := make([]lookupResult, len(builds))
	errList := make([]error, len(builds))
	sem := make(chan struct{}, maxParallelLookups)
	var wg sync.WaitGroup
	for i, b := range builds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res, found, err := r.Lookup(ctx, b)
			results[i], errList[i] = lookupResult{res: res, found: found}, err
		}()
	}
	wg.Wait()
	return results, errors.Join(errList...)
}

func (a *App) openArchive(ctx context.Context, refresh bool) (catalog.View, error) {
	client := archive.NewClient()
	cat, warning, err := catalog.Load(ctx, client, a.Home.CacheDir(), refresh, time.Now())
	if err != nil {
		return catalog.View{}, err
	}
	if warning != "" {
		fmt.Fprintln(a.Err, "warning:", warning)
	}
	installers, err := catalog.OpenInstallers(client, a.Home.CacheDir())
	if err != nil {
		return catalog.View{}, err
	}
	return catalog.View{Catalog: cat, Installers: installers}, nil
}

func (a *App) resolver(view catalog.View) resolve.Resolver {
	return resolve.Resolver{Source: view, Platform: a.Platform, HasRosetta: platform.HasRosetta}
}

// saveLookups persists installer lookups. Failing to save only costs speed
// next time, so it is a warning rather than an error.
func (a *App) saveLookups(view catalog.View) {
	if err := view.Installers.Save(); err != nil {
		fmt.Fprintln(a.Err, "warning: could not save lookup cache:", err)
	}
}

// ensureShim points bin/soffice at the running lovm binary; invoked under
// that name, lovm acts as the shim.
func (a *App) ensureShim() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if self, err = filepath.EvalSymlinks(self); err != nil {
		return err
	}
	link := filepath.Join(a.Home.BinDir(), "soffice")
	if current, err := os.Readlink(link); err == nil && current == self {
		return nil
	}
	if err := os.MkdirAll(a.Home.BinDir(), 0o755); err != nil {
		return err
	}
	if err := os.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return os.Symlink(self, link)
}

func (a *App) printPathHint() {
	if slices.Contains(filepath.SplitList(a.Getenv("PATH")), a.Home.BinDir()) {
		return
	}
	fmt.Fprintf(a.Out, "\nAdd lovm's shim to your PATH (e.g. in ~/.zshrc):\n  export PATH=\"%s:$PATH\"\n", a.Home.BinDir())
}
