// Copyright (C) 2024 Michael J. Fromberger. All Rights Reserved.

// Package semver handles the parsing and formatting of [Semantic Version] strings.
//
// # Usage Outline
//
// Create a new semantic version by providing major, minor, and patch versions:
//
//	v := semver.New(1, 0, 0)
//
// The resulting version has no pre-release or build metadata.
//
// To extend a version with pre-release or build metadata, use:
//
//	v2 := v.WithPreRelease("rc1").WithBuild("unstable")
//
// To format the version as a string in standard notation, use:
//
//	v2.String()
//
// To parse an existing semantic version string:
//
//	v, err := semver.Parse("1.0.0-rc1.2+unstable")
//
// [Semantic Version]: https://semver.org/
package semver

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/creachadair/mds/slice"
)

// V represents a parsed semantic version label. A zero value is ready for use,
// and represents the semantic version "0.0.0".
type V struct {
	major, minor, patch string   // The major, minor, and patch versions (non-empty)
	pre                 []string // Dotted pre-release identifier (if non-empty)
	build               []string // Dotted patch identifier (if non-empty)
}

// New constructs a [V] with the specified major, minor, and patch versions.
// The pre-release and build metadata of the resulting value are empty.
// New will panic if any of these values is negative.
func New(major, minor, patch int) V {
	return V{major: mustItoa(major), minor: mustItoa(minor), patch: mustItoa(patch)}
}

// Before reports whether v is before w in version order.
// See also [Compare].
func (v V) Before(w V) bool { return Compare(v, w) < 0 }

// After reports whether v is after w in version order.
// See also [Compare].
func (v V) After(w V) bool { return Compare(v, w) > 0 }

// Equiv reports whether v and w are equivalent versions. Note that this is
// distinct from equality, because semantic version comparison ignores build
// metadata. See also [Compare].
func (v V) Equiv(w V) bool { return Compare(v, w) == 0 }

// Major reports the major version as an int.
func (v V) Major() int { return mustVal(v.major) }

// Minor reports the minor version as an int.
func (v V) Minor() int { return mustVal(v.minor) }

// Patch reports the patch version as an int.
func (v V) Patch() int { return mustVal(v.patch) }

// Add returns a copy of v with the specified offsets added to core versions.
// Offsets that would cause a version to become negative set it to 0 instead.
func (v V) Add(dmajor, dminor, dpatch int) V {
	m, i, p := max(v.Major()+dmajor, 0), max(v.Minor()+dminor, 0), max(v.Patch()+dpatch, 0)
	return v.WithCore(m, i, p)
}

// Core returns a copy of v with its pre-release and build metadata cleared,
// corresponding to the "core" version ID (major.minor.patch).
func (v V) Core() V { v.pre = nil; v.build = nil; return v }

// WithCore returns a copy of v with its core version (major.minor.patch) set.
// WithCore will panic if any of these values is negative.
func (v V) WithCore(major, minor, patch int) V {
	v.major, v.minor, v.patch = mustItoa(major), mustItoa(minor), mustItoa(patch)
	return v
}

// PreRelease reports the pre-release string, if present.
// The resulting string does not include the "-" prefix.
func (v V) PreRelease() string { return strings.Join(v.pre, ".") }

// WithPreRelease returns a copy of v with its pre-release ID set.
// If id == "", the resulting version has no pre-release ID.
func (v V) WithPreRelease(id string) V { v.pre = cleanWords(id); return v }

// Build reports the build metadata string, if present.
// The resulting string does not include the "+" prefix.
func (v V) Build() string { return strings.Join(v.build, ".") }

// WithBuild returns a copy of v with its build metadata set.
// If meta == "", the resulting version has no build metadata.
func (v V) WithBuild(meta string) V { v.build = cleanWords(meta); return v }

// String returns the complete canonical string representation of v.
func (v V) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s.%s.%s",
		cmp.Or(v.major, "0"), cmp.Or(v.minor, "0"), cmp.Or(v.patch, "0"))
	if pr := v.PreRelease(); pr != "" {
		fmt.Fprint(&sb, "-", pr)
	}
	if b := v.Build(); b != "" {
		fmt.Fprint(&sb, "+", b)
	}
	return sb.String()
}

