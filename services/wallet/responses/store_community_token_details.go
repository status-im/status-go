package responses

import (
	"github.com/status-im/status-go/v10/protocol/communities/token"
)

type DeploymentDetails struct {
	ContractAddress string                `json:"contractAddress"`
	TransactionHash string                `json:"transactionHash"`
	CommunityToken  *token.CommunityToken `json:"communityToken"`
	OwnerToken      *token.CommunityToken `json:"ownerToken"`
	MasterToken     *token.CommunityToken `json:"masterToken"`
}
