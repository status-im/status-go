package communities

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type CommunityPrivilegedMemberSyncMessage struct {
	CommunityPrivateKey                *ecdsa.PrivateKey
	Receivers                          []*ecdsa.PublicKey
	CommunityPrivilegedUserSyncMessage *protobuf.CommunityPrivilegedUserSyncMessage
}

func (m *Manager) HandleRequestToJoinPrivilegedUserSyncMessage(message *protobuf.CommunityPrivilegedUserSyncMessage, communityID types.HexBytes) ([]*RequestToJoin, error) {
	var state RequestToJoinState
	if message.Type == protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ACCEPT_REQUEST_TO_JOIN {
		state = RequestToJoinStateAccepted
	} else {
		state = RequestToJoinStateDeclined
	}

	requestsToJoin := make([]*RequestToJoin, 0)
	for signer, requestToJoinProto := range message.RequestToJoin {
		requestToJoin := &RequestToJoin{
			PublicKey:        signer,
			Clock:            requestToJoinProto.Clock,
			ENSName:          requestToJoinProto.EnsName,
			CommunityID:      requestToJoinProto.CommunityId,
			State:            state,
			RevealedAccounts: requestToJoinProto.RevealedAccounts,
		}
		requestToJoin.CalculateID()

		_, err := m.saveOrUpdateRequestToJoin(communityID, requestToJoin)
		if err != nil {
			return nil, err
		}
		requestsToJoin = append(requestsToJoin, requestToJoin)
	}

	return requestsToJoin, nil
}

func (m *Manager) HandleSyncAllRequestToJoinForNewPrivilegedMember(message *protobuf.CommunityPrivilegedUserSyncMessage, communityID types.HexBytes) ([]*RequestToJoin, error) {
	requestsToJoin := make([]*RequestToJoin, len(message.SyncRequestsToJoin))
	for _, syncRequestToJoin := range message.SyncRequestsToJoin {
		requestToJoin := new(RequestToJoin)
		requestToJoin.InitFromSyncProtobuf(syncRequestToJoin)

		if _, err := m.saveOrUpdateRequestToJoin(communityID, requestToJoin); err != nil {
			return nil, err
		}

		if requestToJoin.RevealedAccounts != nil && len(requestToJoin.RevealedAccounts) > 0 {
			if err := m.persistence.RemoveRequestToJoinRevealedAddresses(requestToJoin.ID); err != nil {
				return nil, err
			}

			if err := m.persistence.SaveRequestToJoinRevealedAddresses(requestToJoin); err != nil {
				return nil, err
			}
		}
		requestsToJoin = append(requestsToJoin, requestToJoin)
	}
	return requestsToJoin, nil
}
