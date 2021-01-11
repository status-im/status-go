package communities

import (
	"crypto/ecdsa"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestCommunitySuite(t *testing.T) {
	suite.Run(t, new(CommunitySuite))
}

const testChatID1 = "chat-id-1"
const testChatID2 = "chat-id-2"

type CommunitySuite struct {
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

func (s *CommunitySuite) SetupTest() {
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

func (s *CommunitySuite) TestInviteUserToOrg() {
	newMember, err := crypto.GenerateKey()
	s.Require().NoError(err)

	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = nil
	// Not an admin
	_, err = org.InviteUserToOrg(&s.member2.PublicKey)
	s.Require().Equal(ErrNotAdmin, err)

	// Add admin to community
	org.config.PrivateKey = s.identity

	response, err := org.InviteUserToOrg(&newMember.PublicKey)
	s.Require().Nil(err)
	s.Require().NotNil(response)

	// Check member has been added
	s.Require().True(org.HasMember(&newMember.PublicKey))

	// Check member has been added to response
	s.Require().NotNil(response.CommunityDescription)

	metadata := &protobuf.ApplicationMetadataMessage{}
	description := &protobuf.CommunityDescription{}

	s.Require().NoError(proto.Unmarshal(response.CommunityDescription, metadata))
	s.Require().NoError(proto.Unmarshal(metadata.Payload, description))

	_, ok := description.Members[common.PubkeyToHex(&newMember.PublicKey)]
	s.Require().True(ok)

	// Check grant validates
	s.Require().NotNil(org.config.ID)
	s.Require().NotNil(response.Grant)

	grant, err := org.VerifyGrantSignature(response.Grant)
	s.Require().NoError(err)
	s.Require().NotNil(grant)
}

func (s *CommunitySuite) TestCreateChat() {
	newChatID := "new-chat-id"
	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = nil

	identity := &protobuf.ChatIdentity{
		DisplayName: "new-chat-display-name",
		Description: "new-chat-description",
	}
	permissions := &protobuf.CommunityPermissions{
		Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	_, err := org.CreateChat(newChatID, &protobuf.CommunityChat{
		Identity:    identity,
		Permissions: permissions,
	})

	s.Require().Equal(ErrNotAdmin, err)

	org.config.PrivateKey = s.identity

	changes, err := org.CreateChat(newChatID, &protobuf.CommunityChat{
		Identity:    identity,
		Permissions: permissions,
	})

	description := org.config.CommunityDescription

	s.Require().NoError(err)
	s.Require().NotNil(description)

	s.Require().NotNil(description.Chats[newChatID])
	s.Require().NotEmpty(description.Clock)
	s.Require().Equal(permissions, description.Chats[newChatID].Permissions)
	s.Require().Equal(identity, description.Chats[newChatID].Identity)

	s.Require().NotNil(changes)
	s.Require().NotNil(changes.ChatsAdded[newChatID])
}

func (s *CommunitySuite) TestDeleteChat() {
	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = nil

	_, err := org.DeleteChat(testChatID1)
	s.Require().Equal(ErrNotAdmin, err)

	org.config.PrivateKey = s.identity

	description, err := org.DeleteChat(testChatID1)
	s.Require().NoError(err)
	s.Require().NotNil(description)

	s.Require().Nil(description.Chats[testChatID1])
	s.Require().Equal(uint64(2), description.Clock)
}

func (s *CommunitySuite) TestInviteUserToChat() {
	newMember, err := crypto.GenerateKey()
	s.Require().NoError(err)

	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = nil
	// Not an admin
	_, err = org.InviteUserToChat(&s.member2.PublicKey, testChatID1)
	s.Require().Equal(ErrNotAdmin, err)

	// Add admin to community
	org.config.PrivateKey = s.identity

	response, err := org.InviteUserToChat(&newMember.PublicKey, testChatID1)
	s.Require().Nil(err)
	s.Require().NotNil(response)

	// Check member has been added
	s.Require().True(org.HasMember(&newMember.PublicKey))
	s.Require().True(org.IsMemberInChat(&newMember.PublicKey, testChatID1))

	// Check member has been added to response
	s.Require().NotNil(response.CommunityDescription)

	metadata := &protobuf.ApplicationMetadataMessage{}
	description := &protobuf.CommunityDescription{}

	s.Require().NoError(proto.Unmarshal(response.CommunityDescription, metadata))
	s.Require().NoError(proto.Unmarshal(metadata.Payload, description))

	_, ok := description.Members[common.PubkeyToHex(&newMember.PublicKey)]
	s.Require().True(ok)

	_, ok = description.Chats[testChatID1].Members[common.PubkeyToHex(&newMember.PublicKey)]
	s.Require().True(ok)

	s.Require().Equal(testChatID1, response.ChatId)

	// Check grant validates
	s.Require().NotNil(org.config.ID)
	s.Require().NotNil(response.Grant)

	grant, err := org.VerifyGrantSignature(response.Grant)
	s.Require().NoError(err)
	s.Require().NotNil(grant)
	s.Require().Equal(testChatID1, grant.ChatId)
}

func (s *CommunitySuite) TestRemoveUserFromChat() {
	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = nil
	// Not an admin
	_, err := org.RemoveUserFromOrg(&s.member1.PublicKey)
	s.Require().Equal(ErrNotAdmin, err)

	// Add admin to community
	org.config.PrivateKey = s.identity

	actualCommunity, err := org.RemoveUserFromChat(&s.member1.PublicKey, testChatID1)
	s.Require().Nil(err)
	s.Require().NotNil(actualCommunity)

	// Check member has not been removed
	s.Require().True(org.HasMember(&s.member1.PublicKey))

	// Check member has not been removed from org
	_, ok := actualCommunity.Members[common.PubkeyToHex(&s.member1.PublicKey)]
	s.Require().True(ok)

	// Check member has been removed from chat
	_, ok = actualCommunity.Chats[testChatID1].Members[common.PubkeyToHex(&s.member1.PublicKey)]
	s.Require().False(ok)
}

func (s *CommunitySuite) TestRemoveUserFormOrg() {
	org := s.buildCommunity(&s.identity.PublicKey)
	org.config.PrivateKey = nil
	// Not an admin
	_, err := org.RemoveUserFromOrg(&s.member1.PublicKey)
	s.Require().Equal(ErrNotAdmin, err)

	// Add admin to community
	org.config.PrivateKey = s.identity

	actualCommunity, err := org.RemoveUserFromOrg(&s.member1.PublicKey)
	s.Require().Nil(err)
	s.Require().NotNil(actualCommunity)

	// Check member has been removed
	s.Require().False(org.HasMember(&s.member1.PublicKey))

	// Check member has been removed from org
	_, ok := actualCommunity.Members[common.PubkeyToHex(&s.member1.PublicKey)]
	s.Require().False(ok)

	// Check member has been removed from chat
	_, ok = actualCommunity.Chats[testChatID1].Members[common.PubkeyToHex(&s.member1.PublicKey)]
	s.Require().False(ok)
}

func (s *CommunitySuite) TestAcceptRequestToJoin() {
	// WHAT TO DO WITH ENS
	// TEST CASE 1: Not an admin
	// TEST CASE 2: No request to join
	// TEST CASE 3: Valid
}

func (s *CommunitySuite) TestDeclineRequestToJoin() {
	// TEST CASE 1: Not an admin
	// TEST CASE 2: No request to join
	// TEST CASE 3: Valid
}

func (s *CommunitySuite) TestHandleRequestJoin() {
	description := &protobuf.CommunityDescription{}

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	signer := &key.PublicKey

	request := &protobuf.CommunityRequestJoin{
		EnsName:     "donvanvliet.stateofus.eth",
		CommunityId: s.communityID,
	}

	requestWithChatID := &protobuf.CommunityRequestJoin{
		EnsName:     "donvanvliet.stateofus.eth",
		CommunityId: s.communityID,
		ChatId:      testChatID1,
	}

	requestWithoutENS := &protobuf.CommunityRequestJoin{
		CommunityId: s.communityID,
	}

	requestWithChatWithoutENS := &protobuf.CommunityRequestJoin{
		CommunityId: s.communityID,
		ChatId:      testChatID1,
	}

	// MATRIX
	// NO_MEMBERHSIP - NO_MEMBERSHIP -> Error -> Anyone can join org, chat is read/write for anyone
	// NO_MEMBRISHIP - INVITATION_ONLY -> Error -> Anyone can join org, chat is invitation only
	// NO_MEMBERSHIP - ON_REQUEST -> Success -> Anyone can join org, chat is on request and needs approval
	// INVITATION_ONLY - NO_MEMBERSHIP -> TODO -> Org is invitation only, chat is read-write for members
	// INVITATION_ONLY - INVITATION_ONLY -> Error -> Org is invitation only, chat is invitation only
	// INVITATION_ONLY - ON_REQUEST -> TODO -> Error -> Org is invitation only, member of the org need to request access for chat
	// ON_REQUEST - NO_MEMBRERSHIP -> TODO -> Error -> Org is on request, chat is read write for members
	// ON_REQUEST - INVITATION_ONLY -> Error -> Org is on request, chat is invitation only for members
	// ON_REQUEST - ON_REQUEST -> Fine -> Org is on request, chat is on request

	testCases := []struct {
		name    string
		config  Config
		request *protobuf.CommunityRequestJoin
		signer  *ecdsa.PublicKey
		err     error
	}{
		{
			name:    "on-request access to community",
			config:  s.configOnRequest(),
			signer:  signer,
			request: request,
			err:     nil,
		},
		{
			name:    "not admin",
			config:  Config{CommunityDescription: description},
			signer:  signer,
			request: request,
			err:     ErrNotAdmin,
		},
		{
			name:    "invitation-only",
			config:  s.configInvitationOnly(),
			signer:  signer,
			request: request,
			err:     ErrCantRequestAccess,
		},
		{
			name:    "ens-only org and missing ens",
			config:  s.configENSOnly(),
			signer:  signer,
			request: requestWithoutENS,
			err:     ErrCantRequestAccess,
		},
		{
			name:    "ens-only chat and missing ens",
			config:  s.configChatENSOnly(),
			signer:  signer,
			request: requestWithChatWithoutENS,
			err:     ErrCantRequestAccess,
		},
		{
			name:    "missing chat",
			config:  s.configOnRequest(),
			signer:  signer,
			request: requestWithChatID,
			err:     ErrChatNotFound,
		},
		// Org-Chat combinations
		// NO_MEMBERSHIP-NO_MEMBERSHIP = error as you should not be
		// requesting access
		{
			name:    "no-membership org with no-membeship chat",
			config:  s.configNoMembershipOrgNoMembershipChat(),
			signer:  signer,
			request: requestWithChatID,
			err:     ErrCantRequestAccess,
		},
		// NO_MEMBERSHIP-INVITATION_ONLY = error as it's invitation only
		{
			name:    "no-membership org with no-membeship chat",
			config:  s.configNoMembershipOrgInvitationOnlyChat(),
			signer:  signer,
			request: requestWithChatID,
			err:     ErrCantRequestAccess,
		},
		// NO_MEMBERSHIP-ON_REQUEST = this is a valid case
		{
			name:    "no-membership org with on-request chat",
			config:  s.configNoMembershipOrgOnRequestChat(),
			signer:  signer,
			request: requestWithChatID,
		},
		// INVITATION_ONLY-INVITATION_ONLY error as it's invitation only
		{
			name:    "invitation-only org with invitation-only chat",
			config:  s.configInvitationOnlyOrgInvitationOnlyChat(),
			signer:  signer,
			request: requestWithChatID,
			err:     ErrCantRequestAccess,
		},
		// ON_REQUEST-INVITATION_ONLY error as it's invitation only
		{
			name:    "on-request org with invitation-only chat",
			config:  s.configOnRequestOrgInvitationOnlyChat(),
			signer:  signer,
			request: requestWithChatID,
			err:     ErrCantRequestAccess,
		},
		// ON_REQUEST-INVITATION_ONLY error as it's invitation only
		{
			name:    "on-request org with invitation-only chat",
			config:  s.configOnRequestOrgInvitationOnlyChat(),
			signer:  signer,
			request: requestWithChatID,
			err:     ErrCantRequestAccess,
		},
		// ON_REQUEST-ON_REQUEST success
		{
			name:    "on-request org with on-request chat",
			config:  s.configOnRequestOrgOnRequestChat(),
			signer:  signer,
			request: requestWithChatID,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			org, err := New(tc.config)
			s.Require().NoError(err)
			err = org.HandleRequestJoin(tc.signer, tc.request)
			s.Require().Equal(tc.err, err)
		})
	}
}

func (s *CommunitySuite) TestCanPost() {
	validGrant := 1
	invalidGrant := 2

	notMember := &s.member3.PublicKey
	member := &s.member1.PublicKey

	// MEMBERSHIP-NO-MEMBERSHIP-Member-> User can post
	// MEMBERSHIP-NO-MEMEBRESHIP->NON member -> User can't post
	// MEMBERSHIP-NO-MEMBERSHIP-Grant -> user can post
	// MEMBERSHIP-NO-MEMBERSHIP-old-grant -> user can't post

	testCases := []struct {
		name    string
		config  Config
		member  *ecdsa.PublicKey
		err     error
		grant   int
		canPost bool
	}{
		{
			name:    "no-membership org with no-membeship chat",
			config:  s.configNoMembershipOrgNoMembershipChat(),
			member:  notMember,
			canPost: true,
		},
		{
			name:    "no-membership org with invitation only chat-not-a-member",
			config:  s.configNoMembershipOrgInvitationOnlyChat(),
			member:  notMember,
			canPost: false,
		},
		{
			name:    "no-membership org with invitation only chat member",
			config:  s.configNoMembershipOrgInvitationOnlyChat(),
			member:  member,
			canPost: true,
		},
		{
			name:    "no-membership org with invitation only chat-not-a-member valid grant",
			config:  s.configNoMembershipOrgInvitationOnlyChat(),
			member:  notMember,
			canPost: true,
			grant:   validGrant,
		},
		{
			name:    "no-membership org with invitation only chat-not-a-member invalid grant",
			config:  s.configNoMembershipOrgInvitationOnlyChat(),
			member:  notMember,
			canPost: false,
			grant:   invalidGrant,
		},
		{
			name:    "membership org with no-membership chat-not-a-member",
			config:  s.configOnRequestOrgNoMembershipChat(),
			member:  notMember,
			canPost: false,
		},
		{
			name:    "membership org with no-membership chat",
			config:  s.configOnRequestOrgNoMembershipChat(),
			member:  member,
			canPost: true,
		},
		{
			name:    "membership org with no-membership chat  not-a-member valid grant",
			config:  s.configOnRequestOrgNoMembershipChat(),
			member:  notMember,
			canPost: true,
			grant:   validGrant,
		},
		{
			name:    "membership org with no-membership chat not-a-member invalid grant",
			config:  s.configOnRequestOrgNoMembershipChat(),
			member:  notMember,
			canPost: false,
			grant:   invalidGrant,
		},
		{
			name:    "monsier creator can always post of course",
			config:  s.configOnRequestOrgNoMembershipChat(),
			member:  &s.identity.PublicKey,
			canPost: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			var grant []byte
			var err error
			org, err := New(tc.config)
			s.Require().NoError(err)

			if tc.grant == validGrant {
				grant, err = org.buildGrant(tc.member, testChatID1)
				// We lower the clock of the description to simulate
				// a valid use case
				org.config.CommunityDescription.Clock--
				s.Require().NoError(err)
			} else if tc.grant == invalidGrant {
				grant, err = org.buildGrant(&s.member2.PublicKey, testChatID1)
				s.Require().NoError(err)
			}
			canPost, err := org.CanPost(tc.member, testChatID1, grant)
			s.Require().Equal(tc.err, err)
			s.Require().Equal(tc.canPost, canPost)
		})
	}
}

func (s *CommunitySuite) TestHandleCommunityDescription() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	signer := &key.PublicKey

	testCases := []struct {
		name        string
		description func(*Community) *protobuf.CommunityDescription
		changes     func(*Community) *CommunityChanges
		signer      *ecdsa.PublicKey
		err         error
	}{
		{
			name:        "updated version but no changes",
			description: s.identicalCommunityDescription,
			signer:      signer,
			changes:     func(_ *Community) *CommunityChanges { return emptyCommunityChanges() },
			err:         nil,
		},
		{
			name:        "updated version but lower clock",
			description: s.oldCommunityDescription,
			signer:      signer,
			changes:     func(_ *Community) *CommunityChanges { return emptyCommunityChanges() },
			err:         nil,
		},
		{
			name:        "removed member from org",
			description: s.removedMemberCommunityDescription,
			signer:      signer,
			changes: func(org *Community) *CommunityChanges {
				changes := emptyCommunityChanges()
				changes.MembersRemoved[s.member1Key] = &protobuf.CommunityMember{}
				changes.ChatsModified[testChatID1] = &CommunityChatChanges{
					MembersAdded:   make(map[string]*protobuf.CommunityMember),
					MembersRemoved: make(map[string]*protobuf.CommunityMember),
				}
				changes.ChatsModified[testChatID1].MembersRemoved[s.member1Key] = &protobuf.CommunityMember{}

				return changes
			},
			err: nil,
		},
		{
			name:        "added member from org",
			description: s.addedMemberCommunityDescription,
			signer:      signer,
			changes: func(org *Community) *CommunityChanges {
				changes := emptyCommunityChanges()
				changes.MembersAdded[s.member3Key] = &protobuf.CommunityMember{}
				changes.ChatsModified[testChatID1] = &CommunityChatChanges{
					MembersAdded:   make(map[string]*protobuf.CommunityMember),
					MembersRemoved: make(map[string]*protobuf.CommunityMember),
				}
				changes.ChatsModified[testChatID1].MembersAdded[s.member3Key] = &protobuf.CommunityMember{}

				return changes
			},
			err: nil,
		},
		{
			name:        "chat added to org",
			description: s.addedChatCommunityDescription,
			signer:      signer,
			changes: func(org *Community) *CommunityChanges {
				changes := emptyCommunityChanges()
				changes.MembersAdded[s.member3Key] = &protobuf.CommunityMember{}
				changes.ChatsAdded[testChatID2] = &protobuf.CommunityChat{Permissions: &protobuf.CommunityPermissions{Access: protobuf.CommunityPermissions_INVITATION_ONLY}, Members: make(map[string]*protobuf.CommunityMember)}
				changes.ChatsAdded[testChatID2].Members[s.member3Key] = &protobuf.CommunityMember{}

				return changes
			},
			err: nil,
		},
		{
			name:        "chat removed from the org",
			description: s.removedChatCommunityDescription,
			signer:      signer,
			changes: func(org *Community) *CommunityChanges {
				changes := emptyCommunityChanges()
				changes.ChatsRemoved[testChatID1] = org.config.CommunityDescription.Chats[testChatID1]

				return changes
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			org := s.buildCommunity(signer)
			org.Join()
			expectedChanges := tc.changes(org)
			actualChanges, err := org.HandleCommunityDescription(tc.signer, tc.description(org), []byte{0x01})
			s.Require().Equal(tc.err, err)
			s.Require().Equal(expectedChanges, actualChanges)
		})
	}
}

func (s *CommunitySuite) TestValidateCommunityDescription() {

	testCases := []struct {
		name        string
		description *protobuf.CommunityDescription
		err         error
	}{
		{
			name:        "valid",
			description: s.buildCommunityDescription(),
			err:         nil,
		},
		{
			name: "empty description",
			err:  ErrInvalidCommunityDescription,
		},
		{
			name:        "empty org permissions",
			description: s.emptyPermissionsCommunityDescription(),
			err:         ErrInvalidCommunityDescriptionNoOrgPermissions,
		},
		{
			name:        "empty chat permissions",
			description: s.emptyChatPermissionsCommunityDescription(),
			err:         ErrInvalidCommunityDescriptionNoChatPermissions,
		},
		{
			name:        "unknown org permissions",
			description: s.unknownOrgPermissionsCommunityDescription(),
			err:         ErrInvalidCommunityDescriptionUnknownOrgAccess,
		},
		{
			name:        "unknown chat permissions",
			description: s.unknownChatPermissionsCommunityDescription(),
			err:         ErrInvalidCommunityDescriptionUnknownChatAccess,
		},
		{
			name:        "member in chat but not in org",
			description: s.memberInChatNotInOrgCommunityDescription(),
			err:         ErrInvalidCommunityDescriptionMemberInChatButNotInOrg,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := ValidateCommunityDescription(tc.description)
			s.Require().Equal(tc.err, err)
		})
	}
}

