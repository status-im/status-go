package validate

import "testing"

func TestIsLikelyENSName(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"alice.eth", true},
		{"a.b", true},
		{"alice.xyz", true},
		{"sub.alice.eth", true},
		{"", false},
		{"a", false},
		{"ab", false},
		{"abc", false}, // no dot
		{"a.b", true},  // exactly 3 chars with dot
		{"a.", false},  // 2 chars: fails the length predicate
		{"noname", false},
	}
	for _, tc := range cases {
		if got := IsLikelyENSName(tc.in); got != tc.want {
			t.Errorf("IsLikelyENSName(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
