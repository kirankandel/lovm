package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/kirankandel/lovm/internal/archive"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/version"
)

type probe struct {
	Found     bool              `json:"found"`
	Installer archive.Installer `json:"installer"`
}

// Installers memoises installer lookups on disk. Archive builds never change,
// so answers, including "no build for this platform", are kept forever.
// Find is safe for concurrent use.
type Installers struct {
	remote Remote
	path   string
	mu     sync.Mutex
	known  map[string]probe
}

func OpenInstallers(remote Remote, cacheDir string) (*Installers, error) {
	s := &Installers{remote: remote, path: filepath.Join(cacheDir, "installers.json"), known: map[string]probe{}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &s.known); err != nil {
		return nil, fmt.Errorf("corrupt cache %s (run: lovm cache clear): %w", s.path, err)
	}
	return s, nil
}

func (s *Installers) Find(ctx context.Context, b version.Build, p platform.Platform) (archive.Installer, bool, error) {
	key := b.String() + " " + p.String()
	s.mu.Lock()
	known, ok := s.known[key]
	s.mu.Unlock()
	if ok {
		return known.Installer, known.Found, nil
	}
	inst, found, err := s.remote.FindInstaller(ctx, b, p)
	if err != nil {
		return archive.Installer{}, false, err
	}
	s.mu.Lock()
	s.known[key] = probe{Found: found, Installer: inst}
	s.mu.Unlock()
	return inst, found, nil
}

// Save writes the lookups learned so far to disk.
func (s *Installers) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeJSONFile(s.path, s.known)
}
