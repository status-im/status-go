package communities

import (
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func createTestCommunity(identity *ecdsa.PrivateKey) (*Community, error) {
	config := Config{
		PrivateKey: identity,
		CommunityDescription: &protobuf.CommunityDescription{
			Members:                 map[string]*protobuf.CommunityMember{},
			Permissions:             &protobuf.CommunityPermissions{},
			Identity:                &protobuf.ChatIdentity{},
			Chats:                   map[string]*protobuf.CommunityChat{},
			BanList:                 []string{},
			Categories:              map[string]*protobuf.CommunityCategory{},
			Encrypted:               false,
			TokenPermissions:        map[string]*protobuf.CommunityTokenPermission{},
			CommunityTokensMetadata: []*protobuf.CommunityTokenMetadata{},
		},
		ID:             &identity.PublicKey,
		Joined:         true,
		MemberIdentity: &identity.PublicKey,
	}

	return New(config, &TimeSourceStub{})
}

func TestCommunityEncryptionKeyActionSuite(t *testing.T) {
	suite.Run(t, new(CommunityEncryptionKeyActionSuite))
}

type CommunityEncryptionKeyActionSuite struct {
	suite.Suite

	identity    *ecdsa.PrivateKey
	communityID []byte

	member1 *ecdsa.PrivateKey
	member2 *ecdsa.PrivateKey
	member3 *ecdsa.PrivateKey

	member1Key string
	member2Key string
	member3Key string
}

func (s *CommunityEncryptionKeyActionSuite) SetupTest() {
	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.identity = identity
	s.communityID = crypto.CompressPubkey(&identity.PublicKey)

	member1, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.member1 = member1

	member2, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.member2 = member2

	member3, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.member3 = member3

	s.member1Key = common.PubkeyToHex(&s.member1.PublicKey)
	s.member2Key = common.PubkeyToHex(&s.member2.PublicKey)
	s.member3Key = common.PubkeyToHex(&s.member3.PublicKey)
}

func (s *CommunityEncryptionKeyActionSuite) TestEncryptionKeyNone() {
	origin, err := createTestCommunity(s.identity)
	s.Require().NoError(err)

	// if there are no changes there should be no actions
	actions := EvaluateCommunityEncryptionKeyActions(origin, origin)
	s.Require().Equal(actions.CommunityKeyAction.ActionType, EncryptionKeyNone)
	s.Require().Len(actions.ChannelKeysActions, 0)
}

func (s *CommunityEncryptionKeyActionSuite) TestCommunityLevelKeyActions_PermissionsCombinations() {
	testCases := []struct {
		name                string
		originPermissions   []*protobuf.CommunityTokenPermission
		modifiedPermissions []*protobuf.CommunityTokenPermission
		expectedActionType  EncryptionKeyActionType
	}{
		{
			name:              "add member permission",
			originPermissions: []*protobuf.CommunityTokenPermission{},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			expectedActionType: EncryptionKeyAdd,
		},
		{
			name:              "add member permissions",
			originPermissions: []*protobuf.CommunityTokenPermission{},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-2",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			expectedActionType: EncryptionKeyAdd,
		},
		{
			name: "add another member permission",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-2",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			expectedActionType: EncryptionKeyNone,
		},
		{
			name: "add another member permission and remove previous one",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-2",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			expectedActionType: EncryptionKeyNone,
		},
		{
			name: "remove member permission",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{},
			expectedActionType:  EncryptionKeyRemove,
		},
		{
			name: "remove one of member permissions",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-2",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			expectedActionType: EncryptionKeyNone,
		},
		{
			name:              "add channel permission",
			originPermissions: []*protobuf.CommunityTokenPermission{},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{"some-chat-id"},
				},
			},
			expectedActionType: EncryptionKeyNone,
		},
		{
			name: "remove channel permission",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{"some-chat-id"},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{},
			expectedActionType:  EncryptionKeyNone,
		},
		{
			name: "add member permission on top of channel permission",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{"some-chat-id"},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{"some-chat-id"},
				},
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-2",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
			},
			expectedActionType: EncryptionKeyAdd,
		},
		{
			name: "add channel permission on top of member permission",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-2",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{"some-chat-id"},
				},
			},
			expectedActionType: EncryptionKeyNone,
		},
		{
			name: "change member permission to channel permission",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
			},
			expectedActionType: EncryptionKeyRemove,
		},
		{
			name: "change channel permission to member permission",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
			},
			expectedActionType: EncryptionKeyAdd,
		},
		{
			name: "change channel permission to member permission on top of member permission",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-2",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-1",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
				&protobuf.CommunityTokenPermission{
					Id:            "some-id-2",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{""},
				},
			},
			expectedActionType: EncryptionKeyNone,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			origin, err := createTestCommunity(s.identity)
			s.Require().NoError(err)
			modified := origin.CreateDeepCopy()

			for _, permission := range tc.originPermissions {
				_, err := origin.UpsertTokenPermission(permission)
				s.Require().NoError(err)
			}

			for _, permission := range tc.modifiedPermissions {
				_, err := modified.UpsertTokenPermission(permission)
				s.Require().NoError(err)
			}

			actions := EvaluateCommunityEncryptionKeyActions(origin, modified)
			s.Require().Equal(tc.expectedActionType, actions.CommunityKeyAction.ActionType)
		})
	}
}

