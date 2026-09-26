package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kirankandel/lovm/internal/archive"
	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/resolve"
	"github.com/kirankandel/lovm/internal/version"
)

func TestParseLsRemoteArgs(t *testing.T) {
	opts, err := parseLsRemoteArgs([]string{"24.8", "--all"})
	if err != nil || !opts.all || opts.refresh || opts.spec.String() != "24.8" {
		t.Fatalf("opts = %+v, err = %v", opts, err)
	}
	opts, err = parseLsRemoteArgs(nil)
	if err != nil || opts.spec.String() != "latest" {
		t.Fatalf("no args: opts = %+v, err = %v", opts, err)
	}
	for _, bad := range [][]string{{"--bogus"}, {"24.8", "25.2"}} {
		_, err := parseLsRemoteArgs(bad)
		wantCode(t, err, errs.CodeUsage)
	}
	_, err = parseLsRemoteArgs([]string{"5.4"})
	wantCode(t, err, errs.CodeBelowFloor)
}

type stubSource map[string]bool // "<build> <platform>" → has installer

func (s stubSource) AllBuilds() []version.Build   { return nil }
func (s stubSource) IsRelease(version.Build) bool { return true }
func (s stubSource) FindInstaller(_ context.Context, b version.Build, p platform.Platform) (archive.Installer, bool, error) {
	return archive.Installer{FileName: b.String()}, s[b.String()+" "+p.String()], nil
}

func TestLookupAllKeepsOrder(t *testing.T) {
	src := stubSource{"24.8.7.2 linux/amd64": true, "26.2.5.2 linux/amd64": true}
	r := resolve.Resolver{Source: src, Platform: platform.Platform{OS: "linux", Arch: "amd64"}}
	builds := []version.Build{{24, 8, 7, 2}, {25, 2, 0, 3}, {26, 2, 5, 2}}
	results, err := lookupAll(context.Background(), r, builds)
	if err != nil {
		t.Fatal(err)
	}
	wantFound := []bool{true, false, true}
	for i, res := range results {
		if res.found != wantFound[i] || (res.found && res.res.Build != builds[i]) {
			t.Errorf("results[%d] = %+v, want found=%v for %v", i, res, wantFound[i], builds[i])
		}
	}
}

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
