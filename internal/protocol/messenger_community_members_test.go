package protocol

import (
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/images"
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
