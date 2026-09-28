package install

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/version"
)

func TestSmokeTest(t *testing.T) {
	b := version.Build{24, 8, 7, 2}
	tests := []struct {
		name           string
		stdout, stderr string
		code           int
		wantErr        bool
		wantHint       string
	}{
		{"matching version", "LibreOffice 24.8.7.2 abc123", "", 0, false, ""},
		{"wrong version", "LibreOffice 7.6.7.2 abc123", "", 0, true, "unexpected version output"},
		{"crashes", "", "error while loading shared libraries", 127, true, "shared libraries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			soffice := writeFakeCommand(t, filepath.Join(t.TempDir(), "soffice"), tt.stdout, tt.stderr, tt.code)
			err := smokeTest(context.Background(), soffice, t.TempDir(), b)
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
