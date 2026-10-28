package wallet

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	w_common "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/pendingtxtracker"
)

func TestPendingERC20TransfersAreTheTokensTheAccountSends(t *testing.T) {
	alice := common.Address{0xa1}
	bob := common.Address{0xb0}
	usdc := common.Address{0x05}
	dai := common.Address{0x06}
	pending := []*pendingtxtracker.PendingTransaction{
		{ChainID: w_common.ChainID(1), From: alice, To: usdc, Data: "0xa9059cbb0000"},
		{ChainID: w_common.ChainID(1), From: alice, To: dai, Data: "0x23B872DD0000"},
		{ChainID: w_common.ChainID(1), From: alice, To: bob, Data: "0x"},
		{ChainID: w_common.ChainID(10), From: alice, To: usdc, Data: "0xa9059cbb0000"},
		{ChainID: w_common.ChainID(1), From: bob, To: usdc, Data: "0xa9059cbb0000"},
	}

	require.Equal(t, []common.Address{usdc, dai}, pendingERC20Transfers(pending, 1, alice))
}
