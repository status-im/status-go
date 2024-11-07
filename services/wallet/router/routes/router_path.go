package routes

import (
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/router/fees"
	walletToken "github.com/status-im/status-go/services/wallet/token"
)

type Path struct {
	ProcessorName  string
	FromChain      *params.Network    // Source chain
	ToChain        *params.Network    // Destination chain
	FromToken      *walletToken.Token // Source token
	ToToken        *walletToken.Token // Destination token, set if applicable
	AmountIn       *hexutil.Big       // Amount that will be sent from the source chain
	AmountInLocked bool               // Is the amount locked
	AmountOut      *hexutil.Big       // Amount that will be received on the destination chain

	SuggestedLevelsForMaxFeesPerGas *fees.MaxFeesLevels // Suggested max fees for the transaction (in ETH WEI)
	MaxFeesPerGas                   *hexutil.Big        // Max fees per gas (determined by client via GasFeeMode, in ETH WEI)

	TxBaseFee     *hexutil.Big // Base fee for the transaction (in ETH WEI)
	TxPriorityFee *hexutil.Big // Priority fee for the transaction (in ETH WEI)
	TxGasAmount   uint64       // Gas used for the transaction
	TxBonderFees  *hexutil.Big // Bonder fees for the transaction - used for Hop bridge (in selected token)
	TxTokenFees   *hexutil.Big // Token fees for the transaction - used for bridges (represent the difference between the amount in and the amount out, in selected token)

	TxFee   *hexutil.Big // fee for the transaction (includes tx fee only, doesn't include approval fees, l1 fees, l1 approval fees, token fees or bonders fees, in ETH WEI)
	TxL1Fee *hexutil.Big // L1 fee for the transaction - used for for transactions placed on L2 chains (in ETH WEI)

	ApprovalRequired        bool            // Is approval required for the transaction
	ApprovalAmountRequired  *hexutil.Big    // Amount required for the approval transaction
	ApprovalContractAddress *common.Address // Address of the contract that needs to be approved
	ApprovalBaseFee         *hexutil.Big    // Base fee for the approval transaction (in ETH WEI)
	ApprovalPriorityFee     *hexutil.Big    // Priority fee for the approval transaction (in ETH WEI)
	ApprovalGasAmount       uint64          // Gas used for the approval transaction

	ApprovalFee   *hexutil.Big // Total fee for the approval transaction (includes approval tx fees only, doesn't include approval l1 fees, in ETH WEI)
	ApprovalL1Fee *hexutil.Big // L1 fee for the approval transaction - used for for transactions placed on L2 chains (in ETH WEI)

	TxTotalFee *hexutil.Big // Total fee for the transaction (includes tx fees, approval fees, l1 fees, l1 approval fees, in ETH WEI)

	EstimatedTime fees.TransactionEstimation

	RequiredTokenBalance  *big.Int // (in selected token)
	RequiredNativeBalance *big.Int // (in ETH WEI)
	SubtractFees          bool
}

func (p *Path) Equal(o *Path) bool {
	return p.FromChain.ChainID == o.FromChain.ChainID && p.ToChain.ChainID == o.ToChain.ChainID
}

