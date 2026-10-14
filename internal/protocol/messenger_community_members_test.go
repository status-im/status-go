package protocol

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/contacts"
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

func (s *MessengerCommunitiesSuite) TestResolveMentionedNamesPrefersVerifiedEns() {
	s.owner.allContacts.Store("0xens", &contacts.Contact{ID: "0xens", DisplayName: "name", EnsName: "name.eth", ENSVerified: true})
	s.owner.allContacts.Store("0xunverified", &contacts.Contact{ID: "0xunverified", DisplayName: "plain", EnsName: "fake.eth"})
	s.owner.allContacts.Store("0xnoname", &contacts.Contact{ID: "0xnoname"})

	msg := common.NewMessage()
	msg.Mentions = []string{"0xens", "0xunverified", "0xnoname"}
	s.owner.resolveMentionedNames(msg)
	s.Require().Equal(map[string]string{"0xens": "name.eth", "0xunverified": "plain"}, msg.MentionedNames)
}
