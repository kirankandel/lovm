//go:build unix

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureShimIsIdempotent(t *testing.T) {
	app, _, _ := newTestApp(t)
	for range 2 {
		if err := app.ensureShim(); err != nil {
			t.Fatal(err)
		}
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(app.Home.BinDir(), "soffice"))
	if err != nil || target != self {
		t.Errorf("shim -> %q (%v), want %q", target, err, self)
	}
}
