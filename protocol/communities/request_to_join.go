package communities

import (
	"fmt"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

const (
	RequestToJoinStatePending uint = iota + 1
	RequestToJoinStateDeclined
	RequestToJoinStateAccepted
)

type RequestToJoin struct {
	ID          types.HexBytes `json:"id"`
	PublicKey   string         `json:"publicKey"`
	Clock       uint64         `json:"clock"`
	ENSName     string         `json:"ensName,omitempty"`
	ChatID      string         `json:"chatId"`
	CommunityID types.HexBytes `json:"communityId"`
	State       uint           `json:"state"`
	Our         bool           `json:"our"`
}

func (r *RequestToJoin) CalculateID() {
	idString := fmt.Sprintf("%s-%s", r.PublicKey, r.CommunityID)
	r.ID = crypto.Keccak256([]byte(idString))
}
