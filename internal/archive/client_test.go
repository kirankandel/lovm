package archive

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/version"
)

// listingHTML renders entries the way Apache autoindex does, including the
// sort, parent and footer links that parseListing must skip.
func listingHTML(entries ...string) string {
	var b strings.Builder
	b.WriteString(`<html><body><table><tr><th><a href="?C=N;O=D">Name</a></th></tr>`)
	b.WriteString(`<tr><td><a href="/libreoffice/old/">Parent Directory</a></td></tr>`)
	for _, e := range entries {
		fmt.Fprintf(&b, `<tr><td><a href="%s">%s</a></td></tr>`, e, e)
	}
	b.WriteString(`</table><address><a href="mailto:hostmaster@example.org">x</a></address></body></html>`)
	return b.String()
}

// fakeArchive serves listings keyed by URL path; unknown paths return 404 and
// paths in status return that status code.
func fakeArchive(t *testing.T, listings map[string][]string, status map[string]int) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if code, ok := status[r.URL.Path]; ok {
			w.WriteHeader(code)
			return
		}
		entries, ok := listings[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, listingHTML(entries...))
	}))
	t.Cleanup(srv.Close)
	return &Client{HTTP: srv.Client(), ArchiveURL: srv.URL + "/old/", StableURL: srv.URL + "/stable/"}
}

func TestBuilds(t *testing.T) {
	c := fakeArchive(t, map[string][]string{
		"/old/": {"24.8.7.2/", "6.0.7.3/", "24.8.0.1/", "README.txt", "24.8/"},
	}, nil)
	got, err := c.Builds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []version.Build{{6, 0, 7, 3}, {24, 8, 0, 1}, {24, 8, 7, 2}}
	if !slices.Equal(got, want) {
		t.Errorf("Builds = %v, want %v", got, want)
	}
}

func TestStableReleases(t *testing.T) {
	c := fakeArchive(t, map[string][]string{"/stable/": {"25.8.7/", "26.8.0/"}}, nil)
	got, err := c.StableReleases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"25.8.7", "26.8.0"}; !slices.Equal(got, want) {
		t.Errorf("StableReleases = %v, want %v", got, want)
	}
}

func TestFindInstaller(t *testing.T) {
	c := fakeArchive(t, map[string][]string{
		"/old/24.8.7.2/deb/x86_64/": {
			"LibreOffice_24.8.7.2_Linux_x86-64_deb_helppack_de.tar.gz",
			"LibreOffice_24.8.7.2_Linux_x86-64_deb.tar.gz.asc",
			"LibreOffice_24.8.7.2_Linux_x86-64_deb.tar.gz",
			"LibreOffice_24.8.7.2_Linux_x86-64_deb_sdk.tar.gz",
		},
		"/old/24.8.7.2/mac/aarch64/": {
			"LibreOffice_24.8.7.2_MacOS_aarch64_langpack_de.dmg",
			"LibreOffice_24.8.7.2_MacOS_aarch64.dmg",
		},
		"/old/6.4.7.2/win/x86_64/": {
			"LibreOffice_6.4.7.2_Win_x64_helppack_de.msi",
			"LibreOffice_6.4.7.2_Win_x64.msi",
		},
		"/old/7.1.8.1/mac/aarch64/": {},
	}, map[string]int{"/old/9.9.9.9/deb/x86_64/": http.StatusInternalServerError})

	linux := platform.Platform{OS: "linux", Arch: "amd64"}
	linuxARM := platform.Platform{OS: "linux", Arch: "arm64"}
	macARM := platform.Platform{OS: "darwin", Arch: "arm64"}
	windows := platform.Platform{OS: "windows", Arch: "amd64"}

	tests := []struct {
		name      string
		build     version.Build
		p         platform.Platform
		wantFile  string
		wantFound bool
		wantErr   bool
	}{
		{"linux main tarball", version.Build{24, 8, 7, 2}, linux, "LibreOffice_24.8.7.2_Linux_x86-64_deb.tar.gz", true, false},
		{"mac skips langpack", version.Build{24, 8, 7, 2}, macARM, "LibreOffice_24.8.7.2_MacOS_aarch64.dmg", true, false},
		{"old windows naming", version.Build{6, 4, 7, 2}, windows, "LibreOffice_6.4.7.2_Win_x64.msi", true, false},
		{"empty folder", version.Build{7, 1, 8, 1}, macARM, "", false, false},
		{"missing folder", version.Build{24, 8, 7, 2}, linuxARM, "", false, false},
		{"server error", version.Build{9, 9, 9, 9}, linux, "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, found, err := c.FindInstaller(context.Background(), tt.build, tt.p)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if found && (inst.FileName != tt.wantFile || !strings.HasSuffix(inst.URL, "/"+tt.wantFile)) {
				t.Errorf("installer = %+v, want file %s", inst, tt.wantFile)
			}
		})
	}
}
