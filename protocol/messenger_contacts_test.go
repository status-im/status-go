package protocol

import (
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
	const timeout = 1 * time.Second

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

	// Set values stored in settings

	changesLatch := SelfContactChangeEvent{}

	setSettingsValues := func() {
		err := s.m.SetDisplayName(displayName)
		s.Require().NoError(err)

		err = s.m.SetBio(bio)
		s.Require().NoError(err)

		err = s.m.settings.SaveSettingField(settings.PreferredName, ensName)
		s.Require().NoError(err)
	}

	eventHandler := func(event *SelfContactChangeEvent) bool {
		if event.DisplayNameChanged {
			changesLatch.DisplayNameChanged = true
		}
		if event.BioChanged {
			changesLatch.BioChanged = true
		}
		if event.PreferredNameChanged {
			changesLatch.PreferredNameChanged = true
		}
		return changesLatch.DisplayNameChanged && changesLatch.BioChanged && changesLatch.PreferredNameChanged
	}

	SetSettingsAndWaitForChange(&s.Suite, s.m, timeout, setSettingsValues, eventHandler)

	// Set values stored in multiaccounts

	setIdentityImages := func() {
		err := s.m.multiAccounts.StoreIdentityImages(s.m.account.KeyUID, identityImages, false)
		s.Require().NoError(err)
	}

	SetIdentityImagesAndWaitForChange(&s.Suite, s.m, timeout, setIdentityImages)

	// Set social links. They are applied immediately, no need to wait.

	err = s.m.AddOrReplaceSocialLinks(socialLinks)
	s.Require().NoError(err)

	// Check values

	selfContact := s.m.GetContactByID(s.m.IdentityPublicKeyString())
	s.Require().NotNil(selfContact)
	s.Require().Equal(displayName, selfContact.DisplayName)
	s.Require().Equal(bio, selfContact.Bio)
	s.Require().Equal(ensName, selfContact.EnsName)
	s.Require().Equal(socialLinks, selfContact.SocialLinks)
	s.Require().Equal(identityImagesMap, selfContact.Images)
}
