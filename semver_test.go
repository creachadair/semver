package semver_test

import (
	"fmt"
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
