package signal

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/protocol/communities/token"
)

const (

	// EventCommunityTokenTransactionStatusChanged is triggered when community token contract
	// transaction changed its status
	EventCommunityTokenTransactionStatusChanged = "communityToken.communityTokenTransactionStatusChanged"
)

type CommunityTokenTransactionSignal struct {
	TransactionType string                `json:"transactionType"`
	Success         bool                  `json:"success"`                  // transaction's status
	Hash            common.Hash           `json:"hash"`                     // transaction hash
	CommunityToken  *token.CommunityToken `json:"communityToken,omitempty"` // community token changed by transaction
	OwnerToken      *token.CommunityToken `json:"ownerToken,omitempty"`     // owner token emitted by deployment transaction
	MasterToken     *token.CommunityToken `json:"masterToken,omitempty"`    // master token emitted by deployment transaction
	ErrorString     string                `json:"errorString"`              // information about failed operation
}

func SendCommunityTokenTransactionStatusSignal(transactionType string, success bool, hash common.Hash,
	communityToken *token.CommunityToken, ownerToken *token.CommunityToken, masterToken *token.CommunityToken, errorString string) {
	send(EventCommunityTokenTransactionStatusChanged, CommunityTokenTransactionSignal{
		TransactionType: transactionType,
		Success:         success,
		Hash:            hash,
		CommunityToken:  communityToken,
		OwnerToken:      ownerToken,
		MasterToken:     masterToken,
		ErrorString:     errorString,
	})
}
