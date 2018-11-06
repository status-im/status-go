package typeddata

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalFull(t *testing.T) {
	data := `
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
    "chainId": 1,
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
	var typed TypedData
	require.NoError(t, json.Unmarshal([]byte(data), &typed))
}

func TestValidateField(t *testing.T) {
	f := Field{}
	require.EqualError(t, f.Validate(), "`name` is required")
	f.Name = "name"
	require.EqualError(t, f.Validate(), "`type` is required")
	f.Type = "type"
	require.NoError(t, f.Validate())
}

func TestValidateTypedData(t *testing.T) {
	d := TypedData{Types: Types{}}
	require.EqualError(t, d.Validate(), "`EIP712Domain` must be in `types`")
	d.Types[eip712Domain] = []Field{}
	require.EqualError(t, d.Validate(), "`primaryType` is required")
	d.PrimaryType = "primary"
	d.Types[d.PrimaryType] = []Field{}
	require.EqualError(t, d.Validate(), "`domain` is required")
	d.Domain = map[string]json.RawMessage{}
	require.EqualError(t, d.Validate(), "`message` is required")
	d.Message = map[string]json.RawMessage{}
	require.NoError(t, d.Validate())
	d.Types[d.PrimaryType] = append(d.Types[d.PrimaryType], Field{Name: "name"})
	require.EqualError(t, d.Validate(), "field 0 from type `primary` is invalid: `type` is required")
	d.Types[d.PrimaryType][0].Type = "tttt"
	require.NoError(t, d.Validate())
}
