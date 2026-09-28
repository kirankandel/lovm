//go:build windows

package install

import (
	"fmt"
	"os"
	"testing"
)

// writeFakeCommand writes a batch file at path+".bat" that prints stdout and
// stderr and exits with code, and returns its path.
func writeFakeCommand(t *testing.T, path, stdout, stderr string, code int) string {
	t.Helper()
	path += ".bat"
	script := fmt.Sprintf("@echo off\r\necho %s\r\necho %s 1>&2\r\nexit /b %d\r\n", stdout, stderr, code)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
