package catalog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// writeJSONFile writes v via a temp file and rename, so a crash never leaves
// a half-written cache behind.
func writeJSONFile(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	encodeErr := json.NewEncoder(tmp).Encode(v)
	if err := errors.Join(encodeErr, tmp.Close()); err != nil {
		return errors.Join(err, os.Remove(tmp.Name()))
	}
	return os.Rename(tmp.Name(), path)
}
