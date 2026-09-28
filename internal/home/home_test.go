package home

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/version"
)

func TestDefaultUsesLovmHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOVM_HOME", root)
	h, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if h.Root != root {
		t.Errorf("Root = %q, want %q", h.Root, root)
	}
}

func TestInstalled(t *testing.T) {
	h := Home{Root: t.TempDir()}
	if got, err := h.Installed(); err != nil || len(got) != 0 {
		t.Fatalf("missing versions dir: %v, %v", got, err)
	}
	for _, name := range []string{"24.8.7.2", "7.6.7.2", ".tmp-26.2.5.2", "not-a-version"} {
		if err := os.MkdirAll(filepath.Join(h.VersionsDir(), name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(h.VersionsDir(), "25.2.0.3"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := h.Installed()
	if err != nil {
		t.Fatal(err)
	}
	want := []version.Build{{7, 6, 7, 2}, {24, 8, 7, 2}}
	if !slices.Equal(got, want) {
		t.Errorf("Installed = %v, want %v", got, want)
	}
}

func TestSofficePath(t *testing.T) {
	linuxDir := t.TempDir()
	linuxSoffice := filepath.Join(linuxDir, "opt", "libreoffice24.8", "program", "soffice")
	mustTouch(t, linuxSoffice)
	if got, err := SofficePath(linuxDir, "linux"); err != nil || got != linuxSoffice {
		t.Errorf("linux: %q, %v", got, err)
	}

	macDir := t.TempDir()
	macSoffice := filepath.Join(macDir, "LibreOffice.app", "Contents", "MacOS", "soffice")
	mustTouch(t, macSoffice)
	if got, err := SofficePath(macDir, "darwin"); err != nil || got != macSoffice {
		t.Errorf("darwin: %q, %v", got, err)
	}

	// msiexec /a mirrors the MSI's folder tree, whose top folder name varies by version.
	winDir := t.TempDir()
	winSoffice := filepath.Join(winDir, "LibreOffice", "program", "soffice.com")
	mustTouch(t, winSoffice)
	mustTouch(t, filepath.Join(winDir, "LibreOffice", "program", "soffice.exe"))
	if got, err := SofficePath(winDir, "windows"); err != nil || got != winSoffice {
		t.Errorf("windows: %q, %v", got, err)
	}
	flatDir := t.TempDir()
	flatSoffice := filepath.Join(flatDir, "program", "soffice.com")
	mustTouch(t, flatSoffice)
	if got, err := SofficePath(flatDir, "windows"); err != nil || got != flatSoffice {
		t.Errorf("windows, flat layout: %q, %v", got, err)
	}

	if _, err := SofficePath(t.TempDir(), "linux"); err == nil {
		t.Error("empty install dir should be an error")
	}
	if _, err := SofficePath(t.TempDir(), "windows"); err == nil {
		t.Error("empty windows install dir should be an error")
	}
	var e *errs.Error
	if _, err := SofficePath(linuxDir, "freebsd"); !errors.As(err, &e) || e.Code != errs.CodeUnsupportedOS {
		t.Errorf("freebsd: %v", err)
	}
}

func TestFileURL(t *testing.T) {
	if got := FileURL("/Users/a b/.lovm/profile"); got != "file:///Users/a%20b/.lovm/profile" {
		t.Errorf("FileURL = %q", got)
	}
	// Windows paths reach url.URL as "C:/..." after ToSlash; the drive letter needs a leading slash.
	if got := FileURL("C:/Users/a b/.lovm/profile"); got != "file:///C:/Users/a%20b/.lovm/profile" {
		t.Errorf("FileURL(drive path) = %q", got)
	}
}

func mustTouch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o755); err != nil {
		t.Fatal(err)
	}
}
