package communities

import (
	"crypto/ecdsa"
	"database/sql"
	"errors"

	"go.uber.org/zap"

	multiaccountscommon "github.com/status-im/status-go/multiaccounts/common"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

var ErrOutdatedSharedRequestToJoinClock = errors.New("outdated clock in shared request to join")
var ErrOutdatedSharedRequestToJoinState = errors.New("outdated state in shared request to join")

type CommunityPrivilegedMemberSyncMessage struct {
	Receivers                          []*ecdsa.PublicKey
	CommunityPrivilegedUserSyncMessage *protobuf.CommunityPrivilegedUserSyncMessage
}

func (m *Manager) HandleRequestToJoinPrivilegedUserSyncMessage(message *protobuf.CommunityPrivilegedUserSyncMessage, community *Community) ([]*RequestToJoin, error) {
	var state RequestToJoinState
	if message.Type == protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ACCEPT_REQUEST_TO_JOIN {
		state = RequestToJoinStateAccepted
	} else {
		state = RequestToJoinStateDeclined
	}

	myPk := common.PubkeyToHex(&m.identity.PublicKey)

	requestsToJoin := make([]*RequestToJoin, 0)
	for signer, requestToJoinProto := range message.RequestToJoin {
		if signer == myPk {
			continue
		}

		requestToJoin := &RequestToJoin{
			PublicKey:          signer,
			Clock:              requestToJoinProto.Clock,
			ENSName:            requestToJoinProto.EnsName,
			CustomizationColor: multiaccountscommon.IDToColorFallbackToBlue(requestToJoinProto.CustomizationColor),
			CommunityID:        requestToJoinProto.CommunityId,
			State:              state,
			RevealedAccounts:   requestToJoinProto.RevealedAccounts,
		}
		requestToJoin.CalculateID()

		err := m.processPrivilegedUserSharedRequestToJoin(community, requestToJoin)
		if err != nil {
			m.logger.Warn("error to handle shared request to join",
				zap.String("communityID", community.IDString()),
				zap.String("requestToJoinID", types.Bytes2Hex(requestToJoin.ID)),
				zap.String("publicKey", requestToJoin.PublicKey),
				zap.String("error", err.Error()))
			continue
		}

		requestsToJoin = append(requestsToJoin, requestToJoin)
	}

	return requestsToJoin, nil
}

func (m *Manager) HandleSyncAllRequestToJoinForNewPrivilegedMember(message *protobuf.CommunityPrivilegedUserSyncMessage, community *Community) ([]*RequestToJoin, error) {
	nonAcceptedRequestsToJoin := []*RequestToJoin{}
	myPk := common.PubkeyToHex(&m.identity.PublicKey)

	for _, syncRequestToJoin := range message.SyncRequestsToJoin {
		if syncRequestToJoin.PublicKey == myPk {
			continue
		}

		requestToJoin := new(RequestToJoin)
		requestToJoin.InitFromSyncProtobuf(syncRequestToJoin)

		err := m.processPrivilegedUserSharedRequestToJoin(community, requestToJoin)
		if err != nil {
			m.logger.Warn("error to handle shared request to join from sync all requests to join msg",
				zap.String("communityID", community.IDString()),
				zap.String("requestToJoinID", types.Bytes2Hex(requestToJoin.ID)),
				zap.String("publicKey", requestToJoin.PublicKey),
				zap.String("error", err.Error()))
			continue
		}

		if requestToJoin.State != RequestToJoinStateAccepted {
			nonAcceptedRequestsToJoin = append(nonAcceptedRequestsToJoin, requestToJoin)
		}
	}
	return nonAcceptedRequestsToJoin, nil
}

func (m *Manager) HandleEditSharedAddressesPrivilegedUserSyncMessage(message *protobuf.CommunityPrivilegedUserSyncMessage, community *Community) error {
	if !(community.IsTokenMaster() || community.IsOwner()) {
		return ErrNotEnoughPermissions
	}

	publicKey := message.SyncEditSharedAddresses.PublicKey
	editSharedAddress := message.SyncEditSharedAddresses.EditSharedAddress
	if err := community.ValidateEditSharedAddresses(publicKey, editSharedAddress); err != nil {
		return err
	}

	return m.handleCommunityEditSharedAddresses(publicKey, community.ID(), editSharedAddress.RevealedAccounts, message.Clock)
}

func (m *Manager) processPrivilegedUserSharedRequestToJoin(community *Community, requestToJoin *RequestToJoin) error {
	existingRequestToJoin, err := m.persistence.GetCommunityRequestToJoinWithRevealedAddresses(requestToJoin.PublicKey, community.ID())
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	statusUpdate := existingRequestToJoin != nil && existingRequestToJoin.Clock == requestToJoin.Clock

	if existingRequestToJoin != nil && existingRequestToJoin.Clock > requestToJoin.Clock {
		return ErrOutdatedSharedRequestToJoinClock
	}

	revealedAccountsExists := requestToJoin.RevealedAccounts != nil && len(requestToJoin.RevealedAccounts) > 0

	if member, memberExists := community.Members()[requestToJoin.PublicKey]; memberExists && member.LastUpdateClock > requestToJoin.Clock {
		return ErrOutdatedSharedRequestToJoinClock
	}

	if statusUpdate {
		isCurrentStateAccepted := existingRequestToJoin.State == RequestToJoinStateAccepted
		isNewRequestAccepted := requestToJoin.State == RequestToJoinStateAccepted
		isNewAcceptedRequestWithoutAccounts := isNewRequestAccepted && !revealedAccountsExists
		isCurrentStateDeclined := existingRequestToJoin.State == RequestToJoinStateDeclined

		if (isCurrentStateAccepted && (!isNewRequestAccepted || isNewAcceptedRequestWithoutAccounts)) ||
			(isCurrentStateDeclined && !isNewRequestAccepted) {
			return ErrOutdatedSharedRequestToJoinState
		}

		err = m.persistence.SetRequestToJoinState(requestToJoin.PublicKey, community.ID(), requestToJoin.State)
		if err != nil {
			return err
		}
	} else {
		err = m.persistence.SaveRequestToJoin(requestToJoin)
		if err != nil {
			return err
		}
	}

	// If we are a token master or owner without private key and we received request to join without
	// revealed accounts - there is a chance, that we lost our role and did't get
	// CommunityDescription update. But it also can indicate, that we received an outdated
	// request to join, when we were admins
	// Decision - is not to delete existing revealed accounts
	if (community.IsTokenMaster() || community.IsOwner()) && !revealedAccountsExists {
		return nil
	}

	err = m.persistence.RemoveRequestToJoinRevealedAddresses(requestToJoin.ID)
	if err != nil {
		return err
	}

	if revealedAccountsExists {
		return m.persistence.SaveRequestToJoinRevealedAddresses(requestToJoin.ID, requestToJoin.RevealedAccounts)
	}

	return nil
}
