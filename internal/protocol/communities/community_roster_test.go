package communities

import (
	"encoding/json"

	"github.com/status-im/status-go/internal/protocol/identity"
	"github.com/status-im/status-go/internal/protocol/identity/alias"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

type memberProfilesStub map[string]MemberProfile

func (s memberProfilesStub) MemberProfile(pubKey string) (MemberProfile, bool) {
	p, ok := s[pubKey]
	return p, ok
}

func (s *CommunitySuite) gateChat(community *Community, chatID string) {
	_, err := community.UpsertTokenPermission(&protobuf.CommunityTokenPermission{
		Id:            "gate-" + chatID,
		Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
		TokenCriteria: []*protobuf.TokenCriteria{{Symbol: "SNT"}},
		ChatIds:       []string{community.ChatID(chatID)},
	})
	s.Require().NoError(err)
	s.Require().True(community.ChannelEncrypted(chatID))
}

func (s *CommunitySuite) rosterCommunity() *Community {
	community := s.buildCommunity(&s.identity.PublicKey)
	description := community.config.CommunityDescription
	description.Members[s.member2Key].Roles = []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_ADMIN}
	description.Chats[testChatID1].Members[s.member1Key].ChannelRole = protobuf.CommunityMember_CHANNEL_ROLE_VIEWER
	description.Chats[testChatID2] = &protobuf.CommunityChat{
		Identity:    &protobuf.ChatIdentity{},
		Permissions: &protobuf.CommunityPermissions{},
		Members:     map[string]*protobuf.CommunityMember{s.member2Key: {}},
	}
	s.gateChat(community, testChatID1)
	return community
}

func (s *CommunitySuite) TestMembersSnapshot() {
	community := s.rosterCommunity()
	profiles := memberProfilesStub{
		s.member1Key: {DisplayName: "alice", EnsName: "alice.stateofus.eth", EnsVerified: true, ThumbnailImageClock: 42},
	}

	snapshot := community.MembersSnapshot(profiles)

	s.Require().Equal(community.Clock(), snapshot.Clock)
	s.Require().Len(snapshot.Members, 2)

	entries := map[string]RosterEntry{}
	for _, entry := range snapshot.Members {
		entries[entry.PubKey] = entry
	}

	alice := entries[s.member1Key]
	s.Require().Equal("alice", alice.DisplayName)
	s.Require().Equal("alice.stateofus.eth", alice.EnsName)
	s.Require().True(alice.EnsVerified)
	s.Require().Empty(alice.Alias)
	s.Require().Equal(uint64(42), alice.ThumbnailImageClock)
	expectedColorID, err := identity.ToColorID(s.member1Key)
	s.Require().NoError(err)
	s.Require().Equal(expectedColorID, alice.ColorID)

	unknown := entries[s.member2Key]
	s.Require().Empty(unknown.DisplayName)
	expectedAlias, err := alias.GenerateFromPublicKeyString(s.member2Key)
	s.Require().NoError(err)
	s.Require().Equal(expectedAlias, unknown.Alias)
	s.Require().Equal([]protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_ADMIN}, unknown.Roles)

	s.Require().Equal(map[string]map[string]protobuf.CommunityMember_ChannelRole{
		testChatID1: {s.member1Key: protobuf.CommunityMember_CHANNEL_ROLE_VIEWER},
	}, snapshot.Channels)
}

func (s *CommunitySuite) TestMembersSnapshotJSON() {
	community := s.rosterCommunity()

	bytes, err := json.Marshal(community.MembersSnapshot(nil))
	s.Require().NoError(err)

	var parsed struct {
		Clock    uint64                       `json:"clock"`
		Members  []map[string]json.RawMessage `json:"members"`
		Channels map[string]map[string]int    `json:"channels"`
	}
	s.Require().NoError(json.Unmarshal(bytes, &parsed))
	s.Require().Equal(community.Clock(), parsed.Clock)
	s.Require().Len(parsed.Members, 2)
	for _, entry := range parsed.Members {
		s.Require().Contains(entry, "pubKey")
		s.Require().Contains(entry, "roles")
		s.Require().Contains(entry, "alias")
		s.Require().Contains(entry, "colorId")
		s.Require().NotContains(entry, "emojiHash")
		s.Require().NotContains(entry, "compressedKey")
	}
	s.Require().Equal(int(protobuf.CommunityMember_CHANNEL_ROLE_VIEWER), parsed.Channels[testChatID1][s.member1Key])
}
