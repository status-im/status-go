package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestMessengerProfileShowcaseSuite(t *testing.T) {
	suite.Run(t, new(MessengerProfileShowcaseSuite))
}

type MessengerProfileShowcaseSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerProfileShowcaseSuite) TestSetAndGetProfileShowcasePreferences() {
	communityEntry := &ProfileShowcaseEntry{
		ID:         "0x01312357798976434",
		Type:       ProfileShowcaseEntryTypeCommunity,
		Visibility: ProfileShowcaseVisibilityContacts,
		Order:      10,
	}
	err := s.m.SetProfileShowcasePreference(communityEntry)
	s.Require().NoError(err)

	accountEntry := &ProfileShowcaseEntry{
		ID:         "0cx34662234",
		Type:       ProfileShowcaseEntryTypeAccount,
		Visibility: ProfileShowcaseVisibilityEveryone,
		Order:      17,
	}
	err = s.m.SetProfileShowcasePreference(accountEntry)
	s.Require().NoError(err)

	collectibleEntry := &ProfileShowcaseEntry{
		ID:         "0x12378534257568678487683576",
		Type:       ProfileShowcaseEntryTypeCollectible,
		Visibility: ProfileShowcaseVisibilityIDVerifiedContacts,
		Order:      17,
	}
	err = s.m.SetProfileShowcasePreference(collectibleEntry)
	s.Require().NoError(err)

	assetEntry := &ProfileShowcaseEntry{
		ID:         "0x139ii4uu423",
		Type:       ProfileShowcaseEntryTypeAsset,
		Visibility: ProfileShowcaseVisibilityNoOne,
		Order:      17,
	}
	err = s.m.SetProfileShowcasePreference(assetEntry)
	s.Require().NoError(err)

	response, err := s.m.GetProfileShowcasePreferences()
	s.Require().NoError(err)

	s.Require().Len(response.Communities, 1)
	s.Require().Equal(response.Communities[0], communityEntry)

	s.Require().Len(response.Communities, 1)
	s.Require().Equal(response.Accounts[0], accountEntry)

	s.Require().Len(response.Communities, 1)
	s.Require().Equal(response.Collectibles[0], collectibleEntry)

	s.Require().Len(response.Assets, 1)
	s.Require().Equal(response.Assets[0], assetEntry)
}
