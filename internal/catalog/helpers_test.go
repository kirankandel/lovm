package catalog

import (
	"testing"

	"github.com/kirankandel/lovm/internal/version"
)

func mustBuild(t *testing.T, s string) version.Build {
	t.Helper()
	b, err := version.ParseBuild(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustBuilds(t *testing.T, ss ...string) []version.Build {
	t.Helper()
	out := make([]version.Build, len(ss))
	for i, s := range ss {
		out[i] = mustBuild(t, s)
	}
	return out
}
