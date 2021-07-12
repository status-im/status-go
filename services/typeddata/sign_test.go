package typeddata

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	fromWallet = `
{
  "name":   "Cow",
  "wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
}
`
	toWallet = `
{
  "name":   "Bob",
  "wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
}
`
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
			map[string]json.RawMessage{chainIDKey: json.RawMessage("1")},
		},
		{
			"ChainIDNotAnInt",
			map[string]json.RawMessage{chainIDKey: json.RawMessage(`"aa"`)},
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