// Compare compares v1 and v2 in standard semantic version order.
// It returns -1 if v1 < v2, 0 if v1 == v2, and +1 if v1 > v2.
func Compare(v1, v2 V) int {
	if c := cmp.Compare(mustVal(v1.major), mustVal(v2.major)); c != 0 {
		return c
	}
	if c := cmp.Compare(mustVal(v1.minor), mustVal(v2.minor)); c != 0 {
		return c
	}
	if c := cmp.Compare(mustVal(v1.patch), mustVal(v2.patch)); c != 0 {
		return c
	}
	n1, n2 := len(v1.pre), len(v2.pre)
	if n1 == 0 && n2 != 0 {
		return 1 // non-empty prerelease precedes empty
	} else if n1 != 0 && n2 == 0 {
		return -1 // non-empty prerelease precedes empty
	}
	return slices.CompareFunc(v1.pre, v2.pre, compareWord)

	// N.B. Build metadata are not considered for comparisons.
}

// CompareStrings compares s1 and s2 in standard semantic version order.
// The strings are cleaned (see [Clean]) before comparison.
// It returns -1 if s1 < s2, 0 if s1 == s2, and +1 if s1 > s2.
// If either string is not a valid semver after cleaning, the two strings are
// compared in ordinary lexicographic order.
func CompareStrings(s1, s2 string) int {
	v1, err1 := Parse(Clean(s1))
	v2, err2 := Parse(Clean(s2))
	if err1 == nil && err2 == nil {
		return Compare(v1, v2)
	}
	return cmp.Compare(s1, s2)
}

// MustParse returns the [V] represented by s, or panics.  This is intended for
// use in program initialization; use [Parse] to check for errors.
func MustParse(s string) V {
	v, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("Parse %q: %v", s, err))
	}
	return v
}

// IsValid reports whether s is a valid semver string.
func IsValid(s string) bool { _, err := Parse(s); return err == nil }

// Parse returns the [V] represented by s.
func Parse(s string) (V, error) {
	// Grammar: https://semver.org/#backusnaur-form-grammar-for-valid-semver-versions

	// Check for pre-release and build labels.
	var pre, build string
	var hasPre, hasBuild bool
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		rest := s[i:] // N.B. keep the marker
		s = s[:i]

		if p, ok := strings.CutPrefix(rest, "-"); ok {
			// rest == "-<pre>[+<build>]"
			hasPre = true
			pre, build, hasBuild = strings.Cut(p, "+")
		} else {
			// rest == "" or rest == "+<build>"
			build, hasBuild = strings.CutPrefix(rest, "+")
		}
	}

	// Parse the base version: major '.' minor '.' patch
	ps := splitWords(s)
	if len(ps) != 3 {
		return V{}, fmt.Errorf("wrong length (got %d, want 3)", len(ps))
	}
	v := V{major: ps[0], minor: ps[1], patch: ps[2]}
	if err := checkVNum(v.major); err != nil {
		return V{}, fmt.Errorf("invalid major: %w", err)
	}
	if err := checkVNum(v.minor); err != nil {
		return V{}, fmt.Errorf("invalid minor: %w", err)
	}
	if err := checkVNum(v.patch); err != nil {
		return V{}, fmt.Errorf("invalid patch: %w", err)
	}

	var err error
	if hasPre {
		if pre == "" {
			return V{}, errors.New("empty pre-release")
		} else if v.pre, err = parseWords(pre); err != nil {
			return V{}, fmt.Errorf("invalid pre-release %q: %w", pre, err)
		}
	}
	if hasBuild {
		if build == "" {
			return V{}, errors.New("empty build metadata")
		} else if v.build, err = parseWords(build); err != nil {
			return V{}, fmt.Errorf("invalid build %q: %w", build, err)
		}
	}
	return v, nil
}

