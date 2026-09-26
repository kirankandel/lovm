package extract

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// untar writes a tar stream into dest. Entries that would land outside dest,
// directly or through a symlink, are errors rather than skipped, so a broken
// package never produces a silently incomplete install.
func untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(dest, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			err = writeFile(target, tr, hdr.FileInfo().Mode().Perm())
		case tar.TypeSymlink:
			err = writeSymlink(dest, target, hdr.Linkname)
		default:
			err = fmt.Errorf("unsupported entry %s (type %q)", hdr.Name, hdr.Typeflag)
		}
		if err != nil {
			return err
		}
	}
}

func writeFile(target string, r io.Reader, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, r)
	return errors.Join(copyErr, f.Close())
}

func writeSymlink(dest, target, linkname string) error {
	if filepath.IsAbs(linkname) || !within(dest, filepath.Join(filepath.Dir(target), linkname)) {
		return fmt.Errorf("symlink %s -> %s points outside the install directory", target, linkname)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Symlink(linkname, target)
}

func safeJoin(dest, name string) (string, error) {
	target := filepath.Join(dest, filepath.FromSlash(name))
	if !within(dest, target) {
		return "", fmt.Errorf("entry %q escapes the install directory", name)
	}
	return target, nil
}

func within(dest, path string) bool {
	rel, err := filepath.Rel(dest, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
