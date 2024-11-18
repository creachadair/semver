// Copyright (C) 2024 Michael J. Fromberger. All Rights Reserved.

package semver_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/creachadair/semver"
)

func mustParse(t *testing.T, s string) semver.V {
	t.Helper()
	v, err := semver.Parse(s)
	if err != nil {
		t.Fatalf("Parse %q: %v", s, err)
	}
	return v
}

func TestOrder(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		// Basic lexicographic ordering on major.minor.patch
		// Includes examples from https://semver.org/#spec-item-11
		{"0.0.0", "0.0.0", 0},
		{"0.0.1", "0.0.0", 1},
		{"0.0.2", "0.0.3", -1},
		{"0.1.2", "0.0.3", 1},
		{"0.1.2", "1.0.3", -1},
		{"2.0.5", "1.30.90", 1},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "2.1.0", -1},
		{"2.1.0", "2.1.1", -1},

		// Core order precedes non-core comparisons.
		{"1.1.5", "1.1.2-rel-blah", 1},
		{"1.1.5-rel-blah", "1.1.2", 1},

		// Pre-releases are compared lexicographically.
		// Examples from https://semver.org/#spec-item-11
		{"1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"1.0.0-alpha.1", "1.0.0-alpha.beta", -1},
		{"1.0.0-alpha.beta", "1.0.0-beta", -1},
		{"1.0.0-beta", "1.0.0-beta.2", -1},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0-beta.11", "1.0.0-rc.1", -1},
		{"1.0.0-rc.1", "1.0.0", -1},
		{"1.57.0", "1.57.0-beta1", 1},

		// Build metadata do not affect comparison.
		{"1.2.3-four+five.six", "1.2.3-four", 0},
		{"1.2.3-four", "1.2.3-four+five", 0},
		{"1.2.3-four+five", "1.2.3-four+six.seven", 0},
	}
	for _, tc := range tests {
		name := fmt.Sprintf("Compare/%s/%s", tc.a, tc.b)
		t.Run(name, func(t *testing.T) {
			a, b := mustParse(t, tc.a), mustParse(t, tc.b)

			got := semver.Compare(a, b)
			if got != tc.want {
				t.Errorf("Compare %v / %v: got %v, want %v", tc.a, tc.b, got, tc.want)
			}

			// Verify that the comparison works in both directions.
			if inv := semver.Compare(b, a); inv != -tc.want {
				t.Errorf("Compare %v / %v: got %v, want %v", tc.b, tc.a, inv, -tc.want)
			}

			// Check the comparison methods.
			if tc.want < 0 && !a.Before(b) {
				t.Errorf("Want [%v].Before(%v), but it is not", tc.a, tc.b)
			}
			if tc.want == 0 && !a.Equiv(b) {
				t.Errorf("Want [%v].Equiv(%v), but it is not", tc.a, tc.b)
			}
			if tc.want > 0 && !a.After(b) {
				t.Errorf("Want [%v].After(%v), but it is not", tc.a, tc.b)
			}
		})
	}

	t.Run("Zero", func(t *testing.T) {
		zero := mustParse(t, "0.0.0")
		if got := semver.Compare(zero, semver.V{}); got != 0 {
			t.Fatalf("Compare to zero V: got %v, want 0", got)
		}
	})
	t.Run("ZeroMeta", func(t *testing.T) {
		zeroMeta := mustParse(t, "0.0.0+fizz.bang")
		if got := semver.Compare(zeroMeta, semver.V{}); got != 0 {
			t.Errorf("Compare %v to zero V: got %v, want 0", zeroMeta, got)
		}
	})
}

func TestFormat(t *testing.T) {
	tests := []struct {
		input semver.V
		want  string
	}{
		{semver.V{}, "0.0.0"},
		{semver.New(0, 0, 0), "0.0.0"},
		{semver.New(1, 2, 120), "1.2.120"},
		{semver.New(1, 0, 2).WithBuild("unstable"), "1.0.2+unstable"},
		{semver.New(5, 1, 0).WithPreRelease("rc1.c030"), "5.1.0-rc1.c030"},
		{semver.New(0, 0, 9).WithBuild("custom").WithPreRelease("alpha5.2"), "0.0.9-alpha5.2+custom"},
		{semver.MustParse("1.2.3-four.five+six").Core(), "1.2.3"},
		{semver.MustParse("1.2.3+four").WithBuild(""), "1.2.3"},
		{semver.V{}.WithPreRelease("rc1.2.3-4"), "0.0.0-rc1.2.3-4"},
	}
	for _, tc := range tests {
		if got := tc.input.String(); got != tc.want {
			t.Errorf("String %#v: got %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "invalid version syntax"},
		{"1", "invalid version syntax"},
		{"1.0", "invalid version syntax"},
		{"..", "major: not a number"},
		{"1..", "minor: not a number"},
		{"1.1.", "patch: not a number"},
		{"q.0.3", "major: not a number"},
		{"1.q.0", "minor: not a number"},
		{"1.0.q", "patch: not a number"},
		{"05.0.0", "major: leading zeroes"},
		{"1.06.1", "minor: leading zeroes"},
		{"1.2.07", "patch: leading zeroes"},
		{"2.4.0-", "empty pre-release"},
		{"1.0.0+", "empty build"},
		{"2.4.0-ok+", "empty build"},
		{"0.1.2-a..b", `pre-release "a..b": empty word (2)`},
		{"1.2.3+a.b.", `build "a.b.": empty word (3)`},
		{"4.5.6-ok.123+.a", `build ".a": empty word (1)`},
		{"1.0.0-bo?gus", `pre-release "bo?gus": invalid char (1)`},
		{"1.4.0+is.b@d", `build "is.b@d": invalid char (2)`},
	}
	for _, tc := range tests {
		got, err := semver.Parse(tc.input)
		if err == nil {
			t.Errorf("Parse %q: got (%v, nil), want %q", tc.input, got, tc.want)
		} else if es := err.Error(); !strings.Contains(es, tc.want) {
			t.Errorf("Parse %q: got %v, want %q", tc.input, err, tc.want)
		}
	}
}

func TestClean(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		// Strings without a major version are not modified.
		{"", ""},
		{".", "."},
		{".1.3", ".1.3"},
		{"v..+b-q", "v..+b-q"},
		{" .5... ", " .5... "},
		{" v-rc1", " v-rc1"},

		// Lesser versions are stubbed to zero.
		{"1", "1.0.0"},
		{"1.5", "1.5.0"},
		{"3.1.4", "3.1.4"},

		// Leading and trailing spaces and leading "v" are removed.
		{" 1 ", "1.0.0"},
		{"v1 ", "1.0.0"},
		{" 1.5 ", "1.5.0"},
		{"v2.79\n", "2.79.0"},
		{"\nv6.5.4\t", "6.5.4"},
		{" v2\t", "2.0.0"},
		{"\tv3.14\r\n", "3.14.0"},

		// Empty fragments are discarded, in various combinations.
		{"1-", "1.0.0"},
		{"1+", "1.0.0"},
		{"1-+", "1.0.0"},
		{"1-foo+", "1.0.0-foo"},
		{"1-+bar", "1.0.0+bar"},
		{"1+bar-", "1.0.0+bar-"},
		{"1.2-a..b+c-d.e.", "1.2.0-a.b+c-d.e"},
	}
	for _, tc := range tests {
		got := semver.Clean(tc.input)
		if got != tc.want {
			t.Errorf("Clean %q: got %q, want %q", tc.input, got, tc.want)
		}
	}
}
