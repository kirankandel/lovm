//go:build windows

package shim

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

// execReplace emulates exec, which Windows lacks: it runs soffice as a child
// with this process's stdio and exits with its status. Like exec it only
// returns on failure, so callers see soffice's exit code and output only.
func execReplace(path string, argv []string) error {
	// Ctrl-C reaches every process on the console; soffice decides how to
	// stop, and the shim must stay alive to report its exit status.
	signal.Ignore(os.Interrupt)
	cmd := exec.Command(path, argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return fmt.Errorf("run %s: %w", path, err)
	}
	os.Exit(cmd.ProcessState.ExitCode())
	return nil
}
