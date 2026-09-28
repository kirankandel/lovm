package install

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/extract"
	"github.com/kirankandel/lovm/internal/home"
	"github.com/kirankandel/lovm/internal/version"
)

const smokeTimeout = 2 * time.Minute

// smokeTest runs `soffice --version` and checks it reports b. This is what
// catches builds that unpack fine but cannot run on this system.
func smokeTest(ctx context.Context, soffice, profileDir string, b version.Build) error {
	ctx, cancel := context.WithTimeout(ctx, smokeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, soffice, "-env:UserInstallation="+home.FileURL(profileDir), "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errs.WontRun(b.String(), fmt.Sprintf("%v\n%s", err, bytes.TrimSpace(out)))
	}
	if !strings.Contains(string(out), "LibreOffice "+b.Triple()) {
		return errs.WontRun(b.String(), fmt.Sprintf("unexpected version output: %s", bytes.TrimSpace(out)))
	}
	return nil
}

func extractFor(ctx context.Context, goos, archivePath, dest string) error {
	switch goos {
	case "linux":
		return extract.DebTarball(archivePath, dest)
	case "darwin":
		return extract.DMG(ctx, archivePath, dest)
	case "windows":
		return extract.MSI(ctx, archivePath, dest)
	default:
		return errs.UnsupportedOS(goos)
	}
}
