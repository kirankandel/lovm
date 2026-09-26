package catalog

import (
	"context"

	"github.com/kirankandel/lovm/internal/archive"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/version"
)

// View combines the catalog and the installer cache into what the resolver needs.
type View struct {
	Catalog    *Catalog
	Installers *Installers
}

func (v View) AllBuilds() []version.Build { return v.Catalog.Builds }

func (v View) IsRelease(b version.Build) bool { return v.Catalog.Releases[b] }

// Channel returns "fresh", "still" or "" for b; see Catalog.Channel.
func (v View) Channel(b version.Build) string { return v.Catalog.Channel(b) }

func (v View) FindInstaller(ctx context.Context, b version.Build, p platform.Platform) (archive.Installer, bool, error) {
	return v.Installers.Find(ctx, b, p)
}
