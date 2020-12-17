package communities

import (
	"crypto/ecdsa"
	"database/sql"

	"github.com/golang/protobuf/proto"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

type Manager struct {
	persistence   *Persistence
	subscriptions []chan *Subscription
	logger        *zap.Logger
}

func NewManager(db *sql.DB, logger *zap.Logger) (*Manager, error) {
	var err error
	if logger == nil {
		if logger, err = zap.NewDevelopment(); err != nil {
			return nil, errors.Wrap(err, "failed to create a logger")
		}
	}

	return &Manager{
		logger: logger,
		persistence: &Persistence{
			logger: logger,
			db:     db,
		},
	}, nil
}

type Subscription struct {
	Community  *Community
	Invitation *protobuf.CommunityInvitation
}

func (m *Manager) Subscribe() chan *Subscription {
	subscription := make(chan *Subscription, 100)
	m.subscriptions = append(m.subscriptions, subscription)
	return subscription
}

func (m *Manager) Stop() error {
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
	return m.persistence.AllCommunities()
}

func (m *Manager) Joined() ([]*Community, error) {
	return m.persistence.JoinedCommunities()
}

func (m *Manager) Created() ([]*Community, error) {
	return m.persistence.CreatedCommunities()
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
		CommunityDescription: description,
	}
	community, err := New(config)
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

func (m *Manager) ExportCommunity(idString string) (*ecdsa.PrivateKey, error) {
	community, err := m.GetByIDString(idString)
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

	community, err := m.persistence.GetByID(communityID)
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

func (m *Manager) CreateChat(idString string, chat *protobuf.CommunityChat) (*Community, *CommunityChanges, error) {
	community, err := m.GetByIDString(idString)
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

func (m *Manager) HandleCommunityDescriptionMessage(signer *ecdsa.PublicKey, description *protobuf.CommunityDescription, payload []byte) (*Community, error) {
	id := crypto.CompressPubkey(signer)
	community, err := m.persistence.GetByID(id)
	if err != nil {
		return nil, err
	}

	if community == nil {
		config := Config{
			CommunityDescription:          description,
			Logger:                        m.logger,
			MarshaledCommunityDescription: payload,
			ID:                            signer,
		}

		community, err = New(config)
		if err != nil {
			return nil, err
		}
	}

	_, err = community.HandleCommunityDescription(signer, description, payload)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	return community, nil
}

func (m *Manager) HandleCommunityInvitation(signer *ecdsa.PublicKey, invitation *protobuf.CommunityInvitation, payload []byte) (*Community, error) {
	m.logger.Debug("Handling wrapped community description message")

	community, err := m.HandleWrappedCommunityDescriptionMessage(payload)
	if err != nil {
		return nil, err
	}

	// Save grant

	return community, nil
}

func (m *Manager) HandleWrappedCommunityDescriptionMessage(payload []byte) (*Community, error) {
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

func (m *Manager) JoinCommunity(idString string) (*Community, error) {
	community, err := m.GetByIDString(idString)
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

func (m *Manager) LeaveCommunity(idString string) (*Community, error) {
	community, err := m.GetByIDString(idString)
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

func (m *Manager) InviteUserToCommunity(idString string, pk *ecdsa.PublicKey) (*Community, error) {
	community, err := m.GetByIDString(idString)
	if err != nil {
		return nil, err
	}
	if community == nil {
		return nil, ErrOrgNotFound
	}

	invitation, err := community.InviteUserToOrg(pk)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveCommunity(community)
	if err != nil {
		return nil, err
	}

	m.publish(&Subscription{Community: community, Invitation: invitation})

	return community, nil
}

func (m *Manager) GetByIDString(idString string) (*Community, error) {
	id, err := types.DecodeHex(idString)
	if err != nil {
		return nil, err
	}
	return m.persistence.GetByID(id)
}

func (m *Manager) CanPost(pk *ecdsa.PublicKey, orgIDString, chatID string, grant []byte) (bool, error) {
	community, err := m.GetByIDString(orgIDString)
	if err != nil {
		return false, err
	}
	if community == nil {
		return false, nil
	}
	return community.CanPost(pk, chatID, grant)
}
