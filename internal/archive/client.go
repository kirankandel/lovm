// Package archive reads the LibreOffice download archive's directory listings.
package archive

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/version"
)

const (
	DefaultArchiveURL = "https://downloadarchive.documentfoundation.org/libreoffice/old/"
	DefaultStableURL  = "https://download.documentfoundation.org/libreoffice/stable/"
)

// Installer is the main installer file for one build on one platform.
type Installer struct {
	URL      string `json:"url"`
	FileName string `json:"file"`
}

// Client fetches listings. ArchiveURL and StableURL must end with "/".
type Client struct {
	HTTP       *http.Client
	ArchiveURL string
	StableURL  string
}

func NewClient() *Client {
	return &Client{
		HTTP:       &http.Client{Timeout: 30 * time.Second},
		ArchiveURL: DefaultArchiveURL,
		StableURL:  DefaultStableURL,
	}
}

// installerPatterns match the main installer in each OS folder. They exclude
// language packs, help packs and the SDK, and accept both the older
// "Win_x64" and newer "Win_x86-64" naming.
var installerPatterns = map[string]*regexp.Regexp{
	"deb": regexp.MustCompile(`^LibreOffice_[0-9.]+_Linux_[A-Za-z0-9-]+_deb\.tar\.gz$`),
	"mac": regexp.MustCompile(`^LibreOffice_[0-9.]+_MacOS_[A-Za-z0-9-]+\.dmg$`),
	"win": regexp.MustCompile(`^LibreOffice_[0-9.]+_Win_[A-Za-z0-9-]+\.msi$`),
}

var errNotFound = errors.New("listing not found")

// Builds lists every four-part build folder in the archive, oldest first.
func (c *Client) Builds(ctx context.Context) ([]version.Build, error) {
	entries, err := c.list(ctx, c.ArchiveURL)
	if err != nil {
		return nil, err
	}
	var builds []version.Build
	for _, e := range entries {
		name, isDir := strings.CutSuffix(e, "/")
		if !isDir {
			continue
		}
		b, err := version.ParseBuild(name)
		if err != nil {
			continue // the archive also holds non-build folders; they are not versions
		}
		builds = append(builds, b)
	}
	slices.SortFunc(builds, version.Build.Compare)
	return builds, nil
}

// StableReleases lists the x.y.z versions currently published as releases.
func (c *Client) StableReleases(ctx context.Context) ([]string, error) {
	entries, err := c.list(ctx, c.StableURL)
	if err != nil {
		return nil, err
	}
	var triples []string
	for _, e := range entries {
		if name, isDir := strings.CutSuffix(e, "/"); isDir {
			triples = append(triples, name)
		}
	}
	return triples, nil
}

// FindInstaller looks for build b's main installer for platform p. found is
// false when the archive has no such installer; folders may exist but be empty.
func (c *Client) FindInstaller(ctx context.Context, b version.Build, p platform.Platform) (inst Installer, found bool, err error) {
	osDir, archDir, ok := p.ArchiveDirs()
	if !ok {
		return Installer{}, false, nil
	}
	dirURL := c.ArchiveURL + b.String() + "/" + osDir + "/" + archDir + "/"
	entries, err := c.list(ctx, dirURL)
	if errors.Is(err, errNotFound) {
		return Installer{}, false, nil
	}
	if err != nil {
		return Installer{}, false, err
	}
	for _, e := range entries {
		if installerPatterns[osDir].MatchString(e) {
			return Installer{URL: dirURL + e, FileName: e}, true, nil
		}
	}
	return Installer{}, false, nil
}

func (c *Client) list(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	return parseListing(resp.Body)
}
