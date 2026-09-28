//go:build unix

package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// installShim symlinks binDir/soffice to self, so updating lovm updates the shim.
func installShim(self, binDir string) error {
	link := filepath.Join(binDir, "soffice")
	if current, err := os.Readlink(link); err == nil && current == self {
		return nil
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	if err := os.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return os.Symlink(self, link)
}
