package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/home"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/selector"
	"github.com/kirankandel/lovm/internal/version"
)

func newTestApp(t *testing.T) (*App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	vars := map[string]string{"PATH": os.Getenv("PATH")}
	app := &App{
		Home:     home.Home{Root: t.TempDir()},
		In:       strings.NewReader(""),
		Out:      out,
		Err:      errOut,
		Getenv:   func(k string) string { return vars[k] },
		Cwd:      t.TempDir(),
		Platform: platform.Platform{OS: "linux", Arch: "amd64"},
	}
	return app, out, errOut
}

// fakeInstall creates version folders with a Linux-style soffice.
func fakeInstall(t *testing.T, h home.Home, builds ...string) {
	t.Helper()
	for _, s := range builds {
		b, err := version.ParseBuild(s)
		if err != nil {
			t.Fatal(err)
		}
		program := filepath.Join(h.InstallDir(b), "opt", "libreoffice"+b.Branch(), "program")
		if err := os.MkdirAll(program, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(program, "soffice"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func run(app *App, args ...string) error { return app.Run(context.Background(), args) }

func wantCode(t *testing.T, err error, code int) {
	t.Helper()
	var e *errs.Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("err = %v, want code %d", err, code)
	}
}

func TestUnknownCommandAndOS(t *testing.T) {
	app, _, _ := newTestApp(t)
	wantCode(t, run(app, "frobnicate"), errs.CodeUsage)
	wantCode(t, run(app), errs.CodeUsage)

	app.Platform = platform.Platform{OS: "windows", Arch: "amd64"}
	wantCode(t, run(app, "ls"), errs.CodeUnsupportedOS)
}

func TestUseWritesLovmrc(t *testing.T) {
	app, _, errOut := newTestApp(t)
	if err := run(app, "use", "24.8"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(app.Cwd, selector.RCFile))
	if err != nil || string(data) != "24.8\n" {
		t.Fatalf(".lovmrc = %q, %v", data, err)
	}
	if !strings.Contains(errOut.String(), "not installed yet") {
		t.Errorf("stderr = %q, want a not-installed note", errOut.String())
	}
	wantCode(t, run(app, "use", "5.4"), errs.CodeBelowFloor)
}

func TestChannelSpecsInLocalCommands(t *testing.T) {
	app, out, errOut := newTestApp(t)
	fakeInstall(t, app.Home, "26.2.6.3", "26.8.0.3")

	// Pinning a channel works before any version list is cached; it just can't say which branch yet.
	if err := run(app, "use", "still"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(app.Cwd, selector.RCFile)); string(data) != "still\n" {
		t.Errorf(".lovmrc = %q", data)
	}
	if !strings.Contains(errOut.String(), "lovm ls-remote") {
		t.Errorf("stderr = %q, want a hint to fetch the version list", errOut.String())
	}

	catalogJSON := `{"fetched_at":"2026-09-26T00:00:00Z","builds":[],"releases":{},"fresh_branch":"26.8","still_branch":"26.2"}`
	if err := os.MkdirAll(app.Home.CacheDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app.Home.CacheDir(), "catalog.json"), []byte(catalogJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run(app, "which", "fresh"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), filepath.Join("versions", "26.8.0.3")) {
		t.Errorf("which fresh = %q, want the 26.8 install", out.String())
	}
	out.Reset()
	if err := run(app, "current"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "26.2.6.3 (still from ") {
		t.Errorf("current = %q", out.String())
	}
}

func TestDefaultAndCurrent(t *testing.T) {
	app, out, _ := newTestApp(t)
	fakeInstall(t, app.Home, "24.8.4.2", "24.8.7.2")
	if err := run(app, "default", "24.8"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run(app, "current"); err != nil {
		t.Fatal(err)
	}
	want := "24.8.7.2 (24.8 from " + app.Home.DefaultFile() + ")\n"
	if out.String() != want {
		t.Errorf("current = %q, want %q", out.String(), want)
	}
}

func TestLsMarksActive(t *testing.T) {
	app, out, _ := newTestApp(t)
	fakeInstall(t, app.Home, "7.6.7.2", "24.8.7.2")
	rc := filepath.Join(app.Cwd, selector.RCFile)
	if err := os.WriteFile(rc, []byte("7.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(app, "ls"); err != nil {
		t.Fatal(err)
	}
	want := "* 7.6.7.2  (7.6 from " + rc + ")\n  24.8.7.2\n"
	if out.String() != want {
		t.Errorf("ls = %q, want %q", out.String(), want)
	}
}

func TestUninstall(t *testing.T) {
	app, _, _ := newTestApp(t)
	fakeInstall(t, app.Home, "24.8.4.2", "24.8.7.2")
	wantCode(t, run(app, "uninstall", "24.8"), errs.CodeUsage) // ambiguous
	if err := run(app, "uninstall", "24.8.4"); err != nil {
		t.Fatal(err)
	}
	installed, _ := app.Home.Installed()
	if len(installed) != 1 || installed[0].String() != "24.8.7.2" {
		t.Errorf("installed = %v", installed)
	}
	wantCode(t, run(app, "uninstall", "25.2"), errs.CodeNotInstalled)
}

func TestWhich(t *testing.T) {
	app, out, _ := newTestApp(t)
	fakeInstall(t, app.Home, "24.8.7.2")
	if err := run(app, "which", "24.8"); err != nil {
		t.Fatal(err)
	}
	b := version.Build{24, 8, 7, 2}
	want := filepath.Join(app.Home.InstallDir(b), "opt", "libreoffice24.8", "program", "soffice") + "\n"
	if out.String() != want {
		t.Errorf("which = %q, want %q", out.String(), want)
	}
}

func TestExec(t *testing.T) {
	app, out, _ := newTestApp(t)
	fakeInstall(t, app.Home, "24.8.7.2")
	if err := run(app, "exec", "24.8", "--", "sh", "-c", `echo "$LOVM_VERSION"; echo "$PATH"`); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out.String(), "\n")
	if lines[0] != "24.8.7.2" || !strings.HasPrefix(lines[1], app.Home.BinDir()+string(os.PathListSeparator)) {
		t.Errorf("exec output = %q", out.String())
	}
	wantCode(t, run(app, "exec", "24.8", "--", "sh", "-c", "exit 3"), 3)
	wantCode(t, run(app, "exec", "24.8", "sh"), errs.CodeUsage)
}

// A system LibreOffice on lovm's own PATH must not win over the shim: exec
// has to resolve the command with the PATH it gives the child.
func TestExecPrefersShimOverSystemSoffice(t *testing.T) {
	app, out, _ := newTestApp(t)
	fakeInstall(t, app.Home, "24.8.7.2")
	writeScript(t, filepath.Join(app.Home.BinDir(), "soffice"), "echo shim")
	decoyDir := t.TempDir()
	writeScript(t, filepath.Join(decoyDir, "soffice"), "echo system")
	t.Setenv("PATH", decoyDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	app.Getenv = func(k string) string { return os.Getenv(k) }

	if err := run(app, "exec", "24.8", "--", "soffice"); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "shim" {
		t.Errorf("exec ran %q, want the shim", got)
	}
}

func writeScript(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
