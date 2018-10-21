package typeddata

import (
	"context"
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

func TestChainIDValidation(t *testing.T) {
	chain := big.NewInt(10)
	type testCase struct {
		description string
		domain      map[string]interface{}
		err         string
	}
	for _, tc := range []testCase{
		{
			"ChainIDMismatch",
			map[string]interface{}{chainIDKey: 1},
			"chainId 1 doesn't match selected chain 10",
		},
		{
			"ChainIDNotAnInt",
			map[string]interface{}{chainIDKey: "10"},
			"chainId is not an int",
		},
		{
			"NoChainIDKey",
			nil,
			"domain misses chain key chainId",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			typed := TypedData{Domain: tc.domain}
			_, err := Sign(typed, nil, chain)
			require.EqualError(t, err, tc.err)
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
	domain := map[string]interface{}{
		"name":              "Ether Mail",
		"version":           "1",
		"chainId":           1,
		"verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
	}
	msg := map[string]interface{}{
		"from": map[string]interface{}{
			"name":   "Cow",
			"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826",
		},
		"to": map[string]interface{}{
			"name":   "Bob",
			"wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB",
		},
		"contents": "Hello, Bob!",
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
