package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/testutils/fake"
)

func TestMessengerContacts(t *testing.T) {
	suite.Run(t, new(MessengerContactsTestSuite))
}

type MessengerContactsTestSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerContactsTestSuite) TestContactWithImagesRestoresStoredPhoto() {
	peer, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contact, err := contacts.BuildContactFromPublicKey(&peer.PublicKey)
	s.Require().NoError(err)
	s.Require().NoError(s.m.persistence.SaveContact(contact, nil))
	photo := fake.IdentityImages()[0]
	_, _, err = s.m.persistence.UpdateContactChatIdentity(contact.ID, &protobuf.ChatIdentity{
		Clock: 1, Images: map[string]*protobuf.IdentityImage{photo.Name: {Payload: photo.Payload}},
	})
	s.Require().NoError(err)
	contact.Images = map[string]images.IdentityImage{photo.Name: {Name: photo.Name, LocalURL: "https://localhost/photo"}}
	s.m.allContacts.Store(contact.ID, contact)

	loaded := s.m.GetContactByIDWithImages(contact.ID)
	s.Require().NotNil(loaded)
	s.Require().Equal(photo.Payload, loaded.Images[photo.Name].Payload)
	s.Require().Empty(s.m.GetContactByID(contact.ID).Images[photo.Name].Payload)
	s.Require().Same(s.m.GetSelfContact(), s.m.GetContactByIDWithImages(s.m.IdentityPublicKeyString()))
	s.Require().Nil(s.m.GetContactByIDWithImages("missing"))
}

func (s *MessengerContactsTestSuite) Test_SelfContact() {
	const timeout = 1 * time.Second

	profileKp, _, _, err := accounts.GetProfileKeypairForTest(true, false, false)
	s.Require().NoError(err)
	profileKp.KeyUID = s.m.account.KeyUID
	profileKp.Accounts[0].KeyUID = s.m.account.KeyUID

	err = s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	// Create values

	displayName := "DisplayName_1"
	bio := "Bio_1"
	ensName := "EnsName_1.eth"
	identityImages := fake.IdentityImages()

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

	// Check values

	selfContact := s.m.GetContactByID(s.m.IdentityPublicKeyString())
	s.Require().NotNil(selfContact)
	s.Require().Equal(displayName, selfContact.DisplayName)
	s.Require().Equal(bio, selfContact.Bio)
	s.Require().Equal(ensName, selfContact.EnsName)
	s.Require().Equal(identityImagesMap, selfContact.Images)
}
