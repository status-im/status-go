package communities

import (
	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

func (s *ManagerSuite) TestGetCommunityMembers() {
	community, err := s.manager.CreateCommunity(&requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}, false)
	s.Require().NoError(err)

	ownerKey := crypto.PubkeyToHex(&s.manager.identity.PublicKey)
	s.manager.SetMemberProfileProvider(memberProfilesStub{ownerKey: {DisplayName: "owner"}})

	snapshot, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(community.Clock(), snapshot.Clock)
	s.Require().Len(snapshot.Members, 1)
	s.Require().Equal(ownerKey, snapshot.Members[0].PubKey)
	s.Require().Equal("owner", snapshot.Members[0].DisplayName)

	member, err := crypto.GenerateKey()
	s.Require().NoError(err)
	community, err = s.manager.GetByID(community.ID())
	s.Require().NoError(err)
	_, err = community.AddMember(&member.PublicKey, nil, 0)
	s.Require().NoError(err)
	s.Require().NoError(s.manager.SaveCommunity(community))

	updated, err := s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Len(updated.Members, 2)
	s.Require().Greater(updated.Clock, snapshot.Clock)
}

func (s *ManagerSuite) TestGetCommunityMembersAfterDelete() {
	community, err := s.manager.CreateCommunity(&requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}, false)
	s.Require().NoError(err)

	_, err = s.manager.GetCommunityMembers(community.ID())
	s.Require().NoError(err)

	s.Require().NoError(s.manager.DeleteCommunity(community.ID()))

	_, err = s.manager.GetCommunityMembers(community.ID())
	s.Require().ErrorIs(err, ErrOrgNotFound)
}
