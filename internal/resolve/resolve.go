// Package resolve turns a version spec into a concrete build and installer
// for a platform, and explains precisely why when it cannot.
package resolve

import (
	"context"

	"github.com/kirankandel/lovm/internal/archive"
	"github.com/kirankandel/lovm/internal/errs"
	"github.com/kirankandel/lovm/internal/platform"
	"github.com/kirankandel/lovm/internal/version"
)

// Source is what the resolver needs to know about the archive.
type Source interface {
	AllBuilds() []version.Build // oldest first
	IsRelease(b version.Build) bool
	FindInstaller(ctx context.Context, b version.Build, p platform.Platform) (archive.Installer, bool, error)
}

type Result struct {
	Build     version.Build
	Installer archive.Installer
	// Platform is the installer's platform: macos/amd64 when an Apple
	// Silicon Mac runs it under Rosetta.
	Platform platform.Platform
	Rosetta  bool
}

type Resolver struct {
	Source     Source
	Platform   platform.Platform
	HasRosetta func() bool
}

func (r Resolver) Resolve(ctx context.Context, spec version.Spec) (Result, error) {
	if spec.BelowFloor() {
		return Result{}, errs.BelowFloor(spec.String())
	}
	matches := newestFirst(r.Source.AllBuilds(), spec)
	if len(matches) == 0 {
		return Result{}, errs.NotFound(spec.String(), r.closest(spec))
	}
	if spec.IsExact() {
		return r.resolveBuild(ctx, matches[0])
	}
	var releases []version.Build
	for _, b := range matches {
		if r.Source.IsRelease(b) {
			releases = append(releases, b)
		}
	}
	if len(releases) == 0 {
		return Result{}, errs.OnlyRC(spec.String(), matches[0].String())
	}
	for _, b := range releases {
		res, found, err := r.Lookup(ctx, b)
		if err != nil {
			return Result{}, err
		}
		if found {
			return r.checkRosetta(res)
		}
	}
	return Result{}, r.noPlatformBuild(ctx, releases[0])
}

// Lookup finds b's installer for r.Platform. On Apple Silicon it falls back
// to the x86_64 build, which runs under Rosetta 2.
func (r Resolver) Lookup(ctx context.Context, b version.Build) (Result, bool, error) {
	inst, found, err := r.Source.FindInstaller(ctx, b, r.Platform)
	if err != nil {
		return Result{}, false, err
	}
	if found {
		return Result{Build: b, Installer: inst, Platform: r.Platform}, true, nil
	}
	if r.Platform.OS != "darwin" || r.Platform.Arch != "arm64" {
		return Result{}, false, nil
	}
	intel := platform.Platform{OS: "darwin", Arch: "amd64"}
	inst, found, err = r.Source.FindInstaller(ctx, b, intel)
	if err != nil || !found {
		return Result{}, false, err
	}
	return Result{Build: b, Installer: inst, Platform: intel, Rosetta: true}, true, nil
}

func (r Resolver) resolveBuild(ctx context.Context, b version.Build) (Result, error) {
	res, found, err := r.Lookup(ctx, b)
	if err != nil {
		return Result{}, err
	}
	if !found {
		return Result{}, r.noPlatformBuild(ctx, b)
	}
	return r.checkRosetta(res)
}

func (r Resolver) checkRosetta(res Result) (Result, error) {
	if res.Rosetta && !r.HasRosetta() {
		return Result{}, errs.RosettaMissing(res.Build.String())
	}
	return res, nil
}

func (r Resolver) noPlatformBuild(ctx context.Context, b version.Build) error {
	var available []string
	for _, p := range platform.All() {
		_, found, err := r.Source.FindInstaller(ctx, b, p)
		if err != nil {
			return err
		}
		if found {
			available = append(available, p.String())
		}
	}
	first, err := r.firstReleaseWith(ctx, b)
	if err != nil {
		return err
	}
	return errs.NoPlatformBuild(b.String(), r.Platform.String(), available, first)
}

// firstReleaseWith finds the first release in a newer branch than after that
// has a native build for r.Platform. Platform support arrives per branch, so
// only the first release of each branch is checked: a few requests, not dozens.
func (r Resolver) firstReleaseWith(ctx context.Context, after version.Build) (string, error) {
	checked := map[string]bool{after.Branch(): true}
	for _, b := range r.Source.AllBuilds() {
		if b.Compare(after) <= 0 || !r.Source.IsRelease(b) || checked[b.Branch()] {
			continue
		}
		checked[b.Branch()] = true
		_, found, err := r.Source.FindInstaller(ctx, b, r.Platform)
		if err != nil {
			return "", err
		}
		if found {
			return b.Triple(), nil
		}
	}
	return "", nil
}

// closest suggests the nearest releases on either side of a spec that matched nothing.
func (r Resolver) closest(spec version.Spec) []string {
	target := spec.Lower()
	var below, above version.Build
	var haveBelow, haveAbove bool
	for _, b := range r.Source.AllBuilds() {
		if !r.Source.IsRelease(b) {
			continue
		}
		if b.Compare(target) < 0 {
			below, haveBelow = b, true
		} else if !haveAbove {
			above, haveAbove = b, true
		}
	}
	var out []string
	if haveBelow {
		out = append(out, below.Triple())
	}
	if haveAbove {
		out = append(out, above.Triple())
	}
	return out
}

func newestFirst(builds []version.Build, spec version.Spec) []version.Build {
	var out []version.Build
	for i := len(builds) - 1; i >= 0; i-- {
		if spec.Matches(builds[i]) {
			out = append(out, builds[i])
		}
	}
	return out
}
