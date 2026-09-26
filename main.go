// Command lovm installs and switches between LibreOffice versions. When
// invoked through its "soffice" symlink it acts as the version-selecting shim.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kirankandel/lovm/internal/cli"
	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/shim"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	var err error
	if invokedAs(os.Args[0]) == "soffice" {
		err = shim.Run(os.Args)
	} else {
		err = cli.Run(ctx, os.Args[1:])
	}
	code := report(os.Stderr, err)
	stop()
	os.Exit(code)
}

func invokedAs(arg0 string) string {
	return strings.TrimSuffix(filepath.Base(arg0), ".exe")
}

// report prints err with its hint and returns the process exit code.
func report(w io.Writer, err error) int {
	if err == nil {
		return 0
	}
	fmt.Fprintf(w, "lovm: %v\n", err)
	var e *errs.Error
	if !errors.As(err, &e) {
		return 1
	}
	if e.Hint != "" {
		fmt.Fprintln(w, e.Hint)
	}
	return e.Code
}
