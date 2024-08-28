package routes

import (
	"math/big"

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
