//go:build darwin

package extract

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDMGCopiesAppBundle(t *testing.T) {
	src := t.TempDir()
	soffice := filepath.Join(src, "LibreOffice.app", "Contents", "MacOS", "soffice")
	if err := os.MkdirAll(filepath.Dir(soffice), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(soffice, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	dmg := filepath.Join(t.TempDir(), "test.dmg")
	if out, err := exec.Command("hdiutil", "create", "-volname", "lovmtest", "-srcfolder", src, "-format", "UDZO", "-o", dmg).CombinedOutput(); err != nil {
		t.Fatalf("hdiutil create: %v\n%s", err, out)
	}

	dest := t.TempDir()
	if err := DMG(context.Background(), dmg, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "LibreOffice.app", "Contents", "MacOS", "soffice")); err != nil {
		t.Fatalf("soffice not copied: %v", err)
	}
}
