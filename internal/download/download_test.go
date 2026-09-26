package download

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func serve(t *testing.T, h http.HandlerFunc) (*httptest.Server, *int) {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestFetchWritesFileAndReportsProgress(t *testing.T) {
	srv, _ := serve(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("installer-bytes")) })
	dest := filepath.Join(t.TempDir(), "downloads", "lo.tar.gz")
	var progress bytes.Buffer
	if err := Fetch(context.Background(), srv.Client(), srv.URL, dest, &progress); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(dest); string(data) != "installer-bytes" {
		t.Errorf("content = %q", data)
	}
	if _, err := os.Stat(dest + ".part"); !errors.Is(err, fs.ErrNotExist) {
		t.Error(".part file left behind")
	}
	if !strings.Contains(progress.String(), "100%") {
		t.Errorf("progress = %q", progress.String())
	}
}

func TestFetchSkipsExistingFile(t *testing.T) {
	srv, calls := serve(t, func(w http.ResponseWriter, r *http.Request) {})
	dest := filepath.Join(t.TempDir(), "lo.tar.gz")
	if err := os.WriteFile(dest, []byte("cached"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Fetch(context.Background(), srv.Client(), srv.URL, dest, nil); err != nil {
		t.Fatal(err)
	}
	if *calls != 0 {
		t.Errorf("server called %d times for a cached file", *calls)
	}
}

func TestFetchReplacesStalePart(t *testing.T) {
	srv, _ := serve(t, func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("new")) })
	dest := filepath.Join(t.TempDir(), "lo.tar.gz")
	if err := os.WriteFile(dest+".part", []byte("leftover from an interrupted download"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Fetch(context.Background(), srv.Client(), srv.URL, dest, nil); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(dest); string(data) != "new" {
		t.Errorf("content = %q", data)
	}
}

func TestFetchFailuresLeaveNothing(t *testing.T) {
	tests := map[string]http.HandlerFunc{
		"not found": func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
		"truncated": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "100")
			w.Write([]byte("short"))
		},
	}
	for name, h := range tests {
		t.Run(name, func(t *testing.T) {
			srv, _ := serve(t, h)
			dest := filepath.Join(t.TempDir(), "lo.tar.gz")
			if err := Fetch(context.Background(), srv.Client(), srv.URL, dest, nil); err == nil {
				t.Fatal("expected error")
			}
			for _, p := range []string{dest, dest + ".part"} {
				if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("%s exists after failure", p)
				}
			}
		})
	}
}
