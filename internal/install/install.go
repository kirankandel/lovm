// Package install runs download → unpack → smoke test → activate.
package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kirankandel/lovm/internal/download"
	"github.com/kirankandel/lovm/internal/home"
	"github.com/kirankandel/lovm/internal/resolve"
	"github.com/kirankandel/lovm/internal/version"
)

// Meta is written to versions/<build>/meta.json.
type Meta struct {
	Build       version.Build `json:"build"`
	Platform    string        `json:"platform"`
	Rosetta     bool          `json:"rosetta"`
	SourceURL   string        `json:"source_url"`
	InstalledAt time.Time     `json:"installed_at"`
}

type Installer struct {
	Home     home.Home
	HTTP     *http.Client
	Progress io.Writer
	// Extract and SmokeTest are fields so tests can replace them; New sets the real ones.
	Extract   func(ctx context.Context, goos, archivePath, dest string) error
	SmokeTest func(ctx context.Context, soffice, profileDir string, b version.Build) error
	Now       func() time.Time
}

func New(h home.Home, progress io.Writer) *Installer {
	return &Installer{
		Home:      h,
		HTTP:      &http.Client{}, // no timeout: installers are ~200 MB; Ctrl-C cancels via ctx
		Progress:  progress,
		Extract:   extractFor,
		SmokeTest: smokeTest,
		Now:       time.Now,
	}
}

// Install makes res available under its version directory. All work happens
// in a temporary directory that is renamed into place only after the smoke
// test passes, so a failed or interrupted install never looks installed.
func (in *Installer) Install(ctx context.Context, res resolve.Result) (err error) {
	archivePath := filepath.Join(in.Home.DownloadsDir(), res.Installer.FileName)
	if err := download.Fetch(ctx, in.HTTP, res.Installer.URL, archivePath, in.Progress); err != nil {
		return err
	}

	tmp := filepath.Join(in.Home.VersionsDir(), ".tmp-"+res.Build.String())
	if err := os.RemoveAll(tmp); err != nil { // leftovers from an interrupted install
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, os.RemoveAll(tmp))
		}
	}()

	installDir := filepath.Join(tmp, "install")
	if err := in.Extract(ctx, res.Platform.OS, archivePath, installDir); err != nil {
		return fmt.Errorf("unpack %s: %w", res.Installer.FileName, err)
	}
	soffice, err := home.SofficePath(installDir, res.Platform.OS)
	if err != nil {
		return err
	}
	profile := filepath.Join(tmp, "profile")
	if err := in.SmokeTest(ctx, soffice, profile, res.Build); err != nil {
		return err
	}
	// The smoke-test profile records absolute paths under tmp; start fresh after the rename.
	if err := os.RemoveAll(profile); err != nil {
		return err
	}
	meta := Meta{
		Build:       res.Build,
		Platform:    res.Platform.String(),
		Rosetta:     res.Rosetta,
		SourceURL:   res.Installer.URL,
		InstalledAt: in.Now(),
	}
	if err := writeMeta(tmp, meta); err != nil {
		return err
	}
	return os.Rename(tmp, in.Home.VersionDir(res.Build))
}

func writeMeta(dir string, m Meta) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "meta.json"), append(data, '\n'), 0o644)
}
