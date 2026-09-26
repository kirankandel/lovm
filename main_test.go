package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kirankandel/lovm/internal/errs"
)

func TestReport(t *testing.T) {
	var buf bytes.Buffer
	if code := report(&buf, nil); code != 0 || buf.Len() != 0 {
		t.Errorf("nil: code %d, output %q", code, buf.String())
	}

	buf.Reset()
	code := report(&buf, fmt.Errorf("wrapped: %w", errs.NotSelected()))
	if code != errs.CodeNotSelected || !strings.Contains(buf.String(), "lovm default") {
		t.Errorf("lovm error: code %d, output %q", code, buf.String())
	}

	buf.Reset()
	if code := report(&buf, errors.New("boom")); code != 1 || buf.String() != "lovm: boom\n" {
		t.Errorf("plain error: code %d, output %q", code, buf.String())
	}
}

// Test binaries have no release version stamped in, so this pins the fallback.
func TestBuildVersionDefaultsToDev(t *testing.T) {
	if got := buildVersion(); got != "dev" {
		t.Errorf("buildVersion() = %q, want dev", got)
	}
}

func TestInvokedAs(t *testing.T) {
	tests := map[string]string{"/home/u/.lovm/bin/soffice": "soffice", "/x/soffice.exe": "soffice", "/usr/local/bin/lovm": "lovm"}
	for arg0, want := range tests {
		if got := invokedAs(arg0); got != want {
			t.Errorf("invokedAs(%q) = %q, want %q", arg0, got, want)
		}
	}
}