func (s *CommunitySuite) emptyCommunityDescription() *protobuf.CommunityDescription {
	return &protobuf.CommunityDescription{
		Permissions: &protobuf.CommunityPermissions{},
	}

}

func (s *CommunitySuite) emptyCommunityDescriptionWithChat() *protobuf.CommunityDescription {
	desc := &protobuf.CommunityDescription{
		Members:     make(map[string]*protobuf.CommunityMember),
		Clock:       1,
		Chats:       make(map[string]*protobuf.CommunityChat),
		Permissions: &protobuf.CommunityPermissions{},
	}

	desc.Chats[testChatID1] = &protobuf.CommunityChat{Permissions: &protobuf.CommunityPermissions{}, Members: make(map[string]*protobuf.CommunityMember)}
	desc.Members[common.PubkeyToHex(&s.member1.PublicKey)] = &protobuf.CommunityMember{}
	desc.Chats[testChatID1].Members[common.PubkeyToHex(&s.member1.PublicKey)] = &protobuf.CommunityMember{}

	return desc

}

func (s *CommunitySuite) configOnRequest() Config {
	description := s.emptyCommunityDescription()
	description.Permissions.Access = protobuf.CommunityPermissions_ON_REQUEST
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) configInvitationOnly() Config {
	description := s.emptyCommunityDescription()
	description.Permissions.Access = protobuf.CommunityPermissions_INVITATION_ONLY
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) configNoMembershipOrgNoMembershipChat() Config {
	description := s.emptyCommunityDescriptionWithChat()
	description.Permissions.Access = protobuf.CommunityPermissions_NO_MEMBERSHIP
	description.Chats[testChatID1].Permissions.Access = protobuf.CommunityPermissions_NO_MEMBERSHIP
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}

}

