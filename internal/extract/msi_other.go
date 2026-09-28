//go:build !windows

package extract

import (
	"context"
	"errors"
)

// MSI is only available on Windows, which provides msiexec.
func MSI(ctx context.Context, msiPath, dest string) error {
	return errors.New("MSI installers can only be unpacked on Windows")
}