// Clean returns a lexically normalized form of a semver-like string.
// The following changes are made, if possible:
//
//   - Leading and trailing whitespace is removed.
//   - A leading "v" is removed, if present.
//   - Omitted minor or patch versions are set to "0".
//   - Empty pre-release and build labels are removed.
//
// If a major version is not present, Clean returns s entirely unmodified.
// Otherwise, except as described above, the input is not modified. In
// particular, if s contains invalid characters or non-numeric version numbers,
// the result may (still) not be a valid version string.
func Clean(s string) string {
	base := strings.TrimPrefix(strings.TrimSpace(s), "v")
	var pre, build string
	if i := strings.IndexAny(base, "-+"); i >= 0 {
		tail := base[i:]
		base = base[:i]
		if p, ok := strings.CutPrefix(tail, "-"); ok {
			pre, build, _ = strings.Cut(p, "+")
		} else {
			build = p[1:] // drop "+"
		}
	}
	ps := strings.SplitN(base, ".", 3)
	if len(ps) == 0 || ps[0] == "" {
		return s
	}
	for i := 1; i < 3; i++ {
		if i >= len(ps) {
			ps = append(ps, "0")
		} else if ps[i] == "" {
			ps[i] = "0"
		}
	}
	out := strings.Join(ps, ".")
	if p := joinCleanWords(pre); p != "" {
		out += "-" + p
	}
	if p := joinCleanWords(build); p != "" {
		out += "+" + p
	}
	return out
}

// mustVal returns the integer represented by s, or panics.
// As a special case, if s == "" it returns 0.
func mustVal(s string) int {
	v, ok := isNum(s)
	if !ok {
		panic(fmt.Sprintf("invalid number %q", s))
	}
	return v
}

// isNum reports whether s comprises only digits, and if so returns the integer
// value represented by s. As a special case, if s == "" it returns (0, true).
func isNum(s string) (int, bool) {
	v := 0
	for i := range s {
		d := s[i]
		if d < '0' || d > '9' {
			return -1, false
		}
		v = (v * 10) + int(d-'0')
	}
	return v, true
}

// isWord reports whether s comprises only digits, letters, and hyphens.
func isWord(s string) bool {
	for i := range s {
		switch d := s[i]; {
		case d >= '0' && d <= '9', d >= 'a' && d <= 'z', d >= 'A' && d <= 'Z', d == '-':
		default:
			return false
		}
	}
	return true
}

// checkVNum reports an error of s is not a valid version number.
func checkVNum(s string) error {
	if _, ok := isNum(s); !ok || s == "" {
		return errors.New("not a number")
	} else if s[0] == '0' && s != "0" {
		return errors.New("leading zeroes")
	}
	return nil
}

// parseWords parses s as a dot-separated sequence of words.
// Precondition: s != ""
func parseWords(s string) ([]string, error) {
	ws := splitWords(s)
	for i, w := range ws {
		if w == "" {
			return nil, fmt.Errorf("empty word (pos %d)", i+1)
		} else if !isWord(w) {
			return nil, fmt.Errorf("invalid char (pos %d)", i+1)
		}
	}
	return ws, nil
}

// compareWord compares a and b. If both comprise only digits, the comparison
// is based on their numeric values; otherwise it is lexicographical.
func compareWord(a, b string) int {
	va, oka := isNum(a)
	vb, okb := isNum(b)
	if oka && okb {
		return cmp.Compare(va, vb)
	}
	return cmp.Compare(a, b)
}

// mustItoa formats v as decimal digits. It panics if v < 0.
func mustItoa(v int) string {
	if v < 0 {
		panic(fmt.Sprintf("negative version: %v", v))
	}
	return strconv.Itoa(v)
}

// splitWords returns the dot-separated words of s, or nil if s == "".
func splitWords(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ".")
}

// cleanWords splits s into words, with all empty words discarded.
func cleanWords(s string) []string {
	return slice.Partition(splitWords(s), func(v string) bool { return v != "" })
}

// joinCleanWords returns a copy of s with all empty words removed.
func joinCleanWords(s string) string { return strings.Join(cleanWords(s), ".") }
