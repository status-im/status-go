package bigint

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVarHexBigInt_UnmarshalJSON_ValidCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *big.Int
	}{
		{
			name:     "simple hex value",
			input:    "0x123",
			expected: big.NewInt(291),
		},
		{
			name:     "hex with leading zeros",
			input:    "0x0001",
			expected: big.NewInt(1),
		},
		{
			name:     "zero value",
			input:    "0x0",
			expected: big.NewInt(0),
		},
		{
			name:     "empty hex after prefix",
			input:    "0x",
			expected: big.NewInt(0),
		},
		{
			name:  "large hex value",
			input: "0x09abc5177d51c36ef4c6a36197d023b60d8fec0100000000000001000000000a",
			expected: func() *big.Int {
				i := new(big.Int)
				i.SetString("09abc5177d51c36ef4c6a36197d023b60d8fec0100000000000001000000000a", 16)
				return i
			}(),
		},
		{
			name:     "uppercase X in prefix",
			input:    "0X123",
			expected: big.NewInt(291),
		},
		{
			name:     "max uint64",
			input:    "0xffffffffffffffff",
			expected: func() *big.Int { i := new(big.Int); i.SetUint64(^uint64(0)); return i }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputBytes, err := json.Marshal(tt.input)
			require.NoError(t, err)

			result := new(VarHexBigInt)
			err = result.UnmarshalJSON(inputBytes)

			require.NoError(t, err)
			require.NotNil(t, result.Int)
			require.Equal(t, tt.expected, result.Int, "expected %s, got %s", tt.expected.String(), result.Int.String())
		})
	}
}

func TestVarHexBigInt_UnmarshalJSON_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError string
	}{
		{
			name:        "empty string",
			input:       "",
			expectError: "hex string too short",
		},
		{
			name:        "single character",
			input:       "0",
			expectError: "hex string too short",
		},
		{
			name:        "missing 0x prefix",
			input:       "123",
			expectError: "hex string must start with 0x prefix",
		},
		{
			name:        "invalid hex characters",
			input:       "0xGHI",
			expectError: "not a valid big integer",
		},
		{
			name:        "wrong prefix",
			input:       "1x123",
			expectError: "hex string must start with 0x prefix",
		},
		{
			name:        "prefix with wrong second character",
			input:       "0y123",
			expectError: "hex string must start with 0x prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputBytes, err := json.Marshal(tt.input)
			require.NoError(t, err)

			result := new(VarHexBigInt)
			err = result.UnmarshalJSON(inputBytes)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectError)
		})
	}
}

func TestVarHexBigInt_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    *big.Int
		expected string
	}{
		{
			name:     "zero",
			input:    big.NewInt(0),
			expected: "0x0",
		},
		{
			name:     "simple value",
			input:    big.NewInt(291),
			expected: "0x123",
		},
		{
			name: "large value",
			input: func() *big.Int {
				i := new(big.Int)
				i.SetString("09abc5177d51c36ef4c6a36197d023b60d8fec0100000000000001000000000a", 16)
				return i
			}(),
			expected: "0x9abc5177d51c36ef4c6a36197d023b60d8fec0100000000000001000000000a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vhbi := &VarHexBigInt{Int: tt.input}
			result, err := vhbi.MarshalJSON()

			require.NoError(t, err)
			require.Equal(t, "\""+tt.expected+"\"", string(result))
		})
	}
}

func TestVarHexBigInt_RoundTrip(t *testing.T) {
	tests := []string{
		"0x0",
		"0x123",
		"0xffffffffffffffff",
		"0x09abc5177d51c36ef4c6a36197d023b60d8fec0100000000000001000000000a",
	}

	for _, hexStr := range tests {
		t.Run(hexStr, func(t *testing.T) {
			// Unmarshal
			inputBytes, err := json.Marshal(hexStr)
			require.NoError(t, err)

			vhbi := new(VarHexBigInt)
			err = vhbi.UnmarshalJSON(inputBytes)
			require.NoError(t, err)

			// Marshal back
			result, err := vhbi.MarshalJSON()
			require.NoError(t, err)

			// Unmarshal again to compare values
			vhbi2 := new(VarHexBigInt)
			err = vhbi2.UnmarshalJSON(result)
			require.NoError(t, err)

			require.Equal(t, vhbi.Int, vhbi2.Int)
		})
	}
}
