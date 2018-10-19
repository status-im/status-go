package typeddata

import (
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/services/typeddata/eip712example"
	"github.com/stretchr/testify/require"
)

func TestTypeString(t *testing.T) {
	types := Types{}
	types["Person"] = []Field{{Name: "name", Type: "string"}, {Name: "wallet", Type: "address"}}
	types["Mail"] = []Field{{Name: "from", Type: "Person"}, {Name: "to", Type: "Person"}}
	rst, err := typeString("Person", types)
	require.NoError(t, err)
	require.Equal(t, "Person(string name,address wallet)", rst)
	rst, err = typeString("Mail", types)
	require.NoError(t, err)
	require.Equal(t, "Mail(Person from,Person to)Person(string name,address wallet)", rst)
}

func TestEncodeData(t *testing.T) {
	types := Types{}
	types["Person"] = []Field{{Name: "name", Type: "string"}, {Name: "wallet", Type: "address"}}
	types["Mail"] = []Field{{Name: "from", Type: "Person"}, {Name: "to", Type: "Person"}}
	message := map[string]interface{}{
		"from": map[string]interface{}{
			"name":   "Cow",
			"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826",
		},
		"to": map[string]interface{}{
			"name":   "Bob",
			"wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB",
		},
	}
	rst, err := encodeData("Person", message["from"].(map[string]interface{}), types)
	require.NoError(t, err)
	bytes32, _ := abi.NewType("bytes32")
	addr, _ := abi.NewType("address")
	args := abi.Arguments{{Type: bytes32}, {Type: bytes32}, {Type: addr}}
	typehash, err := typeHash("Person", types)
	require.NoError(t, err)
	person := message["from"].(map[string]interface{})
	expected, err := args.Pack(typehash,
		crypto.Keccak256Hash([]byte(person["name"].(string))),
		common.HexToAddress(person["wallet"].(string)))
	require.NoError(t, err)
	require.Equal(t, crypto.Keccak256Hash(expected), rst)
}

func TestInteroparableWithSolidity(t *testing.T) {
	key, _ := crypto.GenerateKey()
	testaddr := crypto.PubkeyToAddress(key.PublicKey)
	genesis := core.GenesisAlloc{
		testaddr: {Balance: new(big.Int).SetInt64(math.MaxInt64)},
	}
	backend := backends.NewSimulatedBackend(genesis, math.MaxInt64)
	_, _, example, err := eip712example.DeployExample(bind.NewKeyedTransactor(key), backend)
	require.NoError(t, err)
	backend.Commit()
	domainSol, err := example.DOMAINSEPARATOR(nil)
	require.NoError(t, err)
	types := Types{eip712Domain: []Field{
		{Name: "name", Type: "string"},
		{Name: "version", Type: "string"},
		{Name: "chainId", Type: "uint256"},
		{Name: "verifyingContract", Type: "address"},
	}}
	msg := map[string]interface{}{
		"name":              "Ether Mail",
		"version":           "1",
		"chainId":           1,
		"verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
	}
	domain, err := encodeData(eip712Domain, msg, types)
	require.NoError(t, err)
	require.Equal(t, domainSol[:], domain[:])
}
