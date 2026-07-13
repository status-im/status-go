package lifi

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

const sampleQuoteResponse = `{
	"id": "0x123",
	"type": "swap",
	"tool": "1inch",
	"estimate": {
		"fromAmount": "1000000000000000000",
		"toAmount": "3000000000",
		"toAmountMin": "2985000000",
		"approvalAddress": "0x1111111254EEB25477B68fb85Ed929f73A960582"
	},
	"transactionRequest": {
		"from": "0xAbC0000000000000000000000000000000000001",
		"to": "0x1111111254EEB25477B68fb85Ed929f73A960582",
		"value": "0xde0b6b3a7640000",
		"data": "0xabcdef",
		"gasPrice": "0x3b9aca00",
		"gasLimit": "0x30d40",
		"chainId": 1
	}
}`

func TestDecodeQuote(t *testing.T) {
	var quote Quote
	err := json.Unmarshal([]byte(sampleQuoteResponse), &quote)
	require.NoError(t, err)

	require.Equal(t, "swap", quote.Type)
	require.Equal(t, "1inch", quote.Tool)

	require.NotNil(t, quote.Estimate.ToAmount)
	require.Equal(t, "3000000000", quote.Estimate.ToAmount.String())
	require.NotNil(t, quote.Estimate.ToAmountMin)
	require.Equal(t, "2985000000", quote.Estimate.ToAmountMin.String())
	require.Equal(t, common.HexToAddress("0x1111111254EEB25477B68fb85Ed929f73A960582"), quote.Estimate.ApprovalAddress)

	require.Equal(t, "0xde0b6b3a7640000", quote.TransactionRequest.Value)
	require.Equal(t, "0xabcdef", quote.TransactionRequest.Data)
	require.Equal(t, "0x30d40", quote.TransactionRequest.GasLimit)
	require.Equal(t, uint64(1), quote.TransactionRequest.ChainID)
}
