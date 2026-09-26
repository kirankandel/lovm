//go:build windows

package shim

import "github.com/kirankandel/lovm/internal/errs"

func execReplace(path string, argv []string) error {
	return errs.UnsupportedOS("windows")
}
