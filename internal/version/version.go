// Package version parses LibreOffice build IDs and the version specs users type.
package version

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// MinMajor is the oldest LibreOffice major version lovm supports.
const MinMajor = 6

// Build is a full four-part LibreOffice build ID such as 24.8.4.2.
type Build [4]int

// ParseBuild parses a four-part build ID such as "24.8.4.2".
func ParseBuild(s string) (Build, error) {
	parts, err := parseParts(s)
	if err != nil {
		return Build{}, err
	}
	if len(parts) != 4 {
		return Build{}, fmt.Errorf("invalid build %q: want four parts like 24.8.4.2", s)
	}
	var b Build
	copy(b[:], parts)
	return b, nil
}

func (b Build) String() string { return fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3]) }

// Triple is the release number users talk about, e.g. "24.8.4".
func (b Build) Triple() string { return fmt.Sprintf("%d.%d.%d", b[0], b[1], b[2]) }

// Branch is the release branch, e.g. "24.8".
func (b Build) Branch() string { return fmt.Sprintf("%d.%d", b[0], b[1]) }

// Compare returns -1, 0 or +1 when b is older than, equal to or newer than o.
func (b Build) Compare(o Build) int { return slices.Compare(b[:], o[:]) }

// MarshalText makes builds appear as "24.8.4.2" in JSON, including as map keys.
func (b Build) MarshalText() ([]byte, error) { return []byte(b.String()), nil }

func (b *Build) UnmarshalText(text []byte) error {
	parsed, err := ParseBuild(string(text))
	if err != nil {
		return err
	}
	*b = parsed
	return nil
}

// Spec selects one or more builds: "latest", "fresh", "still", "24.8",
// "24.8.4" or "24.8.4.2". Create it with ParseSpec.
type Spec struct {
	latest  bool
	channel string // "fresh" or "still"; expanded to a branch by the catalog
	parts   []int
}

// ParseSpec parses a user-supplied version spec, ignoring surrounding whitespace.
func ParseSpec(s string) (Spec, error) {
	s = strings.TrimSpace(s)
	switch s {
	case "latest":
		return Spec{latest: true}, nil
	case "fresh", "still":
		return Spec{channel: s}, nil
	}
	parts, err := parseParts(s)
	if err != nil || len(parts) < 2 || len(parts) > 4 {
		return Spec{}, fmt.Errorf("invalid version %q: use latest, fresh, still, 24.8, 24.8.4 or 24.8.4.2", s)
	}
	return Spec{parts: parts}, nil
}

func (s Spec) String() string {
	if s.latest {
		return "latest"
	}
	if s.channel != "" {
		return s.channel
	}
	strs := make([]string, len(s.parts))
	for i, p := range s.parts {
		strs[i] = strconv.Itoa(p)
	}
	return strings.Join(strs, ".")
}

// IsExact reports whether the spec names one four-part build.
func (s Spec) IsExact() bool { return len(s.parts) == 4 }

// Channel returns "fresh" or "still" for a channel spec, and "" otherwise.
func (s Spec) Channel() string { return s.channel }

// BelowFloor reports whether the spec asks for a version older than MinMajor.
func (s Spec) BelowFloor() bool { return len(s.parts) > 0 && s.parts[0] < MinMajor }

// Matches reports whether b falls under the spec. A channel spec matches
// nothing: which branch it means comes from the catalog, so expand it first.
func (s Spec) Matches(b Build) bool {
	if s.latest {
		return true
	}
	if s.channel != "" {
		return false
	}
	return slices.Equal(s.parts, b[:len(s.parts)])
}

// Lower returns the oldest build the spec could match (missing parts are zero).
func (s Spec) Lower() Build {
	var b Build
	copy(b[:], s.parts)
	return b
}

func parseParts(s string) ([]int, error) {
	fields := strings.Split(s, ".")
	parts := make([]int, 0, len(fields))
	for _, f := range fields {
		if f == "" || strings.Trim(f, "0123456789") != "" {
			return nil, fmt.Errorf("invalid version %q", s)
		}
		n, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("invalid version %q: %w", s, err)
		}
		parts = append(parts, n)
	}
	return parts, nil
}
