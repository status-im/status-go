package protocol

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/identity"
)

func TestMessengerContacts(t *testing.T) {
	suite.Run(t, new(MessengerContactsTestSuite))
}

type MessengerContactsTestSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerContactsTestSuite) Test_SelfContact() {
	profileKp := accounts.GetProfileKeypairForTest(true, false, false)
	profileKp.KeyUID = s.m.account.KeyUID
	profileKp.Accounts[0].KeyUID = s.m.account.KeyUID

	err := s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	// Create values

	displayName := "DisplayName_1"
	bio := "Bio_1"
	ensName := "EnsName_1.eth"
	socialLinks := identity.SocialLinks{{Text: identity.TelegramID, URL: "dummy.telegram"}}
	identityImages := images.SampleIdentityImages()

	identityImagesMap := make(map[string]images.IdentityImage)
	for _, img := range identityImages {
		img.KeyUID = s.m.account.KeyUID
		identityImagesMap[img.Name] = img
	}

	// Create change subscriptions

	const timeout = 100 * time.Second
	var wg sync.WaitGroup
	displayNameChanged := false
	preferredNameChanged := false
	bioChanged := false
	identityImageChanged := false

	wg.Add(1)
	go func() {
		channel := s.m.settings.SubscribeToChanges()
		defer func() {
			wg.Done()
		}()
		for !displayNameChanged || !preferredNameChanged || !bioChanged {
			select {
			case setting := <-channel:
				switch setting.GetReactName() {
				case settings.DisplayName.GetReactName():
					displayNameChanged = true
				case settings.PreferredName.GetReactName():
					preferredNameChanged = true
				case settings.Bio.GetReactName():
					bioChanged = true
				}
			case <-time.After(timeout):
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-s.m.multiAccounts.SubscribeToIdentityImageChanges():
			identityImageChanged = true
		case <-time.After(timeout):
			return
		}
	}()

	// Set settings

	err = s.m.SetDisplayName(displayName)
	s.Require().NoError(err)

	err = s.m.SetBio(bio)
	s.Require().NoError(err)

	err = s.m.settings.SaveSettingField(settings.PreferredName, ensName)
	s.Require().NoError(err)

	err = s.m.AddOrReplaceSocialLinks(socialLinks)
	s.Require().NoError(err)

	err = s.m.multiAccounts.StoreIdentityImages(s.m.account.KeyUID, identityImages, false)
	s.Require().NoError(err)

	// Wait for changes

	wg.Wait()
	s.Require().True(displayNameChanged)
	s.Require().True(preferredNameChanged)
	s.Require().True(bioChanged)
	s.Require().True(identityImageChanged)

	// Check values

	selfContact := s.m.GetContactByID(s.m.IdentityPublicKeyString())
	s.Require().NotNil(selfContact)
	s.Require().Equal(displayName, selfContact.DisplayName)
	s.Require().Equal(bio, selfContact.Bio)
	s.Require().Equal(ensName, selfContact.EnsName)
	s.Require().Equal(socialLinks, selfContact.SocialLinks)
	s.Require().Equal(identityImagesMap, selfContact.Images)
}
