// Package home knows where lovm keeps things on disk.
package home

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/version"
)

type Home struct{ Root string }

// Default is $LOVM_HOME, or ~/.lovm when it is unset.
func Default() (Home, error) {
	root := os.Getenv("LOVM_HOME")
	if root == "" {
		dir, err := os.UserHomeDir()
		if err != nil {
			return Home{}, fmt.Errorf("find home directory: %w", err)
		}
		root = filepath.Join(dir, ".lovm")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Home{}, err
	}
	return Home{Root: abs}, nil
}

func (h Home) BinDir() string { return filepath.Join(h.Root, "bin") }

func (h Home) CacheDir() string { return filepath.Join(h.Root, "cache") }

func (h Home) DownloadsDir() string { return filepath.Join(h.CacheDir(), "downloads") }

func (h Home) VersionsDir() string { return filepath.Join(h.Root, "versions") }

func (h Home) VersionDir(b version.Build) string { return filepath.Join(h.VersionsDir(), b.String()) }

func (h Home) InstallDir(b version.Build) string { return filepath.Join(h.VersionDir(b), "install") }

func (h Home) ProfileDir(b version.Build) string { return filepath.Join(h.VersionDir(b), "profile") }

func (h Home) DefaultFile() string { return filepath.Join(h.Root, "default") }

// Installed lists installed builds, oldest first. In-progress installs
// (".tmp-*" folders) are not listed.
func (h Home) Installed() ([]version.Build, error) {
	entries, err := os.ReadDir(h.VersionsDir())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var builds []version.Build
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := version.ParseBuild(e.Name())
		if err != nil {
			continue // not a finished install
		}
		builds = append(builds, b)
	}
	slices.SortFunc(builds, version.Build.Compare)
	return builds, nil
}

// SofficePath locates the real soffice executable inside an install directory.
func SofficePath(installDir, goos string) (string, error) {
	switch goos {
	case "darwin":
		path := filepath.Join(installDir, "LibreOffice.app", "Contents", "MacOS", "soffice")
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("soffice not found in %s: %w", installDir, err)
		}
		return path, nil
	case "linux":
		matches, err := filepath.Glob(filepath.Join(installDir, "opt", "libreoffice*", "program", "soffice"))
		if err != nil {
			return "", err
		}
		if len(matches) != 1 {
			return "", fmt.Errorf("expected one soffice under %s, found %d", installDir, len(matches))
		}
		return matches[0], nil
	case "windows":
		return windowsSoffice(installDir)
	default:
		return "", errs.UnsupportedOS(goos)
	}
}

// windowsSoffice finds program\soffice.com, the console launcher: soffice.exe
// is a GUI program, so its output never reaches a terminal and callers could
// not see --version or conversion errors. The folder above program\ comes
// from the MSI and differs between versions, so a shallow search finds it.
func windowsSoffice(installDir string) (string, error) {
	var matches []string
	for _, pattern := range []string{"program", "*/program", "*/*/program"} {
		found, err := filepath.Glob(filepath.Join(installDir, pattern, "soffice.com"))
		if err != nil {
			return "", err
		}
		matches = append(matches, found...)
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("expected one soffice.com under %s, found %d", installDir, len(matches))
	}
	return matches[0], nil
}

// FileURL turns an absolute path into the file:// URL LibreOffice expects,
// percent-encoding characters such as spaces.
func FileURL(path string) string {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p // "C:/x" must become file:///C:/x, not a URL with host "C:"
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}
