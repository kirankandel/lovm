package catalog

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kirankandel/lovm/internal/version"
)

// stableInfo summarises the /stable/ listing: which x.y.z versions are
// currently released and which branches are still maintained.
type stableInfo struct {
	triples      map[string]bool // e.g. "26.8.0"
	oldestBranch version.Build   // e.g. 25.8.0.0
	branches     []version.Build // distinct branches, newest first, e.g. 26.8.0.0, 26.2.0.0
}

func newStableInfo(triples []string) (stableInfo, error) {
	info := stableInfo{triples: map[string]bool{}}
	for _, t := range triples {
		b, err := version.ParseBuild(t + ".0")
		if err != nil {
			return stableInfo{}, fmt.Errorf("unexpected entry %q in stable listing: %w", t, err)
		}
		branch := version.Build{b[0], b[1], 0, 0}
		if !slices.Contains(info.branches, branch) {
			info.branches = append(info.branches, branch)
		}
		info.triples[t] = true
	}
	if len(info.triples) == 0 {
		return stableInfo{}, errors.New("stable listing is empty")
	}
	slices.SortFunc(info.branches, func(a, b version.Build) int { return b.Compare(a) })
	info.oldestBranch = info.branches[len(info.branches)-1]
	return info, nil
}

// channels names TDF's "fresh" (newest) and "still" (previous) branches. They
// are the two newest branches in /stable/, matching the two versions offered
// on libreoffice.org; older branches can linger there after end of life.
func (s stableInfo) channels() (fresh, still string) {
	fresh = s.branches[0].Branch()
	if len(s.branches) > 1 {
		still = s.branches[1].Branch()
	}
	return fresh, still
}
