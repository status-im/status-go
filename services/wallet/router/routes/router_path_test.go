package routes

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/permit2"
	"github.com/status-im/status-go/services/wallet/router/fees"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

func TestCopyPath(t *testing.T) {
	addr := common.HexToAddress("0x123")
	path := &Path{
		ProcessorName: "test",
		FromChain:     &params.Network{ChainID: 1},
		ToChain:       &params.Network{ChainID: 2},
		FromToken:     &tokentypes.Token{Token: &types.Token{Symbol: "symbol1"}},
		ToToken:       &tokentypes.Token{Token: &types.Token{Symbol: "symbol2"}},
		AmountIn:      (*hexutil.Big)(big.NewInt(100)),
		AmountOut:     (*hexutil.Big)(big.NewInt(200)),
		SuggestedLevelsForMaxFeesPerGas: &fees.MaxFeesLevels{
			Low:    (*hexutil.Big)(big.NewInt(100)),
			Medium: (*hexutil.Big)(big.NewInt(200)),
			High:   (*hexutil.Big)(big.NewInt(300)),
		},
		TxMaxFeesPerGas:         (*hexutil.Big)(big.NewInt(100)),
		TxBaseFee:               (*hexutil.Big)(big.NewInt(100)),
		TxPriorityFee:           (*hexutil.Big)(big.NewInt(100)),
		TxGasAmount:             100,
		TxBonderFees:            (*hexutil.Big)(big.NewInt(100)),
		TxTokenFees:             (*hexutil.Big)(big.NewInt(100)),
		TxEstimatedTime:         100,
		TxFee:                   (*hexutil.Big)(big.NewInt(100)),
		TxL1Fee:                 (*hexutil.Big)(big.NewInt(100)),
		ApprovalRequired:        true,
		ApprovalAmountRequired:  (*hexutil.Big)(big.NewInt(100)),
		ApprovalContractAddress: &addr,
		ApprovalMaxFeesPerGas:   (*hexutil.Big)(big.NewInt(100)),
		ApprovalBaseFee:         (*hexutil.Big)(big.NewInt(100)),
		ApprovalPriorityFee:     (*hexutil.Big)(big.NewInt(100)),
		ApprovalGasAmount:       100,
		ApprovalEstimatedTime:   100,
		ApprovalFee:             (*hexutil.Big)(big.NewInt(100)),
		ApprovalL1Fee:           (*hexutil.Big)(big.NewInt(100)),
		TxTotalFee:              (*hexutil.Big)(big.NewInt(100)),
		RequiredTokenBalance:    big.NewInt(100),
		RequiredNativeBalance:   big.NewInt(100),
		SubtractFees:            true,
		PermitDetails: &permit2.Details{
			Type:     permit2.TypePermit2,
			ChainID:  1,
			Owner:    common.HexToAddress("0x456"),
			Token:    common.HexToAddress("0x789"),
			Amount:   big.NewInt(100),
			Spender:  common.HexToAddress("0xabc"),
			Permit2:  common.HexToAddress("0xdef"),
			Nonce:    big.NewInt(1),
			Deadline: big.NewInt(2),
		},
	}

	newPath := path.Copy()

	assert.True(t, reflect.DeepEqual(path, newPath))

	// The permit is mutated in place when its signature arrives, so the copy must not
	// share it with the original.
	newPath.PermitDetails.Amount.SetInt64(1)
	newPath.PermitDetails.Nonce.SetInt64(99)
	assert.Equal(t, int64(100), path.PermitDetails.Amount.Int64())
	assert.Equal(t, int64(1), path.PermitDetails.Nonce.Int64())
}
