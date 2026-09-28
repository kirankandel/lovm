//go:build windows

package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// defaultPathExt is what cmd.exe uses when PATHEXT is unset.
const defaultPathExt = ".com;.exe;.bat;.cmd"

// executableCandidates follows cmd.exe: Windows marks programs by extension,
// so "soffice" means the first of soffice.com, soffice.exe, ... per PATHEXT.
func executableCandidates(name string) []string {
	pathExt := os.Getenv("PATHEXT")
	if pathExt == "" {
		pathExt = defaultPathExt
	}
	exts := strings.Split(strings.ToLower(pathExt), ";")
	if slices.Contains(exts, strings.ToLower(filepath.Ext(name))) {
		return []string{name}
	}
	candidates := make([]string, 0, len(exts))
	for _, ext := range exts {
		if ext != "" {
			candidates = append(candidates, name+ext)
		}
	}
	return candidates
}

func isExecutable(info fs.FileInfo) bool { return info.Mode().IsRegular() }
