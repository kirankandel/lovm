//go:build windows

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// installShim copies self to binDir\soffice.exe. Symlinks need admin rights
// or Developer Mode on Windows, and a hardlink would keep pointing at the old
// file once `go install` replaces lovm.exe, so a copy it is; every install
// refreshes it.
func installShim(self, binDir string) error {
	dest := filepath.Join(binDir, "soffice.exe")
	want, err := os.ReadFile(self)
	if err != nil {
		return err
	}
	have, err := os.ReadFile(dest)
	if err == nil && bytes.Equal(have, want) {
		return nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	tmp := dest + ".new"
	if err := os.WriteFile(tmp, want, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return errors.Join(
			fmt.Errorf("update %s (close any LibreOffice started through lovm and retry): %w", dest, err),
			os.Remove(tmp),
		)
	}
	return nil
}
