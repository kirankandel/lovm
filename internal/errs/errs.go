// Package errs defines the failures lovm explains to users. Each carries a
// hint and a distinct exit code so scripts and CI can tell them apart.
package errs

import (
	"fmt"
	"strings"

	"github.com/kirankandel/lovm/internal/version"
)

// Exit codes. 1 is used for unexpected errors.
const (
	CodeUsage           = 2
	CodeNotFound        = 3
	CodeBelowFloor      = 4
	CodeNoPlatformBuild = 5
	CodeRosettaMissing  = 6
	CodeOnlyRC          = 7
	CodeUnsupportedOS   = 8
	CodeWontRun         = 9
	CodeNotSelected     = 10
	CodeNotInstalled    = 11
	CodeChannelUnknown  = 12
)

// Error is a failure lovm explains: Msg says what went wrong, Hint what to do.
type Error struct {
	Code int
	Msg  string
	Hint string
}

func (e *Error) Error() string { return e.Msg }

func Usage(msg string) *Error {
	return &Error{Code: CodeUsage, Msg: msg, Hint: "Run 'lovm help' for usage."}
}

func NotFound(spec string, closest []string) *Error {
	hint := "See available versions with: lovm ls-remote"
	if len(closest) > 0 {
		hint = "Closest: " + strings.Join(closest, ", ") + "\n" + hint
	}
	return &Error{Code: CodeNotFound, Msg: fmt.Sprintf("LibreOffice %s not found", spec), Hint: hint}
}

func BelowFloor(spec string) *Error {
	return &Error{
		Code: CodeBelowFloor,
		Msg:  fmt.Sprintf("LibreOffice %s is not supported by lovm (minimum %d.0)", spec, version.MinMajor),
	}
}

func NoPlatformBuild(build, platform string, available []string, firstWith string) *Error {
	var hint []string
	if len(available) > 0 {
		hint = append(hint, "Available for: "+strings.Join(available, ", "))
	}
	if firstWith != "" {
		hint = append(hint, fmt.Sprintf("First version with %s: %s", platform, firstWith))
	}
	return &Error{
		Code: CodeNoPlatformBuild,
		Msg:  fmt.Sprintf("LibreOffice %s has no build for %s", build, platform),
		Hint: strings.Join(hint, "\n"),
	}
}

func RosettaMissing(build string) *Error {
	return &Error{
		Code: CodeRosettaMissing,
		Msg:  fmt.Sprintf("LibreOffice %s has no Apple Silicon build; the x86_64 build needs Rosetta 2", build),
		Hint: "Install it with: softwareupdate --install-rosetta",
	}
}

func OnlyRC(spec, latestRC string) *Error {
	return &Error{
		Code: CodeOnlyRC,
		Msg:  fmt.Sprintf("%s is not released yet (latest RC: %s)", spec, latestRC),
		Hint: "To install the RC: lovm install " + latestRC,
	}
}

func UnsupportedOS(goos string) *Error {
	return &Error{Code: CodeUnsupportedOS, Msg: fmt.Sprintf("lovm does not support %s yet", goos), Hint: "Supported: Linux and macOS."}
}

func WontRun(build, output string) *Error {
	return &Error{
		Code: CodeWontRun,
		Msg:  fmt.Sprintf("LibreOffice %s was unpacked but fails to start on this system", build),
		Hint: output,
	}
}

func NotSelected() *Error {
	return &Error{
		Code: CodeNotSelected,
		Msg:  "No LibreOffice version selected (checked LOVM_VERSION, .lovmrc, default)",
		Hint: "Pick one with: lovm default <version>  or  lovm use <version>",
	}
}

// NotInstalled reports a spec with no matching installed build. origin names
// where the spec came from (a .lovmrc path, "LOVM_VERSION", ...); it is empty
// when the user typed the spec on the command line.
func NotInstalled(spec, origin string) *Error {
	msg := fmt.Sprintf("LibreOffice %s is not installed", spec)
	if origin != "" {
		msg = fmt.Sprintf("%s is pinned by %s but not installed", spec, origin)
	}
	return &Error{Code: CodeNotInstalled, Msg: msg, Hint: "Run: lovm install " + spec}
}

// ChannelUnknown reports a "fresh"/"still" spec that can't be mapped to a
// branch: no version list is cached yet, or TDF currently maintains one branch.
func ChannelUnknown(channel string) *Error {
	return &Error{
		Code: CodeChannelUnknown,
		Msg:  fmt.Sprintf("don't know which LibreOffice branch is %q", channel),
		Hint: "Refresh the version list with: lovm ls-remote --refresh",
	}
}

// ChildExit carries the exit status of a command run by `lovm exec`.
func ChildExit(code int) *Error {
	return &Error{Code: code, Msg: fmt.Sprintf("command exited with status %d", code)}
}
