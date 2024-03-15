package requests

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type TokenCriteriaRequest struct {
	ContractAddresses map[uint64]string           `json:"contractAddresses"`
	Type              protobuf.CommunityTokenType `json:"type"`
	Symbol            string                      `json:"symbol"`
	Name              string                      `json:"name"`
	ENSPattern        string                      `json:"ensPattern"`
	Decimals          uint64                      `json:"decimals"`
	AmountInWei       string                      `json:"amountInWei"`
}

func (t *TokenCriteriaRequest) ToTokenCriteriaProtobuf() *protobuf.TokenCriteria {
	return &protobuf.TokenCriteria{
		ContractAddresses: t.ContractAddresses,
		Type:              t.Type,
		Symbol:            t.Symbol,
		Name:              t.Name,
		EnsPattern:        t.ENSPattern,
		Decimals:          t.Decimals,
		AmountInWei:       t.AmountInWei,
	}
}

type CreateCommunityTokenPermissionV2 struct {
	CommunityID   types.HexBytes                         `json:"communityId"`
	Type          protobuf.CommunityTokenPermission_Type `json:"type"`
	TokenCriteria []TokenCriteriaRequest                 `json:"tokenCriteria"`
	IsPrivate     bool                                   `json:"isPrivate"`
	ChatIDs       []string                               `json:"chatIds"`
}

func (p CreateCommunityTokenPermissionV2) ToCreateCommunityTokenPermission() *CreateCommunityTokenPermission {
	permission := &CreateCommunityTokenPermission{
		CommunityID: p.CommunityID,
		Type:        p.Type,
		IsPrivate:   p.IsPrivate,
		ChatIDs:     p.ChatIDs,
	}
	for _, c := range p.TokenCriteria {
		permission.TokenCriteria = append(permission.TokenCriteria, c.ToTokenCriteriaProtobuf())
	}
	return permission
}
