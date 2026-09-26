package platform

import "os/exec"

// HasRosetta reports whether this Mac can run x86_64 binaries. Only meaningful
// on macOS/arm64; running a trivial x86_64 binary is the most direct check.
func HasRosetta() bool {
	return exec.Command("/usr/bin/arch", "-x86_64", "/usr/bin/true").Run() == nil
}
