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
	addresses := []common.Address{
		common.HexToAddress("0xd8da6bf26964af9d7eed9e03e53415d37aa96045"),
		common.HexToAddress("0xa1e277ea6b97effc5b61b3bf5de03f438981247e"),
	}
	for i := range data {
		var response alchemy.GetAssetTranfersResponse
		err := json.Unmarshal([]byte(data[i]), &response)
		require.NoError(t, err)
		commonResponse := alchemy.TransfersToActivityTransactions(response.Transfers, 1, addresses[i])
		require.Equal(t, len(commonResponse), len(response.Transfers))
	}
}
