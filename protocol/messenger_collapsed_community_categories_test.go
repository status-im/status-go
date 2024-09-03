package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerCollapsedCommunityCategoriesSuite(t *testing.T) {
	suite.Run(t, new(MessengerCollapsedCommunityCategoriesSuite))
}

type MessengerCollapsedCommunityCategoriesSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerCollapsedCommunityCategoriesSuite) TestUpsertCollapsedCommunityCategories() {
	communityID := "community-id"
	categoryID := "category-id"
	request := &requests.ToggleCollapsedCommunityCategory{
		CommunityID: communityID,
		CategoryID:  categoryID,
		Collapsed:   true,
	}

	s.Require().NoError(s.m.ToggleCollapsedCommunityCategory(request))

	categories, err := s.m.CollapsedCommunityCategories()
	s.Require().NoError(err)
	s.Require().Len(categories, 1)
	s.Require().Equal(communityID, categories[0].CommunityID)
	s.Require().Equal(categoryID, categories[0].CategoryID)

	request.Collapsed = false

	s.Require().NoError(s.m.ToggleCollapsedCommunityCategory(request))

	categories, err = s.m.CollapsedCommunityCategories()
	s.Require().NoError(err)
	s.Require().Len(categories, 0)
}
