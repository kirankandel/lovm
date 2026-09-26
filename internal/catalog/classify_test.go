package catalog

import (
	"slices"
	"testing"

	"github.com/kirankandel/lovm/internal/version"
)

func TestNewStableInfo(t *testing.T) {
	info, err := newStableInfo([]string{"26.2.5", "25.8.7", "26.8.0"})
	if err != nil {
		t.Fatal(err)
	}
	if info.oldestBranch != (version.Build{25, 8, 0, 0}) {
		t.Errorf("oldestBranch = %v", info.oldestBranch)
	}
	if !info.triples["26.2.5"] {
		t.Errorf("triples = %v", info.triples)
	}
	if _, err := newStableInfo(nil); err == nil {
		t.Error("empty stable listing should be an error")
	}
	if _, err := newStableInfo([]string{"latest"}); err == nil {
		t.Error("non-version entry should be an error")
	}
}

func TestClassifyReleases(t *testing.T) {
	stable, err := newStableInfo([]string{"25.8.7", "26.2.5", "26.2.6", "26.8.0"})
	if err != nil {
		t.Fatal(err)
	}
	builds := mustBuilds(t,
		"7.6.6.1", "7.6.6.3", "7.6.7.1", "7.6.7.2", // end-of-life branch
		"25.8.6.1", "25.8.6.2", "25.8.7.1", // 25.8.6 superseded, 25.8.7 in stable
		"26.2.5.1", "26.2.5.2", "26.2.6.1", "26.2.6.2", // both in stable
		"26.8.0.1", "26.8.0.2", "26.8.0.3", "26.8.1.1", // 26.8.1 still in RC
		"27.2.0.1", // new branch, not released yet
	)
	got := classifyReleases(builds, stable)

	var gotList []version.Build
	for b, isRelease := range got {
		if isRelease {
			gotList = append(gotList, b)
		}
	}
	slices.SortFunc(gotList, version.Build.Compare)
	want := mustBuilds(t, "7.6.6.3", "7.6.7.2", "25.8.6.2", "25.8.7.1", "26.2.5.2", "26.2.6.2", "26.8.0.3")
	if !slices.Equal(gotList, want) {
		t.Errorf("releases = %v\nwant       %v", gotList, want)
	}
}
