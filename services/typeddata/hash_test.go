package typeddata

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestTypeString(t *testing.T) {
	type testCase struct {
		description string
		typeString  string
		types       Types
		target      string
	}
	for _, tc := range []testCase{
		{
			"WithoutDeps",
			"Person(string name,address wallet)",
			Types{"Person": []Field{{Name: "name", Type: "string"}, {Name: "wallet", Type: "address"}}},
			"Person",
		},
		{
			"SingleDep",
			"Mail(Person from,Person to)Person(string name,address wallet)",
			Types{
				"Person": []Field{{Name: "name", Type: "string"}, {Name: "wallet", Type: "address"}},
				"Mail":   []Field{{Name: "from", Type: "Person"}, {Name: "to", Type: "Person"}},
			},
			"Mail",
		},
		{
			"DepsOrdered",
			"Z(A a,B b)A(string name)B(string name)",
			Types{
				"A": []Field{{Name: "name", Type: "string"}},
				"B": []Field{{Name: "name", Type: "string"}},
				"Z": []Field{{Name: "a", Type: "A"}, {Name: "b", Type: "B"}},
			},
			"Z",
		},
		{
			"RecursiveDepsIgnored",
			"Z(A a)A(Z z)",
			Types{
				"A": []Field{{Name: "z", Type: "Z"}},
				"Z": []Field{{Name: "a", Type: "A"}},
			},
			"Z",
		},
	} {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			require.Equal(t, tc.typeString, typeString(tc.target, tc.types))
		})
	}
}

