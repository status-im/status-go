package protocol

import (
	"reflect"
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

	preferences := &identity.ProfileShowcasePreferences{
		Communities: []*identity.ProfileShowcaseCommunityPreference{
			&identity.ProfileShowcaseCommunityPreference{
				CommunityID:        "0x32433445133424",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
		},
		Accounts: []*identity.ProfileShowcaseAccountPreference{
			&identity.ProfileShowcaseAccountPreference{
				Address:            "0x0000000000000000000000000032433445133422",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
			&identity.ProfileShowcaseAccountPreference{
				Address:            "0x0000000000000000000000000032433445133424",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              1,
			},
		},
		Collectibles: []*identity.ProfileShowcaseCollectiblePreference{
			&identity.ProfileShowcaseCollectiblePreference{
				ContractAddress:    "0x12378534257568678487683576",
				ChainID:            11155111,
				TokenID:            "123213895929994903",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
		},
		VerifiedTokens: []*identity.ProfileShowcaseVerifiedTokenPreference{
			&identity.ProfileShowcaseVerifiedTokenPreference{
				Symbol:             "ETH",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              1,
			},
			&identity.ProfileShowcaseVerifiedTokenPreference{
				Symbol:             "DAI",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityIDVerifiedContacts,
				Order:              2,
			},
			&identity.ProfileShowcaseVerifiedTokenPreference{
				Symbol:             "SNT",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityNoOne,
				Order:              3,
			},
		},
		UnverifiedTokens: []*identity.ProfileShowcaseUnverifiedTokenPreference{
			&identity.ProfileShowcaseUnverifiedTokenPreference{
				ContractAddress:    "0x454525452023452",
				ChainID:            1,
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
			&identity.ProfileShowcaseUnverifiedTokenPreference{
				ContractAddress:    "0x12312323323233",
				ChainID:            11155111,
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              1,
			},
		},
		SocialLinks: []*identity.ProfileShowcaseSocialLinkPreference{
			&identity.ProfileShowcaseSocialLinkPreference{
				Text:               identity.TwitterID,
				URL:                "https://twitter.com/ethstatus",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              1,
			},
			&identity.ProfileShowcaseSocialLinkPreference{
				Text:               identity.TwitterID,
				URL:                "https://twitter.com/StatusIMBlog",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityIDVerifiedContacts,
				Order:              2,
			},
			&identity.ProfileShowcaseSocialLinkPreference{
				Text:               identity.GithubID,
				URL:                "https://github.com/status-im",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              3,
			},
		},
	}

	err = persistence.SaveProfileShowcasePreferences(preferences)
	s.Require().NoError(err)

	preferencesBack, err := persistence.GetProfileShowcasePreferences()
	s.Require().NoError(err)
	s.Require().True(reflect.DeepEqual(preferences, preferencesBack))

	newPreferences := &identity.ProfileShowcasePreferences{
		Communities: []*identity.ProfileShowcaseCommunityPreference{
			&identity.ProfileShowcaseCommunityPreference{
				CommunityID:        "0x32433445133424",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              0,
			},
		},
		Accounts: []*identity.ProfileShowcaseAccountPreference{
			&identity.ProfileShowcaseAccountPreference{
				Address:            "0x0000000000000000000000000032433445133422",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              0,
			},
		},
		Collectibles: []*identity.ProfileShowcaseCollectiblePreference{},
		VerifiedTokens: []*identity.ProfileShowcaseVerifiedTokenPreference{
			&identity.ProfileShowcaseVerifiedTokenPreference{
				Symbol:             "ETH",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              1,
			},
		},
		UnverifiedTokens: []*identity.ProfileShowcaseUnverifiedTokenPreference{
			&identity.ProfileShowcaseUnverifiedTokenPreference{
				ContractAddress:    "0x12312323323233",
				ChainID:            11155111,
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
		},
		SocialLinks: []*identity.ProfileShowcaseSocialLinkPreference{
			&identity.ProfileShowcaseSocialLinkPreference{
				Text:               identity.TwitterID,
				URL:                "https://twitter.com/ethstatus",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              1,
			},
		},
	}

	err = persistence.SaveProfileShowcasePreferences(newPreferences)
	s.Require().NoError(err)

	preferencesBack, err = persistence.GetProfileShowcasePreferences()
	s.Require().NoError(err)
	s.Require().True(reflect.DeepEqual(newPreferences, preferencesBack))
}

func (s *TestProfileShowcasePersistence) TestProfileShowcaseContacts() {
	db, err := openTestDB()
	s.Require().NoError(err)
	persistence := newSQLitePersistence(db)

	showcase1 := &identity.ProfileShowcase{
		ContactID: "contact_1",
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
		Accounts: []*identity.ProfileShowcaseAccount{
			&identity.ProfileShowcaseAccount{
				ContactID: "contact_1",
				Address:   "0x32433445133424",
				Name:      "Status Account",
				ColorID:   "blue",
				Emoji:     "-_-",
				Order:     0,
			},
			&identity.ProfileShowcaseAccount{
				ContactID: "contact_1",
				Address:   "0x0000000000000000000000000032433445133424",
				Name:      "Money Box",
				ColorID:   "red",
				Emoji:     ":o)",
				Order:     1,
			},
		},
		Collectibles: []*identity.ProfileShowcaseCollectible{
			&identity.ProfileShowcaseCollectible{
				ContractAddress: "0x12378534257568678487683576",
				ChainID:         1,
				TokenID:         "123213895929994903",
				Order:           0,
			},
		},
		VerifiedTokens: []*identity.ProfileShowcaseVerifiedToken{
			&identity.ProfileShowcaseVerifiedToken{
				Symbol: "ETH",
				Order:  1,
			},
			&identity.ProfileShowcaseVerifiedToken{
				Symbol: "DAI",
				Order:  2,
			},
			&identity.ProfileShowcaseVerifiedToken{
				Symbol: "SNT",
				Order:  3,
			},
		},
		UnverifiedTokens: []*identity.ProfileShowcaseUnverifiedToken{
			&identity.ProfileShowcaseUnverifiedToken{
				ContractAddress: "0x454525452023452",
				ChainID:         1,
				Order:           0,
			},
			&identity.ProfileShowcaseUnverifiedToken{
				ContractAddress: "0x12312323323233",
				ChainID:         11155111,
				Order:           1,
			},
		},
		SocialLinks: []*identity.ProfileShowcaseSocialLink{
			&identity.ProfileShowcaseSocialLink{
				URL:   "https://status.app/",
				Text:  "Status",
				Order: 1,
			},
			&identity.ProfileShowcaseSocialLink{
				URL:   "https://github.com/status-im",
				Text:  "Github",
				Order: 2,
			},
		},
	}
	err = persistence.SaveProfileShowcaseForContact(showcase1)
	s.Require().NoError(err)

	showcase2 := &identity.ProfileShowcase{
		ContactID: "contact_2",
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
				ContractAddress: "0x12378534257568678487683576",
				ChainID:         1,
				TokenID:         "123213895929994903",
				Order:           1,
			},
		},
	}
	err = persistence.SaveProfileShowcaseForContact(showcase2)
	s.Require().NoError(err)

	showcase1Back, err := persistence.GetProfileShowcaseForContact("contact_1")
	s.Require().NoError(err)

	s.Require().Equal(len(showcase1.Communities), len(showcase1Back.Communities))
	for i := 0; i < len(showcase1.Communities); i++ {
		s.Require().Equal(*showcase1.Communities[i], *showcase1Back.Communities[i])
	}
	s.Require().Equal(len(showcase1.Accounts), len(showcase1Back.Accounts))
	for i := 0; i < len(showcase1.Accounts); i++ {
		s.Require().Equal(*showcase1.Accounts[i], *showcase1Back.Accounts[i])
	}
	s.Require().Equal(len(showcase1.Collectibles), len(showcase1Back.Collectibles))
	for i := 0; i < len(showcase1.Collectibles); i++ {
		s.Require().Equal(*showcase1.Collectibles[i], *showcase1Back.Collectibles[i])
	}
	s.Require().Equal(len(showcase1.VerifiedTokens), len(showcase1Back.VerifiedTokens))
	for i := 0; i < len(showcase1.VerifiedTokens); i++ {
		s.Require().Equal(*showcase1.VerifiedTokens[i], *showcase1Back.VerifiedTokens[i])
	}
	s.Require().Equal(len(showcase1.UnverifiedTokens), len(showcase1Back.UnverifiedTokens))
	for i := 0; i < len(showcase1.UnverifiedTokens); i++ {
		s.Require().Equal(*showcase1.UnverifiedTokens[i], *showcase1Back.UnverifiedTokens[i])
	}
	s.Require().Equal(len(showcase1.SocialLinks), len(showcase1Back.SocialLinks))
	for i := 0; i < len(showcase1.SocialLinks); i++ {
		s.Require().Equal(*showcase1.SocialLinks[i], *showcase1Back.SocialLinks[i])
	}

	showcase2Back, err := persistence.GetProfileShowcaseForContact("contact_2")
	s.Require().NoError(err)

	s.Require().Equal(len(showcase2.Communities), len(showcase2Back.Communities))
	s.Require().Equal(*showcase2.Communities[0], *showcase2Back.Communities[0])
	s.Require().Equal(*showcase2.Communities[1], *showcase2Back.Communities[1])
	s.Require().Equal(len(showcase2.Collectibles), len(showcase2Back.Collectibles))
	s.Require().Equal(*showcase2.Collectibles[0], *showcase2Back.Collectibles[0])
	s.Require().Equal(0, len(showcase2Back.Accounts))
	s.Require().Equal(0, len(showcase2Back.VerifiedTokens))
	s.Require().Equal(0, len(showcase2Back.UnverifiedTokens))
	s.Require().Equal(0, len(showcase2Back.SocialLinks))
}

func (s *TestProfileShowcasePersistence) TestFetchingProfileShowcaseAccountsByAddress() {
	db, err := openTestDB()
	s.Require().NoError(err)
	persistence := newSQLitePersistence(db)

	conatacts := []*Contact{
		&Contact{
			ID: "contact_1",
		},
		&Contact{
			ID: "contact_2",
		},
		&Contact{
			ID: "contact_3",
		},
	}

	err = persistence.SaveContacts(conatacts)
	s.Require().NoError(err)

	showcase1 := &identity.ProfileShowcase{
		ContactID: "contact_1",
		Accounts: []*identity.ProfileShowcaseAccount{
			&identity.ProfileShowcaseAccount{
				ContactID: "contact_1",
				Address:   "0x0000000000000000000000000000000000000001",
				Name:      "Contact1-Account1",
				ColorID:   "blue",
				Emoji:     "-_-",
				Order:     0,
			},
			&identity.ProfileShowcaseAccount{
				ContactID: "contact_1",
				Address:   "0x0000000000000000000000000000000000000002",
				Name:      "Contact1-Account2",
				ColorID:   "blue",
				Emoji:     "-_-",
				Order:     1,
			},
		},
	}
	showcase2 := &identity.ProfileShowcase{
		ContactID: "contact_2",
		Accounts: []*identity.ProfileShowcaseAccount{
			&identity.ProfileShowcaseAccount{
				ContactID: "contact_2",
				Address:   "0x0000000000000000000000000000000000000001",
				Name:      "Contact2-Account1",
				ColorID:   "blue",
				Emoji:     "-_-",
				Order:     0,
			},
			&identity.ProfileShowcaseAccount{
				ContactID: "contact_2",
				Address:   "0x0000000000000000000000000000000000000002",
				Name:      "Contact2-Account2",
				ColorID:   "blue",
				Emoji:     "-_-",
				Order:     1,
			},
		},
	}
	showcase3 := &identity.ProfileShowcase{
		ContactID: "contact_3",
		Accounts: []*identity.ProfileShowcaseAccount{
			&identity.ProfileShowcaseAccount{
				ContactID: "contact_3",
				Address:   "0x0000000000000000000000000000000000000001",
				Name:      "Contact3-Account1",
				ColorID:   "blue",
				Emoji:     "-_-",
				Order:     0,
			},
		},
	}

	err = persistence.SaveProfileShowcaseForContact(showcase1)
	s.Require().NoError(err)
	err = persistence.SaveProfileShowcaseForContact(showcase2)
	s.Require().NoError(err)
	err = persistence.SaveProfileShowcaseForContact(showcase3)
	s.Require().NoError(err)

	showcaseAccounts, err := persistence.GetProfileShowcaseAccountsByAddress(showcase1.Accounts[0].Address)
	s.Require().NoError(err)

	s.Require().Equal(3, len(showcaseAccounts))
	for i := 0; i < len(showcaseAccounts); i++ {
		if showcaseAccounts[i].ContactID == showcase1.ContactID {
			s.Require().Equal(showcase1.Accounts[0].Address, showcase1.Accounts[0].Address)
		} else if showcaseAccounts[i].ContactID == showcase2.ContactID {
			s.Require().Equal(showcase2.Accounts[0].Address, showcase2.Accounts[0].Address)
		} else if showcaseAccounts[i].ContactID == showcase3.ContactID {
			s.Require().Equal(showcase3.Accounts[0].Address, showcase3.Accounts[0].Address)
		} else {
			s.Require().Fail("unexpected contact id")
		}
	}

	showcaseAccounts, err = persistence.GetProfileShowcaseAccountsByAddress(showcase1.Accounts[1].Address)
	s.Require().NoError(err)

	s.Require().Equal(2, len(showcaseAccounts))
	for i := 0; i < len(showcaseAccounts); i++ {
		if showcaseAccounts[i].ContactID == showcase1.ContactID {
			s.Require().Equal(showcase1.Accounts[0].Address, showcase1.Accounts[0].Address)
		} else if showcaseAccounts[i].ContactID == showcase2.ContactID {
			s.Require().Equal(showcase2.Accounts[0].Address, showcase2.Accounts[0].Address)
		} else {
			s.Require().Fail("unexpected contact id")
		}
	}
}

func (s *TestProfileShowcasePersistence) TestUpdateProfileShowcaseAccountOnWalletAccountChange() {
	db, err := openTestDB()
	s.Require().NoError(err)
	persistence := newSQLitePersistence(db)

	deleteAccountAddress := "0x0000000000000000000000000033433445133423"
	updateAccountAddress := "0x0000000000000000000000000032433445133424"

	preferences := &identity.ProfileShowcasePreferences{
		Accounts: []*identity.ProfileShowcaseAccountPreference{
			&identity.ProfileShowcaseAccountPreference{
				Address:            deleteAccountAddress,
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
			&identity.ProfileShowcaseAccountPreference{
				Address:            updateAccountAddress,
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              1,
			},
		},
	}

	err = persistence.SaveProfileShowcasePreferences(preferences)
	s.Require().NoError(err)

	account, err := persistence.GetProfileShowcaseAccountPreference(updateAccountAddress)
	s.Require().NoError(err)
	s.Require().NotNil(account)
	s.Require().Equal(*account, *preferences.Accounts[1])

	account.ShowcaseVisibility = identity.ProfileShowcaseVisibilityIDVerifiedContacts
	account.Order = 7

	err = persistence.SaveProfileShowcaseAccountPreference(account)
	s.Require().NoError(err)

	deleted, err := persistence.DeleteProfileShowcaseAccountPreference(deleteAccountAddress)
	s.Require().NoError(err)
	s.Require().True(deleted)

	// One more time to check correct error handling
	deleted, err = persistence.DeleteProfileShowcaseAccountPreference(deleteAccountAddress)
	s.Require().NoError(err)
	s.Require().False(deleted)

	preferencesBack, err := persistence.GetProfileShowcasePreferences()
	s.Require().NoError(err)

	s.Require().Len(preferencesBack.Accounts, 1)
	s.Require().Equal(*preferencesBack.Accounts[0], *account)
}

func (s *TestProfileShowcasePersistence) TestUpdateProfileShowcaseCommunityOnChange() {
	db, err := openTestDB()
	s.Require().NoError(err)
	persistence := newSQLitePersistence(db)

	deleteCommunityID := "0x3243344513424"

	preferences := &identity.ProfileShowcasePreferences{
		Communities: []*identity.ProfileShowcaseCommunityPreference{
			&identity.ProfileShowcaseCommunityPreference{
				CommunityID:        "0x32433445133424",
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
			&identity.ProfileShowcaseCommunityPreference{
				CommunityID:        deleteCommunityID,
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              1,
			},
		},
	}

	err = persistence.SaveProfileShowcasePreferences(preferences)
	s.Require().NoError(err)

	deleted, err := persistence.DeleteProfileShowcaseCommunityPreference(deleteCommunityID)
	s.Require().NoError(err)
	s.Require().True(deleted)

	// One more time to check correct error handling
	deleted, err = persistence.DeleteProfileShowcaseCommunityPreference(deleteCommunityID)
	s.Require().NoError(err)
	s.Require().False(deleted)

	preferencesBack, err := persistence.GetProfileShowcasePreferences()
	s.Require().NoError(err)

	s.Require().Len(preferencesBack.Communities, 1)
	s.Require().Equal(*preferencesBack.Communities[0], *preferences.Communities[0])
}
