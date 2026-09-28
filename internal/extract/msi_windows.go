//go:build windows

package extract

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// errInstallInProgress is msiexec's ERROR_INSTALL_ALREADY_RUNNING.
const errInstallInProgress = 1618

// MSI unpacks an installer with an administrative install (msiexec /a): the
// files are copied into dest with no registry entries, shortcuts or file
// associations, so it needs no admin rights and cannot clash with a regular
// LibreOffice install.
func MSI(ctx context.Context, msiPath, dest string) error {
	msiPath, err := filepath.Abs(msiPath)
	if err != nil {
		return err
	}
	if dest, err = filepath.Abs(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	logFile, err := os.CreateTemp("", "lovm-msiexec-*.log")
	if err != nil {
		return err
	}
	logPath := logFile.Name()
	if err := logFile.Close(); err != nil {
		return err
	}

	msiexec := filepath.Join(os.Getenv("SystemRoot"), "System32", "msiexec.exe")
	cmd := exec.CommandContext(ctx, msiexec)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: msiexecCmdLine(msiexec, msiPath, dest, logPath)}
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == errInstallInProgress {
			return errors.New("another Windows installation is in progress; wait for it to finish and retry")
		}
		return fmt.Errorf("msiexec /a: %w (log: %s)", err, logPath)
	}
	if err := os.Remove(logPath); err != nil {
		return err
	}
	return removeImageCopies(dest)
}

// msiexecCmdLine builds the raw command line. msiexec parses it itself and
// wants PROPERTY="value", which Go's per-argument quoting ("PROPERTY=value")
// does not produce. Windows paths cannot contain quotes, so no escaping is needed.
func msiexecCmdLine(msiexec, msiPath, dest, logPath string) string {
	return fmt.Sprintf(`"%s" /a "%s" /qn TARGETDIR="%s" /l* "%s"`, msiexec, msiPath, dest, logPath)
}

// removeImageCopies deletes the stripped copy of the .msi that an
// administrative install leaves next to the files; it is only needed to
// install from the image, which lovm never does.
func removeImageCopies(dest string) error {
	copies, err := filepath.Glob(filepath.Join(dest, "*.msi"))
	if err != nil {
		return err
	}
	for _, c := range copies {
		if err := os.Remove(c); err != nil {
			return err
		}
	}
	return nil
}
