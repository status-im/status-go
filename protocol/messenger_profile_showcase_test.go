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
	communityEntry1 := &ProfileShowcaseEntry{
		ID:                 "0x01312357798976434",
		EntryType:          ProfileShowcaseEntryTypeCommunity,
		ShowcaseVisibility: ProfileShowcaseVisibilityContacts,
		Order:              10,
	}

	communityEntry2 := &ProfileShowcaseEntry{
		ID:                 "0x01312357798976535",
		EntryType:          ProfileShowcaseEntryTypeCommunity,
		ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
		Order:              11,
	}

	accountEntry := &ProfileShowcaseEntry{
		ID:                 "0cx34662234",
		EntryType:          ProfileShowcaseEntryTypeAccount,
		ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
		Order:              17,
	}

	collectibleEntry := &ProfileShowcaseEntry{
		ID:                 "0x12378534257568678487683576",
		EntryType:          ProfileShowcaseEntryTypeCollectible,
		ShowcaseVisibility: ProfileShowcaseVisibilityIDVerifiedContacts,
		Order:              17,
	}

	assetEntry := &ProfileShowcaseEntry{
		ID:                 "0x139ii4uu423",
		EntryType:          ProfileShowcaseEntryTypeAsset,
		ShowcaseVisibility: ProfileShowcaseVisibilityNoOne,
		Order:              17,
	}

	request := ProfileShowcasePreferences{
		Communities:  []*ProfileShowcaseEntry{communityEntry1, communityEntry2},
		Accounts:     []*ProfileShowcaseEntry{accountEntry},
		Collectibles: []*ProfileShowcaseEntry{collectibleEntry},
		Assets:       []*ProfileShowcaseEntry{assetEntry},
	}

	err := s.m.SetProfileShowcasePreferences(request)
	s.Require().NoError(err)

	response, err := s.m.GetProfileShowcasePreferences()
	s.Require().NoError(err)

	s.Require().Len(response.Communities, 2)
	s.Require().Equal(response.Communities[0], communityEntry1)
	s.Require().Equal(response.Communities[1], communityEntry2)

	s.Require().Len(response.Accounts, 1)
	s.Require().Equal(response.Accounts[0], accountEntry)

	s.Require().Len(response.Collectibles, 1)
	s.Require().Equal(response.Collectibles[0], collectibleEntry)

	s.Require().Len(response.Assets, 1)
	s.Require().Equal(response.Assets[0], assetEntry)
}
