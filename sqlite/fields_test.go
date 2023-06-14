package sqlite

import (
	"math/big"
	"testing"
)

func strToPtr(s string) *string {
	res := new(string)
	*res = s
	return res
}

func TestBigIntToPadded128BitsStr(t *testing.T) {
	testCases := []struct {
		name     string
		input    *big.Int
		expected *string
	}{
		{
			name:     "case small",
			input:    big.NewInt(123456),
			expected: strToPtr("0000000000000000000000000001e240"),
		},
		{
			name:     "case zero",
			input:    big.NewInt(0),
			expected: strToPtr("00000000000000000000000000000000"),
		},
		{
			name:     "case very large",
			input:    new(big.Int).Exp(big.NewInt(10), big.NewInt(26), nil),
			expected: strToPtr("000000000052b7d2dcc80cd2e4000000"),
		},
		{
			name:     "case max",
			input:    new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1)),
			expected: strToPtr("ffffffffffffffffffffffffffffffff"),
		},
		{
			name:     "case 3",
			input:    nil,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := BigIntToPadded128BitsStr(tc.input)
			if result != nil && tc.expected != nil {
				if *result != *tc.expected {
					t.Errorf("expected %s, got %s", *tc.expected, *result)
				}
			} else if result != nil || tc.expected != nil {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestInt64ToPadded128BitsStr(t *testing.T) {
	testCases := []struct {
		name     string
		input    int64
		expected *string
	}{
		{
			name:     "case nonzero",
			input:    123456,
			expected: strToPtr("0000000000000000000000000001e240"),
		},
		{
			name:     "case zero",
			input:    0,
			expected: strToPtr("00000000000000000000000000000000"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Int64ToPadded128BitsStr(tc.input)
			if result != nil && tc.expected != nil {
				if *result != *tc.expected {
					t.Errorf("expected %s, got %s", *tc.expected, *result)
				}
			} else if result != nil || tc.expected != nil {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}
