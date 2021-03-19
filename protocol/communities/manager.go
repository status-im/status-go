package communities

import (
	"crypto/ecdsa"
	"database/sql"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

type Manager struct {
	persistence     *Persistence
	ensSubscription chan []*ens.VerificationRecord
	subscriptions   []chan *Subscription
	ensVerifier     *ens.Verifier
	identity        *ecdsa.PublicKey
	logger          *zap.Logger
	quit            chan struct{}
}

func NewManager(identity *ecdsa.PublicKey, db *sql.DB, logger *zap.Logger, verifier *ens.Verifier) (*Manager, error) {
	if identity == nil {
		return nil, errors.New("empty identity")
	}

	var err error
	if logger == nil {
		if logger, err = zap.NewDevelopment(); err != nil {
			return nil, errors.Wrap(err, "failed to create a logger")
		}
	}

	manager := &Manager{
		logger:   logger,
		identity: identity,
		quit:     make(chan struct{}),
		persistence: &Persistence{
			logger: logger,
			db:     db,
		},
	}

	if verifier != nil {

		sub := verifier.Subscribe()
		manager.ensSubscription = sub
		manager.ensVerifier = verifier
	}

	return manager, nil
}

type Subscription struct {
	Community   *Community
	Invitations []*protobuf.CommunityInvitation
}

type CommunityResponse struct {
	Community *Community        `json:"community"`
	Changes   *CommunityChanges `json:"changes"`
}

func (m *Manager) Subscribe() chan *Subscription {
	subscription := make(chan *Subscription, 100)
	m.subscriptions = append(m.subscriptions, subscription)
	return subscription
}

func (m *Manager) Start() error {
	if m.ensVerifier != nil {
		m.runENSVerificationLoop()
	}
	return nil
}

func (m *Manager) runENSVerificationLoop() {
	go func() {
		for {
			select {
			case <-m.quit:
				m.logger.Debug("quitting ens verification loop")
				return
			case records, more := <-m.ensSubscription:
				if !more {
					m.logger.Debug("no more ens records, quitting")
					return
				}
				m.logger.Info("received records", zap.Any("records", records))

			}
		}
	}()
}

func (m *Manager) Stop() error {
	close(m.quit)
	for _, c := range m.subscriptions {
		close(c)
	}
	return nil
}

func (m *Manager) publish(subscription *Subscription) {
	for _, s := range m.subscriptions {
		select {
		case s <- subscription:
		default:
			m.logger.Warn("subscription channel full, dropping message")
		}
	}
}

func (m *Manager) All() ([]*Community, error) {
	return m.persistence.AllCommunities(m.identity)
}

func (m *Manager) Joined() ([]*Community, error) {
	return m.persistence.JoinedCommunities(m.identity)
}

func (m *Manager) Created() ([]*Community, error) {
	return m.persistence.CreatedCommunities(m.identity)
}

// CreateCommunity takes a description, generates an ID for it, saves it and return it
func (m *Manager) CreateCommunity(description *protobuf.CommunityDescription) (*Community, error) {
	err := ValidateCommunityDescription(description)
	if err != nil {
		return nil, err
	}

	description.Clock = 1

	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	config := Config{
		ID:                   &key.PublicKey,
		PrivateKey:           key,
		Logger:               m.logger,
		Joined:               true,
		MemberIdentity:       m.identity,
		CommunityDescription: description,
	}
	community, err := New(config)
	if err != nil {
		return nil, err
	}

	// We join any community we create
	community.Join()

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community})

	return community, nil
}

func (m *Manager) ExportCommunity(id types.HexBytes) (*ecdsa.PrivateKey, error) {
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}

	if community.config.PrivateKey == nil {
		return nil, errors.New("not an admin")
	}

	return community.config.PrivateKey, nil
}

func (m *Manager) ImportCommunity(key *ecdsa.PrivateKey) (*Community, error) {
	communityID := crypto.CompressPubkey(&key.PublicKey)

	community, err := m.persistence.GetByID(m.identity, communityID)
	if err != nil {
		return nil, err
	}

	if community == nil {
		description := &protobuf.CommunityDescription{
			Permissions: &protobuf.CommunityPermissions{},
		}

		config := Config{
			ID:                   &key.PublicKey,
			PrivateKey:           key,
			Logger:               m.logger,
			Joined:               true,
			MemberIdentity:       m.identity,
			CommunityDescription: description,
		}
		community, err = New(config)
		if err != nil {
			return nil, err
		}
	} else {
		community.config.PrivateKey = key
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) CreateChat(communityID types.HexBytes, chat *protobuf.CommunityChat) (*Community, *CommunityChanges, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, nil, err
	}
	if community == nil {
		return nil, nil, ErrOrgNotFound
	}
	chatID := uuid.New().String()
	changes, err := community.CreateChat(chatID, chat)
	if err != nil {
		return nil, nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, nil, err
	}

	// Advertise changes
	m.publish(&Subscription{Community: community})

	return community, changes, nil
}

