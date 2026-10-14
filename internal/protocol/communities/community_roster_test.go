package communities

import (
	"encoding/json"

	"github.com/golang/protobuf/proto"

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

func (s *CommunitySuite) TestEvaluateChangesDetectsRoleChanges() {
	community := s.rosterCommunity()
	origin := community.config.CommunityDescription
	modified := proto.Clone(origin).(*protobuf.CommunityDescription)
	modified.Members[s.member1Key].Roles = []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_TOKEN_MASTER}
	modified.Chats[testChatID1].Members[s.member1Key].ChannelRole = protobuf.CommunityMember_CHANNEL_ROLE_POSTER

	changes := evaluateCommunityChangesByDescription(origin, modified)

	s.Require().Empty(changes.MembersAdded)
	s.Require().Empty(changes.MembersRemoved)
	s.Require().Len(changes.MembersModified, 1)
	s.Require().Contains(changes.MembersModified, s.member1Key)
	s.Require().NotNil(changes.ChatsModified[testChatID1])
	s.Require().Empty(changes.ChatsModified[testChatID1].MembersAdded)
	s.Require().Equal(protobuf.CommunityMember_CHANNEL_ROLE_POSTER,
		changes.ChatsModified[testChatID1].MembersModified[s.member1Key].ChannelRole)
}

func (s *CommunitySuite) TestEvaluateChangesIgnoresRoleOrder() {
	community := s.rosterCommunity()
	origin := community.config.CommunityDescription
	origin.Members[s.member2Key].Roles = []protobuf.CommunityMember_Roles{
		protobuf.CommunityMember_ROLE_ADMIN, protobuf.CommunityMember_ROLE_TOKEN_MASTER}
	modified := proto.Clone(origin).(*protobuf.CommunityDescription)
	modified.Members[s.member2Key].Roles = []protobuf.CommunityMember_Roles{
		protobuf.CommunityMember_ROLE_TOKEN_MASTER, protobuf.CommunityMember_ROLE_ADMIN}

	changes := evaluateCommunityChangesByDescription(origin, modified)

	s.Require().Empty(changes.MembersModified)
}

func (s *CommunitySuite) TestCommunityChangesJSON() {
	community := s.rosterCommunity()
	community.memberProfiles = memberProfilesStub{s.member3Key: {DisplayName: "carol"}}
	snapshot := community.MembersSnapshot(nil)

	changes, err := community.AddMember(&s.member3.PublicKey, nil, 0)
	s.Require().NoError(err)

	bytes, err := json.Marshal(changes)
	s.Require().NoError(err)

	var delta struct {
		CommunityID string                 `json:"communityId"`
		Clock       uint64                 `json:"clock"`
		RosterAdded map[string]RosterEntry `json:"rosterAdded"`
	}
	s.Require().NoError(json.Unmarshal(bytes, &delta))
	s.Require().Equal(community.IDString(), delta.CommunityID)
	s.Require().Greater(delta.Clock, snapshot.Clock)
	s.Require().Equal(RosterEntry{
		PubKey:      s.member3Key,
		Roles:       []protobuf.CommunityMember_Roles{},
		DisplayName: "carol",
		ColorID:     delta.RosterAdded[s.member3Key].ColorID,
	}, delta.RosterAdded[s.member3Key])
}

