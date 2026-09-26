//go:build unix

package shim

import (
	"fmt"
	"os"
	"syscall"
)

// execReplace replaces this process, so the caller sees soffice's own PID,
// signals and exit status with no wrapper in between.
func execReplace(path string, argv []string) error {
	if err := syscall.Exec(path, argv, os.Environ()); err != nil {
		return fmt.Errorf("exec %s: %w", path, err)
	}
	return nil
}
