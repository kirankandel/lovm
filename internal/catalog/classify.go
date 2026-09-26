package catalog

import "github.com/kirankandel/lovm/internal/version"

// classifyReleases decides which builds are finished releases rather than
// release candidates. For each x.y.z only its highest build can be a release,
// and it is one if any of these hold:
//  1. x.y.z is listed in /stable/
//  2. a newer x.y.z' exists in the same x.y branch
//  3. the x.y branch is older than every branch in /stable/ (end-of-life)
//
// The returned map holds only releases (all values true).
func classifyReleases(builds []version.Build, stable stableInfo) map[version.Build]bool {
	highestPerTriple := map[string]version.Build{}
	newestPerBranch := map[string]version.Build{}
	for _, b := range builds {
		keepHighest(highestPerTriple, b.Triple(), b)
		keepHighest(newestPerBranch, b.Branch(), b)
	}
	releases := map[version.Build]bool{}
	for triple, b := range highestPerTriple {
		superseded := newestPerBranch[b.Branch()].Triple() != triple
		endOfLife := version.Build{b[0], b[1], 0, 0}.Compare(stable.oldestBranch) < 0
		if stable.triples[triple] || superseded || endOfLife {
			releases[b] = true
		}
	}
	return releases
}

func keepHighest(m map[string]version.Build, key string, b version.Build) {
	if cur, ok := m[key]; !ok || b.Compare(cur) > 0 {
		m[key] = b
	}
}
