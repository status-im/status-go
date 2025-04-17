package routes

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/router/fees"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

type Path struct {
	RouterInputParamsUuid string
	ProcessorName         string
	FromChain             *params.Network   // Source chain
	ToChain               *params.Network   // Destination chain
	FromToken             *tokenTypes.Token // Source token
	ToToken               *tokenTypes.Token // Destination token, set if applicable
	AmountIn              *hexutil.Big      // Amount that will be sent from the source chain
	AmountOut             *hexutil.Big      // Amount that will be received on the destination chain

	SuggestedLevelsForMaxFeesPerGas *fees.MaxFeesLevels // Suggested max fees by the network (in ETH WEI)
	SuggestedMinPriorityFee         *hexutil.Big        // Suggested min priority fee by the network (in ETH WEI)
	SuggestedMaxPriorityFee         *hexutil.Big        // Suggested max priority fee by the network (in ETH WEI)
	SuggestedTxNonce                *hexutil.Uint64     // Suggested nonce for the transaction
	SuggestedTxGasAmount            uint64              // Suggested gas amount for the transaction
	SuggestedApprovalTxNonce        *hexutil.Uint64     // Suggested nonce for the approval transaction
	SuggestedApprovalGasAmount      uint64              // Suggested gas amount for the approval transaction
	CurrentBaseFee                  *hexutil.Big        // Current network base fee (in ETH WEI)
	UsedContractAddress             *common.Address     // Address of the contract that will be used for the transaction

	TxPackedData    []byte          // Packed data for the transaction
	TxNonce         *hexutil.Uint64 // Nonce for the transaction
	TxGasFeeMode    fees.GasFeeMode // Gas fee mode for the transaction
	TxMaxFeesPerGas *hexutil.Big    // Max fees per gas (determined by client via GasFeeMode, in ETH WEI)
	TxBaseFee       *hexutil.Big    // Base fee for the transaction (in ETH WEI)
	TxPriorityFee   *hexutil.Big    // Priority fee for the transaction (in ETH WEI)
	TxGasAmount     uint64          // Gas used for the transaction
	TxBonderFees    *hexutil.Big    // Bonder fees for the transaction - used for Hop bridge (in selected token)
	TxTokenFees     *hexutil.Big    // Token fees for the transaction - used for bridges (represent the difference between the amount in and the amount out, in selected token)
	TxEstimatedTime uint            // Estimated time for the transaction in seconds

	TxFee   *hexutil.Big // fee for the transaction (includes tx fee only, doesn't include approval fees, l1 fees, l1 approval fees, token fees or bonders fees, in ETH WEI)
	TxL1Fee *hexutil.Big // L1 fee for the transaction - used for for transactions placed on L2 chains (in ETH WEI)

	ApprovalRequired        bool            // Is approval required for the transaction
	ApprovalAmountRequired  *hexutil.Big    // Amount required for the approval transaction
	ApprovalContractAddress *common.Address // Address of the contract that will be used for the approval transaction, the same as UsedContractAddress. We can remove this field and use UsedContractAddress instead.
	ApprovalPackedData      []byte          // Packed data for the approval transaction
	ApprovalTxNonce         *hexutil.Uint64 // Nonce for the transaction
	ApprovalGasFeeMode      fees.GasFeeMode // Gas fee mode for the approval transaction
	ApprovalMaxFeesPerGas   *hexutil.Big    // Max fees per gas (determined by client via GasFeeMode, in ETH WEI)
	ApprovalBaseFee         *hexutil.Big    // Base fee for the approval transaction (in ETH WEI)
	ApprovalPriorityFee     *hexutil.Big    // Priority fee for the approval transaction (in ETH WEI)
	ApprovalGasAmount       uint64          // Gas used for the approval transaction
	ApprovalEstimatedTime   uint            // Estimated time for the approval transaction in seconds

	ApprovalFee   *hexutil.Big // Total fee for the approval transaction (includes approval tx fees only, doesn't include approval l1 fees, in ETH WEI)
	ApprovalL1Fee *hexutil.Big // L1 fee for the approval transaction - used for for transactions placed on L2 chains (in ETH WEI)

	TxTotalFee *hexutil.Big // Total fee for the transaction (includes tx fees, approval fees, l1 fees, l1 approval fees, in ETH WEI)

	RequiredTokenBalance  *big.Int // (in selected token)
	RequiredNativeBalance *big.Int // (in ETH WEI)
	SubtractFees          bool

	// used internally
	communityParams *requests.CommunityRouteInputParams
}

func (p *Path) PathIdentity() string {
	var communityID string
	if p.communityParams != nil {
		communityID = p.communityParams.ID()
	}
	return fmt.Sprintf("%s-%s-%d-%s", p.RouterInputParamsUuid, p.ProcessorName, p.FromChain.ChainID, communityID)
}

func (p *Path) TxIdentityKey(approval bool) string {
	return fmt.Sprintf("%s-%v", p.PathIdentity(), approval)
}

func (p *Path) Equal(o *Path) bool {
	return p.FromChain.ChainID == o.FromChain.ChainID && p.ToChain.ChainID == o.ToChain.ChainID
}

func (p *Path) SetCommunityParams(params *requests.CommunityRouteInputParams) {
	p.communityParams = params
}

func (p *Path) GetCommunityParams() *requests.CommunityRouteInputParams {
	return p.communityParams
}