func (p *Path) Copy() *Path {
	newPath := &Path{
		ProcessorName:     p.ProcessorName,
		AmountInLocked:    p.AmountInLocked,
		TxGasAmount:       p.TxGasAmount,
		ApprovalRequired:  p.ApprovalRequired,
		ApprovalGasAmount: p.ApprovalGasAmount,
		EstimatedTime:     p.EstimatedTime,
		SubtractFees:      p.SubtractFees,
	}

	if p.FromChain != nil {
		newPath.FromChain = &params.Network{}
		*newPath.FromChain = *p.FromChain
	}

	if p.ToChain != nil {
		newPath.ToChain = &params.Network{}
		*newPath.ToChain = *p.ToChain
	}

	if p.FromToken != nil {
		newPath.FromToken = &walletToken.Token{}
		*newPath.FromToken = *p.FromToken
	}

	if p.ToToken != nil {
		newPath.ToToken = &walletToken.Token{}
		*newPath.ToToken = *p.ToToken
	}

	if p.AmountIn != nil {
		newPath.AmountIn = (*hexutil.Big)(big.NewInt(0).Set(p.AmountIn.ToInt()))
	}

	if p.AmountOut != nil {
		newPath.AmountOut = (*hexutil.Big)(big.NewInt(0).Set(p.AmountOut.ToInt()))
	}

	if p.SuggestedLevelsForMaxFeesPerGas != nil {
		newPath.SuggestedLevelsForMaxFeesPerGas = &fees.MaxFeesLevels{
			Low:    (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedLevelsForMaxFeesPerGas.Low.ToInt())),
			Medium: (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedLevelsForMaxFeesPerGas.Medium.ToInt())),
			High:   (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedLevelsForMaxFeesPerGas.High.ToInt())),
		}
	}

	if p.MaxFeesPerGas != nil {
		newPath.MaxFeesPerGas = (*hexutil.Big)(big.NewInt(0).Set(p.MaxFeesPerGas.ToInt()))
	}

	if p.TxBaseFee != nil {
		newPath.TxBaseFee = (*hexutil.Big)(big.NewInt(0).Set(p.TxBaseFee.ToInt()))
	}

	if p.TxPriorityFee != nil {
		newPath.TxPriorityFee = (*hexutil.Big)(big.NewInt(0).Set(p.TxPriorityFee.ToInt()))
	}

	if p.TxBonderFees != nil {
		newPath.TxBonderFees = (*hexutil.Big)(big.NewInt(0).Set(p.TxBonderFees.ToInt()))
	}

	if p.TxTokenFees != nil {
		newPath.TxTokenFees = (*hexutil.Big)(big.NewInt(0).Set(p.TxTokenFees.ToInt()))
	}

	if p.TxFee != nil {
		newPath.TxFee = (*hexutil.Big)(big.NewInt(0).Set(p.TxFee.ToInt()))
	}

	if p.TxL1Fee != nil {
		newPath.TxL1Fee = (*hexutil.Big)(big.NewInt(0).Set(p.TxL1Fee.ToInt()))
	}

	if p.ApprovalAmountRequired != nil {
		newPath.ApprovalAmountRequired = (*hexutil.Big)(big.NewInt(0).Set(p.ApprovalAmountRequired.ToInt()))
	}

	if p.ApprovalContractAddress != nil {
		addr := common.HexToAddress(p.ApprovalContractAddress.Hex())
		newPath.ApprovalContractAddress = &addr
	}

	if p.ApprovalBaseFee != nil {
		newPath.ApprovalBaseFee = (*hexutil.Big)(big.NewInt(0).Set(p.ApprovalBaseFee.ToInt()))
	}

	if p.ApprovalPriorityFee != nil {
		newPath.ApprovalPriorityFee = (*hexutil.Big)(big.NewInt(0).Set(p.ApprovalPriorityFee.ToInt()))
	}

	if p.ApprovalFee != nil {
		newPath.ApprovalFee = (*hexutil.Big)(big.NewInt(0).Set(p.ApprovalFee.ToInt()))
	}

	if p.ApprovalL1Fee != nil {
		newPath.ApprovalL1Fee = (*hexutil.Big)(big.NewInt(0).Set(p.ApprovalL1Fee.ToInt()))
	}

	if p.TxTotalFee != nil {
		newPath.TxTotalFee = (*hexutil.Big)(big.NewInt(0).Set(p.TxTotalFee.ToInt()))
	}

	if p.RequiredTokenBalance != nil {
		newPath.RequiredTokenBalance = big.NewInt(0).Set(p.RequiredTokenBalance)
	}

	if p.RequiredNativeBalance != nil {
		newPath.RequiredNativeBalance = big.NewInt(0).Set(p.RequiredNativeBalance)
	}

	return newPath
}

// ID that uniquely identifies a path in a given route
func (p *Path) ID() string {
	// A route will contain at most a single path from a given processor for a given origin chain
	return p.ProcessorName + "-" + strconv.Itoa(int(p.FromChain.ChainID))
}
