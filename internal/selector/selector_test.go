package selector

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/home"
	"github.com/kirankandel/lovm/internal/version"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func wantCode(t *testing.T, err error, code int) {
	t.Helper()
	var e *errs.Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("err = %v, want code %d", err, code)
	}
}

func TestSelectOrder(t *testing.T) {
	h := home.Home{Root: t.TempDir()}
	project := t.TempDir()
	cwd := filepath.Join(project, "a", "b")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, h.DefaultFile(), "7.6\n")
	write(t, filepath.Join(project, RCFile), "25.2\n")
	write(t, filepath.Join(project, "a", RCFile), "24.8\n")

	tests := []struct {
		name       string
		vars       map[string]string
		cwd        string
		wantSpec   string
		wantOrigin string
	}{
		{"env wins", map[string]string{"LOVM_VERSION": "26.2"}, cwd, "26.2", "LOVM_VERSION"},
		{"nearest .lovmrc", nil, cwd, "24.8", filepath.Join(project, "a", RCFile)},
		{"default", nil, t.TempDir(), "7.6", h.DefaultFile()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sel, err := Select(h, env(tt.vars), tt.cwd)
			if err != nil {
				t.Fatal(err)
			}
			if sel.Spec.String() != tt.wantSpec || sel.Origin != tt.wantOrigin {
				t.Errorf("got %s from %s, want %s from %s", sel.Spec, sel.Origin, tt.wantSpec, tt.wantOrigin)
			}
		})
	}
}

func TestSelectNothing(t *testing.T) {
	_, err := Select(home.Home{Root: t.TempDir()}, env(nil), t.TempDir())
	wantCode(t, err, errs.CodeNotSelected)
}

func TestReadSpecFileToleratesEditors(t *testing.T) {
	path := filepath.Join(t.TempDir(), RCFile)
	write(t, path, "\ufeff24.8\r\nignored second line\n")
	spec, err := ReadSpecFile(path)
	if err != nil || spec.String() != "24.8" {
		t.Fatalf("spec = %v, err = %v", spec, err)
	}
}

func TestReadSpecFileNamesPathOnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), RCFile)
	write(t, path, "banana\n")
	if _, err := ReadSpecFile(path); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("err = %v, want it to name %s", err, path)
	}
}

func TestPickInstalled(t *testing.T) {
	installed := []version.Build{{7, 6, 7, 2}, {24, 8, 4, 2}, {24, 8, 7, 2}}
	tests := map[string]string{"24.8": "24.8.7.2", "24.8.4": "24.8.4.2", "latest": "24.8.7.2", "25.2": ""}
	for spec, want := range tests {
		s, err := version.ParseSpec(spec)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := PickInstalled(s, installed)
		if (want == "" && ok) || (want != "" && got.String() != want) {
			t.Errorf("PickInstalled(%s) = %v, %v; want %q", spec, got, ok, want)
		}
	}
}

func TestActiveNotInstalled(t *testing.T) {
	h := home.Home{Root: t.TempDir()}
	cwd := t.TempDir()
	write(t, filepath.Join(cwd, RCFile), "24.8\n")
	_, _, err := Active(h, env(nil), cwd)
	wantCode(t, err, errs.CodeNotInstalled)
	if !strings.Contains(err.Error(), filepath.Join(cwd, RCFile)) {
		t.Errorf("err = %v, want it to name the .lovmrc", err)
	}
}
