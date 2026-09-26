// Package download fetches installer files into lovm's cache.
package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// Fetch downloads url to dest unless dest already exists. Data goes to
// dest+".part" and is renamed into place only once the full body arrived, so
// dest never holds a partial file.
func Fetch(ctx context.Context, client *http.Client, url, dest string, progress io.Writer) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	part := dest + ".part"
	if err := writeBody(part, resp.Body, resp.ContentLength, progress, filepath.Base(dest)); err != nil {
		return errors.Join(fmt.Errorf("download %s: %w", url, err), removeIfExists(part))
	}
	return os.Rename(part, dest)
}

func writeBody(path string, body io.Reader, size int64, progress io.Writer, label string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	src := body
	if progress != nil {
		src = &progressReader{r: body, total: size, out: progress, label: label}
	}
	n, copyErr := io.Copy(f, src)
	if err := errors.Join(copyErr, f.Close()); err != nil {
		return err
	}
	if size >= 0 && n != size {
		return fmt.Errorf("incomplete download: got %d of %d bytes", n, size)
	}
	if progress != nil {
		fmt.Fprintln(progress)
	}
	return nil
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// progressReader prints "Downloading <file>  42%" each time the percentage changes.
type progressReader struct {
	r       io.Reader
	total   int64
	done    int64
	lastPct int
	out     io.Writer
	label   string
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.done += int64(n)
	if p.total > 0 {
		if pct := int(p.done * 100 / p.total); pct != p.lastPct {
			p.lastPct = pct
			fmt.Fprintf(p.out, "\rDownloading %s %3d%%", p.label, pct)
		}
	}
	return n, err
}
