package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/version"
)

func fakeSoffice(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "soffice")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSmokeTest(t *testing.T) {
	b := version.Build{24, 8, 7, 2}
	tests := []struct {
		name     string
		script   string
		wantErr  bool
		wantHint string
	}{
		{"matching version", `echo "LibreOffice 24.8.7.2 abc123"`, false, ""},
		{"wrong version", `echo "LibreOffice 7.6.7.2 abc123"`, true, "unexpected version output"},
		{"crashes", `echo "error while loading shared libraries" >&2; exit 127`, true, "shared libraries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := smokeTest(context.Background(), fakeSoffice(t, tt.script), t.TempDir(), b)
			if !tt.wantErr {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var e *errs.Error
			if !errors.As(err, &e) || e.Code != errs.CodeWontRun {
				t.Fatalf("err = %v, want WontRun", err)
			}
			if !strings.Contains(e.Hint, tt.wantHint) {
				t.Errorf("Hint = %q, want it to mention %q", e.Hint, tt.wantHint)
			}
		})
	}
}
