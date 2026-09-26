package catalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kirankandel/lovm/internal/archive"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/version"
)

type fakeRemote struct {
	builds     []version.Build
	stable     []string
	err        error
	installers map[string]archive.Installer // key: "<build> <platform>"
	fetches    int
	finds      int
}

func (f *fakeRemote) Builds(context.Context) ([]version.Build, error) {
	f.fetches++
	if f.err != nil {
		return nil, f.err
	}
	return f.builds, nil
}

func (f *fakeRemote) StableReleases(context.Context) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.stable, nil
}

func (f *fakeRemote) FindInstaller(_ context.Context, b version.Build, p platform.Platform) (archive.Installer, bool, error) {
	f.finds++
	inst, ok := f.installers[b.String()+" "+p.String()]
	return inst, ok, nil
}

var now = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

func TestLoadFetchesThenUsesCache(t *testing.T) {
	ctx, dir := context.Background(), t.TempDir()
	remote := &fakeRemote{builds: mustBuilds(t, "5.4.7.2", "24.8.7.2"), stable: []string{"24.8.7"}}

	cat, warning, err := Load(ctx, remote, dir, false, now)
	if err != nil || warning != "" {
		t.Fatalf("Load: %v, warning %q", err, warning)
	}
	if !slices.Equal(cat.Builds, mustBuilds(t, "24.8.7.2")) {
		t.Errorf("Builds = %v, want 5.4 dropped by the floor", cat.Builds)
	}
	if !cat.Releases[mustBuild(t, "24.8.7.2")] {
		t.Errorf("Releases = %v", cat.Releases)
	}

	steps := []struct {
		at          time.Time
		refresh     bool
		wantFetches int
	}{
		{now.Add(time.Hour), false, 1},      // fresh cache
		{now.Add(25 * time.Hour), false, 2}, // expired
		{now.Add(25 * time.Hour), true, 3},  // forced
	}
	for _, s := range steps {
		if _, _, err := Load(ctx, remote, dir, s.refresh, s.at); err != nil {
			t.Fatal(err)
		}
		if remote.fetches != s.wantFetches {
			t.Errorf("at %v refresh=%v: fetches = %d, want %d", s.at, s.refresh, remote.fetches, s.wantFetches)
		}
	}
}

func TestLoadFallsBackToStaleCache(t *testing.T) {
	ctx, dir := context.Background(), t.TempDir()
	remote := &fakeRemote{builds: mustBuilds(t, "24.8.7.2"), stable: []string{"24.8.7"}}
	if _, _, err := Load(ctx, remote, dir, false, now); err != nil {
		t.Fatal(err)
	}
	remote.err = errors.New("offline")
	cat, warning, err := Load(ctx, remote, dir, false, now.Add(48*time.Hour))
	if err != nil {
		t.Fatalf("stale cache should be used: %v", err)
	}
	if !strings.Contains(warning, "offline") || len(cat.Builds) != 1 {
		t.Errorf("warning = %q, builds = %v", warning, cat.Builds)
	}
}

func TestLoadWithoutCacheFails(t *testing.T) {
	remote := &fakeRemote{err: errors.New("offline")}
	if _, _, err := Load(context.Background(), remote, t.TempDir(), false, now); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRejectsCorruptCache(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "catalog.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := Load(context.Background(), &fakeRemote{}, dir, false, now)
	if err == nil || !strings.Contains(err.Error(), "lovm cache clear") {
		t.Fatalf("err = %v, want a hint to clear the cache", err)
	}
}

func TestInstallersMemoiseLookups(t *testing.T) {
	ctx, dir := context.Background(), t.TempDir()
	b := mustBuild(t, "24.8.7.2")
	linux := platform.Platform{OS: "linux", Arch: "amd64"}
	linuxARM := platform.Platform{OS: "linux", Arch: "arm64"}
	remote := &fakeRemote{installers: map[string]archive.Installer{
		"24.8.7.2 linux/amd64": {URL: "https://example.test/lo.tar.gz", FileName: "lo.tar.gz"},
	}}

	s, err := OpenInstallers(remote, dir)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, found, err := s.Find(ctx, b, linux); err != nil || !found {
			t.Fatalf("linux: found=%v err=%v", found, err)
		}
		if _, found, err := s.Find(ctx, b, linuxARM); err != nil || found {
			t.Fatalf("arm64: found=%v err=%v", found, err)
		}
	}
	if remote.finds != 2 {
		t.Errorf("finds = %d, want 2 (second round from memory)", remote.finds)
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenInstallers(remote, dir)
	if err != nil {
		t.Fatal(err)
	}
	inst, found, err := reopened.Find(ctx, b, linux)
	if err != nil || !found || inst.FileName != "lo.tar.gz" {
		t.Fatalf("reopened linux: %+v found=%v err=%v", inst, found, err)
	}
	if _, found, _ := reopened.Find(ctx, b, linuxARM); found {
		t.Error("negative result should be cached too")
	}
	if remote.finds != 2 {
		t.Errorf("finds = %d, want 2 (answers from disk)", remote.finds)
	}
}
