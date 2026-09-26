package archive

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

var hrefPattern = regexp.MustCompile(`href="([^"]+)"`)

// parseListing extracts entry names from an Apache autoindex page. Folder
// entries keep their trailing slash. Sort links ("?C=..."), absolute links
// (parent folder, stylesheet) and external links are dropped.
func parseListing(r io.Reader) ([]string, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read listing: %w", err)
	}
	var entries []string
	for _, m := range hrefPattern.FindAllSubmatch(body, -1) {
		href := string(m[1])
		if strings.HasPrefix(href, "?") || strings.HasPrefix(href, "/") || strings.Contains(href, ":") {
			continue
		}
		entries = append(entries, href)
	}
	return entries, nil
}
