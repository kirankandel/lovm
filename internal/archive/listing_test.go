package archive

import (
	"os"
	"slices"
	"testing"
)

func TestParseListingRealPage(t *testing.T) {
	f, err := os.Open("testdata/version_dir.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := parseListing(f)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"deb/", "mac/", "portable/", "rpm/", "src/", "win/"}
	if !slices.Equal(got, want) {
		t.Errorf("parseListing = %v, want %v", got, want)
	}
}
