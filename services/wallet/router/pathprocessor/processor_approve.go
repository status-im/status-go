package pathprocessor

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Placeholder for Approve transaction until handling gets fully done status-go side
type ApproveTxArgs struct {
	ApprovalSpender common.Address `json:"approvalSpender"`
	ApprovalAmount  *hexutil.Big   `json:"approvalAmount"`
}
