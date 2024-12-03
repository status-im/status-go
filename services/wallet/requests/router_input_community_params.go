package requests

import (
	"fmt"
	"math/big"

	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/services/wallet/bigint"
)

const maxSupply = 999999999

var (
	ErrNoNameSet        = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-001"), Details: "name is not set"}
	ErrNoSymbolSet      = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-002"), Details: "symbol is not set"}
	ErrWrongSupplyValue = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-003"), Details: "wrong supply value: %v"}
)

type DeploymentParameters struct {
	Name               string               `json:"name"`
	Symbol             string               `json:"symbol"`
	Supply             *bigint.BigInt       `json:"supply"`
	InfiniteSupply     bool                 `json:"infiniteSupply"`
	Transferable       bool                 `json:"transferable"`
	RemoteSelfDestruct bool                 `json:"remoteSelfDestruct"`
	TokenURI           string               `json:"tokenUri"`
	OwnerTokenAddress  string               `json:"ownerTokenAddress"`
	MasterTokenAddress string               `json:"masterTokenAddress"`
	CommunityID        string               `json:"communityId"`
	Description        string               `json:"description"`
	CroppedImage       *images.CroppedImage `json:"croppedImage,omitempty"` // for community tokens
	Base64Image        string               `json:"base64image"`            // for owner & master tokens
	Decimals           int                  `json:"decimals"`
}

func (d *DeploymentParameters) GetSupply() *big.Int {
	if d.InfiniteSupply {
		return d.GetInfiniteSupply()
	}
	return d.Supply.Int
}

// infinite supply for ERC721 is 2^256-1
func (d *DeploymentParameters) GetInfiniteSupply() *big.Int {
	return GetInfiniteSupply()
}

func GetInfiniteSupply() *big.Int {
	max := new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil)
	max.Sub(max, big.NewInt(1))
	return max
}

func (d *DeploymentParameters) Validate(isAsset bool) error {
	if len(d.Name) <= 0 {
		return ErrNoNameSet
	}
	if len(d.Symbol) <= 0 {
		return ErrNoSymbolSet
	}
	var maxForType = big.NewInt(maxSupply)
	if isAsset {
		assetMultiplier, _ := big.NewInt(0).SetString("1000000000000000000", 10)
		maxForType = maxForType.Mul(maxForType, assetMultiplier)
	}
	if !d.InfiniteSupply && (d.Supply.Cmp(big.NewInt(0)) < 0 || d.Supply.Cmp(maxForType) > 0) {
		return &errors.ErrorResponse{
			Code:    ErrWrongSupplyValue.Code,
			Details: fmt.Sprintf(ErrWrongSupplyValue.Details, d.Supply),
		}
	}
	return nil
}
