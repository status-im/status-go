package protocol

import (
	_ "github.com/mutecomm/go-sqlcipher/v4" // require go-sqlcipher that overrides default implementation

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/protobuf"
)

func isImageWithNamePresent(imgs map[string]*protobuf.IdentityImage, name string) bool {
	for k, v := range imgs {
		if k == name && len(v.Payload) > 0 {
			return true
		}
	}

	return false
}

func (s *MessengerSuite) retrieveIdentityImages(alice, bob *Messenger, chat *Chat) map[string]*protobuf.IdentityImage {
	s.Require().NoError(alice.settings.SaveSettingField(settings.DisplayName, "alice"))

	identityImages := images.SampleIdentityImages()
	identityImagesMap := make(map[string]images.IdentityImage)
	for _, img := range identityImages {
		img.KeyUID = s.m.account.KeyUID
		identityImagesMap[img.Name] = img
	}

	err := s.m.multiAccounts.StoreIdentityImages(s.m.account.KeyUID, identityImages, true)
	s.Require().NoError(err)
	s.Require().NoError(alice.SaveChat(chat))
	s.Require().NoError(bob.settings.SaveSettingField(settings.DisplayName, "bob"))
	s.Require().NoError(bob.SaveChat(chat))

	chatContext := GetChatContextFromChatType(chat.ChatType)

	chatIdentity, err := alice.createChatIdentity(chatContext)
	s.Require().NoError(err)

	imgs := chatIdentity.Images
	s.Require().NoError(err)

	return imgs
}

func (s *MessengerSuite) TestTwoImagesAreAddedToChatIdentityForPrivateChat() {
	alice := s.m
	bob := s.newMessenger()
	defer TearDownMessenger(&s.Suite, bob)

	bobPkString := types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))

	chat := CreateOneToOneChat(bobPkString, &bob.identity.PublicKey, alice.transport)
	s.Require().Equal(privateChat, GetChatContextFromChatType(chat.ChatType))

	imgs := s.retrieveIdentityImages(alice, bob, chat)
	s.Require().Len(imgs, 2)
	s.Require().Equal(true, isImageWithNamePresent(imgs, "thumbnail"))
	s.Require().Equal(true, isImageWithNamePresent(imgs, "large"))
}

func (s *MessengerSuite) TestOneImageIsAddedToChatIdentityForPublicChat() {
	alice := s.m
	bob := s.newMessenger()
	defer TearDownMessenger(&s.Suite, bob)

	chat := CreatePublicChat("alic-and-bob-chat", &testTimeSource{})
	s.Require().Equal(publicChat, GetChatContextFromChatType(chat.ChatType))

	imgs := s.retrieveIdentityImages(alice, bob, chat)
	s.Require().Len(imgs, 1)
	s.Require().Equal(true, isImageWithNamePresent(imgs, "thumbnail"))
	s.Require().Equal(false, isImageWithNamePresent(imgs, "large"))
}
