package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegrationInstallAndConvert downloads real LibreOffice builds, so it
// only runs with LOVM_INTEGRATION=1. LOVM_INTEGRATION_VERSIONS picks the
// versions (default "24.8"); on Linux use "6.4 24.8" to cover both gzip and
// xz packages.
func TestIntegrationInstallAndConvert(t *testing.T) {
	if os.Getenv("LOVM_INTEGRATION") != "1" {
		t.Skip("set LOVM_INTEGRATION=1 to run (downloads ~200 MB per version)")
	}
	versions := strings.Fields(os.Getenv("LOVM_INTEGRATION_VERSIONS"))
	if len(versions) == 0 {
		versions = []string{"24.8"}
	}

	bin := filepath.Join(t.TempDir(), "lovm")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	lovmHome := t.TempDir()
	input := filepath.Join(t.TempDir(), "hello.txt")
	if err := os.WriteFile(input, []byte("hello from lovm\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, v := range versions {
		t.Run(v, func(t *testing.T) {
			lovm := func(args ...string) {
				t.Helper()
				cmd := exec.Command(bin, args...)
				cmd.Env = append(os.Environ(), "LOVM_HOME="+lovmHome)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("lovm %s: %v\n%s", strings.Join(args, " "), err, out)
				}
			}
			lovm("install", v)
			outDir := t.TempDir()
			lovm("exec", v, "--", "soffice", "--headless", "--convert-to", "pdf", "--outdir", outDir, input)
			if _, err := os.Stat(filepath.Join(outDir, "hello.pdf")); err != nil {
				t.Fatalf("no PDF produced: %v", err)
			}
		})
	}
}
