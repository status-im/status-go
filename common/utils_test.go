package common

import (
	"testing"
)

func TestTruncateWithDotN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		n        int
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			n:        4,
			expected: "",
		},
		{
			name:     "string shorter than n",
			input:    "123",
			n:        4,
			expected: "123",
		},
		{
			name:     "string equal to n",
			input:    "1234",
			n:        4,
			expected: "1234",
		},
		{
			name:     "hex string with n=4 (even split: 2+2)",
			input:    "0x123456789",
			n:        4,
			expected: "0x..89",
		},
		{
			name:     "hex string with n=3 (odd split: 2+1)",
			input:    "0x123456789",
			n:        3,
			expected: "0x..9",
		},
		{
			name:     "hex string with n=5 (odd split: 3+2)",
			input:    "0x123456789",
			n:        5,
			expected: "0x1..89",
		},
		{
			name:     "normal string with n=3 (odd split: 2+1)",
			input:    "acdkef099",
			n:        3,
			expected: "ac..9",
		},
		{
			name:     "hex address with n=6 (even split: 3+3)",
			input:    "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
			n:        6,
			expected: "0xd..045",
		},
		{
			name:     "very short n=2 (even split: 1+1)",
			input:    "abcdef",
			n:        2,
			expected: "a..f",
		},
		{
			name:     "n=1 adjusted to minimum 2 (even split: 1+1)",
			input:    "abcdef",
			n:        1,
			expected: "a..f",
		},
		{
			name:     "negative n",
			input:    "test",
			n:        -1,
			expected: "test",
		},
		{
			name:     "zero n",
			input:    "test",
			n:        0,
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateWithDotN(tt.input, tt.n)
			if result != tt.expected {
				t.Errorf("TruncateWithDotN(%q, %d) = %q, want %q",
					tt.input, tt.n, result, tt.expected)
			}
		})
	}
}