func (s *CommunitySuite) configNoMembershipOrgInvitationOnlyChat() Config {
	description := s.emptyCommunityDescriptionWithChat()
	description.Permissions.Access = protobuf.CommunityPermissions_NO_MEMBERSHIP
	description.Chats[testChatID1].Permissions.Access = protobuf.CommunityPermissions_INVITATION_ONLY
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) configInvitationOnlyOrgInvitationOnlyChat() Config {
	description := s.emptyCommunityDescriptionWithChat()
	description.Permissions.Access = protobuf.CommunityPermissions_INVITATION_ONLY
	description.Chats[testChatID1].Permissions.Access = protobuf.CommunityPermissions_INVITATION_ONLY
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) configNoMembershipOrgOnRequestChat() Config {
	description := s.emptyCommunityDescriptionWithChat()
	description.Permissions.Access = protobuf.CommunityPermissions_NO_MEMBERSHIP
	description.Chats[testChatID1].Permissions.Access = protobuf.CommunityPermissions_ON_REQUEST
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) configOnRequestOrgOnRequestChat() Config {
	description := s.emptyCommunityDescriptionWithChat()
	description.Permissions.Access = protobuf.CommunityPermissions_ON_REQUEST
	description.Chats[testChatID1].Permissions.Access = protobuf.CommunityPermissions_ON_REQUEST
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) configOnRequestOrgInvitationOnlyChat() Config {
	description := s.emptyCommunityDescriptionWithChat()
	description.Permissions.Access = protobuf.CommunityPermissions_ON_REQUEST
	description.Chats[testChatID1].Permissions.Access = protobuf.CommunityPermissions_INVITATION_ONLY
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) configOnRequestOrgNoMembershipChat() Config {
	description := s.emptyCommunityDescriptionWithChat()
	description.Permissions.Access = protobuf.CommunityPermissions_ON_REQUEST
	description.Chats[testChatID1].Permissions.Access = protobuf.CommunityPermissions_NO_MEMBERSHIP
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) configChatENSOnly() Config {
	description := s.emptyCommunityDescriptionWithChat()
	description.Permissions.Access = protobuf.CommunityPermissions_ON_REQUEST
	description.Chats[testChatID1].Permissions.Access = protobuf.CommunityPermissions_ON_REQUEST
	description.Chats[testChatID1].Permissions.EnsOnly = true
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) configENSOnly() Config {
	description := s.emptyCommunityDescription()
	description.Permissions.Access = protobuf.CommunityPermissions_ON_REQUEST
	description.Permissions.EnsOnly = true
	return Config{
		ID:                   &s.identity.PublicKey,
		CommunityDescription: description,
		PrivateKey:           s.identity,
	}
}

