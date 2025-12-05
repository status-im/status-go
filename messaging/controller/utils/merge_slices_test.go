package utils

import (
	"bytes"
	"testing"
)

func TestMergeByteSlices(t *testing.T) {
	tests := []struct {
		name     string
		slices   [][]byte
		expected []byte
	}{
		{
			name:     "NilInput",
			slices:   nil,
			expected: nil,
		},
		{
			name:     "EmptyInput",
			slices:   [][]byte{},
			expected: nil,
		},
		{
			name: "LexicographicOrdering",
			slices: [][]byte{
				[]byte("c"),
				[]byte("ab"),
				[]byte("aa"),
			},
			expected: []byte("aaabc"),
		},
		{
			name: "HandlesNilAndDuplicateSlices",
			slices: [][]byte{
				[]byte("bb"),
				nil,
				[]byte("aa"),
				[]byte("aa"),
			},
			expected: []byte("aaaabb"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := MergeByteSlices(tc.slices)
			if tc.expected == nil {
				if got != nil {
					t.Fatalf("expected nil, got %q", string(got))
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %q, got nil", string(tc.expected))
			}
			if !bytes.Equal(got, tc.expected) {
				t.Fatalf("expected %q, got %q", string(tc.expected), string(got))
			}
		})
	}
}
