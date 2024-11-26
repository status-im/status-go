package requests

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/services/wallet/router/fees"
)

var (
	ErrMaxFeesPerGasRequired = &errors.ErrorResponse{Code: errors.ErrorCode("WRC-001"), Details: "maxFeesPerGas is required"}
	ErrPriorityFeeRequired   = &errors.ErrorResponse{Code: errors.ErrorCode("WRC-002"), Details: "priorityFee is required"}
)

type PathTxCustomParams struct {
	GasFeeMode    fees.GasFeeMode `json:"gasFeeMode" validate:"required"`
	Nonce         uint64          `json:"nonce"`
	GasAmount     uint64          `json:"gasAmount"`
	MaxFeesPerGas *hexutil.Big    `json:"maxFeesPerGas"`
	PriorityFee   *hexutil.Big    `json:"priorityFee"`
}

type PathTxIdentity struct {
	RouterInputParamsUuid string `json:"routerInputParamsUuid" validate:"required"`
	PathName              string `json:"pathName" validate:"required"`
	ChainID               uint64 `json:"chainID" validate:"required"`
	IsApprovalTx          bool   `json:"isApprovalTx"`
}

func (p *PathTxIdentity) PathIdentity() string {
	return fmt.Sprintf("%s-%s-%d", p.RouterInputParamsUuid, p.PathName, p.ChainID)
}

func (p *PathTxIdentity) TxIdentityKey() string {
	return fmt.Sprintf("%s-%v", p.PathIdentity(), p.IsApprovalTx)
}

func (p *PathTxCustomParams) Validate() error {
	if p.GasFeeMode != fees.GasFeeCustom {
		return nil
	}
	if p.MaxFeesPerGas == nil {
		return ErrMaxFeesPerGasRequired
	}
	if p.PriorityFee == nil {
		return ErrPriorityFeeRequired
	}
	return nil
}