func (s *CommunitySuite) config() Config {
	config := s.configOnRequestOrgInvitationOnlyChat()
	return config
}

func (s *CommunitySuite) buildCommunityDescription() *protobuf.CommunityDescription {
	config := s.configOnRequestOrgInvitationOnlyChat()
	desc := config.CommunityDescription
	desc.Clock = 1
	desc.Members = make(map[string]*protobuf.CommunityMember)
	desc.Members[s.member1Key] = &protobuf.CommunityMember{}
	desc.Members[s.member2Key] = &protobuf.CommunityMember{}
	desc.Chats[testChatID1].Members = make(map[string]*protobuf.CommunityMember)
	desc.Chats[testChatID1].Members[s.member1Key] = &protobuf.CommunityMember{}
	return desc
}

func (s *CommunitySuite) emptyPermissionsCommunityDescription() *protobuf.CommunityDescription {
	desc := s.buildCommunityDescription()
	desc.Permissions = nil
	return desc
}

func (s *CommunitySuite) emptyChatPermissionsCommunityDescription() *protobuf.CommunityDescription {
	desc := s.buildCommunityDescription()
	desc.Chats[testChatID1].Permissions = nil
	return desc
}

func (s *CommunitySuite) unknownOrgPermissionsCommunityDescription() *protobuf.CommunityDescription {
	desc := s.buildCommunityDescription()
	desc.Permissions.Access = protobuf.CommunityPermissions_UNKNOWN_ACCESS
	return desc
}

