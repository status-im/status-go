package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

func (s *MessengerCommunitiesSuite) TestGetCommunityMembers() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	snapshot, err := s.owner.GetCommunityMembers(community.ID())
	s.Require().NoError(err)

	entries := map[string]communities.RosterEntry{}
	for _, entry := range snapshot.Members {
		entries[entry.PubKey] = entry
	}
	s.Require().Len(entries, 2)
	s.Require().NotEmpty(s.owner.selfContact.DisplayName)
	s.Require().Equal(s.owner.selfContact.DisplayName, entries[s.owner.IdentityPublicKeyString()].DisplayName)
	s.Require().Contains(entries, s.alice.IdentityPublicKeyString())

	stored, err := s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(stored.Clock(), snapshot.Clock)
}

func (s *MessengerCommunitiesSuite) TestContactMapMemberProfile() {
	me := &contacts.Contact{ID: "0xme", DisplayName: "me"}
	cm := &contactMap{me: me, logger: zap.NewNop()}
	cm.Store("0xknown", &contacts.Contact{
		ID:          "0xknown",
		DisplayName: "known",
		EnsName:     "known.eth",
		ENSVerified: true,
		Images: map[string]images.IdentityImage{
			images.SmallDimName: {Name: images.SmallDimName, Clock: 7},
		},
	})

	profile, ok := cm.MemberProfile("0xknown")
	s.Require().True(ok)
	s.Require().Equal(communities.MemberProfile{
		DisplayName:         "known",
		EnsName:             "known.eth",
		EnsVerified:         true,
		ThumbnailImageClock: 7,
	}, profile)

	profile, ok = cm.MemberProfile("0xme")
	s.Require().True(ok)
	s.Require().Equal("me", profile.DisplayName)

	_, ok = cm.MemberProfile("0xunknown")
	s.Require().False(ok)
}

func (s *MessengerCommunitiesSuite) TestMentionedNames() {
	community, chat := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)
	s.advertiseCommunityTo(community, s.owner, s.bob)
	s.joinCommunity(community, s.owner, s.bob)

	ownerID := s.owner.IdentityPublicKeyString()
	bobID := s.bob.IdentityPublicKeyString()
	unknownKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	unknownID := crypto.PubkeyToHex(&unknownKey.PublicKey)

	// For alice, the owner is an added contact and bob a non-contact member.
	s.alice.allContacts.Store(ownerID, &contacts.Contact{
		ID:                       ownerID,
		DisplayName:              "owner-contact",
		ContactRequestLocalState: contacts.ContactRequestStateSent,
	})
	s.alice.allContacts.Store(bobID, &contacts.Contact{ID: bobID, DisplayName: "Bobby"})
	contact, ok := s.alice.allContacts.Load(bobID)
	s.Require().True(ok)
	s.Require().False(contact.Added())

	// alice drops bob's message as "can't post" until she has the
	// description that adds bob.
	_, err = WaitOnMessengerResponse(s.alice, func(*MessengerResponse) bool {
		aliceCommunity, err := s.alice.GetCommunityByID(community.ID())
		return err == nil && aliceCommunity.HasMember(&s.bob.identity.PublicKey)
	}, "alice did not see bob join")
	s.Require().NoError(err)

	text := fmt.Sprintf("@%s @%s @%s @%s", ownerID, bobID, unknownID, common.EveryoneMentionTag)
	sent := sendChatMessage(&s.Suite, s.bob, chat.ID, text)

	expected := map[string]string{ownerID: "owner-contact", bobID: "Bobby"}
	_, err = WaitOnMessengerResponse(s.alice, func(r *MessengerResponse) bool {
		for _, msg := range r.Messages() {
			if msg.ID == sent.ID {
				return len(msg.Mentions) == 4 && fmt.Sprint(msg.MentionedNames) == fmt.Sprint(expected)
			}
		}
		return false
	}, "message with mentioned names not received")
	s.Require().NoError(err)

	msgs, _, err := s.alice.MessageByChatID(chat.ID, "", "", 10)
	s.Require().NoError(err)
	var fetched *common.Message
	for _, msg := range msgs {
		if msg.ID == sent.ID {
			fetched = msg
		}
	}
	s.Require().NotNil(fetched)
	s.Require().Equal(expected, fetched.MentionedNames)

	byID, err := s.alice.MessageByID(sent.ID)
	s.Require().NoError(err)
	s.Require().Equal(expected, byID.MentionedNames)
}