func (s *CommunityEncryptionKeyActionSuite) TestCommunityLevelKeyActions_MembersCombinations() {
	testCases := []struct {
		name            string
		permissions     []*protobuf.CommunityTokenPermission
		originMembers   []*ecdsa.PublicKey
		modifiedMembers []*ecdsa.PublicKey
		expectedAction  EncryptionKeyAction
	}{
		{
			name:            "add member to open community",
			permissions:     []*protobuf.CommunityTokenPermission{},
			originMembers:   []*ecdsa.PublicKey{},
			modifiedMembers: []*ecdsa.PublicKey{&s.member1.PublicKey},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeyNone,
				Members:    map[string]*protobuf.CommunityMember{},
			},
		},
		{
			name:            "remove member from open community",
			permissions:     []*protobuf.CommunityTokenPermission{},
			originMembers:   []*ecdsa.PublicKey{&s.member1.PublicKey},
			modifiedMembers: []*ecdsa.PublicKey{},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeyNone,
				Members:    map[string]*protobuf.CommunityMember{},
			},
		},
		{
			name: "add member to token-gated community",
			permissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			originMembers:   []*ecdsa.PublicKey{},
			modifiedMembers: []*ecdsa.PublicKey{&s.member1.PublicKey},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeySendToMembers,
				Members: map[string]*protobuf.CommunityMember{
					s.member1Key: &protobuf.CommunityMember{},
				},
			},
		},
		{
			name: "add multiple members to token-gated community",
			permissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			originMembers:   []*ecdsa.PublicKey{},
			modifiedMembers: []*ecdsa.PublicKey{&s.member1.PublicKey, &s.member2.PublicKey},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeySendToMembers,
				Members: map[string]*protobuf.CommunityMember{
					s.member1Key: &protobuf.CommunityMember{},
					s.member2Key: &protobuf.CommunityMember{},
				},
			},
		},
		{
			name: "remove member from token-gated community",
			permissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			originMembers:   []*ecdsa.PublicKey{&s.member1.PublicKey, &s.member2.PublicKey},
			modifiedMembers: []*ecdsa.PublicKey{&s.member1.PublicKey},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeyRekey,
				Members: map[string]*protobuf.CommunityMember{
					s.member1Key: &protobuf.CommunityMember{},
				},
			},
		},
		{
			name: "add and remove members from token-gated community",
			permissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			originMembers:   []*ecdsa.PublicKey{&s.member1.PublicKey},
			modifiedMembers: []*ecdsa.PublicKey{&s.member2.PublicKey, &s.member3.PublicKey},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeyRekey,
				Members: map[string]*protobuf.CommunityMember{
					s.member2Key: &protobuf.CommunityMember{},
					s.member3Key: &protobuf.CommunityMember{},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			origin, err := createTestCommunity(s.identity)
			s.Require().NoError(err)

			for _, permission := range tc.permissions {
				_, err := origin.UpsertTokenPermission(permission)
				s.Require().NoError(err)
			}
			modified := origin.CreateDeepCopy()

			for _, member := range tc.originMembers {
				_, err := origin.AddMember(member, []protobuf.CommunityMember_Roles{})
				s.Require().NoError(err)
			}

			for _, member := range tc.modifiedMembers {
				_, err := modified.AddMember(member, []protobuf.CommunityMember_Roles{})
				s.Require().NoError(err)
			}

			actions := EvaluateCommunityEncryptionKeyActions(origin, modified)
			s.Require().Equal(tc.expectedAction.ActionType, actions.CommunityKeyAction.ActionType)
			s.Require().Len(tc.expectedAction.Members, len(actions.CommunityKeyAction.Members))
			for memberKey := range tc.expectedAction.Members {
				_, exists := actions.CommunityKeyAction.Members[memberKey]
				s.Require().True(exists)
			}
		})
	}
}

