//go:build windows

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

var (
	printVersionAndPath = []string{"cmd", "/c", "echo %LOVM_VERSION%&echo %PATH%"}
	exitThree           = []string{"cmd", "/c", "exit 3"}
)

// writeFakeCommand writes name.bat in dir that prints output; PATHEXT lets
// it run as plain name.
func writeFakeCommand(t *testing.T, dir, name, output string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".bat"), []byte("@echo off\r\necho "+output+"\r\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
