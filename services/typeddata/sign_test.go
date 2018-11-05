package typeddata

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/services/typeddata/eip712example"
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

func TestInteroparableWithSolidity(t *testing.T) {
	key, _ := crypto.GenerateKey()
	testaddr := crypto.PubkeyToAddress(key.PublicKey)
	genesis := core.GenesisAlloc{
		testaddr: {Balance: new(big.Int).SetInt64(math.MaxInt64)},
	}
	backend := backends.NewSimulatedBackend(genesis, math.MaxInt64)
	opts := bind.NewKeyedTransactor(key)
	_, _, example, err := eip712example.DeployExample(opts, backend)
	require.NoError(t, err)
	backend.Commit()

	domainSol, err := example.DOMAINSEPARATOR(nil)
	require.NoError(t, err)
	mailSol, err := example.MAIL(nil)
	require.NoError(t, err)

	mytypes := Types{
		eip712Domain: []Field{
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"Person": []Field{
			{Name: "name", Type: "string"},
			{Name: "wallet", Type: "address"},
		},
		"Mail": []Field{
			{Name: "from", Type: "Person"},
			{Name: "to", Type: "Person"},
			{Name: "contents", Type: "string"},
		},
	}
	domain := map[string]json.RawMessage{
		"name":              json.RawMessage(`"Ether Mail"`),
		"version":           json.RawMessage(`"1"`),
		"chainId":           json.RawMessage("1"),
		"verifyingContract": json.RawMessage(`"0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"`),
	}
	msg := map[string]json.RawMessage{
		"from":     json.RawMessage(fromWallet),
		"to":       json.RawMessage(toWallet),
		"contents": json.RawMessage(`"Hello, Bob!"`),
	}
	typed := TypedData{
		Types:       mytypes,
		PrimaryType: "Mail",
		Domain:      domain,
		Message:     msg,
	}

	domainHash, err := hashStruct(eip712Domain, typed.Domain, typed.Types)
	require.NoError(t, err)
	require.Equal(t, domainSol[:], domainHash[:])

	mailHash, err := hashStruct(typed.PrimaryType, typed.Message, typed.Types)
	require.NoError(t, err)
	require.Equal(t, mailSol[:], mailHash[:])

	signature, err := Sign(typed, key, big.NewInt(1))
	require.NoError(t, err)
	require.Len(t, signature, 65)

	r := [32]byte{}
	copy(r[:], signature[:32])
	s := [32]byte{}
	copy(s[:], signature[32:64])
	v := signature[64]
	tx, err := example.Verify(opts, v, r, s)
	require.NoError(t, err)
	backend.Commit()
	receipt, err := bind.WaitMined(context.TODO(), backend, tx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
}
