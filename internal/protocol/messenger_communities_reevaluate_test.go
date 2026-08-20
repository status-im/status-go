package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/requests"
)

func TestMessengerCommunitiesReevaluateSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunitiesReevaluateSuite))
}

type MessengerCommunitiesReevaluateSuite struct {
	CommunitiesMessengerTestSuiteBase
}

func (s *MessengerCommunitiesReevaluateSuite) newMessengerWithForceReevalAllowed() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	communityManagerOptions := []communities.ManagerOption{
		communities.WithAllowForcingCommunityMembersReevaluation(true),
	}

	return s.newMessengerWithConfig(testMessengerConfig{
		privateKey:   privateKey,
		extraOptions: []Option{WithCommunityManagerOptions(communityManagerOptions)},
	}, "", []string{})
}

func (s *MessengerCommunitiesReevaluateSuite) TestReevaluateMembersPermissions_ForceRejectedWhenNotAllowed() {
	owner := s.newMessenger("", []string{})
	community, _ := createCommunity(&s.Suite, owner)

	_, err := owner.ReevaluateCommunityMembersPermissions(&requests.ReevaluateCommunityMembersPermissions{
		CommunityID: community.ID(),
		Force:       true,
	})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "forcing members reevaluation is not allowed")
}

func (s *MessengerCommunitiesReevaluateSuite) TestReevaluateMembersPermissions_ForceAllowedOnControlNode() {
	owner := s.newMessengerWithForceReevalAllowed()
	community, _ := createCommunity(&s.Suite, owner)

	_, err := owner.ReevaluateCommunityMembersPermissions(&requests.ReevaluateCommunityMembersPermissions{
		CommunityID: community.ID(),
		Force:       true,
	})
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesReevaluateSuite) TestReevaluateMembersPermissions_ScheduleWithoutForce() {
	owner := s.newMessenger("", []string{})
	community, _ := createCommunity(&s.Suite, owner)

	_, err := owner.ReevaluateCommunityMembersPermissions(&requests.ReevaluateCommunityMembersPermissions{
		CommunityID: community.ID(),
		Force:       false,
	})
	s.Require().NoError(err)
}