func (m *Manager) HandleCommunityDescriptionMessage(signer *ecdsa.PublicKey, description *protobuf.CommunityDescription, payload []byte) (*CommunityResponse, error) {
	id := crypto.CompressPubkey(signer)
	community, err := m.persistence.GetByID(m.identity, id)
	if err != nil {
		return nil, err
	}

	if community == nil {
		config := Config{
			CommunityDescription:          description,
			Logger:                        m.logger,
			MarshaledCommunityDescription: payload,
			MemberIdentity:                m.identity,
			ID:                            signer,
		}

		community, err = New(config)
		if err != nil {
			return nil, err
		}
	}

	changes, err := community.UpdateCommunityDescription(signer, description, payload)
	if err != nil {
		return nil, err
	}

	pkString := common.PubkeyToHex(m.identity)

	// If the community require membership, we set whether we should leave/join the community after a state change
	if community.InvitationOnly() || community.OnRequest() {
		if changes.HasNewMember(pkString) {
			hasPendingRequest, err := m.persistence.HasPendingRequestsToJoinForUserAndCommunity(pkString, changes.Community.ID())
			if err != nil {
				return nil, err
			}
			// If there's any pending request, we should join the community
			// automatically
			changes.ShouldMemberJoin = hasPendingRequest
		}

		if changes.HasMemberLeft(pkString) {
			// If we joined previously the community, we should leave it
			changes.ShouldMemberLeave = community.Joined()
		}
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	// We mark our requests as completed, though maybe we should mark
	// any request for any user that has been added as completed
	if err := m.markRequestToJoin(m.identity, community); err != nil {
		return nil, err
	}
	// Check if there's a change and we should be joining

	return &CommunityResponse{
		Community: community,
		Changes:   changes,
	}, nil
}

// TODO: This is not fully implemented, we want to save the grant passed at
// this stage and make sure it's used when publishing.
func (m *Manager) HandleCommunityInvitation(signer *ecdsa.PublicKey, invitation *protobuf.CommunityInvitation, payload []byte) (*CommunityResponse, error) {
	m.logger.Debug("Handling wrapped community description message")

	community, err := m.HandleWrappedCommunityDescriptionMessage(payload)
	if err != nil {
		return nil, err
	}

	// Save grant

	return community, nil
}

// markRequestToJoin marks all the pending requests to join as completed
// if we are members
func (m *Manager) markRequestToJoin(pk *ecdsa.PublicKey, community *Community) error {
	if community.HasMember(pk) {
		return m.persistence.SetRequestToJoinState(common.PubkeyToHex(pk), community.ID(), RequestToJoinStateAccepted)
	}
	return nil
}

func (m *Manager) AcceptRequestToJoin(request *requests.AcceptRequestToJoinCommunity) (*Community, error) {
	dbRequest, err := m.persistence.GetRequestToJoin(request.ID)
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(dbRequest.CommunityID)
	if err != nil {
		return nil, err
	}

	pk, err := common.HexToPubkey(dbRequest.PublicKey)
	if err != nil {
		return nil, err
	}

	return m.inviteUsersToCommunity(community, []*ecdsa.PublicKey{pk})
}

func (m *Manager) DeclineRequestToJoin(request *requests.DeclineRequestToJoinCommunity) error {
	dbRequest, err := m.persistence.GetRequestToJoin(request.ID)
	if err != nil {
		return err
	}

	return m.persistence.SetRequestToJoinState(dbRequest.PublicKey, dbRequest.CommunityID, RequestToJoinStateDeclined)

}

func (m *Manager) HandleCommunityRequestToJoin(signer *ecdsa.PublicKey, request *protobuf.CommunityRequestToJoin) (*RequestToJoin, error) {
	community, err := m.persistence.GetByID(m.identity, request.CommunityId)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	// If they are already a member, ignore
	if community.HasMember(signer) {
		return nil, ErrAlreadyMember
	}

	if err := community.ValidateRequestToJoin(signer, request); err != nil {
		return nil, err
	}

	requestToJoin := &RequestToJoin{
		PublicKey:   common.PubkeyToHex(signer),
		Clock:       request.Clock,
		ENSName:     request.EnsName,
		CommunityID: request.CommunityId,
		State:       RequestToJoinStatePending,
	}

	requestToJoin.CalculateID()

	if err := m.persistence.SaveRequestToJoin(requestToJoin); err != nil {
		return nil, err
	}

	return requestToJoin, nil
}

func (m *Manager) HandleWrappedCommunityDescriptionMessage(payload []byte) (*CommunityResponse, error) {
	m.logger.Debug("Handling wrapped community description message")

	applicationMetadataMessage := &protobuf.ApplicationMetadataMessage{}
	err := proto.Unmarshal(payload, applicationMetadataMessage)
	if err != nil {
		return nil, err
	}
	if applicationMetadataMessage.Type != protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION {
		return nil, ErrInvalidMessage
	}
	signer, err := applicationMetadataMessage.RecoverKey()
	if err != nil {
		return nil, err
	}

	description := &protobuf.CommunityDescription{}

	err = proto.Unmarshal(applicationMetadataMessage.Payload, description)
	if err != nil {
		return nil, err
	}

	return m.HandleCommunityDescriptionMessage(signer, description, payload)
}

func (m *Manager) JoinCommunity(id types.HexBytes) (*Community, error) {
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}
	community.Join()
	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}
	return community, nil
}

