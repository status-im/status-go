package typeddata

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
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
		message     map[string]json.RawMessage
		types       Types
		target      string
		result      func(testCase) common.Hash
	}

	bytes32, _ := abi.NewType("bytes32")
	addr, _ := abi.NewType("address")
	boolT, _ := abi.NewType("bool")

	for _, tc := range []testCase{
		{
			"HexAddressConvertedToBytes",
			map[string]json.RawMessage{"wallet": json.RawMessage(`"0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"`)},
			Types{"A": []Field{{Name: "wallet", Type: "address"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: addr}}
				typehash := typeHash(tc.target, tc.types)
				var data common.Address
				assert.NoError(t, json.Unmarshal(tc.message["wallet"], &data))
				packed, _ := args.Pack(typehash, data)
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"StringHashed",
			map[string]json.RawMessage{"name": json.RawMessage(`"AAA"`)},
			Types{"A": []Field{{Name: "name", Type: "string"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: bytes32}}
				typehash := typeHash(tc.target, tc.types)
				var data string
				assert.NoError(t, json.Unmarshal(tc.message["name"], &data))
				packed, _ := args.Pack(typehash, crypto.Keccak256Hash([]byte(data)))
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"BytesHashed",
			map[string]json.RawMessage{"name": json.RawMessage(`"0x010203"`)}, // []byte{1,2,3}
			Types{"A": []Field{{Name: "name", Type: "bytes"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: bytes32}}
				typehash := typeHash(tc.target, tc.types)
				var data hexutil.Bytes
				assert.NoError(t, json.Unmarshal(tc.message["name"], &data))
				packed, _ := args.Pack(typehash, crypto.Keccak256Hash(data))
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"FixedBytesAsIs",
			map[string]json.RawMessage{"name": json.RawMessage(`"0x010203"`)}, // []byte{1,2,3}
			Types{"A": []Field{{Name: "name", Type: "bytes32"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: bytes32}}
				typehash := typeHash(tc.target, tc.types)
				var data hexutil.Bytes
				assert.NoError(t, json.Unmarshal(tc.message["name"], &data))
				rst := [32]byte{}
				copy(rst[:], data)
				packed, _ := args.Pack(typehash, rst)
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"BoolAsIs",
			map[string]json.RawMessage{"flag": json.RawMessage("true")},
			Types{"A": []Field{{Name: "flag", Type: "bool"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: boolT}}
				typehash := typeHash(tc.target, tc.types)
				var data bool
				assert.NoError(t, json.Unmarshal(tc.message["flag"], &data))
				packed, _ := args.Pack(typehash, data)
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"Int32Uint32AsIs",
			map[string]json.RawMessage{"I": json.RawMessage("-10"), "UI": json.RawMessage("10")},
			Types{"A": []Field{{Name: "I", Type: "int32"}, {Name: "UI", Type: "uint32"}}},
			"A",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: int256Type}, {Type: int256Type}}
				typehash := typeHash(tc.target, tc.types)
				packed, _ := args.Pack(typehash, big.NewInt(-10), big.NewInt(10))
				return crypto.Keccak256Hash(packed)
			},
		},
		{
			"SignedUnsignedIntegersBiggerThen64",
			map[string]json.RawMessage{
				"i128":  json.RawMessage("1"),
				"i256":  json.RawMessage("1"),
				"ui128": json.RawMessage("1"),
				"ui256": json.RawMessage("1"),
			},
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
			map[string]json.RawMessage{"a": json.RawMessage(`{"name":"AAA"}`)},
			Types{"A": []Field{{Name: "name", Type: "string"}}, "Z": []Field{{Name: "a", Type: "A"}}},
			"Z",
			func(tc testCase) common.Hash {
				args := abi.Arguments{{Type: bytes32}, {Type: bytes32}}
				zhash := typeHash(tc.target, tc.types)
				ahash := typeHash("A", tc.types)
				var A map[string]string
				assert.NoError(t, json.Unmarshal(tc.message["a"], &A))
				apacked, _ := args.Pack(ahash, crypto.Keccak256Hash([]byte(A["name"])))
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
		message     map[string]json.RawMessage
		types       Types
		target      string
	}

	for _, tc := range []testCase{
		{
			"FailedUnmxarshalAsAString",
			map[string]json.RawMessage{"a": json.RawMessage("1")},
			Types{"A": []Field{{Name: "name", Type: "string"}}},
			"A",
		},
		{
			"FailedUnmarshalToHexBytesToABytes",
			map[string]json.RawMessage{"a": {1, 2, 3}},
			Types{"A": []Field{{Name: "name", Type: "bytes"}}},
			"A",
		},
		{
			"CompositeTypeIsNotAnObject",
			map[string]json.RawMessage{"a": json.RawMessage(`"AAA"`)},
			Types{"A": []Field{{Name: "name", Type: "string"}}, "Z": []Field{{Name: "a", Type: "A"}}},
			"Z",
		},
		{
			"CompositeTypesFailed",
			map[string]json.RawMessage{"a": json.RawMessage(`{"name":10}`)},
			Types{"A": []Field{{Name: "name", Type: "string"}}, "Z": []Field{{Name: "a", Type: "A"}}},
			"Z",
		},
		{
			"ArraysNotSupported",
			map[string]json.RawMessage{"a": json.RawMessage("[1,2]")},
			Types{"A": []Field{{Name: "name", Type: "int8[2]"}}},
			"A",
		},
		{
			"SlicesNotSupported",
			map[string]json.RawMessage{"a": json.RawMessage("[1,2]")},
			Types{"A": []Field{{Name: "name", Type: "int[]"}}},
			"A",
		},
		{
			"FailedToUnmarshalInteger",
			map[string]json.RawMessage{"a": json.RawMessage("x00x")},
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
