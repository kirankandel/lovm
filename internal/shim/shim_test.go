package shim

import (
	"slices"
	"testing"
)

func TestWithProfileAddsProfile(t *testing.T) {
	got := WithProfile([]string{"--headless", "--convert-to", "pdf", "a.docx"}, "/h/.lovm/versions/24.8.7.2/profile")
	want := []string{
		"-env:UserInstallation=file:///h/.lovm/versions/24.8.7.2/profile",
		"--headless", "--convert-to", "pdf", "a.docx",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestWithProfileKeepsCallerProfile(t *testing.T) {
	args := []string{"-env:UserInstallation=file:///tmp/aop-profile", "--headless"}
	if got := WithProfile(args, "/h/profile"); !slices.Equal(got, args) {
		t.Errorf("caller's profile was changed: %v", got)
	}
}

func TestWithProfileEncodesSpaces(t *testing.T) {
	got := WithProfile(nil, "/Users/a b/.lovm/versions/24.8.7.2/profile")
	if got[0] != "-env:UserInstallation=file:///Users/a%20b/.lovm/versions/24.8.7.2/profile" {
		t.Errorf("got %q", got[0])
	}
}
