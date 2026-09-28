//go:build unix

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

var (
	printVersionAndPath = []string{"sh", "-c", `echo "$LOVM_VERSION"; echo "$PATH"`}
	exitThree           = []string{"sh", "-c", "exit 3"}
)

// writeFakeCommand writes a program named name in dir that prints output.
func writeFakeCommand(t *testing.T, dir, name, output string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\necho "+output+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
