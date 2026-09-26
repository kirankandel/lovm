// Package catalog caches what lovm has learned about the LibreOffice archive.
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/kirankandel/lovm/internal/archive"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/version"
)

const catalogTTL = 24 * time.Hour

// Remote is the part of archive.Client the catalog needs.
type Remote interface {
	Builds(ctx context.Context) ([]version.Build, error)
	StableReleases(ctx context.Context) ([]string, error)
	FindInstaller(ctx context.Context, b version.Build, p platform.Platform) (archive.Installer, bool, error)
}

// Catalog lists every supported build in the archive and which are releases.
type Catalog struct {
	FetchedAt time.Time              `json:"fetched_at"`
	Builds    []version.Build        `json:"builds"` // oldest first
	Releases  map[version.Build]bool `json:"releases"`
}

// Load returns the cached catalog, refetching it when it is older than a day
// or refresh is set. When refetching fails but a cached copy exists, that copy
// is returned with a warning for the user.
func Load(ctx context.Context, remote Remote, cacheDir string, refresh bool, now time.Time) (*Catalog, string, error) {
	path := filepath.Join(cacheDir, "catalog.json")
	cached, err := readCatalog(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, "", err
	}
	if cached != nil && !refresh && now.Sub(cached.FetchedAt) < catalogTTL {
		return cached, "", nil
	}
	fresh, err := fetch(ctx, remote, now)
	if err != nil {
		if cached == nil {
			return nil, "", fmt.Errorf("fetch LibreOffice version list: %w", err)
		}
		warning := fmt.Sprintf("could not refresh the version list (%v); using the copy from %s",
			err, cached.FetchedAt.Format(time.DateOnly))
		return cached, warning, nil
	}
	if err := writeJSONFile(path, fresh); err != nil {
		return nil, "", err
	}
	return fresh, "", nil
}

func readCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("corrupt cache %s (run: lovm cache clear): %w", path, err)
	}
	return &c, nil
}

func fetch(ctx context.Context, remote Remote, now time.Time) (*Catalog, error) {
	all, err := remote.Builds(ctx)
	if err != nil {
		return nil, err
	}
	triples, err := remote.StableReleases(ctx)
	if err != nil {
		return nil, err
	}
	stable, err := newStableInfo(triples)
	if err != nil {
		return nil, err
	}
	var builds []version.Build
	for _, b := range all {
		if b[0] >= version.MinMajor {
			builds = append(builds, b)
		}
	}
	return &Catalog{FetchedAt: now, Builds: builds, Releases: classifyReleases(builds, stable)}, nil
}
