package requests

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/services/wallet/router/fees"

	"gopkg.in/go-playground/validator.v9"
)

var (
	ErrMaxFeesPerGasRequired = &errors.ErrorResponse{Code: errors.ErrorCode("WRC-001"), Details: "maxFeesPerGas is required"}
	ErrPriorityFeeRequired   = &errors.ErrorResponse{Code: errors.ErrorCode("WRC-002"), Details: "priorityFee is required"}
)

type PathTxCustomParams struct {
	GasFeeMode    fees.GasFeeMode `json:"gasFeeMode" validate:"gasFeeModeValid"`
	Nonce         uint64          `json:"nonce"`
	GasAmount     uint64          `json:"gasAmount"`
	MaxFeesPerGas *hexutil.Big    `json:"maxFeesPerGas"`
	PriorityFee   *hexutil.Big    `json:"priorityFee"`
	GasPrice      *hexutil.Big    `json:"gasPrice"` // used for non-EIP1559 chains
}

func gasFeeModeValid(fl validator.FieldLevel) bool {
	mode := fl.Field().Interface().(fees.GasFeeMode)
	switch mode {
	case fees.GasFeeLow, fees.GasFeeMedium, fees.GasFeeHigh, fees.GasFeeCustom:
		return true
	default:
		return false
	}
}

type PathTxIdentity struct {
	RouterInputParamsUuid string `json:"routerInputParamsUuid" validate:"required"`
	PathName              string `json:"pathName" validate:"required"`
	ChainID               uint64 `json:"chainID" validate:"required"`
	IsApprovalTx          bool   `json:"isApprovalTx"`
	CommunityID           string `json:"communityId"`
}

func (p *PathTxIdentity) PathIdentity() string {
	return fmt.Sprintf("%s-%s-%d-%s", p.RouterInputParamsUuid, p.PathName, p.ChainID, p.CommunityID)
}

func (p *PathTxIdentity) TxIdentityKey() string {
	return fmt.Sprintf("%s-%v", p.PathIdentity(), p.IsApprovalTx)
}

func (p *PathTxIdentity) Validate() error {
	validate := validator.New()
	return validate.Struct(p)
}

func (p *PathTxCustomParams) Validate() error {
	validate := validator.New()
	err := validate.RegisterValidation("gasFeeModeValid", gasFeeModeValid)
	if err != nil {
		return err
	}
	err = validate.Struct(p)
	if err != nil {
		return err
	}
	if p.GasFeeMode != fees.GasFeeCustom {
		return nil
	}
	if p.GasPrice != nil { // used for non-EIP1559 chains
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