func TestEncodeData(t *testing.T) {
	type testCase struct {
		description string
		message     map[string]interface{}
		types       Types
		target      string
		result      func(testCase) common.Hash
	}

	bytes32, _ := abi.NewType("bytes32")
	addr, _ := abi.NewType("address")
	bool, _ := abi.NewType("bool")

	for _, tc := range []testCase{
		{
			"HexAddressConvertedToBytes",
			map[string]interface{}{"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"},
			Types{"A": []Field{{Name: "wallet", Type: "address"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: addr}}
				typehash := typeHash(tc.target, tc.types)
				packed, _ := args.Pack(typehash,
					common.HexToAddress(tc.message["wallet"].(string)))
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"StringHashed",
			map[string]interface{}{"name": "AAA"},
			Types{"A": []Field{{Name: "name", Type: "string"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: bytes32}}
				typehash := typeHash(tc.target, tc.types)
				packed, _ := args.Pack(typehash,
					crypto.Keccak256Hash([]byte(tc.message["name"].(string))))
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"BytesHashed",
			map[string]interface{}{"name": []byte{1, 2, 3}},
			Types{"A": []Field{{Name: "name", Type: "bytes"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: bytes32}}
				typehash := typeHash(tc.target, tc.types)
				packed, _ := args.Pack(typehash,
					crypto.Keccak256Hash(tc.message["name"].([]byte)))
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"FixedBytesAsIs",
			map[string]interface{}{"name": [32]byte{1, 2, 3}},
			Types{"A": []Field{{Name: "name", Type: "bytes32"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: bytes32}}
				typehash := typeHash(tc.target, tc.types)
				packed, _ := args.Pack(typehash, tc.message["name"])
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"BoolAsIs",
			map[string]interface{}{"flag": true},
			Types{"A": []Field{{Name: "flag", Type: "bool"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: bool}}
				typehash := typeHash(tc.target, tc.types)
				packed, _ := args.Pack(typehash, tc.message["flag"])
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"Int32Uint32AsIs",
			map[string]interface{}{"i": -10, "ui": uint(10)},
			Types{"A": []Field{{Name: "i", Type: "int32"}, {Name: "ui", Type: "uint32"}}},
			"A",
			func(tc testCase) common.Hash {
				intT, _ := abi.NewType("int32")
				uintT, _ := abi.NewType("uint32")
				args := abi.Arguments{{Type: bytes32}, {Type: intT}, {Type: uintT}}
				typehash := typeHash(tc.target, tc.types)
				packed, _ := args.Pack(typehash, int32(tc.message["i"].(int)), uint32(tc.message["ui"].(uint)))
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"SignedUnsignedIntegersBiggerThen64",
			map[string]interface{}{"i128": "1", "i256": "1", "ui128": "1", "ui256": "1"},
			Types{"A": []Field{
				{Name: "i128", Type: "int128"}, {Name: "i256", Type: "int256"},
				{Name: "ui128", Type: "uint128"}, {Name: "ui256", Type: "uint256"},
			}},
			"A",
			func(tc testCase) common.Hash {
				intBig, _ := abi.NewType("int128")
				uintBig, _ := abi.NewType("uint128")
				args := abi.Arguments{{Type: bytes32},
					{Type: intBig}, {Type: intBig}, {Type: uintBig}, {Type: uintBig}}
				typehash := typeHash(tc.target, tc.types)
				val := big.NewInt(1)
				packed, _ := args.Pack(typehash, val, val, val, val)
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"CompositeTypesAreRecursivelyEncoded",
			map[string]interface{}{"a": map[string]interface{}{
				"name": "AAA",
			}},
			Types{"A": []Field{{Name: "name", Type: "string"}}, "Z": []Field{{Name: "a", Type: "A"}}},
			"Z",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: bytes32}}
				zhash := typeHash(tc.target, tc.types)
				ahash := typeHash("A", tc.types)
				apacked, _ := args.Pack(ahash,
					crypto.Keccak256Hash([]byte(tc.message["a"].(map[string]interface{})["name"].(string))))
				packed, _ := args.Pack(zhash, crypto.Keccak256Hash(apacked))
				return crypto.Keccak256Hash(packed)
			},
		},
	} {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			encoded, err := hashStruct(tc.target, tc.message, tc.types)
			require.NoError(t, err)
			require.Equal(t, tc.result(tc), encoded)
		})
	}
}

func TestEncodeDataErrors(t *testing.T) {
	type testCase struct {
		description string
		message     map[string]interface{}
		types       Types
		target      string
	}

	for _, tc := range []testCase{
		{
			"FailedToCastToAString",
			map[string]interface{}{"a": 1},
			Types{"A": []Field{{Name: "name", Type: "string"}}},
			"A",
		},
		{
			"FailedToCastToABytes",
			map[string]interface{}{"a": 1},
			Types{"A": []Field{{Name: "name", Type: "bytes"}}},
			"A",
		},
		{
			"CompositeTypeIsNotAnObject",
			map[string]interface{}{"a": "AAA"},
			Types{"A": []Field{{Name: "name", Type: "string"}}, "Z": []Field{{Name: "a", Type: "A"}}},
			"Z",
		},
		{
			"CompositeTypesFailed",
			map[string]interface{}{"a": map[string]interface{}{
				"name": 10,
			}},
			Types{"A": []Field{{Name: "name", Type: "string"}}, "Z": []Field{{Name: "a", Type: "A"}}},
			"Z",
		},
		{
			"ArraysNotSupported",
			map[string]interface{}{"a": []string{"A", "B"}},
			Types{"A": []Field{{Name: "name", Type: "string[2]"}}},
			"A",
		},
		{
			"SlicesNotSupported",
			map[string]interface{}{"a": []string{"A", "B"}},
			Types{"A": []Field{{Name: "name", Type: "string[]"}}},
			"A",
		},
		{
			"FailedToSetABigInt",
			map[string]interface{}{"a": "x00x"},
			Types{"A": []Field{{Name: "name", Type: "uint256"}}},
			"A",
		},
	} {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			encoded, err := hashStruct(tc.target, tc.message, tc.types)
			require.Error(t, err)
			require.Equal(t, common.Hash{}, encoded)
		})
	}
}