func (p *Path) Copy() *Path {
	newPath := &Path{
		RouterInputParamsUuid:      p.RouterInputParamsUuid,
		ProcessorName:              p.ProcessorName,
		SuggestedTxGasAmount:       p.SuggestedTxGasAmount,
		SuggestedApprovalGasAmount: p.SuggestedApprovalGasAmount,
		TxGasFeeMode:               p.TxGasFeeMode,
		TxGasAmount:                p.TxGasAmount,
		TxEstimatedTime:            p.TxEstimatedTime,
		ApprovalRequired:           p.ApprovalRequired,
		ApprovalGasFeeMode:         p.ApprovalGasFeeMode,
		ApprovalGasAmount:          p.ApprovalGasAmount,
		ApprovalEstimatedTime:      p.ApprovalEstimatedTime,
		SubtractFees:               p.SubtractFees,
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
		newPath.FromToken = &tokenTypes.Token{}
		*newPath.FromToken = *p.FromToken
	}

	if p.ToToken != nil {
		newPath.ToToken = &tokenTypes.Token{}
		*newPath.ToToken = *p.ToToken
	}

	if p.AmountIn != nil {
		newPath.AmountIn = (*hexutil.Big)(big.NewInt(0).Set(p.AmountIn.ToInt()))
	}

	if p.AmountOut != nil {
		newPath.AmountOut = (*hexutil.Big)(big.NewInt(0).Set(p.AmountOut.ToInt()))
	}

	if p.SuggestedLevelsForMaxFeesPerGas != nil {
		newPath.SuggestedLevelsForMaxFeesPerGas = &fees.MaxFeesLevels{}
		if p.SuggestedLevelsForMaxFeesPerGas.Low != nil {
			newPath.SuggestedLevelsForMaxFeesPerGas.Low = (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedLevelsForMaxFeesPerGas.Low.ToInt()))
		}
		if p.SuggestedLevelsForMaxFeesPerGas.LowPriority != nil {
			newPath.SuggestedLevelsForMaxFeesPerGas.LowPriority = (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedLevelsForMaxFeesPerGas.LowPriority.ToInt()))
		}
		newPath.SuggestedLevelsForMaxFeesPerGas.LowEstimatedTime = p.SuggestedLevelsForMaxFeesPerGas.LowEstimatedTime
		if p.SuggestedLevelsForMaxFeesPerGas.Medium != nil {
			newPath.SuggestedLevelsForMaxFeesPerGas.Medium = (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedLevelsForMaxFeesPerGas.Medium.ToInt()))
		}
		if p.SuggestedLevelsForMaxFeesPerGas.MediumPriority != nil {
			newPath.SuggestedLevelsForMaxFeesPerGas.MediumPriority = (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedLevelsForMaxFeesPerGas.MediumPriority.ToInt()))
		}
		newPath.SuggestedLevelsForMaxFeesPerGas.MediumEstimatedTime = p.SuggestedLevelsForMaxFeesPerGas.MediumEstimatedTime
		if p.SuggestedLevelsForMaxFeesPerGas.High != nil {
			newPath.SuggestedLevelsForMaxFeesPerGas.High = (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedLevelsForMaxFeesPerGas.High.ToInt()))
		}
		if p.SuggestedLevelsForMaxFeesPerGas.HighPriority != nil {
			newPath.SuggestedLevelsForMaxFeesPerGas.HighPriority = (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedLevelsForMaxFeesPerGas.HighPriority.ToInt()))
		}
		newPath.SuggestedLevelsForMaxFeesPerGas.HighEstimatedTime = p.SuggestedLevelsForMaxFeesPerGas.HighEstimatedTime
	}

	if p.SuggestedMinPriorityFee != nil {
		newPath.SuggestedMinPriorityFee = (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedMinPriorityFee.ToInt()))
	}

	if p.SuggestedMaxPriorityFee != nil {
		newPath.SuggestedMaxPriorityFee = (*hexutil.Big)(big.NewInt(0).Set(p.SuggestedMaxPriorityFee.ToInt()))
	}

	if p.SuggestedTxNonce != nil {
		SuggestedTxNonce := *p.SuggestedTxNonce
		newPath.SuggestedTxNonce = &SuggestedTxNonce
	}

	if p.SuggestedApprovalTxNonce != nil {
		SuggestedApprovalTxNonce := *p.SuggestedApprovalTxNonce
		newPath.SuggestedApprovalTxNonce = &SuggestedApprovalTxNonce
	}

	if p.CurrentBaseFee != nil {
		newPath.CurrentBaseFee = (*hexutil.Big)(big.NewInt(0).Set(p.CurrentBaseFee.ToInt()))
	}

	if p.TxPackedData != nil {
		newPath.TxPackedData = make([]byte, len(p.TxPackedData))
		copy(newPath.TxPackedData, p.TxPackedData)
	}

	if p.TxNonce != nil {
		txNonce := *p.TxNonce
		newPath.TxNonce = &txNonce
	}

	if p.TxMaxFeesPerGas != nil {
		newPath.TxMaxFeesPerGas = (*hexutil.Big)(big.NewInt(0).Set(p.TxMaxFeesPerGas.ToInt()))
	}

	if p.UsedContractAddress != nil {
		addr := common.HexToAddress(p.UsedContractAddress.Hex())
		newPath.UsedContractAddress = &addr
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

	if p.ApprovalPackedData != nil {
		newPath.ApprovalPackedData = make([]byte, len(p.ApprovalPackedData))
		copy(newPath.ApprovalPackedData, p.ApprovalPackedData)
	}

	if p.ApprovalTxNonce != nil {
		approvalTxNonce := *p.ApprovalTxNonce
		newPath.ApprovalTxNonce = &approvalTxNonce
	}

	if p.ApprovalMaxFeesPerGas != nil {
		newPath.ApprovalMaxFeesPerGas = (*hexutil.Big)(big.NewInt(0).Set(p.ApprovalMaxFeesPerGas.ToInt()))
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

	if p.communityParams != nil {
		newPath.communityParams = p.communityParams.Copy()
	}

	return newPath
}