func (s *MessengerCommunitiesSuite) TestResolveMentionedNamesPriority() {
	s.owner.allContacts.Store("0xnick", &contacts.Contact{ID: "0xnick", Alias: "a", LocalNickname: "nick", DisplayName: "name", EnsName: "nick.eth", ENSVerified: true})
	s.owner.allContacts.Store("0xens", &contacts.Contact{ID: "0xens", Alias: "a", DisplayName: "name", EnsName: "name.eth", ENSVerified: true})
	s.owner.allContacts.Store("0xunverified", &contacts.Contact{ID: "0xunverified", Alias: "a", DisplayName: "plain", EnsName: "fake.eth"})
	s.owner.allContacts.Store("0xaliasonly", &contacts.Contact{ID: "0xaliasonly", Alias: "a"})

	msg := common.NewMessage()
	msg.Mentions = []string{"0xnick", "0xens", "0xunverified", "0xaliasonly", s.owner.IdentityPublicKeyString()}
	s.owner.resolveMentionedNames(msg)
	s.Require().Equal(map[string]string{
		"0xnick":                          "nick",
		"0xens":                           "name.eth",
		"0xunverified":                    "plain",
		s.owner.IdentityPublicKeyString(): "Charlie",
	}, msg.MentionedNames)
}

func rosterAddedHas(r *MessengerResponse, communityID types.HexBytes, pubKey string) bool {
	return rosterDeltaHas(r, communityID, "rosterAdded", pubKey)
}

func rosterRemovedHas(r *MessengerResponse, communityID types.HexBytes, pubKey string) bool {
	return rosterDeltaHas(r, communityID, "rosterRemoved", pubKey)
}

func rosterModifiedHas(r *MessengerResponse, communityID types.HexBytes, pubKey string) bool {
	return rosterDeltaHas(r, communityID, "rosterModified", pubKey)
}

// rosterDeltaHas reports whether the serialized changes of the community in
// the response carry pubKey in the given roster delta key.
func rosterDeltaHas(r *MessengerResponse, communityID types.HexBytes, key string, pubKey string) bool {
	if r == nil {
		return false
	}
	for _, changes := range r.CommunityChanges {
		if changes.Community == nil || !bytes.Equal(changes.Community.ID(), communityID) {
			continue
		}
		payload, err := json.Marshal(changes)
		if err != nil {
			return false
		}
		var wire map[string]json.RawMessage
		if json.Unmarshal(payload, &wire) != nil {
			return false
		}
		var entries map[string]communities.RosterEntry
		if json.Unmarshal(wire[key], &entries) != nil {
			return false
		}
		if _, ok := entries[pubKey]; ok {
			return true
		}
	}
	return false
}

func (s *MessengerCommunitiesSuite) TestControlNodeGetsRosterDeltaOnAutoAcceptedJoin() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)

	_, err := s.alice.RequestToJoinCommunity(s.createRequestToJoinCommunity(community.ID(), s.alice))
	s.Require().NoError(err)

	aliceID := s.alice.IdentityPublicKeyString()
	_, err = WaitOnMessengerResponse(s.owner, func(r *MessengerResponse) bool {
		return rosterAddedHas(r, community.ID(), aliceID)
	}, "control node did not get alice as a roster delta")
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestManualAcceptReturnsRosterDelta() {
	community, _ := createOnRequestCommunity(&s.Suite, s.owner)
	s.advertiseCommunityTo(community, s.owner, s.alice)

	requestID := requestToJoinCommunity(&s.Suite, s.owner, s.alice, s.createRequestToJoinCommunity(community.ID(), s.alice))
	response, err := s.owner.AcceptRequestToJoinCommunity(&requests.AcceptRequestToJoinCommunity{ID: requestID})
	s.Require().NoError(err)
	s.Require().True(rosterAddedHas(response, community.ID(), s.alice.IdentityPublicKeyString()))
}

