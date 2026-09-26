// Package platform names the OS/architecture pairs LibreOffice publishes builds for.
package platform

import "runtime"

// Platform uses Go's GOOS/GOARCH values: OS is "linux", "darwin" or "windows";
// Arch is "amd64" or "arm64".
type Platform struct {
	OS   string
	Arch string
}

// Current is the platform lovm is running on.
func Current() Platform { return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH} }

// All returns every platform the LibreOffice archive publishes builds for.
func All() []Platform {
	return []Platform{
		{"linux", "amd64"}, {"linux", "arm64"},
		{"darwin", "amd64"}, {"darwin", "arm64"},
		{"windows", "amd64"}, {"windows", "arm64"},
	}
}

// String renders the platform the way users talk about it, e.g. "macos/arm64".
func (p Platform) String() string {
	name := p.OS
	if name == "darwin" {
		name = "macos"
	}
	return name + "/" + p.Arch
}

var (
	archiveOSDirs   = map[string]string{"linux": "deb", "darwin": "mac", "windows": "win"}
	archiveArchDirs = map[string]string{"amd64": "x86_64", "arm64": "aarch64"}
)

// ArchiveDirs returns the folder names the archive uses for p, e.g. ("deb",
// "x86_64") for linux/amd64. ok is false for platforms the archive lacks.
func (p Platform) ArchiveDirs() (osDir, archDir string, ok bool) {
	osDir, ok = archiveOSDirs[p.OS]
	if !ok {
		return "", "", false
	}
	archDir, ok = archiveArchDirs[p.Arch]
	if !ok {
		return "", "", false
	}
	return osDir, archDir, true
}
