//go:build darwin

package extract

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DMG copies LibreOffice.app out of a disk image into dest. The image is
// always detached, even when copying fails.
func DMG(ctx context.Context, dmgPath, dest string) (err error) {
	mountPoint, err := os.MkdirTemp("", "lovm-dmg-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.Remove(mountPoint)) }()

	attach := exec.CommandContext(ctx, "hdiutil", "attach", "-nobrowse", "-readonly", "-noautoopen", "-mountpoint", mountPoint, dmgPath)
	if out, err := attach.CombinedOutput(); err != nil {
		return fmt.Errorf("hdiutil attach: %w: %s", err, out)
	}
	defer func() { err = errors.Join(err, detach(mountPoint)) }()

	app := filepath.Join(mountPoint, "LibreOffice.app")
	if _, err := os.Stat(app); err != nil {
		return fmt.Errorf("LibreOffice.app not found in %s: %w", dmgPath, err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if out, err := exec.CommandContext(ctx, "ditto", app, filepath.Join(dest, "LibreOffice.app")).CombinedOutput(); err != nil {
		return fmt.Errorf("copy LibreOffice.app: %w: %s", err, out)
	}
	return nil
}

// detach deliberately ignores the install's context: the image must be
// detached even when the user pressed Ctrl-C.
func detach(mountPoint string) error {
	out, err := exec.Command("hdiutil", "detach", mountPoint).CombinedOutput()
	if err == nil {
		return nil
	}
	if forceOut, forceErr := exec.Command("hdiutil", "detach", "-force", mountPoint).CombinedOutput(); forceErr != nil {
		return fmt.Errorf("hdiutil detach %s: %w: %s %s", mountPoint, forceErr, out, forceOut)
	}
	return nil
}
