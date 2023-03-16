package communities

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type RequestToJoinState uint

const (
	RequestToJoinStatePending RequestToJoinState = iota + 1
	RequestToJoinStateDeclined
	RequestToJoinStateAccepted
	RequestToJoinStateCanceled
)

type RequestToJoin struct {
	ID                types.HexBytes     `json:"id"`
	PublicKey         string             `json:"publicKey"`
	Clock             uint64             `json:"clock"`
	ENSName           string             `json:"ensName,omitempty"`
	ChatID            string             `json:"chatId"`
	CommunityID       types.HexBytes     `json:"communityId"`
	State             RequestToJoinState `json:"state"`
	Our               bool               `json:"our"`
	RevealedAddresses map[string][]byte  `json:"revealedAddresses,omitempty"`
}

func (r *RequestToJoin) CalculateID() {
	r.ID = CalculateRequestID(r.PublicKey, r.CommunityID)
}

func (r *RequestToJoin) ToSyncProtobuf() *protobuf.SyncCommunityRequestsToJoin {
	return &protobuf.SyncCommunityRequestsToJoin{
		Id:                r.ID,
		PublicKey:         r.PublicKey,
		Clock:             r.Clock,
		EnsName:           r.ENSName,
		ChatId:            r.ChatID,
		CommunityId:       r.CommunityID,
		State:             uint64(r.State),
		RevealedAddresses: r.RevealedAddresses,
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
	r.RevealedAddresses = proto.RevealedAddresses
}

func (r *RequestToJoin) Empty() bool {
	return len(r.ID)+len(r.PublicKey)+int(r.Clock)+len(r.ENSName)+len(r.ChatID)+len(r.CommunityID)+int(r.State) == 0
}
