package cli

import (
	"context"
	"os"
	"strings"
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

func TestPrintPathHint(t *testing.T) {
	app, out, _ := newTestApp(t)
	setPath := func(dirs ...string) {
		path := strings.Join(dirs, string(os.PathListSeparator))
		app.Getenv = func(k string) string { return map[string]string{"PATH": path}[k] }
	}

	setPath("/usr/bin")
	app.printPathHint()
	if !strings.Contains(out.String(), `export PATH="`+app.Home.BinDir()+`:$PATH"`) {
		t.Errorf("unix hint = %q", out.String())
	}

	app.Platform = platform.Platform{OS: "windows", Arch: "amd64"}
	out.Reset()
	app.printPathHint()
	if !strings.Contains(out.String(), "SetEnvironmentVariable") || !strings.Contains(out.String(), app.Home.BinDir()) {
		t.Errorf("windows hint = %q", out.String())
	}

	// Windows folder names ignore case, so a differently cased entry already counts.
	setPath("/usr/bin", strings.ToUpper(app.Home.BinDir()))
	out.Reset()
	app.printPathHint()
	if out.Len() != 0 {
		t.Errorf("hint printed although the shim is on PATH: %q", out.String())
	}
}
