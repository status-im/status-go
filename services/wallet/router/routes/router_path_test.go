package routes

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/router/fees"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func TestCopyPath(t *testing.T) {
	addr := common.HexToAddress("0x123")
	path := &Path{
		ProcessorName: "test",
		FromChain:     &params.Network{ChainID: 1},
		ToChain:       &params.Network{ChainID: 2},
		FromToken:     &tokenTypes.Token{Symbol: "symbol1"},
		ToToken:       &tokenTypes.Token{Symbol: "symbol2"},
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
	}

	newPath := path.Copy()

	assert.True(t, reflect.DeepEqual(path, newPath))
}