func (s *MessengerCommunitiesSuite) TestMemberGetsRosterDeltaForAnotherJoin() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)
	s.advertiseCommunityTo(community, s.owner, s.bob)

	_, err := s.bob.RequestToJoinCommunity(s.createRequestToJoinCommunity(community.ID(), s.bob))
	s.Require().NoError(err)

	bobID := s.bob.IdentityPublicKeyString()
	_, err = WaitOnMessengerResponse(s.owner, func(r *MessengerResponse) bool {
		return rosterAddedHas(r, community.ID(), bobID)
	}, "control node did not get bob as a roster delta")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.alice, func(r *MessengerResponse) bool {
		return rosterAddedHas(r, community.ID(), bobID)
	}, "member did not get bob as a roster delta")
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestKickReturnsRosterDelta() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	response, err := s.owner.RemoveUserFromCommunity(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().True(rosterRemovedHas(response, community.ID(), s.alice.IdentityPublicKeyString()))
}

func (s *MessengerCommunitiesSuite) TestBanReturnsRosterDelta() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	aliceID := s.alice.IdentityPublicKeyString()
	response, err := s.owner.BanUserFromCommunity(context.Background(), &requests.BanUserFromCommunity{
		CommunityID: community.ID(),
		User:        crypto.PubkeyToHexBytes(&s.alice.identity.PublicKey),
	})
	s.Require().NoError(err)
	s.Require().True(rosterRemovedHas(response, community.ID(), aliceID))
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Contains(response.CommunityChanges[0].MembersBanned, aliceID)

	// Unbanning does not re-add the member.
	response, err = s.owner.UnbanUserFromCommunity(&requests.UnbanUserFromCommunity{
		CommunityID: community.ID(),
		User:        crypto.PubkeyToHexBytes(&s.alice.identity.PublicKey),
	})
	s.Require().NoError(err)
	s.Require().False(rosterAddedHas(response, community.ID(), aliceID))
}

func (s *MessengerCommunitiesSuite) TestRoleChangeReturnsRosterDelta() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	response, err := s.owner.AddRoleToMember(&requests.AddRoleToMember{
		CommunityID: community.ID(),
		User:        crypto.PubkeyToHexBytes(&s.alice.identity.PublicKey),
		Role:        protobuf.CommunityMember_ROLE_ADMIN,
	})
	s.Require().NoError(err)
	s.Require().True(rosterModifiedHas(response, community.ID(), s.alice.IdentityPublicKeyString()))
}

func (s *MessengerCommunitiesSuite) TestControlNodeGetsRosterDeltaOnLeave() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.owner, s.alice)
	s.joinCommunity(community, s.owner, s.alice)

	_, err := s.alice.LeaveCommunity(community.ID())
	s.Require().NoError(err)

	aliceID := s.alice.IdentityPublicKeyString()
	_, err = WaitOnMessengerResponse(s.owner, func(r *MessengerResponse) bool {
		return rosterRemovedHas(r, community.ID(), aliceID)
	}, "control node did not get alice's leave as a roster delta")
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSuite) TestProcessCommunityChangesCoalescesPerCommunity() {
	community, _ := s.createCommunity()
	aliceID := s.alice.IdentityPublicKeyString()
	bobID := s.bob.IdentityPublicKeyString()

	batch := make([]*communities.CommunityChanges, 0, 3)
	for _, pubKey := range []string{aliceID, bobID} {
		changes := communities.EmptyCommunityChanges()
		changes.Community = community
		changes.MembersAdded[pubKey] = &protobuf.CommunityMember{}
		batch = append(batch, changes)
	}
	removal := communities.EmptyCommunityChanges()
	removal.Community = community
	removal.MembersRemoved[aliceID] = &protobuf.CommunityMember{}
	batch = append(batch, removal)

	state := &ReceivedMessageState{Response: &MessengerResponse{CommunityChanges: batch}}
	s.owner.processCommunityChanges(state)

	s.Require().Len(state.Response.CommunityChanges, 1)
	s.Require().True(rosterAddedHas(state.Response, community.ID(), bobID))
	s.Require().False(rosterAddedHas(state.Response, community.ID(), aliceID))
	s.Require().True(rosterRemovedHas(state.Response, community.ID(), aliceID))
}
