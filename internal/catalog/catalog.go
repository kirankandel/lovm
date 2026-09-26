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
	"github.com/kirankandel/lovm/internal/errs"
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

// Catalog lists every supported build in the archive, which are releases, and
// which branches TDF currently offers as "fresh" and "still".
type Catalog struct {
	FetchedAt   time.Time              `json:"fetched_at"`
	Builds      []version.Build        `json:"builds"` // oldest first
	Releases    map[version.Build]bool `json:"releases"`
	FreshBranch string                 `json:"fresh_branch"` // e.g. "26.8"
	StillBranch string                 `json:"still_branch"` // e.g. "26.2"; empty if only one branch is maintained
}

// Channel returns "fresh" or "still" when b is the newest release of the
// branch TDF offers under that name, and "" for every other build.
func (c *Catalog) Channel(b version.Build) string {
	var channel string
	switch b.Branch() {
	case c.FreshBranch:
		channel = "fresh"
	case c.StillBranch:
		channel = "still"
	default:
		return ""
	}
	if !c.Releases[b] {
		return ""
	}
	for _, other := range c.Builds {
		if other.Branch() == b.Branch() && c.Releases[other] && other.Compare(b) > 0 {
			return ""
		}
	}
	return channel
}

// Expand turns a "fresh"/"still" spec into its branch, e.g. still → 26.2.
// Other specs are returned unchanged.
func (c *Catalog) Expand(spec version.Spec) (version.Spec, error) {
	var branch string
	switch spec.Channel() {
	case "":
		return spec, nil
	case "fresh":
		branch = c.FreshBranch
	case "still":
		branch = c.StillBranch
	}
	if branch == "" {
		return version.Spec{}, errs.ChannelUnknown(spec.Channel())
	}
	return version.ParseSpec(branch)
}

// ReadCached returns the catalog saved by the last Load without touching the
// network, for commands that must work offline such as the shim. It returns
// an fs.ErrNotExist error when nothing has been cached yet.
func ReadCached(cacheDir string) (*Catalog, error) {
	return readCatalog(filepath.Join(cacheDir, "catalog.json"))
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
	// A catalog without FreshBranch was written before channels existed; refetch it.
	if cached != nil && !refresh && cached.FreshBranch != "" && now.Sub(cached.FetchedAt) < catalogTTL {
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
	fresh, still := stable.channels()
	return &Catalog{
		FetchedAt:   now,
		Builds:      builds,
		Releases:    classifyReleases(builds, stable),
		FreshBranch: fresh,
		StillBranch: still,
	}, nil
}