func (m *Manager) LeaveCommunity(id types.HexBytes) (*Community, error) {
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}
	community.Leave()
	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}
	return community, nil
}

func (m *Manager) inviteUsersToCommunity(community *Community, pks []*ecdsa.PublicKey) (*Community, error) {
	var invitations []*protobuf.CommunityInvitation
	for _, pk := range pks {
		invitation, err := community.InviteUserToOrg(pk)
		if err != nil {
			return nil, err
		}
		// We mark the user request (if any) as completed
		if err := m.markRequestToJoin(pk, community); err != nil {
			return nil, err
		}

		invitations = append(invitations, invitation)
	}

	err := m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community, Invitations: invitations})

	return community, nil
}

func (m *Manager) InviteUsersToCommunity(communityID types.HexBytes, pks []*ecdsa.PublicKey) (*Community, error) {
	community, err := m.GetByID(communityID)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	return m.inviteUsersToCommunity(community, pks)
}

func (m *Manager) RemoveUserFromCommunity(id types.HexBytes, pk *ecdsa.PublicKey) (*Community, error) {
	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	_, err = community.RemoveUserFromOrg(pk)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community})

	return community, nil
}

func (m *Manager) BanUserFromCommunity(request *requests.BanUserFromCommunity) (*Community, error) {
	id := request.CommunityID

	publicKey, err := common.HexToPubkey(request.User.String())
	if err != nil {
		return nil, err
	}

	community, err := m.GetByID(id)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	_, err = community.BanUserFromCommunity(publicKey)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community})

	return community, nil
}

func (m *Manager) GetByID(id []byte) (*Community, error) {
	return m.persistence.GetByID(m.identity, id)
}

func (m *Manager) GetByIDString(idString string) (*Community, error) {
	id, err := types.DecodeHex(idString)
	if err != nil {
		return nil, err
	}
	return m.GetByID(id)
}

func (m *Manager) RequestToJoin(requester *ecdsa.PublicKey, request *requests.RequestToJoinCommunity) (*Community, *RequestToJoin, error) {
	community, err := m.persistence.GetByID(m.identity, request.CommunityID)
	if err != nil {
		return nil, nil, err
	}

	// We don't allow requesting access if already a member
	if community.HasMember(m.identity) {
		return nil, nil, ErrAlreadyMember
	}

	clock := uint64(time.Now().Unix())
	requestToJoin := &RequestToJoin{
		PublicKey:   common.PubkeyToHex(requester),
		Clock:       clock,
		ENSName:     request.ENSName,
		CommunityID: request.CommunityID,
		State:       RequestToJoinStatePending,
		Our:         true,
	}

	requestToJoin.CalculateID()

	if err := m.persistence.SaveRequestToJoin(requestToJoin); err != nil {
		return nil, nil, err
	}
	community.config.RequestedToJoinAt = uint64(time.Now().Unix())

	return community, requestToJoin, nil
}

func (m *Manager) PendingRequestsToJoinForUser(pk *ecdsa.PublicKey) ([]*RequestToJoin, error) {
	return m.persistence.PendingRequestsToJoinForUser(common.PubkeyToHex(pk))
}

func (m *Manager) PendingRequestsToJoinForCommunity(id types.HexBytes) ([]*RequestToJoin, error) {
	m.logger.Info("fetching pending invitations", zap.String("community-id", id.String()))
	return m.persistence.PendingRequestsToJoinForCommunity(id)
}

func (m *Manager) CanPost(pk *ecdsa.PublicKey, communityID string, chatID string, grant []byte) (bool, error) {
	community, err := m.GetByIDString(communityID)
	if err != nil {
		return false, err
	}
	if community == nil {
		return false, nil
	}
	return community.CanPost(pk, chatID, grant)
}
