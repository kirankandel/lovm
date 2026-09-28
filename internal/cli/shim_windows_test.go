//go:build windows

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInstallShimCopiesAndRefreshes(t *testing.T) {
	src := filepath.Join(t.TempDir(), "lovm.exe")
	if err := os.WriteFile(src, []byte("lovm v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(t.TempDir(), "bin")
	shim := filepath.Join(binDir, "soffice.exe")
	if err := installShim(src, binDir); err != nil {
		t.Fatal(err)
	}
	assertSameFile(t, shim, src)

	// Unchanged lovm: the shim is left alone rather than rewritten.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(shim, old, old); err != nil {
		t.Fatal(err)
	}
	if err := installShim(src, binDir); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(shim); err != nil || !info.ModTime().Equal(old) {
		t.Errorf("shim was rewritten although lovm did not change (%v)", err)
	}

	// Updated lovm: the shim follows.
	if err := os.WriteFile(src, []byte("lovm v2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installShim(src, binDir); err != nil {
		t.Fatal(err)
	}
	assertSameFile(t, shim, src)
}

func assertSameFile(t *testing.T, got, want string) {
	t.Helper()
	g, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	w, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(g, w) {
		t.Errorf("%s = %q, want %q", got, g, w)
	}
}
