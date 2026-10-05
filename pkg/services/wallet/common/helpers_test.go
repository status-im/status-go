package common

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
)

func TestPackApprovalInputData(t *testing.T) {

	expectedData := "095ea7b3000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa0000000000000000000000000000000000000000000000000000000000000064"

	addr := common.HexToAddress("0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa")
	data, err := PackApprovalInputData(big.NewInt(100), &addr)
	require.NoError(t, err)
	require.Equal(t, expectedData, hex.EncodeToString(data))
}

func TestUnpackApprovalInputData(t *testing.T) {
	spender := common.HexToAddress("0xaaaaaaae92cc1ceef79a038017889fdd26d23d4d")
	amount := big.NewInt(123456789)

	packed, err := PackApprovalInputData(amount, &spender)
	require.NoError(t, err)

	gotSpender, gotAmount, err := UnpackApprovalInputData(packed)
	require.NoError(t, err)
	require.Equal(t, spender, gotSpender)
	require.Equal(t, 0, amount.Cmp(gotAmount))

	_, _, err = UnpackApprovalInputData(packed[:10])
	require.Error(t, err)

	wrongSelector := append([]byte{0xa9, 0x05, 0x9c, 0xbb}, packed[4:]...) // transfer(address,uint256)
	_, _, err = UnpackApprovalInputData(wrongSelector)
	require.Error(t, err)
}

func TestRelayProcessorClassification(t *testing.T) {
	require.True(t, IsProcessorSwap(pathProcessorCommon.ProcessorRelayName))
	require.True(t, IsProcessorBridge(pathProcessorCommon.ProcessorRelayName))
}
