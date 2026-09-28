//go:build windows

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookPathInUsesPathExt(t *testing.T) {
	t.Setenv("PATHEXT", ".COM;.EXE;.BAT")
	dir := t.TempDir()
	for _, name := range []string{"soffice.exe", "tool.bat", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tests := map[string]string{
		"soffice":     filepath.Join(dir, "soffice.exe"),
		"SOFFICE.EXE": filepath.Join(dir, "SOFFICE.EXE"),
		"tool":        filepath.Join(dir, "tool.bat"),
	}
	for name, want := range tests {
		got, err := lookPathIn(name, dir)
		if err != nil || got != want {
			t.Errorf("lookPathIn(%q) = %q, %v; want %q", name, got, err, want)
		}
	}
	if _, err := lookPathIn("notes", dir); err == nil {
		t.Error("found notes.txt, which is not a program")
	}
}