// The client in production reads the develop-era keys, so they must keep
// their names and shapes; roster entries only appear under new keys.
func (s *CommunitySuite) TestCommunityChangesJSONKeepsExistingKeys() {
	community := s.rosterCommunity()
	member := func() *protobuf.CommunityMember {
		return &protobuf.CommunityMember{Roles: []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_ADMIN}}
	}
	chat := &protobuf.CommunityChat{Identity: &protobuf.ChatIdentity{DisplayName: "chat"}}
	category := &protobuf.CommunityCategory{CategoryId: testCategoryID1, Name: testCategoryName1}
	permission := NewCommunityTokenPermission(&protobuf.CommunityTokenPermission{Id: "permission"})

	changes := EmptyCommunityChanges()
	changes.Community = community
	changes.ControlNodeChanged = &s.identity.PublicKey
	changes.MembersAdded[s.member1Key] = member()
	changes.MembersRemoved[s.member2Key] = member()
	changes.MembersModified[s.member3Key] = member()
	changes.MembersBanned[s.member1Key] = true
	changes.MembersUnbanned[s.member2Key] = true
	changes.TokenPermissionsAdded["added"] = permission
	changes.TokenPermissionsModified["modified"] = permission
	changes.TokenPermissionsRemoved["removed"] = permission
	changes.ChatsAdded[testChatID2] = chat
	changes.ChatsRemoved[testChatID2] = chat
	chatChanges := changes.chatChanges(testChatID1)
	chatChanges.ChatModified = chat
	chatChanges.MembersAdded[s.member1Key] = member()
	chatChanges.MembersRemoved[s.member2Key] = member()
	chatChanges.MembersModified[s.member3Key] = member()
	chatChanges.CategoryModified = testCategoryID1
	chatChanges.PositionModified = 2
	chatChanges.FirstMessageTimestampModified = 3
	changes.CategoriesAdded[testCategoryID1] = category
	changes.CategoriesModified[testCategoryID1] = category
	changes.CategoriesRemoved = []string{testCategoryID1}
	changes.MemberWalletsAdded[s.member1Key] = []*protobuf.RevealedAccount{{Address: "0x1"}}
	changes.MemberWalletsRemoved = []string{s.member2Key}
	changes.ShouldMemberJoin = true
	changes.MemberKicked = true
	changes.MemberSoftKicked = true
	changes.ImageModified = true

	bytes, err := json.Marshal(changes)
	s.Require().NoError(err)

	var parsed map[string]json.RawMessage
	s.Require().NoError(json.Unmarshal(bytes, &parsed))
	for _, key := range []string{
		"controlNodeChanged",
		"membersAdded", "membersRemoved", "membersBanned", "membersUnbanned",
		"tokenPermissionsAdded", "tokenPermissionsModified", "tokenPermissionsRemoved",
		"chatsRemoved", "chatsAdded", "chatsModified",
		"categoriesRemoved", "categoriesAdded", "categoriesModified",
		"memberWalletsRemoved", "memberWalletsAdded",
		"memberAdded", "memberRemoved", "memberSoftRemoved", "communityImageModified",
		"membersModified", "communityId", "clock",
		"rosterAdded", "rosterRemoved", "rosterModified",
	} {
		_, ok := parsed[key]
		s.Require().True(ok, key)
	}

	// the community itself is sent once, in the response's communities
	s.Require().NotContains(parsed, "community")
	s.Require().Equal(`"`+community.IDString()+`"`, string(parsed["communityId"]))

	for key, pubKey := range map[string]string{
		"membersAdded":    s.member1Key,
		"membersRemoved":  s.member2Key,
		"membersModified": s.member3Key,
	} {
		var members map[string]map[string]json.RawMessage
		s.Require().NoError(json.Unmarshal(parsed[key], &members), key)
		var roles []protobuf.CommunityMember_Roles
		s.Require().NoError(json.Unmarshal(members[pubKey]["roles"], &roles), key)
		s.Require().Equal([]protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_ADMIN}, roles, key)
		s.Require().NotContains(members[pubKey], "pubKey", key)
	}

	var chatsModified map[string]map[string]json.RawMessage
	s.Require().NoError(json.Unmarshal(parsed["chatsModified"], &chatsModified))
	for _, key := range []string{
		"ChatModified", "MembersAdded", "MembersRemoved", "MembersModified",
		"CategoryModified", "PositionModified", "FirstMessageTimestampModified",
	} {
		s.Require().Contains(chatsModified[testChatID1], key)
	}
	s.Require().JSONEq(`"`+testCategoryID1+`"`, string(chatsModified[testChatID1]["CategoryModified"]))
	s.Require().JSONEq(`2`, string(chatsModified[testChatID1]["PositionModified"]))

	for key, pubKey := range map[string]string{
		"rosterAdded":    s.member1Key,
		"rosterRemoved":  s.member2Key,
		"rosterModified": s.member3Key,
	} {
		var roster map[string]RosterEntry
		s.Require().NoError(json.Unmarshal(parsed[key], &roster), key)
		s.Require().Equal(pubKey, roster[pubKey].PubKey, key)
		s.Require().Equal([]protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_ADMIN}, roster[pubKey].Roles, key)
	}
}

// Admin events patch the description without bumping its clock, so the
// roster clock must account for them to keep deltas ordered after snapshots.
func (s *CommunitySuite) TestRosterClockFollowsAppliedEvents() {
	community := s.rosterCommunity()
	before := community.MembersSnapshot(nil)
	origin := community.CreateDeepCopy()

	eventClock := community.Clock() + 5
	community.config.EventsData = &EventsData{Events: []CommunityEvent{{CommunityEventClock: eventClock}}}
	community.config.CommunityDescription.Chats[testChatID1].Members[s.member2Key] = &protobuf.CommunityMember{}
	changes := EvaluateCommunityChanges(origin, community)
	changes.Community = community

	bytes, err := json.Marshal(changes)
	s.Require().NoError(err)
	var delta struct {
		Clock uint64 `json:"clock"`
	}
	s.Require().NoError(json.Unmarshal(bytes, &delta))

	s.Require().Greater(delta.Clock, before.Clock)
	s.Require().Equal(eventClock, delta.Clock)
	s.Require().Equal(eventClock, community.MembersSnapshot(nil).Clock)
}

// The control node must publish a description newer than every event it
// applied, otherwise members that saw those events would see the clock regress.
func (s *CommunitySuite) TestIncreaseClockPassesAppliedEvents() {
	community := s.rosterCommunity()
	eventClock := community.timesource.GetCurrentTime() + 1000
	community.config.EventsData = &EventsData{Events: []CommunityEvent{{CommunityEventClock: eventClock}}}

	community.increaseClock()

	s.Require().Greater(community.Clock(), eventClock)
}

func (s *CommunitySuite) TestMergeKeepsMembersModified() {
	changes := EmptyCommunityChanges()
	other := EmptyCommunityChanges()
	other.MembersModified[s.member1Key] = &protobuf.CommunityMember{}

	changes.Merge(other)

	s.Require().Contains(changes.MembersModified, s.member1Key)
}

// Members are read through GetCommunityMembers and roster deltas; the
// community JSON only carries their count.
func (s *CommunitySuite) TestMarshalJSONCarriesNoMembers() {
	community := s.rosterCommunity()
	s.Require().True(community.ChannelEncrypted(testChatID1))

	bytes, err := json.Marshal(community)
	s.Require().NoError(err)

	var parsed struct {
		MembersCount int                                   `json:"membersCount"`
		Members      json.RawMessage                       `json:"members"`
		Chats        map[string]map[string]json.RawMessage `json:"chats"`
	}
	s.Require().NoError(json.Unmarshal(bytes, &parsed))
	s.Require().Equal(2, parsed.MembersCount)
	s.Require().Nil(parsed.Members)
	s.Require().NotEmpty(parsed.Chats)
	for chatID, chat := range parsed.Chats {
		s.Require().NotContains(chat, "members", chatID)
	}
}
