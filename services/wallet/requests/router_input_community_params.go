package requests

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/images"
	communitiestoken "github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
)

const maxSupply = 999999999

var (
	ErrNoNameSet                                     = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-001"), Details: "name is not set"}
	ErrNoSymbolSet                                   = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-002"), Details: "symbol is not set"}
	ErrWrongSupplyValue                              = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-003"), Details: "wrong supply value: %v"}
	ErrWalletAddressesEmpty                          = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-004"), Details: "wallet addresses list is empty"}
	ErrNoCommunityAmount                             = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-005"), Details: "amount is required"}
	ErrCommunityAmountMustBePositive                 = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-006"), Details: "amount must be positive"}
	ErrNoCommunityIdProvided                         = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-007"), Details: "community id is required for community related transfers"}
	ErrNoCommunitySignerPubKey                       = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-008"), Details: "signer pub key is required"}
	ErrNoCommunityTokenDeploymentSignature           = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-009"), Details: "signature is required"}
	ErrNoCommunityOwnerTokenParameters               = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-010"), Details: "owner token parameters are required"}
	ErrNoCommunityMasterTokenParameters              = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-011"), Details: "master token parameters are required"}
	ErrNoCommunityDeploymentParameters               = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-012"), Details: "deployment parameters are required"}
	ErrNoCommunityTransferDetails                    = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-013"), Details: "transfer details are required"}
	ErrNoCommunityContractAddress                    = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-014"), Details: "contract address is required"}
	ErrCommunityTokenIdsListEmpty                    = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-015"), Details: "token list is empty"}
	ErrProvidedIndexForSettingInternalDataOutOfRange = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-016"), Details: "provided index for setting internal data is out of range"}
	ErrSetSignerPubKeyWithMultipleTransferDetails    = &errors.ErrorResponse{Code: errors.ErrorCode("WRRC-017"), Details: "signer pub key can be set only with one transfer detail"}
)

type CommunityRouteInputParams struct {
	CommunityID              string                `json:"communityID"`
	TransferDetails          []*TransferDetails    `json:"transferDetails"`
	SignerPubKey             string                `json:"signerPubKey"`
	TokenIds                 []*hexutil.Big        `json:"tokenIds"`
	WalletAddresses          []common.Address      `json:"walletAddresses"`
	TokenDeploymentSignature string                `json:"tokenDeploymentSignature"`
	OwnerTokenParameters     *DeploymentParameters `json:"ownerTokenParameters"`
	MasterTokenParameters    *DeploymentParameters `json:"masterTokenParameters"`
	DeploymentParameters     *DeploymentParameters `json:"deploymentParameters"`
	// used internally
	tokenContractAddress common.Address                   `json:"-"` // contract address used in a single processor
	amount               *hexutil.Big                     `json:"-"` // amount used in a single processor
	tokenType            protobuf.CommunityTokenType      `json:"-"`
	privilegeLevel       communitiestoken.PrivilegesLevel `json:"-"`
}

type TransferDetails struct {
	TokenType            protobuf.CommunityTokenType      `json:"tokenType"`
	PrivilegeLevel       communitiestoken.PrivilegesLevel `json:"privilegeLevel"`
	TokenContractAddress common.Address                   `json:"tokenContractAddress"`
	Amount               *hexutil.Big                     `json:"amount"`
}

type DeploymentParameters struct {
	Name               string               `json:"name"`
	Symbol             string               `json:"symbol"`
	Supply             *bigint.BigInt       `json:"supply"`
	InfiniteSupply     bool                 `json:"infiniteSupply"`
	Transferable       bool                 `json:"transferable"`
	RemoteSelfDestruct bool                 `json:"remoteSelfDestruct"`
	TokenURI           string               `json:"tokenUri"`
	OwnerTokenAddress  common.Address       `json:"ownerTokenAddress"`
	MasterTokenAddress common.Address       `json:"masterTokenAddress"`
	CommunityID        string               `json:"communityId"`
	Description        string               `json:"description"`
	CroppedImage       *images.CroppedImage `json:"croppedImage,omitempty"` // for community tokens
	Base64Image        string               `json:"base64image"`            // for owner & master tokens
	Decimals           int                  `json:"decimals"`
}

