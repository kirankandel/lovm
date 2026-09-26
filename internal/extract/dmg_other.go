//go:build !darwin

package extract

import (
	"context"
	"errors"
)

// DMG is only available on macOS, which provides hdiutil.
func DMG(ctx context.Context, dmgPath, dest string) error {
	return errors.New("disk images can only be unpacked on macOS")
}
