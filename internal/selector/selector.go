// Package selector decides which installed LibreOffice version is active.
package selector

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/kirankandel/lovm/internal/catalog"
	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/home"
	"github.com/kirankandel/lovm/internal/version"
)

const RCFile = ".lovmrc"

// utf8BOM is the byte-order mark some Windows editors put at the start of text files.
const utf8BOM = "\xef\xbb\xbf"

type Selection struct {
	Spec   version.Spec
	Origin string // "LOVM_VERSION", or the path of the .lovmrc or default file
}

// Select finds the active spec: LOVM_VERSION, then the nearest .lovmrc from
// cwd upwards, then the global default.
func Select(h home.Home, getenv func(string) string, cwd string) (Selection, error) {
	if v := getenv("LOVM_VERSION"); v != "" {
		spec, err := version.ParseSpec(v)
		if err != nil {
			return Selection{}, fmt.Errorf("LOVM_VERSION: %w", err)
		}
		return Selection{Spec: spec, Origin: "LOVM_VERSION"}, nil
	}
	rc, found, err := findRC(cwd)
	if err != nil {
		return Selection{}, err
	}
	if found {
		return selectFile(rc)
	}
	if _, err := os.Stat(h.DefaultFile()); err == nil {
		return selectFile(h.DefaultFile())
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Selection{}, err
	}
	return Selection{}, errs.NotSelected()
}

// Active resolves the selected spec to the newest matching installed build.
func Active(h home.Home, getenv func(string) string, cwd string) (version.Build, Selection, error) {
	sel, err := Select(h, getenv, cwd)
	if err != nil {
		return version.Build{}, Selection{}, err
	}
	spec, err := ExpandChannel(h, sel.Spec)
	if err != nil {
		return version.Build{}, sel, err
	}
	installed, err := h.Installed()
	if err != nil {
		return version.Build{}, Selection{}, err
	}
	b, ok := PickInstalled(spec, installed)
	if !ok {
		return version.Build{}, sel, errs.NotInstalled(sel.Spec.String(), sel.Origin)
	}
	return b, sel, nil
}

// ExpandChannel turns "fresh"/"still" into a branch using the version list
// cached by the last online command, so it works offline. Other specs are
// returned unchanged.
func ExpandChannel(h home.Home, spec version.Spec) (version.Spec, error) {
	if spec.Channel() == "" {
		return spec, nil
	}
	cat, err := catalog.ReadCached(h.CacheDir())
	if errors.Is(err, fs.ErrNotExist) {
		return version.Spec{}, errs.ChannelUnknown(spec.Channel())
	}
	if err != nil {
		return version.Spec{}, err
	}
	return cat.Expand(spec)
}

// PickInstalled returns the newest installed build matching spec. installed
// must be sorted oldest first, as home.Installed returns it.
func PickInstalled(spec version.Spec, installed []version.Build) (version.Build, bool) {
	for i := len(installed) - 1; i >= 0; i-- {
		if spec.Matches(installed[i]) {
			return installed[i], true
		}
	}
	return version.Build{}, false
}

// ReadSpecFile reads a one-line spec file such as .lovmrc. Only the first
// line counts; whitespace, CRLF endings and a UTF-8 byte-order mark are ignored.
func ReadSpecFile(path string) (version.Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return version.Spec{}, err
	}
	line, _, _ := strings.Cut(string(data), "\n")
	spec, err := version.ParseSpec(strings.TrimPrefix(line, utf8BOM))
	if err != nil {
		return version.Spec{}, fmt.Errorf("%s: %w", path, err)
	}
	return spec, nil
}

func selectFile(path string) (Selection, error) {
	spec, err := ReadSpecFile(path)
	if err != nil {
		return Selection{}, err
	}
	return Selection{Spec: spec, Origin: path}, nil
}

func findRC(dir string) (string, bool, error) {
	for {
		path := filepath.Join(dir, RCFile)
		_, err := os.Stat(path)
		if err == nil {
			return path, true, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", false, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
		dir = parent
	}
}
