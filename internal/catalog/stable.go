package catalog

import (
	"errors"
	"fmt"

	"github.com/kirankandel/lovm/internal/version"
)

// stableInfo summarises the /stable/ listing: which x.y.z versions are
// currently released and the oldest branch still maintained.
type stableInfo struct {
	triples      map[string]bool // e.g. "26.8.0"
	oldestBranch version.Build   // e.g. 25.8.0.0
}

func newStableInfo(triples []string) (stableInfo, error) {
	info := stableInfo{triples: map[string]bool{}}
	for _, t := range triples {
		b, err := version.ParseBuild(t + ".0")
		if err != nil {
			return stableInfo{}, fmt.Errorf("unexpected entry %q in stable listing: %w", t, err)
		}
		branch := version.Build{b[0], b[1], 0, 0}
		if len(info.triples) == 0 || branch.Compare(info.oldestBranch) < 0 {
			info.oldestBranch = branch
		}
		info.triples[t] = true
	}
	if len(info.triples) == 0 {
		return stableInfo{}, errors.New("stable listing is empty")
	}
	return info, nil
}
