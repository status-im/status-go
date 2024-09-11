package pathprocessor

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type TransactionInputData struct {
	ProcessorName   string
	FromAsset       *string
	FromAmount      *hexutil.Big
	ToAsset         *string
	ToAmount        *hexutil.Big
	Side            *SwapSide
	SlippageBps     *uint16
	ApprovalAmount  *hexutil.Big
	ApprovalSpender *common.Address
}

func NewInputData() *TransactionInputData {
	return &TransactionInputData{
		FromAmount:     (*hexutil.Big)(new(big.Int)),
		ToAmount:       (*hexutil.Big)(new(big.Int)),
		ApprovalAmount: (*hexutil.Big)(new(big.Int)),
	}
}
