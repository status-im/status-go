package integration

import (
	"time"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol"
)

func (s *LinkPreviewsTestSuite) setProfileParameters(messenger *protocol.Messenger, displayName string, bio string, identityImages []images.IdentityImage) {
	const timeout = 1 * time.Second

	changes := protocol.SelfContactChangeEvent{}

	protocol.SetSettingsAndWaitForChange(&s.Suite, messenger, timeout, func() {
		err := messenger.SetDisplayName(displayName)
		s.Require().NoError(err)
		err = messenger.SetBio(bio)
		s.Require().NoError(err)
	}, func(event *protocol.SelfContactChangeEvent) bool {
		if event.DisplayNameChanged {
			changes.DisplayNameChanged = true
		}
		if event.BioChanged {
			changes.BioChanged = true
		}
		return changes.DisplayNameChanged && changes.BioChanged
	})

	protocol.SetIdentityImagesAndWaitForChange(&s.Suite, messenger, timeout, func() {
		err := messenger.multiAccounts.StoreIdentityImages(messenger.account.KeyUID, identityImages, false)
		s.Require().NoError(err)
	})

	selfContact := messenger.GetSelfContact()
	s.Require().Equal(selfContact.DisplayName, displayName)
	s.Require().Equal(selfContact.Bio, bio)

	for _, image := range identityImages {
		saved, ok := selfContact.Images[image.Name]
		s.Require().True(ok)
		s.Require().Equal(saved, image)
	}
	s.Require().Equal(selfContact.DisplayName, displayName)
}

func (s *LinkPreviewsTestSuite) Test_UnfurlURLs_SelfLink() {
	profileKp, _, _, err := accounts.GetProfileKeypairForTest(true, false, false)
	s.Require().NoError(err)
	profileKp.KeyUID = s.m.account.KeyUID
	profileKp.Accounts[0].KeyUID = s.m.account.KeyUID

	err = s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	// Set initial profile parameters
	identityImages := images.SampleIdentityImages()
	s.setProfileParameters(s.m, "TestDisplayName_3", "TestBio_3", identityImages)

	// Generate a shared URL
	u, err := s.m.ShareUserURLWithData(s.m.IdentityPublicKeyString())
	s.Require().NoError(err)

	// Update contact info locally after creating the shared URL
	// This is required to test that URL-decoded data is not used in the preview.
	iconPayload, err := images.GetPayloadFromURI(exampleIdenticonURI)
	s.Require().NoError(err)
	icon := images.IdentityImage{
		Name:    images.SmallDimName,
		Width:   50,
		Height:  50,
		Payload: iconPayload,
	}
	s.setProfileParameters(s.m, "TestDisplayName_4", "TestBio_4", []images.IdentityImage{icon})

	r, err := s.m.UnfurlURLs(nil, []string{u})
	s.Require().NoError(err)
	s.Require().Len(r.StatusLinkPreviews, 1)
	s.Require().Len(r.LinkPreviews, 0)

	userSettings, err := s.m.getSettings()
	s.Require().NoError(err)

	preview := r.StatusLinkPreviews[0]
	s.Require().Equal(u, preview.URL)
	s.Require().Nil(preview.Community)
	s.Require().Nil(preview.Channel)
	s.Require().NotNil(preview.Contact)
	s.Require().Equal(s.m.IdentityPublicKeyString(), preview.Contact.PublicKey)
	s.Require().Equal(userSettings.DisplayName, preview.Contact.DisplayName)
	s.Require().Equal(userSettings.Bio, preview.Contact.Description)

	s.Require().Equal(icon.Width, preview.Contact.Icon.Width)
	s.Require().Equal(icon.Height, preview.Contact.Icon.Height)
	s.Require().Equal("", preview.Contact.Icon.URL)

	expectedDataURI, err := images.GetPayloadDataURI(icon.Payload)
	s.Require().NoError(err)
	s.Require().Equal(expectedDataURI, preview.Contact.Icon.DataURI)
}
