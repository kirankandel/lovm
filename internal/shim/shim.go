// Package shim is what runs when lovm is invoked as "soffice": it finds the
// active version and replaces itself with that version's real soffice.
package shim

import (
	"os"
	"runtime"
	"strings"

	"github.com/kirankandel/lovm/internal/home"
	"github.com/kirankandel/lovm/internal/selector"
)

const profileFlag = "-env:UserInstallation="

// Run execs the active version's soffice. args is os.Args. It only returns
// on failure; there is deliberately no fallback to a system soffice.
func Run(args []string) error {
	h, err := home.Default()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	b, _, err := selector.Active(h, os.Getenv, cwd)
	if err != nil {
		return err
	}
	soffice, err := home.SofficePath(h.InstallDir(b), runtime.GOOS)
	if err != nil {
		return err
	}
	argv := append([]string{soffice}, WithProfile(args[1:], h.ProfileDir(b))...)
	return execReplace(soffice, argv)
}

// WithProfile gives each version its own LibreOffice profile, unless the
// caller chose one. Separate profiles also mean separate single-instance
// locks, so different versions can run at the same time.
func WithProfile(args []string, profileDir string) []string {
	for _, a := range args {
		if strings.HasPrefix(a, profileFlag) {
			return args
		}
	}
	return append([]string{profileFlag + home.FileURL(profileDir)}, args...)
}
