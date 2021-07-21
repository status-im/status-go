package communities

import (
	"fmt"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type RequestToJoinState uint

const (
	RequestToJoinStatePending RequestToJoinState = iota + 1
	RequestToJoinStateDeclined
	RequestToJoinStateAccepted
)

type RequestToJoin struct {
	ID          types.HexBytes     `json:"id"`
	PublicKey   string             `json:"publicKey"`
	Clock       uint64             `json:"clock"`
	ENSName     string             `json:"ensName,omitempty"`
	ChatID      string             `json:"chatId"`
	CommunityID types.HexBytes     `json:"communityId"`
	State       RequestToJoinState `json:"state"`
	Our         bool               `json:"our"`
}

func (r *RequestToJoin) CalculateID() {
	idString := fmt.Sprintf("%s-%s", r.PublicKey, r.CommunityID)
	r.ID = crypto.Keccak256([]byte(idString))
}

func (r *RequestToJoin) ToSyncProtobuf() *protobuf.SyncCommunityRequestsToJoin {
	return &protobuf.SyncCommunityRequestsToJoin{
		Id:          r.ID,
		PublicKey:   r.PublicKey,
		Clock:       r.Clock,
		EnsName:     r.ENSName,
		ChatId:      r.ChatID,
		CommunityId: r.CommunityID,
		State:       uint64(r.State),
	}
}

func (r *RequestToJoin) InitFromSyncProtobuf(proto *protobuf.SyncCommunityRequestsToJoin) {
	r.ID = proto.Id
	r.PublicKey = proto.PublicKey
	r.Clock = proto.Clock
	r.ENSName = proto.EnsName
	r.ChatID = proto.ChatId
	r.CommunityID = proto.CommunityId
	r.State = RequestToJoinState(proto.State)
}

func (r *RequestToJoin) Empty() bool {
	return len(r.ID)+len(r.PublicKey)+int(r.Clock)+len(r.ENSName)+len(r.ChatID)+len(r.CommunityID)+int(r.State) == 0
}
