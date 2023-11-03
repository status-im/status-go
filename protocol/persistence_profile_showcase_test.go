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

	preferences := &ProfileShowcasePreferences{
		Communities: []*ProfileShowcaseCommunityPreference{
			&ProfileShowcaseCommunityPreference{
				CommunityID:        "0x32433445133424",
				ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
		},
		Accounts: []*ProfileShowcaseAccountPreference{
			&ProfileShowcaseAccountPreference{
				Address:            "0x32433445133424",
				Name:               "Status Account",
				ColorID:            "blue",
				Emoji:              "-_-",
				ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
			&ProfileShowcaseAccountPreference{
				Address:            "0x3845354643324",
				Name:               "Money Box",
				ColorID:            "red",
				Emoji:              ":o)",
				ShowcaseVisibility: ProfileShowcaseVisibilityContacts,
				Order:              1,
			},
		},
		Assets: []*ProfileShowcaseAssetPreference{
			&ProfileShowcaseAssetPreference{
				Symbol:             "ETH",
				ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
			&ProfileShowcaseAssetPreference{
				Symbol:             "DAI",
				ShowcaseVisibility: ProfileShowcaseVisibilityIDVerifiedContacts,
				Order:              2,
			},
			&ProfileShowcaseAssetPreference{
				Symbol:             "SNT",
				ShowcaseVisibility: ProfileShowcaseVisibilityNoOne,
				Order:              3,
			},
		},
	}

	err = persistence.SaveProfileShowcasePreferences(preferences)
	s.Require().NoError(err)

	preferencesBack, err := persistence.GetProfileShowcasePreferences()
	s.Require().NoError(err)

	s.Require().Len(preferencesBack.Communities, 1)
	s.Require().Equal(preferences.Communities[0].CommunityID, preferencesBack.Communities[0].CommunityID)
	s.Require().Equal(preferences.Communities[0].ShowcaseVisibility, preferencesBack.Communities[0].ShowcaseVisibility)
	s.Require().Equal(preferences.Communities[0].Order, preferencesBack.Communities[0].Order)

	s.Require().Len(preferencesBack.Accounts, 2)
	s.Require().Equal(preferences.Accounts[0].Address, preferencesBack.Accounts[0].Address)
	s.Require().Equal(preferences.Accounts[0].Name, preferencesBack.Accounts[0].Name)
	s.Require().Equal(preferences.Accounts[0].ColorID, preferencesBack.Accounts[0].ColorID)
	s.Require().Equal(preferences.Accounts[0].Emoji, preferencesBack.Accounts[0].Emoji)
	s.Require().Equal(preferences.Accounts[0].ShowcaseVisibility, preferencesBack.Accounts[0].ShowcaseVisibility)
	s.Require().Equal(preferences.Accounts[0].Order, preferencesBack.Accounts[0].Order)

	s.Require().Equal(preferences.Accounts[1].Address, preferencesBack.Accounts[1].Address)
	s.Require().Equal(preferences.Accounts[1].Name, preferencesBack.Accounts[1].Name)
	s.Require().Equal(preferences.Accounts[1].ColorID, preferencesBack.Accounts[1].ColorID)
	s.Require().Equal(preferences.Accounts[1].Emoji, preferencesBack.Accounts[1].Emoji)
	s.Require().Equal(preferences.Accounts[1].ShowcaseVisibility, preferencesBack.Accounts[1].ShowcaseVisibility)
	s.Require().Equal(preferences.Accounts[1].Order, preferencesBack.Accounts[1].Order)

	s.Require().Len(preferencesBack.Collectibles, 0)

	s.Require().Len(preferencesBack.Assets, 3)
	s.Require().Equal(preferences.Assets[0].Symbol, preferencesBack.Assets[0].Symbol)
	s.Require().Equal(preferences.Assets[0].ShowcaseVisibility, preferencesBack.Assets[0].ShowcaseVisibility)
	s.Require().Equal(preferences.Assets[0].Order, preferencesBack.Assets[0].Order)

	s.Require().Equal(preferences.Assets[1].Symbol, preferencesBack.Assets[1].Symbol)
	s.Require().Equal(preferences.Assets[1].ShowcaseVisibility, preferencesBack.Assets[1].ShowcaseVisibility)
	s.Require().Equal(preferences.Assets[1].Order, preferencesBack.Assets[1].Order)

	s.Require().Equal(preferences.Assets[2].Symbol, preferencesBack.Assets[2].Symbol)
	s.Require().Equal(preferences.Assets[2].ShowcaseVisibility, preferencesBack.Assets[2].ShowcaseVisibility)
	s.Require().Equal(preferences.Assets[2].Order, preferencesBack.Assets[2].Order)
}

func (s *TestProfileShowcasePersistence) TestProfileShowcaseContacts() {
	db, err := openTestDB()
	s.Require().NoError(err)
	persistence := newSQLitePersistence(db)

	showcase1 := &identity.ProfileShowcase{
		Communities: []*identity.ProfileShowcaseCommunity{
			&identity.ProfileShowcaseCommunity{
				CommunityID: "0x012312234234234",
				Order:       6,
			},
			&identity.ProfileShowcaseCommunity{
				CommunityID: "0x04523233466753",
				Order:       7,
			},
		},
		Assets: []*identity.ProfileShowcaseAsset{
			&identity.ProfileShowcaseAsset{
				Symbol: "ETH",
				Order:  1,
			},
		},
	}
	err = persistence.SaveProfileShowcaseForContact("contact_1", showcase1)
	s.Require().NoError(err)

	showcase2 := &identity.ProfileShowcase{
		Communities: []*identity.ProfileShowcaseCommunity{
			&identity.ProfileShowcaseCommunity{
				CommunityID: "0x012312234234234", // same id to check query
				Order:       3,
			},
			&identity.ProfileShowcaseCommunity{
				CommunityID: "0x096783478384593",
				Order:       7,
			},
		},
		Collectibles: []*identity.ProfileShowcaseCollectible{
			&identity.ProfileShowcaseCollectible{
				UID:   "d378662f-3d71-44e0-81ee-ff7f1778c13a",
				Order: 1,
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