func (s *CommunitySuite) unknownChatPermissionsCommunityDescription() *protobuf.CommunityDescription {
	desc := s.buildCommunityDescription()
	desc.Chats[testChatID1].Permissions.Access = protobuf.CommunityPermissions_UNKNOWN_ACCESS
	return desc
}

func (s *CommunitySuite) memberInChatNotInOrgCommunityDescription() *protobuf.CommunityDescription {
	desc := s.buildCommunityDescription()
	desc.Chats[testChatID1].Members[s.member3Key] = &protobuf.CommunityMember{}
	return desc
}

func (s *CommunitySuite) buildCommunity(owner *ecdsa.PublicKey) *Community {

	config := s.config()
	config.ID = owner
	config.CommunityDescription = s.buildCommunityDescription()

	org, err := New(config)
	s.Require().NoError(err)
	return org
}

func (s *CommunitySuite) identicalCommunityDescription(org *Community) *protobuf.CommunityDescription {
	description := proto.Clone(org.config.CommunityDescription).(*protobuf.CommunityDescription)
	description.Clock++
	return description
}

func (s *CommunitySuite) oldCommunityDescription(org *Community) *protobuf.CommunityDescription {
	description := proto.Clone(org.config.CommunityDescription).(*protobuf.CommunityDescription)
	description.Clock--
	delete(description.Members, s.member1Key)
	delete(description.Chats[testChatID1].Members, s.member1Key)
	return description
}

