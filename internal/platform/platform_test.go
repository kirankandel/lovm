package platform

import "testing"

func TestString(t *testing.T) {
	tests := map[Platform]string{
		{OS: "darwin", Arch: "arm64"}:  "macos/arm64",
		{OS: "linux", Arch: "amd64"}:   "linux/amd64",
		{OS: "windows", Arch: "amd64"}: "windows/amd64",
	}
	for p, want := range tests {
		if got := p.String(); got != want {
			t.Errorf("%#v.String() = %q, want %q", p, got, want)
		}
	}
}

func TestArchiveDirs(t *testing.T) {
	tests := []struct {
		p               Platform
		wantOS, wantArc string
		wantOK          bool
	}{
		{Platform{OS: "linux", Arch: "amd64"}, "deb", "x86_64", true},
		{Platform{OS: "linux", Arch: "arm64"}, "deb", "aarch64", true},
		{Platform{OS: "darwin", Arch: "arm64"}, "mac", "aarch64", true},
		{Platform{OS: "windows", Arch: "amd64"}, "win", "x86_64", true},
		{Platform{OS: "linux", Arch: "386"}, "", "", false},
		{Platform{OS: "freebsd", Arch: "amd64"}, "", "", false},
	}
	for _, tt := range tests {
		osDir, archDir, ok := tt.p.ArchiveDirs()
		if osDir != tt.wantOS || archDir != tt.wantArc || ok != tt.wantOK {
			t.Errorf("%v.ArchiveDirs() = %q, %q, %v", tt.p, osDir, archDir, ok)
		}
	}
}

func TestAllPlatformsHaveArchiveDirs(t *testing.T) {
	seen := map[Platform]bool{}
	for _, p := range All() {
		if _, _, ok := p.ArchiveDirs(); !ok {
			t.Errorf("%v has no archive folders", p)
		}
		if seen[p] {
			t.Errorf("%v listed twice", p)
		}
		seen[p] = true
	}
}
