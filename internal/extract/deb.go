// Package extract unpacks LibreOffice installer files without root.
package extract

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ulikunitz/xz"
)

// DebTarball unpacks every .deb directly inside the DEBS folder of a
// LibreOffice "_deb.tar.gz" into dest, giving dest/opt/libreofficeX.Y/...
// Debs in subfolders (desktop-integration) are skipped: they only add system
// menu entries and point at absolute paths.
func DebTarball(tarGzPath, dest string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("read %s: %w", tarGzPath, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	found := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", tarGzPath, err)
		}
		if hdr.Typeflag != tar.TypeReg || !isTopLevelDeb(hdr.Name) {
			continue
		}
		if err := extractDeb(tr, dest); err != nil {
			return fmt.Errorf("%s: %w", path.Base(hdr.Name), err)
		}
		found++
	}
	if found == 0 {
		return fmt.Errorf("no .deb packages found in %s", tarGzPath)
	}
	return nil
}

func isTopLevelDeb(name string) bool {
	return strings.HasSuffix(name, ".deb") && path.Base(path.Dir(name)) == "DEBS"
}

const arMagic = "!<arch>\n"

// extractDeb reads a .deb (an ar archive) and unpacks its data.tar.* member.
func extractDeb(r io.Reader, dest string) error {
	magic := make([]byte, len(arMagic))
	if _, err := io.ReadFull(r, magic); err != nil {
		return fmt.Errorf("read ar header: %w", err)
	}
	if string(magic) != arMagic {
		return errors.New("not an ar archive")
	}
	for {
		name, size, err := readArHeader(r)
		if err == io.EOF {
			return errors.New("no data.tar member")
		}
		if err != nil {
			return err
		}
		member := io.LimitReader(r, size)
		if strings.HasPrefix(name, "data.tar") {
			return untarCompressed(member, name, dest)
		}
		if _, err := io.Copy(io.Discard, member); err != nil {
			return err
		}
		if size%2 == 1 { // members are 2-byte aligned
			if _, err := io.CopyN(io.Discard, r, 1); err != nil {
				return err
			}
		}
	}
}

// readArHeader reads one 60-byte ar member header: name(16) mtime(12) uid(6)
// gid(6) mode(8) size(10) magic(2).
func readArHeader(r io.Reader) (name string, size int64, err error) {
	var hdr [60]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return "", 0, err
	}
	name = strings.TrimRight(strings.TrimSpace(string(hdr[0:16])), "/")
	size, err = strconv.ParseInt(strings.TrimSpace(string(hdr[48:58])), 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("bad ar size for %q: %w", name, err)
	}
	return name, size, nil
}

// untarCompressed handles the payload compressions seen in LibreOffice debs:
// gzip (6.x) and xz (24.x).
func untarCompressed(r io.Reader, name, dest string) error {
	switch {
	case strings.HasSuffix(name, ".gz"):
		gz, err := gzip.NewReader(r)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		defer gz.Close()
		return untar(gz, dest)
	case strings.HasSuffix(name, ".xz"):
		xr, err := xz.NewReader(r)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return untar(xr, dest)
	default:
		return fmt.Errorf("unsupported deb payload %s", name)
	}
}
