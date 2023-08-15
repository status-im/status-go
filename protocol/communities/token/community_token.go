package token

import (
	"fmt"
	"math/big"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/wallet/bigint"
)

type DeployState uint8

const (
	Failed DeployState = iota
	InProgress
	Deployed
)

type PrivilegesLevel uint8

const (
	OwnerLevel PrivilegesLevel = iota
	MasterLevel
	CommunityLevel
)

type CommunityToken struct {
	TokenType          protobuf.CommunityTokenType `json:"tokenType"`
	CommunityID        string                      `json:"communityId"`
	Address            string                      `json:"address"`
	Name               string                      `json:"name"`
	Symbol             string                      `json:"symbol"`
	Description        string                      `json:"description"`
	Supply             *bigint.BigInt              `json:"supply"`
	InfiniteSupply     bool                        `json:"infiniteSupply"`
	Transferable       bool                        `json:"transferable"`
	RemoteSelfDestruct bool                        `json:"remoteSelfDestruct"`
	ChainID            int                         `json:"chainId"`
	DeployState        DeployState                 `json:"deployState"`
	Base64Image        string                      `json:"image"`
	Decimals           int                         `json:"decimals"`
	Deployer           string                      `json:"deployer"`
	PrivilegesLevel    PrivilegesLevel             `json:"privilegesLevel"`
}

func ToCommunityTokenProtobuf(token *CommunityToken) *protobuf.CommunityToken {
	return &protobuf.CommunityToken{
		TokenType:          token.TokenType,
		CommunityId:        token.CommunityID,
		Address:            token.Address,
		Name:               token.Name,
		Symbol:             token.Symbol,
		Description:        token.Description,
		Supply:             token.Supply.String(),
		InfiniteSupply:     token.InfiniteSupply,
		Transferable:       token.Transferable,
		RemoteSelfDestruct: token.RemoteSelfDestruct,
		ChainId:            int32(token.ChainID),
		DeployState:        protobuf.CommunityToken_DeployState(token.DeployState),
		Base64Image:        token.Base64Image,
		Decimals:           int32(token.Decimals),
		Deployer:           token.Deployer,
		PrivilegesLevel:    protobuf.CommunityToken_PrivilegesLevel(token.PrivilegesLevel),
	}
}

func FromCommunityTokenProtobuf(pToken *protobuf.CommunityToken) *CommunityToken {
	token := &CommunityToken{
		TokenType:          pToken.TokenType,
		CommunityID:        pToken.CommunityId,
		Address:            pToken.Address,
		Name:               pToken.Name,
		Symbol:             pToken.Symbol,
		Description:        pToken.Description,
		InfiniteSupply:     pToken.InfiniteSupply,
		Transferable:       pToken.Transferable,
		RemoteSelfDestruct: pToken.RemoteSelfDestruct,
		ChainID:            int(pToken.ChainId),
		DeployState:        DeployState(pToken.DeployState),
		Base64Image:        pToken.Base64Image,
		Decimals:           int(pToken.Decimals),
		Deployer:           pToken.Deployer,
		PrivilegesLevel:    PrivilegesLevel(pToken.PrivilegesLevel),
	}

	supplyBigInt, ok := new(big.Int).SetString(pToken.Supply, 10)
	if ok {
		token.Supply = &bigint.BigInt{Int: supplyBigInt}
	} else {
		token.Supply = &bigint.BigInt{Int: big.NewInt(0)}
		fmt.Println("can't create supply bigInt from string")
	}

	return token
}
