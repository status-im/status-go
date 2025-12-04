package alchemy_test

import (
	"encoding/json"
	"testing"

	"github.com/status-im/status-go/services/wallet/thirdparty/activity/alchemy"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

func TestGetAssetTransfers(t *testing.T) {
	var response alchemy.GetAssetTranfersResponse
	err := json.Unmarshal([]byte(getAssetTransfersResponseData), &response)
	require.NoError(t, err)
	require.Equal(t, len(response.Transfers), alchemy.MaxAssetTransfersCount)

	firstTransfer := response.Transfers[0]
	require.Equal(t, firstTransfer.BlockNum.Text(16), "14dafb6")
	require.Equal(t, firstTransfer.Hash.String(), "0x7258e2eee7b13bc185c49bc5741555d5252e966e17b6fca1d7b6752a30d9ac4c")
	require.Equal(t, firstTransfer.FromAddress.String(), "0x871def1A32D52e6eD580B2422d51e5A9d5C3B8C1")
	require.Equal(t, firstTransfer.ToAddress.String(), "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	require.Nil(t, firstTransfer.Erc1155Metadata)
	require.Nil(t, firstTransfer.TokenID)
	require.Equal(t, firstTransfer.Asset, "ETH")
	require.Equal(t, firstTransfer.Category, alchemy.TransferCategoryExternal)
}

func TestTransferToCommon(t *testing.T) {
	data := []string{
		getAssetTransfersResponseData,
		getAssetTransfersResponseData2,
	}
	result := []int{
		1000,
		48,
	}
	addresses := []common.Address{
		common.HexToAddress("0xd8da6bf26964af9d7eed9e03e53415d37aa96045"),
		common.HexToAddress("0xa1e277ea6b97effc5b61b3bf5de03f438981247e"),
	}
	for i := range data {
		var response alchemy.GetAssetTranfersResponse
		err := json.Unmarshal([]byte(data[i]), &response)
		require.NoError(t, err)
		commonResponse := alchemy.TransfersToThirdpartyActivityEntries(response.Transfers, 1, addresses[i])
		require.Equal(t, result[i], len(commonResponse))
	}
}

var malformedErc721Response = `{
	"transfers": [{
		"blockNum": "0x6a7f8e",
		"uniqueId": "0x0e0c4834de556d22a03c1727435f88fb38cefd7c67ee0ab98c6aa1ccfb19a7cb:log:66",
		"hash": "0x0e0c4834de556d22a03c1727435f88fb38cefd7c67ee0ab98c6aa1ccfb19a7cb",
		"from": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e",
		"to": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7",
		"value": null,
		"erc721TokenId": null,
		"erc1155Metadata": null,
		"tokenId": null,
		"asset": "FA721",
		"category": "erc721",
		"rawContract": {
			"value": null,
			"address": "0x9f64932be34d5d897c4253d17707b50921f372b6",
			"decimal": null
		},
		"metadata": {
			"blockTimestamp": "2024-10-30T22:40:00.000Z"
		}
	}]
}`

var malformedErc1155Response = `{
	"transfers": [{
		"blockNum": "0x6a7f93",
		"uniqueId": "0xde60d8ac4248630da5ec7f1547e55b64f432204799c16b44384e493590cd5a70:log:68",
		"hash": "0xde60d8ac4248630da5ec7f1547e55b64f432204799c16b44384e493590cd5a70",
		"from": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e",
		"to": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7",
		"value": null,
		"erc721TokenId": null,
		"erc1155Metadata": [
			{
				"tokenId": null,
				"value": "0x01"
			}
		],
		"tokenId": null,
		"asset": "FA1155",
		"category": "erc1155",
		"rawContract": {
			"value": null,
			"address": "0x1ed60fedff775d500dde21a974cd4e92e0047cc8",
			"decimal": null
		},
		"metadata": {
			"blockTimestamp": "2024-10-30T22:41:00.000Z"
		}
	}]
}`

var erc20NullAddress = `{
	"transfers": [{
		"blockNum": "0x100",
		"uniqueId": "test:erc20:null:address",
		"hash": "0x1234567890123456789012345678901234567890123456789012345678901234",
		"from": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e",
		"to": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7",
		"category": "erc20",
		"rawContract": {
			"value": "0x100",
			"address": null
		},
		"metadata": {
			"blockTimestamp": "2024-10-30T22:40:00.000Z"
		}
	}]
}`

var erc721NullAddress = `{
	"transfers": [{
		"blockNum": "0x100",
		"uniqueId": "test:erc721:null:address",
		"hash": "0x1234567890123456789012345678901234567890123456789012345678901234",
		"from": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e",
		"to": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7",
		"category": "erc721",
		"tokenId": "0x1",
		"rawContract": {
			"value": null,
			"address": null
		},
		"metadata": {
			"blockTimestamp": "2024-10-30T22:40:00.000Z"
		}
	}]
}`

var erc1155NullAddress = `{
	"transfers": [{
		"blockNum": "0x100",
		"uniqueId": "test:erc1155:null:address",
		"hash": "0x1234567890123456789012345678901234567890123456789012345678901234",
		"from": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e",
		"to": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7",
		"category": "erc1155",
		"erc1155Metadata": [{"tokenId": "0x1", "value": "0x1"}],
		"rawContract": {
			"value": null,
			"address": null
		},
		"metadata": {
			"blockTimestamp": "2024-10-30T22:40:00.000Z"
		}
	}]
}`

var externalNullValue = `{
	"transfers": [{
		"blockNum": "0x100",
		"uniqueId": "test:external:null:value",
		"hash": "0x1234567890123456789012345678901234567890123456789012345678901234",
		"from": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e",
		"to": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7",
		"category": "external",
		"rawContract": {
			"value": null,
			"address": null
		},
		"metadata": {
			"blockTimestamp": "2024-10-30T22:40:00.000Z"
		}
	}]
}`

var nullBlockNum = `{
	"transfers": [{
		"blockNum": null,
		"uniqueId": "test:null:blocknum",
		"hash": "0x1234567890123456789012345678901234567890123456789012345678901234",
		"from": "0xa1e277ea6b97effc5b61b3bf5de03f438981247e",
		"to": "0x0adb6caa256a5375c638c00e2ff80a9ac1b2d3a7",
		"category": "external",
		"rawContract": {
			"value": "0x100",
			"address": null
		},
		"metadata": {
			"blockTimestamp": "2024-10-30T22:40:00.000Z"
		}
	}]
}`

func TestMalformedAlchemyResponses(t *testing.T) {
	testCases := []struct {
		name string
		data string
	}{
		{"empty transfers array", `{"transfers": []}`},
		{"missing transfers field", `{}`},
		{"ERC721 with nil tokenId", malformedErc721Response},
		{"ERC1155 with nil tokenId", malformedErc1155Response},
		{"ERC20 with null rawContract.address", erc20NullAddress},
		{"ERC721 with null rawContract.address", erc721NullAddress},
		{"ERC1155 with null rawContract.address", erc1155NullAddress},
		{"external with null rawContract.value", externalNullValue},
		{"null blockNum", nullBlockNum},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var response alchemy.GetAssetTranfersResponse
			err := json.Unmarshal([]byte(tc.data), &response)
			require.NoError(t, err)

			// Filter valid transfers (simulates what FetchTransfers does)
			validTransfers := make([]alchemy.Transfer, 0)
			for _, t := range response.Transfers {
				if t.IsValid() {
					validTransfers = append(validTransfers, t)
				}
			}

			// Should not panic with filtered transfers
			result := alchemy.TransfersToThirdpartyActivityEntries(
				validTransfers, 1, common.HexToAddress("0xa1e277ea6b97effc5b61b3bf5de03f438981247e"))
			require.NotNil(t, result)
		})
	}
}
