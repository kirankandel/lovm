// Package cli implements lovm's subcommands.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/home"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/version"
)

const usage = `Usage: lovm <command> [arguments]

Commands:
  ls-remote [prefix] [--all] [--refresh]   list versions available for this platform
  install <version>                        download and install a version
  uninstall <version>                      remove an installed version
  ls                                       list installed versions
  use <version>                            pin a version for this directory (.lovmrc)
  default <version>                        set the global default version
  current                                  show the active version and why
  which [version]                          print the path of the real soffice
  exec <version> -- <command> [args...]    run a command with that version active
  cache clear                              delete cached downloads and version lists

Versions: latest, 24.8, 24.8.4 or 24.8.4.2
`

// App holds everything a command touches, so tests can point it at temp dirs.
type App struct {
	Home     home.Home
	In       io.Reader
	Out, Err io.Writer
	Getenv   func(string) string
	Cwd      string
	Platform platform.Platform
}

// Run builds an App for the real environment and runs one command.
func Run(ctx context.Context, args []string) error {
	h, err := home.Default()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	app := &App{Home: h, In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Getenv: os.Getenv, Cwd: cwd, Platform: platform.Current()}
	return app.Run(ctx, args)
}

func (a *App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errs.Usage("no command given")
	}
	cmd, rest := args[0], args[1:]
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		fmt.Fprint(a.Out, usage)
		return nil
	}
	if a.Platform.OS != "linux" && a.Platform.OS != "darwin" {
		return errs.UnsupportedOS(a.Platform.OS)
	}
	switch cmd {
	case "uninstall":
		return a.uninstall(rest)
	case "ls":
		return a.ls(rest)
	case "use":
		return a.use(rest)
	case "default":
		return a.setDefault(rest)
	case "current":
		return a.current(rest)
	case "which":
		return a.which(rest)
	case "exec":
		return a.exec(ctx, rest)
	case "cache":
		return a.cache(rest)
	default:
		return errs.Usage(fmt.Sprintf("unknown command %q", cmd))
	}
}

// oneSpec parses the single version argument most commands take.
func oneSpec(cmd string, args []string) (version.Spec, error) {
	if len(args) != 1 {
		return version.Spec{}, errs.Usage(fmt.Sprintf("usage: lovm %s <version>", cmd))
	}
	spec, err := version.ParseSpec(args[0])
	if err != nil {
		return version.Spec{}, err
	}
	if spec.BelowFloor() {
		return version.Spec{}, errs.BelowFloor(spec.String())
	}
	return spec, nil
}