func (s *CommunityEncryptionKeyActionSuite) TestCommunityLevelKeyActions_PermissionsMembersCombinations() {
	testCases := []struct {
		name                string
		originPermissions   []*protobuf.CommunityTokenPermission
		modifiedPermissions []*protobuf.CommunityTokenPermission
		originMembers       []*ecdsa.PublicKey
		modifiedMembers     []*ecdsa.PublicKey
		expectedActionType  EncryptionKeyActionType
	}{
		{
			name:              "add member permission, add members",
			originPermissions: []*protobuf.CommunityTokenPermission{},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			originMembers:      []*ecdsa.PublicKey{},
			modifiedMembers:    []*ecdsa.PublicKey{&s.member1.PublicKey},
			expectedActionType: EncryptionKeyAdd,
		},
		{
			name:              "add member permission, remove members",
			originPermissions: []*protobuf.CommunityTokenPermission{},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			originMembers:      []*ecdsa.PublicKey{&s.member1.PublicKey},
			modifiedMembers:    []*ecdsa.PublicKey{},
			expectedActionType: EncryptionKeyAdd,
		},
		{
			name: "remove member permission, add members",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{},
			originMembers:       []*ecdsa.PublicKey{},
			modifiedMembers:     []*ecdsa.PublicKey{&s.member1.PublicKey},
			expectedActionType:  EncryptionKeyRemove,
		},
		{
			name: "remove member permission, remove members",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{},
			originMembers:       []*ecdsa.PublicKey{&s.member1.PublicKey},
			modifiedMembers:     []*ecdsa.PublicKey{},
			expectedActionType:  EncryptionKeyRemove,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			origin, err := createTestCommunity(s.identity)
			s.Require().NoError(err)
			modified := origin.CreateDeepCopy()

			for _, permission := range tc.originPermissions {
				_, err := origin.UpsertTokenPermission(permission)
				s.Require().NoError(err)
			}
			for _, member := range tc.originMembers {
				_, err := origin.AddMember(member, []protobuf.CommunityMember_Roles{})
				s.Require().NoError(err)
			}

			for _, permission := range tc.modifiedPermissions {
				_, err := modified.UpsertTokenPermission(permission)
				s.Require().NoError(err)
			}
			for _, member := range tc.modifiedMembers {
				_, err := modified.AddMember(member, []protobuf.CommunityMember_Roles{})
				s.Require().NoError(err)
			}

			actions := EvaluateCommunityEncryptionKeyActions(origin, modified)
			s.Require().Equal(tc.expectedActionType, actions.CommunityKeyAction.ActionType)
		})
	}
}

