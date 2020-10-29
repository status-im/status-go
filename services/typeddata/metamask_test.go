package typeddata

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/eth-node/types"

	signercore "github.com/ethereum/go-ethereum/signer/core"
)

func TestTypedDataSuite(t *testing.T) {
	suite.Run(t, new(TypedDataSuite))
}

type TypedDataSuite struct {
	suite.Suite

	privateKey  *ecdsa.PrivateKey
	typedDataV3 signercore.TypedData
	typedDataV4 signercore.TypedData
}

func (s *TypedDataSuite) SetupTest() {
	pk, err := crypto.ToECDSA(crypto.Keccak256([]byte("cow")))
	s.Require().NoError(err)

	s.privateKey = pk

	s.Require().NoError(json.Unmarshal([]byte(typedDataV3), &s.typedDataV3))
	s.Require().NoError(json.Unmarshal([]byte(typedDataV4), &s.typedDataV4))
}

func (s *TypedDataSuite) TestTypedDataV3() {
	signature, err := SignTypedDataV4(s.typedDataV3, s.privateKey, big.NewInt(1))
	s.Require().NoError(err)
	s.Require().Equal("0x4355c47d63924e8a72e509b65029052eb6c299d53a04e167c5775fd466751c9d07299936d304c153f6443dfa05f40ff007d72911b6f72307f996231605b915621c", types.EncodeHex(signature))
}

func (s *TypedDataSuite) TestTypedDataV4() {

	expected := "0xfabfe1ed996349fc6027709802be19d047da1aa5d6894ff5f6486d92db2e6860"
	actual := s.typedDataV4.TypeHash("Person")
	s.Require().Equal(expected, actual.String())

	fromTypedData := signercore.TypedData{}
	s.Require().NoError(json.Unmarshal([]byte(fromJSON), &fromTypedData))

	actual, err := s.typedDataV4.HashStruct("Person", fromTypedData.Message)
	s.Require().NoError(err)
	expected = "0x9b4846dd48b866f0ac54d61b9b21a9e746f921cefa4ee94c4c0a1c49c774f67f"
	s.Require().Equal(expected, actual.String())

	encodedData, err := s.typedDataV4.EncodeData(s.typedDataV4.PrimaryType, s.typedDataV4.Message, 1)
	s.Require().NoError(err)

	expected = "0x4bd8a9a2b93427bb184aca81e24beb30ffa3c747e2a33d4225ec08bf12e2e7539b4846dd48b866f0ac54d61b9b21a9e746f921cefa4ee94c4c0a1c49c774f67fefa62530c7ae3a290f8a13a5fc20450bdb3a6af19d9d9d2542b5a94e631a9168b5aadf3154a261abdd9086fc627b61efca26ae5702701d05cd2305f7c52a2fc8"
	s.Require().Equal(expected, encodedData.String())

	actual, err = s.typedDataV4.HashStruct(s.typedDataV4.PrimaryType, s.typedDataV4.Message)
	s.Require().NoError(err)

	expected = "0x99b97a26b830a26d5ca27ced87ba4d73c6276a2b8315656882a771d6f98b01f3"
	s.Require().Equal(expected, actual.String())

	signature, err := SignTypedDataV4(s.typedDataV4, s.privateKey, big.NewInt(1))
	s.Require().NoError(err)
	s.Require().Equal("0xf632e305033e23de75545fcdd0a481d83d9d41954e12c07004327cddf4e3c762757652b04d11dd022e0018e6160723f322d0b4bd9b41c87db93755405f5548391b", types.EncodeHex(signature))
}

const typedDataV3 = `
{
	"types": {
		"EIP712Domain": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "version",
				"type": "string"
			},
			{
				"name": "chainId",
				"type": "uint256"
			},
			{
				"name": "verifyingContract",
				"type": "address"
			}
		],
		"Person": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "wallet",
				"type": "address"
			}
		],
		"Mail": [
			{
				"name": "from",
				"type": "Person"
			},
			{
				"name": "to",
				"type": "Person"
			},
			{
				"name": "contents",
				"type": "string"
			}
		]
	},
	"primaryType": "Mail",
	"domain": {
		"name": "Ether Mail",
		"version": "1",
		"chainId": "1",
		"verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
	},
	"message": {
		"from": {
			"name": "Cow",
			"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
		},
		"to": {
			"name": "Bob",
			"wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
		},
		"contents": "Hello, Bob!"
	}
}
`

const typedDataV4 = `
{
	"types": {
		"EIP712Domain": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "version",
				"type": "string"
			},
			{
				"name": "chainId",
				"type": "uint256"
			},
			{
				"name": "verifyingContract",
				"type": "address"
			}
		],
		"Person": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "wallets",
				"type": "address[]"
			}
		],
		"Mail": [
			{
				"name": "from",
				"type": "Person"
			},
			{
				"name": "to",
				"type": "Person[]"
			},
			{
				"name": "contents",
				"type": "string"
			}
		],
		"Group": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "members",
				"type": "Person[]"
			}
		]
	},
	"domain": {
		"name": "Ether Mail",
		"version": "1",
		"chainId": "1",
		"verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
	},
	"primaryType": "Mail",
	"message": {
		"from": {
			"name": "Cow",
			"wallets": [
				"0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826",
				"0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF"
			]
		},
		"to": [
			{
				"name": "Bob",
				"wallets": [
					"0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB",
					"0xB0BdaBea57B0BDABeA57b0bdABEA57b0BDabEa57",
					"0xB0B0b0b0b0b0B000000000000000000000000000"
				]
			}
		],
		"contents": "Hello, Bob!"
	}
}
`
const fromJSON = `
{
	"types": {
		"EIP712Domain": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "version",
				"type": "string"
			},
			{
				"name": "chainId",
				"type": "uint256"
			},
			{
				"name": "verifyingContract",
				"type": "address"
			}
		],
		"Person": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "wallets",
				"type": "address[]"
			}
		],
		"Mail": [
			{
				"name": "from",
				"type": "Person"
			},
			{
				"name": "to",
				"type": "Person[]"
			},
			{
				"name": "contents",
				"type": "string"
			}
		],
		"Group": [
			{
				"name": "name",
				"type": "string"
			},
			{
				"name": "members",
				"type": "Person[]"
			}
		]
	},
	"domain": {
		"name": "Ether Mail",
		"version": "1",
		"chainId": "1",
		"verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
	},
	"primaryType": "Mail",
	"message": {
			"name": "Cow",
			"wallets": [
				"0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826",
				"0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF"
			]
		}
	}
`