// ID that uniquely identifies community route input params
func (c *CommunityRouteInputParams) ID() string {
	return c.CommunityID + "-" + c.tokenContractAddress.String()
}

func (c *CommunityRouteInputParams) UseTransferDetails() bool {
	return len(c.TransferDetails) > 0
}

func (c *CommunityRouteInputParams) SetInternalParams(detailsIndex int) error {
	if detailsIndex < 0 || detailsIndex >= len(c.TransferDetails) {
		return ErrProvidedIndexForSettingInternalDataOutOfRange
	}

	c.tokenType = c.TransferDetails[detailsIndex].TokenType
	c.privilegeLevel = c.TransferDetails[detailsIndex].PrivilegeLevel
	c.tokenContractAddress = c.TransferDetails[detailsIndex].TokenContractAddress
	c.amount = c.TransferDetails[detailsIndex].Amount
	return nil
}

func (c *CommunityRouteInputParams) GetTokenType() protobuf.CommunityTokenType {
	return c.tokenType
}

func (c *CommunityRouteInputParams) GetPrivilegeLevel() communitiestoken.PrivilegesLevel {
	return c.privilegeLevel
}

func (c *CommunityRouteInputParams) GetTokenContractAddress() common.Address {
	return c.tokenContractAddress
}

func (c *CommunityRouteInputParams) GetAmount() *big.Int {
	return c.amount.ToInt()
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

func (c *CommunityRouteInputParams) validateCommunityRelatedInputs(sendType sendtype.SendType) error {
	if c.CommunityID == "" {
		return ErrNoCommunityIdProvided
	}

	if sendType == sendtype.CommunityBurn {
		if len(c.TransferDetails) == 0 {
			return ErrNoCommunityTransferDetails
		}
		for _, td := range c.TransferDetails {
			if td.TokenContractAddress.String() == "" || (td.TokenContractAddress == common.Address{}) {
				return ErrNoCommunityContractAddress
			}
			if td.Amount == nil {
				return ErrNoCommunityAmount
			}
			if td.Amount.ToInt().Cmp(big.NewInt(0)) <= 0 {
				return ErrCommunityAmountMustBePositive
			}
		}
	}

	if sendType == sendtype.CommunityDeployAssets {
		if c.DeploymentParameters == nil {
			return ErrNoCommunityDeploymentParameters
		}
		err := c.DeploymentParameters.Validate(true)
		if err != nil {
			return err
		}
	}

	if sendType == sendtype.CommunityDeployCollectibles {
		if c.DeploymentParameters == nil {
			return ErrNoCommunityDeploymentParameters
		}
		err := c.DeploymentParameters.Validate(false)
		if err != nil {
			return err
		}
	}

	if sendType == sendtype.CommunityDeployOwnerToken {
		if c.SignerPubKey == "" {
			return ErrNoCommunitySignerPubKey
		}
		if c.TokenDeploymentSignature == "" {
			return ErrNoCommunityTokenDeploymentSignature
		}
		if c.OwnerTokenParameters == nil {
			return ErrNoCommunityOwnerTokenParameters
		}
		if c.MasterTokenParameters == nil {
			return ErrNoCommunityMasterTokenParameters
		}
	}

	if sendType == sendtype.CommunityMintTokens {
		if len(c.WalletAddresses) == 0 {
			return ErrWalletAddressesEmpty
		}
		if len(c.TransferDetails) == 0 {
			return ErrNoCommunityTransferDetails
		}
		for _, td := range c.TransferDetails {
			if td.TokenContractAddress.String() == "" || (td.TokenContractAddress == common.Address{}) {
				return ErrNoCommunityContractAddress
			}
			if td.Amount == nil {
				return ErrNoCommunityAmount
			}
			if td.Amount.ToInt().Cmp(big.NewInt(0)) <= 0 {
				return ErrCommunityAmountMustBePositive
			}
		}
	}

	if sendType == sendtype.CommunityRemoteBurn {
		if len(c.TokenIds) == 0 {
			return ErrCommunityTokenIdsListEmpty
		}
	}

	if sendType == sendtype.CommunitySetSignerPubKey {
		if c.SignerPubKey == "" {
			return ErrNoCommunitySignerPubKey
		}

		if len(c.TransferDetails) != 1 {
			return ErrSetSignerPubKeyWithMultipleTransferDetails
		}
		for _, td := range c.TransferDetails {
			if td.TokenContractAddress.String() == "" || (td.TokenContractAddress == common.Address{}) {
				return ErrNoCommunityContractAddress
			}
		}
	}

	return nil
}

func (td *TransferDetails) copy() *TransferDetails {
	newParams := &TransferDetails{
		TokenType:            td.TokenType,
		PrivilegeLevel:       td.PrivilegeLevel,
		TokenContractAddress: td.TokenContractAddress,
	}

	if td.Amount != nil {
		newParams.Amount = (*hexutil.Big)(big.NewInt(0).Set(td.Amount.ToInt()))
	}
	return newParams
}

func (d *DeploymentParameters) copy() *DeploymentParameters {
	newParams := &DeploymentParameters{
		Name:               d.Name,
		Symbol:             d.Symbol,
		InfiniteSupply:     d.InfiniteSupply,
		Transferable:       d.Transferable,
		RemoteSelfDestruct: d.RemoteSelfDestruct,
		TokenURI:           d.TokenURI,
		OwnerTokenAddress:  d.OwnerTokenAddress,
		MasterTokenAddress: d.MasterTokenAddress,
		CommunityID:        d.CommunityID,
		Description:        d.Description,
		Base64Image:        d.Base64Image,
		Decimals:           d.Decimals,
	}

	if d.Supply != nil {
		newParams.Supply = &bigint.BigInt{Int: new(big.Int).Set(d.Supply.Int)}
	}
	if d.CroppedImage != nil {
		ci := *d.CroppedImage
		newParams.CroppedImage = &ci
	}
	return newParams
}

func (c *CommunityRouteInputParams) Copy() *CommunityRouteInputParams {
	newParams := &CommunityRouteInputParams{
		CommunityID:              c.CommunityID,
		SignerPubKey:             c.SignerPubKey,
		TokenDeploymentSignature: c.TokenDeploymentSignature,
	}

	if c.TokenIds != nil {
		newParams.TokenIds = make([]*hexutil.Big, len(c.TokenIds))
		for i, id := range c.TokenIds {
			newParams.TokenIds[i] = (*hexutil.Big)(big.NewInt(0).Set(id.ToInt()))
		}
	}
	if c.TransferDetails != nil {
		newParams.TransferDetails = make([]*TransferDetails, len(c.TransferDetails))
		for i, td := range c.TransferDetails {
			newParams.TransferDetails[i] = td.copy()
		}
	}
	if c.WalletAddresses != nil {
		newParams.WalletAddresses = make([]common.Address, len(c.WalletAddresses))
		copy(newParams.WalletAddresses, c.WalletAddresses)
	}
	if c.OwnerTokenParameters != nil {
		newParams.OwnerTokenParameters = c.OwnerTokenParameters.copy()
	}
	if c.MasterTokenParameters != nil {
		newParams.MasterTokenParameters = c.MasterTokenParameters.copy()
	}
	if c.DeploymentParameters != nil {
		newParams.DeploymentParameters = c.DeploymentParameters.copy()
	}

	// internal fields
	newParams.tokenType = c.tokenType
	newParams.privilegeLevel = c.privilegeLevel
	newParams.tokenContractAddress = c.tokenContractAddress
	if c.amount != nil {
		newParams.amount = (*hexutil.Big)(big.NewInt(0).Set(c.amount.ToInt()))
	}

	return newParams
}
