package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kirankandel/lovm/internal/archive"
	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/home"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/resolve"
	"github.com/kirankandel/lovm/internal/version"
)

var build = version.Build{24, 8, 7, 2}

func fakeExtract(_ context.Context, _, _, dest string) error {
	program := filepath.Join(dest, "opt", "libreoffice24.8", "program")
	if err := os.MkdirAll(program, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(program, "soffice"), []byte("#!/bin/sh\n"), 0o755)
}

func newTestInstaller(t *testing.T, smoke func(context.Context, string, string, version.Build) error) (*Installer, resolve.Result) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "archive-bytes")
	}))
	t.Cleanup(srv.Close)
	in := &Installer{
		Home:      home.Home{Root: t.TempDir()},
		HTTP:      srv.Client(),
		Extract:   fakeExtract,
		SmokeTest: smoke,
		Now:       func() time.Time { return time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC) },
	}
	res := resolve.Result{
		Build:     build,
		Installer: archive.Installer{URL: srv.URL + "/lo.tar.gz", FileName: "lo.tar.gz"},
		Platform:  platform.Platform{OS: "linux", Arch: "amd64"},
	}
	return in, res
}

func passingSmoke(_ context.Context, _, profile string, _ version.Build) error {
	return os.MkdirAll(profile, 0o755) // LibreOffice creates the profile on first run
}

func TestInstallSuccess(t *testing.T) {
	in, res := newTestInstaller(t, passingSmoke)
	if err := in.Install(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	h := in.Home
	if _, err := home.SofficePath(h.InstallDir(build), "linux"); err != nil {
		t.Errorf("installed tree: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(h.VersionDir(build), "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Build != build || meta.Platform != "linux/amd64" || meta.SourceURL != res.Installer.URL {
		t.Errorf("meta = %+v", meta)
	}
	if _, err := os.Stat(h.ProfileDir(build)); !errors.Is(err, fs.ErrNotExist) {
		t.Error("smoke-test profile should not be kept")
	}
	entries, err := os.ReadDir(h.VersionsDir())
	if err != nil || len(entries) != 1 || entries[0].Name() != build.String() {
		t.Errorf("versions dir = %v, %v; want only %s", entries, err, build)
	}
}

func TestInstallFailureLeavesNoVersionButKeepsDownload(t *testing.T) {
	in, res := newTestInstaller(t, func(context.Context, string, string, version.Build) error {
		return errs.WontRun(build.String(), "missing libX11")
	})
	err := in.Install(context.Background(), res)
	var e *errs.Error
	if !errors.As(err, &e) || e.Code != errs.CodeWontRun {
		t.Fatalf("err = %v, want WontRun", err)
	}
	h := in.Home
	for _, p := range []string{h.VersionDir(build), filepath.Join(h.VersionsDir(), ".tmp-"+build.String())} {
		if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s should not exist after a failed install", p)
		}
	}
	if _, err := os.Stat(filepath.Join(h.DownloadsDir(), "lo.tar.gz")); err != nil {
		t.Errorf("download should stay cached: %v", err)
	}
}

func TestInstallClearsStaleTempDir(t *testing.T) {
	in, res := newTestInstaller(t, passingSmoke)
	junk := filepath.Join(in.Home.VersionsDir(), ".tmp-"+build.String(), "install", "junk")
	if err := os.MkdirAll(junk, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := in.Install(context.Background(), res); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(in.Home.InstallDir(build), "junk")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("leftovers from an interrupted install ended up in the version")
	}
}
