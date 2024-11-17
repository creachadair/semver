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
	"regexp"
	"slices"
	"strconv"
	"strings"
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
	return V{major: mustFmt(major), minor: mustFmt(minor), patch: mustFmt(patch)}
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

// Core returns a copy of v with its pre-release and build metadata cleared,
// corresponding to the "core" version ID (major.minor.patch).
func (v V) Core() V { v.pre = nil; v.build = nil; return v }

// PreRelease reports the pre-release string, if present.
// The resulting string does not include the "-" prefix.
func (v V) PreRelease() string { return strings.Join(v.pre, ".") }

// WithPreRelease returns a copy of v with its pre-release ID set.
// If id == "", the resulting version has no pre-release ID.
func (v V) WithPreRelease(id string) V { v.pre = dotWords(id); return v }

// Build reports the build metadata string, if present.
// The resulting string does not include the "+" prefix.
func (v V) Build() string { return strings.Join(v.build, ".") }

// WithBuild returns a copy of v with its build metadata set.
// If meta == "", the resulting version has no build metadata.
func (v V) WithBuild(meta string) V { v.build = dotWords(meta); return v }

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

// MustParse returns the [V] represented by s, or panics.  This is intended for
// use in program initialization; use [Parse] to check for errors.
func MustParse(s string) V {
	v, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("Parse %q: %v", s, err))
	}
	return v
}

// Parse returns the [V] represented by s.
func Parse(s string) (V, error) {
	m := svRE.FindStringSubmatch(s)
	if m == nil {
		return V{}, errSyntax
	}
	out := V{
		major: m[posMajor],
		minor: m[posMinor],
		patch: m[posPatch],
	}
	if v := m[posPre]; v != "" {
		out.pre = strings.Split(v, ".")
	}
	if v := m[posBuild]; v != "" {
		out.build = strings.Split(v, ".")
	}
	return out, nil
}

const (
	// Grammar: https://semver.org/#backusnaur-form-grammar-for-valid-semver-versions

	expr = `` + // semver → major '.' minor '.' patch ['-' pre-release] ['+' build]
		`(?P<major>` + numericID + `)` +
		`\.(?P<minor>` + numericID + `)` +
		`\.(?P<patch>` + numericID + `)` +
		preRelease +
		build +
		`$`

	// pre-release → pr-id {'.' pr-id}
	preRelease = `(?:-(?P<pre>` + prID + `(?:\.` + prID + `)*))?`
	// build → build-id {'.' build-id}
	build = `(?:\+(?P<build>` + buildID + `(?:\.` + buildID + `)*))?`

	numericID = `(?:0|[1-9]\d*)`                          // zero or positive
	alphaID   = `(?:[-a-zA-Z0-9]*[-a-zA-Z][-a-zA-Z0-9]*)` // at least one non-digit
	prID      = `(?:` + alphaID + `|` + numericID + `)`
	buildID   = `(?:` + alphaID + `|[0-9]+)`
)

var (
	svRE = regexp.MustCompile(expr)

	posMajor = svRE.SubexpIndex("major")
	posMinor = svRE.SubexpIndex("minor")
	posPatch = svRE.SubexpIndex("patch")
	posPre   = svRE.SubexpIndex("pre")
	posBuild = svRE.SubexpIndex("build")

	errSyntax = errors.New("invalid semver format")
)

// mustVal returns the integer represented by s, or panics.
// As a special case, if s == "" it returns 0.
func mustVal(s string) int {
	v, ok := isNum(s)
	if !ok {
		panic(fmt.Sprintf("invalid number %q", s))
	}
	return v
}

// isNum reports whether s comprises only digits, and if so the integer value
// represented by them. As a special case, if s == "" it returns (0, true).
func isNum(s string) (int, bool) {
	v := 0
	for i := range s {
		d := s[i]
		if d >= '0' && d <= '9' {
			v = (v * 10) + int(d-'0')
		} else {
			return -1, false
		}
	}
	return v, true
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

// mustFmt formats v as decimal digits. It panics if v < 0.
func mustFmt(v int) string {
	if v < 0 {
		panic(fmt.Sprintf("negative version: %v", v))
	}
	return strconv.Itoa(v)
}

// dotWords returns the dot-separated words of s, or nil if s == "".
func dotWords(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ".")
}
