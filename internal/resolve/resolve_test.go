package resolve

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kirankandel/lovm/internal/archive"
	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/version"
)

type fakeSource struct {
	builds     []version.Build
	releases   map[version.Build]bool
	installers map[string]bool // "<build> <platform>"
}

func (f fakeSource) AllBuilds() []version.Build     { return f.builds }
func (f fakeSource) IsRelease(b version.Build) bool { return f.releases[b] }
func (f fakeSource) FindInstaller(_ context.Context, b version.Build, p platform.Platform) (archive.Installer, bool, error) {
	if !f.installers[b.String()+" "+p.String()] {
		return archive.Installer{}, false, nil
	}
	return archive.Installer{URL: "https://example.test/" + b.String(), FileName: b.String() + "-" + p.Arch}, true, nil
}

var (
	linux    = platform.Platform{OS: "linux", Arch: "amd64"}
	linuxARM = platform.Platform{OS: "linux", Arch: "arm64"}
	macARM   = platform.Platform{OS: "darwin", Arch: "arm64"}
	desktops = []string{"linux/amd64", "macos/amd64", "windows/amd64"}
	modern   = []string{"linux/amd64", "macos/amd64", "macos/arm64", "windows/amd64"}
)

func testSource(t *testing.T) fakeSource {
	t.Helper()
	rows := []struct {
		build     string
		release   bool
		platforms []string
	}{
		{"6.4.7.2", true, desktops},
		{"7.1.8.1", true, desktops}, // no Apple Silicon build yet
		{"24.8.7.1", false, modern},
		{"24.8.7.2", true, modern},
		{"25.2.0.3", true, modern},
		{"26.2.0.3", true, []string{"linux/amd64", "linux/arm64", "macos/arm64"}},
		{"26.2.5.2", true, []string{"linux/amd64", "linux/arm64", "macos/arm64"}},
		{"26.2.6.2", true, []string{"linux/amd64", "macos/arm64"}}, // arm64 build missing
		{"26.8.0.3", true, modern},
		{"26.8.1.1", false, modern}, // RC
	}
	src := fakeSource{releases: map[version.Build]bool{}, installers: map[string]bool{}}
	for _, r := range rows {
		b, err := version.ParseBuild(r.build)
		if err != nil {
			t.Fatal(err)
		}
		src.builds = append(src.builds, b)
		src.releases[b] = r.release
		for _, p := range r.platforms {
			src.installers[r.build+" "+p] = true
		}
	}
	return src
}

func resolveSpec(t *testing.T, p platform.Platform, rosetta bool, spec string) (Result, error) {
	t.Helper()
	s, err := version.ParseSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	r := Resolver{Source: testSource(t), Platform: p, HasRosetta: func() bool { return rosetta }}
	return r.Resolve(context.Background(), s)
}

func wantCode(t *testing.T, err error, code int) *errs.Error {
	t.Helper()
	var e *errs.Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("err = %v, want code %d", err, code)
	}
	return e
}

func TestResolveFindsBuild(t *testing.T) {
	tests := []struct {
		name      string
		p         platform.Platform
		spec      string
		wantBuild string
	}{
		{"latest skips RC", linux, "latest", "26.8.0.3"},
		{"branch picks newest release", linux, "24.8", "24.8.7.2"},
		{"exact RC is allowed", linux, "26.8.1.1", "26.8.1.1"},
		{"platform filter skips build without arm64", linuxARM, "26.2", "26.2.5.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := resolveSpec(t, tt.p, true, tt.spec)
			if err != nil {
				t.Fatal(err)
			}
			if res.Build.String() != tt.wantBuild || res.Rosetta {
				t.Errorf("got %v (rosetta=%v), want %s", res.Build, res.Rosetta, tt.wantBuild)
			}
		})
	}
}

func TestResolveOnlyRC(t *testing.T) {
	_, err := resolveSpec(t, linux, true, "26.8.1")
	e := wantCode(t, err, errs.CodeOnlyRC)
	if !strings.Contains(e.Hint, "lovm install 26.8.1.1") {
		t.Errorf("Hint = %q", e.Hint)
	}
}

func TestResolveNoPlatformBuild(t *testing.T) {
	for _, spec := range []string{"24.8.7.2", "24.8"} {
		_, err := resolveSpec(t, linuxARM, true, spec)
		e := wantCode(t, err, errs.CodeNoPlatformBuild)
		wantHint := "Available for: linux/amd64, macos/amd64, macos/arm64, windows/amd64\n" +
			"First version with linux/arm64: 26.2.0"
		if e.Hint != wantHint {
			t.Errorf("%s: Hint = %q\nwant %q", spec, e.Hint, wantHint)
		}
	}
}

func TestResolveRosetta(t *testing.T) {
	res, err := resolveSpec(t, macARM, true, "7.1")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Rosetta || res.Platform != (platform.Platform{OS: "darwin", Arch: "amd64"}) {
		t.Errorf("got %+v, want the x86_64 build under Rosetta", res)
	}

	_, err = resolveSpec(t, macARM, false, "7.1")
	wantCode(t, err, errs.CodeRosettaMissing)
}

func TestResolveBelowFloor(t *testing.T) {
	_, err := resolveSpec(t, linux, true, "5.4")
	wantCode(t, err, errs.CodeBelowFloor)
}

func TestResolveNotFoundSuggestsNeighbours(t *testing.T) {
	_, err := resolveSpec(t, linux, true, "24.9")
	e := wantCode(t, err, errs.CodeNotFound)
	if !strings.HasPrefix(e.Hint, "Closest: 24.8.7, 25.2.0") {
		t.Errorf("Hint = %q", e.Hint)
	}
}
