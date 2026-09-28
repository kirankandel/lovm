//go:build unix

package cli

import "io/fs"

func executableCandidates(name string) []string { return []string{name} }

func isExecutable(info fs.FileInfo) bool {
	return info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}