func (s *CommunitySuite) removedMemberCommunityDescription(org *Community) *protobuf.CommunityDescription {
	description := proto.Clone(org.config.CommunityDescription).(*protobuf.CommunityDescription)
	description.Clock++
	delete(description.Members, s.member1Key)
	delete(description.Chats[testChatID1].Members, s.member1Key)
	return description
}

func (s *CommunitySuite) addedMemberCommunityDescription(org *Community) *protobuf.CommunityDescription {
	description := proto.Clone(org.config.CommunityDescription).(*protobuf.CommunityDescription)
	description.Clock++
	description.Members[s.member3Key] = &protobuf.CommunityMember{}
	description.Chats[testChatID1].Members[s.member3Key] = &protobuf.CommunityMember{}

	return description
}

func (s *CommunitySuite) addedChatCommunityDescription(org *Community) *protobuf.CommunityDescription {
	description := proto.Clone(org.config.CommunityDescription).(*protobuf.CommunityDescription)
	description.Clock++
	description.Members[s.member3Key] = &protobuf.CommunityMember{}
	description.Chats[testChatID2] = &protobuf.CommunityChat{Permissions: &protobuf.CommunityPermissions{Access: protobuf.CommunityPermissions_INVITATION_ONLY}, Members: make(map[string]*protobuf.CommunityMember)}
	description.Chats[testChatID2].Members[s.member3Key] = &protobuf.CommunityMember{}

	return description
}

func (s *CommunitySuite) removedChatCommunityDescription(org *Community) *protobuf.CommunityDescription {
	description := proto.Clone(org.config.CommunityDescription).(*protobuf.CommunityDescription)
	description.Clock++
	delete(description.Chats, testChatID1)

	return description
}
