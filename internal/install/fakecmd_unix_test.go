//go:build unix

package install

import (
	"fmt"
	"os"
	"testing"
)

// writeFakeCommand writes a program at path that prints stdout and stderr and
// exits with code, and returns its path.
func writeFakeCommand(t *testing.T, path, stdout, stderr string, code int) string {
	t.Helper()
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' '%s'\nprintf '%%s\\n' '%s' >&2\nexit %d\n", stdout, stderr, code)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