func (s *CommunityEncryptionKeyActionSuite) TestChannelLevelKeyActions() {
	channelID := "1234"
	chatID := types.EncodeHex(crypto.CompressPubkey(&s.identity.PublicKey)) + channelID
	testCases := []struct {
		name                string
		originPermissions   []*protobuf.CommunityTokenPermission
		modifiedPermissions []*protobuf.CommunityTokenPermission
		originMembers       []*ecdsa.PublicKey
		modifiedMembers     []*ecdsa.PublicKey
		expectedAction      EncryptionKeyAction
	}{
		{
			name:              "add channel permission",
			originPermissions: []*protobuf.CommunityTokenPermission{},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{chatID},
				},
			},
			originMembers:   []*ecdsa.PublicKey{},
			modifiedMembers: []*ecdsa.PublicKey{},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeyAdd,
				Members:    map[string]*protobuf.CommunityMember{},
			},
		},
		{
			name: "remove channel permission",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{chatID},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{},
			originMembers:       []*ecdsa.PublicKey{},
			modifiedMembers:     []*ecdsa.PublicKey{},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeyRemove,
				Members:    map[string]*protobuf.CommunityMember{},
			},
		},
		{
			name: "add members to token-gated channel",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{chatID},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{chatID},
				},
			},
			originMembers:   []*ecdsa.PublicKey{},
			modifiedMembers: []*ecdsa.PublicKey{&s.member1.PublicKey, &s.member2.PublicKey},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeySendToMembers,
				Members: map[string]*protobuf.CommunityMember{
					s.member1Key: &protobuf.CommunityMember{},
					s.member2Key: &protobuf.CommunityMember{},
				},
			},
		},
		{
			name: "remove members from token-gated channel",
			originPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{chatID},
				},
			},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{
				&protobuf.CommunityTokenPermission{
					Id:            "some-id",
					Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
					TokenCriteria: make([]*protobuf.TokenCriteria, 0),
					ChatIds:       []string{chatID},
				},
			},
			originMembers:   []*ecdsa.PublicKey{&s.member1.PublicKey, &s.member2.PublicKey},
			modifiedMembers: []*ecdsa.PublicKey{},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeyRekey,
				Members:    map[string]*protobuf.CommunityMember{},
			},
		},
		{
			name:                "add members to open channel",
			originPermissions:   []*protobuf.CommunityTokenPermission{},
			modifiedPermissions: []*protobuf.CommunityTokenPermission{},
			originMembers:       []*ecdsa.PublicKey{},
			modifiedMembers:     []*ecdsa.PublicKey{&s.member1.PublicKey, &s.member2.PublicKey},
			expectedAction: EncryptionKeyAction{
				ActionType: EncryptionKeyNone,
				Members:    map[string]*protobuf.CommunityMember{},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			origin, err := createTestCommunity(s.identity)
			s.Require().NoError(err)

			_, err = origin.CreateChat(channelID, &protobuf.CommunityChat{
				Members:     map[string]*protobuf.CommunityMember{},
				Permissions: &protobuf.CommunityPermissions{Access: protobuf.CommunityPermissions_NO_MEMBERSHIP},
				Identity:    &protobuf.ChatIdentity{},
			})
			s.Require().NoError(err)

			modified := origin.CreateDeepCopy()

			for _, permission := range tc.originPermissions {
				_, err := origin.UpsertTokenPermission(permission)
				s.Require().NoError(err)
			}
			for _, member := range tc.originMembers {
				_, err := origin.AddMember(member, []protobuf.CommunityMember_Roles{})
				s.Require().NoError(err)
				_, err = origin.AddMemberToChat(channelID, member, []protobuf.CommunityMember_Roles{})
				s.Require().NoError(err)
			}

			for _, permission := range tc.modifiedPermissions {
				_, err := modified.UpsertTokenPermission(permission)
				s.Require().NoError(err)
			}
			for _, member := range tc.modifiedMembers {
				_, err := modified.AddMember(member, []protobuf.CommunityMember_Roles{})
				s.Require().NoError(err)
				_, err = modified.AddMemberToChat(channelID, member, []protobuf.CommunityMember_Roles{})
				s.Require().NoError(err)
			}

			actions := EvaluateCommunityEncryptionKeyActions(origin, modified)
			channelAction, ok := actions.ChannelKeysActions[channelID]
			s.Require().True(ok)
			s.Require().Equal(tc.expectedAction.ActionType, channelAction.ActionType)
			s.Require().Len(tc.expectedAction.Members, len(channelAction.Members))
			for memberKey := range tc.expectedAction.Members {
				_, exists := channelAction.Members[memberKey]
				s.Require().True(exists)
			}
		})
	}
}

func (s *CommunityEncryptionKeyActionSuite) TestNilOrigin() {
	newCommunity, err := createTestCommunity(s.identity)
	s.Require().NoError(err)

	channelID := "0x1234"
	chatID := types.EncodeHex(crypto.CompressPubkey(&s.identity.PublicKey)) + channelID

	_, err = newCommunity.CreateChat(channelID, &protobuf.CommunityChat{
		Members:     map[string]*protobuf.CommunityMember{},
		Permissions: &protobuf.CommunityPermissions{Access: protobuf.CommunityPermissions_NO_MEMBERSHIP},
		Identity:    &protobuf.ChatIdentity{},
	})
	s.Require().NoError(err)

	newCommunityPermissions := []*protobuf.CommunityTokenPermission{
		&protobuf.CommunityTokenPermission{
			Id:            "some-id-1",
			Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
			TokenCriteria: make([]*protobuf.TokenCriteria, 0),
			ChatIds:       []string{},
		},
		&protobuf.CommunityTokenPermission{
			Id:            "some-id-2",
			Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
			TokenCriteria: make([]*protobuf.TokenCriteria, 0),
			ChatIds:       []string{chatID},
		},
	}
	for _, permission := range newCommunityPermissions {
		_, err := newCommunity.UpsertTokenPermission(permission)
		s.Require().NoError(err)
	}

	actions := EvaluateCommunityEncryptionKeyActions(nil, newCommunity)
	s.Require().Equal(actions.CommunityKeyAction.ActionType, EncryptionKeyAdd)
	s.Require().Len(actions.ChannelKeysActions, 1)
	s.Require().NotNil(actions.ChannelKeysActions[channelID])
	s.Require().Equal(actions.ChannelKeysActions[channelID].ActionType, EncryptionKeyAdd)
}
