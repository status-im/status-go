package typeddata

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChainIDValidation(t *testing.T) {
	chain := big.NewInt(10)
	type testCase struct {
		description string
		domain      map[string]json.RawMessage
	}
	for _, tc := range []testCase{
		{
			"ChainIDMismatch",
			map[string]json.RawMessage{ChainIDKey: json.RawMessage("1")},
		},
		{
			"ChainIDNotAnInt",
			map[string]json.RawMessage{ChainIDKey: json.RawMessage(`"aa"`)},
		},
		{
			"NoChainIDKey",
			nil,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			typed := TypedData{Domain: tc.domain}
			_, err := Sign(typed, nil, chain)
			require.Error(t, err)
		})
	}
}
