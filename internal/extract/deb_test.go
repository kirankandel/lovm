package extract

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

type tarEntry struct {
	name string
	body string
	mode int64
	link string // non-empty makes a symlink
	dir  bool
}

func tarBytes(t *testing.T, entries ...tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: e.mode, Size: int64(len(e.body)), Typeflag: tar.TypeReg}
		switch {
		case e.dir:
			hdr.Typeflag, hdr.Size = tar.TypeDir, 0
		case e.link != "":
			hdr.Typeflag, hdr.Linkname, hdr.Size = tar.TypeSymlink, e.link, 0
		}
		if hdr.Mode == 0 {
			hdr.Mode = 0o644
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func xzBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type arMember struct {
	name string
	data []byte
}

// debBytes builds a .deb, which is an ar archive of its members.
func debBytes(members ...arMember) []byte {
	var buf bytes.Buffer
	buf.WriteString("!<arch>\n")
	for _, m := range members {
		fmt.Fprintf(&buf, "%-16s%-12d%-6d%-6d%-8o%-10d`\n", m.name, 0, 0, 0, 0o644, len(m.data))
		buf.Write(m.data)
		if len(m.data)%2 == 1 {
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes()
}

// writeTarball writes a LibreOffice-style _deb.tar.gz holding the given entries.
func writeTarball(t *testing.T, entries ...tarEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "LibreOffice_24.8.7.2_Linux_x86-64_deb.tar.gz")
	if err := os.WriteFile(path, gzipBytes(t, tarBytes(t, entries...)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const top = "LibreOffice_24.8.7.2_Linux_x86-64_deb/"

func TestDebTarballExtractsGzipAndXzDebs(t *testing.T) {
	core := debBytes(
		arMember{"debian-binary", []byte("2.0\n")},
		arMember{"odd-sized", []byte("abc")}, // exercises ar's 2-byte alignment padding
		arMember{"data.tar.gz", gzipBytes(t, tarBytes(t,
			tarEntry{name: "./", dir: true},
			tarEntry{name: "./opt/libreoffice24.8/program/soffice", body: "#!/bin/sh\n", mode: 0o755},
			tarEntry{name: "./opt/libreoffice24.8/program/libfoo.so", link: "libfoo.so.1"},
		))},
	)
	calc := debBytes(
		arMember{"debian-binary", []byte("2.0\n")},
		arMember{"data.tar.xz", xzBytes(t, tarBytes(t,
			tarEntry{name: "./opt/libreoffice24.8/program/calc.txt", body: "calc"},
		))},
	)
	menus := debBytes(arMember{"data.tar.gz", gzipBytes(t, tarBytes(t,
		tarEntry{name: "./usr/bin/libreoffice24.8", body: "x"},
	))})

	tarball := writeTarball(t,
		tarEntry{name: top + "readmes/README_en-US", body: "readme"},
		tarEntry{name: top + "DEBS/core.deb", body: string(core)},
		tarEntry{name: top + "DEBS/calc.deb", body: string(calc)},
		tarEntry{name: top + "DEBS/desktop-integration/menus.deb", body: string(menus)},
	)
	dest := t.TempDir()
	if err := DebTarball(tarball, dest); err != nil {
		t.Fatal(err)
	}

	program := filepath.Join(dest, "opt", "libreoffice24.8", "program")
	info, err := os.Stat(filepath.Join(program, "soffice"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("soffice mode = %v, want executable", info.Mode())
	}
	if data, _ := os.ReadFile(filepath.Join(program, "calc.txt")); string(data) != "calc" {
		t.Errorf("calc.txt = %q (xz payload)", data)
	}
	if link, err := os.Readlink(filepath.Join(program, "libfoo.so")); err != nil || link != "libfoo.so.1" {
		t.Errorf("symlink = %q, %v", link, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "usr")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("desktop-integration debs should be skipped")
	}
}

func TestDebTarballRejectsUnsafeEntries(t *testing.T) {
	tests := map[string]tarEntry{
		"path traversal":   {name: "../escape.txt", body: "x"},
		"absolute symlink": {name: "./opt/link", link: "/etc/passwd"},
		"escaping symlink": {name: "./opt/link", link: "../../escape.txt"},
	}
	for name, entry := range tests {
		t.Run(name, func(t *testing.T) {
			deb := debBytes(arMember{"data.tar.gz", gzipBytes(t, tarBytes(t, entry))})
			tarball := writeTarball(t, tarEntry{name: top + "DEBS/a.deb", body: string(deb)})
			parent := t.TempDir()
			err := DebTarball(tarball, filepath.Join(parent, "install"))
			if err == nil {
				t.Fatal("expected error")
			}
			if _, err := os.Lstat(filepath.Join(parent, "escape.txt")); !errors.Is(err, fs.ErrNotExist) {
				t.Error("wrote outside the install directory")
			}
		})
	}
}

func TestDebTarballRejectsUnknownPayloadCompression(t *testing.T) {
	deb := debBytes(arMember{"data.tar.zst", []byte("zstd data")})
	tarball := writeTarball(t, tarEntry{name: top + "DEBS/a.deb", body: string(deb)})
	err := DebTarball(tarball, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "data.tar.zst") {
		t.Fatalf("err = %v, want it to name the unsupported payload", err)
	}
}

func TestDebTarballWithoutDebsFails(t *testing.T) {
	tarball := writeTarball(t, tarEntry{name: top + "readmes/README", body: "x"})
	if err := DebTarball(tarball, t.TempDir()); err == nil {
		t.Fatal("expected error for a tarball with no packages")
	}
}
