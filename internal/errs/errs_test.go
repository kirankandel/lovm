package errs

import (
	"strings"
	"testing"
)

func TestNoPlatformBuild(t *testing.T) {
	e := NoPlatformBuild("24.8.7.2", "linux/arm64", []string{"linux/amd64", "macos/arm64"}, "26.2.0")
	if e.Code != CodeNoPlatformBuild {
		t.Errorf("Code = %d", e.Code)
	}
	if e.Error() != "LibreOffice 24.8.7.2 has no build for linux/arm64" {
		t.Errorf("Error() = %q", e.Error())
	}
	wantHint := "Available for: linux/amd64, macos/arm64\nFirst version with linux/arm64: 26.2.0"
	if e.Hint != wantHint {
		t.Errorf("Hint = %q, want %q", e.Hint, wantHint)
	}
}

func TestNoPlatformBuildWithoutSuggestions(t *testing.T) {
	if e := NoPlatformBuild("6.0.7.3", "linux/arm64", nil, ""); e.Hint != "" {
		t.Errorf("Hint = %q, want empty", e.Hint)
	}
}

func TestNotFoundListsClosest(t *testing.T) {
	e := NotFound("24.9", []string{"24.8.7", "25.2.0"})
	if !strings.HasPrefix(e.Hint, "Closest: 24.8.7, 25.2.0\n") {
		t.Errorf("Hint = %q", e.Hint)
	}
}

func TestNotInstalledNamesOrigin(t *testing.T) {
	if got := NotInstalled("24.8", "/p/.lovmrc").Msg; got != "24.8 is pinned by /p/.lovmrc but not installed" {
		t.Errorf("with origin: %q", got)
	}
	if got := NotInstalled("24.8", "").Msg; got != "LibreOffice 24.8 is not installed" {
		t.Errorf("without origin: %q", got)
	}
}
