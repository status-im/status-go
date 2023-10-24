package protocol

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/identity"
)

func TestProfileShowcasePersistenceSuite(t *testing.T) {
	suite.Run(t, new(TestProfileShowcasePersistence))
}

type TestProfileShowcasePersistence struct {
	suite.Suite
}

func (s *TestProfileShowcasePersistence) TestProfileShowcasePreferences() {
	db, err := openTestDB()
	s.Require().NoError(err)
	persistence := newSQLitePersistence(db)

	preferences := []*ProfileShowcaseEntry{
		&ProfileShowcaseEntry{
			ID:                 "0x32433445133424",
			EntryType:          ProfileShowcaseEntryTypeCommunity,
			ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
			Order:              0,
		},
		&ProfileShowcaseEntry{
			ID:                 "0x12333245443413412",
			EntryType:          ProfileShowcaseEntryTypeAccount,
			ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
			Order:              0,
		},
		&ProfileShowcaseEntry{
			ID:                 "b4ef5ce9-5a10-4c88-a2ff-5bc371f82930",
			EntryType:          ProfileShowcaseEntryTypeCollectible,
			ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
			Order:              0,
		},
		&ProfileShowcaseEntry{
			ID:                 "ETH",
			EntryType:          ProfileShowcaseEntryTypeAsset,
			ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
			Order:              0,
		},
	}

	err = persistence.SaveProfileShowcasePreferences(preferences)
	s.Require().NoError(err)

	preferencesBack, err := persistence.GetAllProfileShowcasePreferences()
	s.Require().NoError(err)

	s.Require().Equal(len(preferences), len(preferencesBack))

	for i, entry := range preferences {
		s.Require().Equal(entry.ID, preferencesBack[i].ID)
		s.Require().Equal(entry.EntryType, preferencesBack[i].EntryType)
		s.Require().Equal(entry.ShowcaseVisibility, preferencesBack[i].ShowcaseVisibility)
		s.Require().Equal(entry.Order, preferencesBack[i].Order)
	}

	preferencesBack, err = persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeCommunity)
	s.Require().NoError(err)
	s.Require().Equal(1, len(preferencesBack))
	s.Require().Equal(preferences[0].ID, preferencesBack[0].ID)

	preferencesBack, err = persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeAccount)
	s.Require().NoError(err)
	s.Require().Equal(1, len(preferencesBack))
	s.Require().Equal(preferences[1].ID, preferencesBack[0].ID)

	preferencesBack, err = persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeCollectible)
	s.Require().NoError(err)
	s.Require().Equal(1, len(preferencesBack))
	s.Require().Equal(preferences[2].ID, preferencesBack[0].ID)

	preferencesBack, err = persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeAsset)
	s.Require().NoError(err)
	s.Require().Equal(1, len(preferencesBack))
	s.Require().Equal(preferences[3].ID, preferencesBack[0].ID)
}

func (s *TestProfileShowcasePersistence) TestProfileShowcaseContacts() {
	db, err := openTestDB()
	s.Require().NoError(err)
	persistence := newSQLitePersistence(db)

	showcase1 := &identity.ProfileShowcase{
		Communities: []*identity.VisibleProfileShowcaseEntry{
			&identity.VisibleProfileShowcaseEntry{
				EntryID: "0x012312234234234",
				Order:   6,
			},
			&identity.VisibleProfileShowcaseEntry{
				EntryID: "0x04523233466753",
				Order:   7,
			},
		},
		Assets: []*identity.VisibleProfileShowcaseEntry{
			&identity.VisibleProfileShowcaseEntry{
				EntryID: "ETH",
				Order:   1,
			},
		},
	}
	err = persistence.SaveProfileShowcaseForContact("contact_1", showcase1)
	s.Require().NoError(err)

	showcase2 := &identity.ProfileShowcase{
		Communities: []*identity.VisibleProfileShowcaseEntry{
			&identity.VisibleProfileShowcaseEntry{
				EntryID: "0x012312234234234", // same id to check query
				Order:   3,
			},
			&identity.VisibleProfileShowcaseEntry{
				EntryID: "0x096783478384593",
				Order:   7,
			},
		},
		Collectibles: []*identity.VisibleProfileShowcaseEntry{
			&identity.VisibleProfileShowcaseEntry{
				EntryID: "d378662f-3d71-44e0-81ee-ff7f1778c13a",
				Order:   1,
			},
		},
	}
	err = persistence.SaveProfileShowcaseForContact("contact_2", showcase2)
	s.Require().NoError(err)

	showcase1Back, err := persistence.GetProfileShowcaseForContact("contact_1")
	s.Require().NoError(err)

	s.Require().Equal(len(showcase1.Communities), len(showcase1Back.Communities))
	s.Require().Equal(*showcase1.Communities[0], *showcase1Back.Communities[0])
	s.Require().Equal(*showcase1.Communities[1], *showcase1Back.Communities[1])
	s.Require().Equal(len(showcase1.Assets), len(showcase1Back.Assets))
	s.Require().Equal(*showcase1.Assets[0], *showcase1Back.Assets[0])
	s.Require().Equal(0, len(showcase1Back.Accounts))
	s.Require().Equal(0, len(showcase1Back.Collectibles))

	showcase2Back, err := persistence.GetProfileShowcaseForContact("contact_2")
	s.Require().NoError(err)

	s.Require().Equal(len(showcase2.Communities), len(showcase2Back.Communities))
	s.Require().Equal(*showcase2.Communities[0], *showcase2Back.Communities[0])
	s.Require().Equal(*showcase2.Communities[1], *showcase2Back.Communities[1])
	s.Require().Equal(len(showcase2.Collectibles), len(showcase2Back.Collectibles))
	s.Require().Equal(*showcase2.Collectibles[0], *showcase2Back.Collectibles[0])
	s.Require().Equal(0, len(showcase2Back.Accounts))
	s.Require().Equal(0, len(showcase2Back.Assets))
}
