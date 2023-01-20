package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerProfilePictureHandlerSuite(t *testing.T) {
	suite.Run(t, new(MessengerProfilePictureHandlerSuite))
}

type MessengerProfilePictureHandlerSuite struct {
	suite.Suite
	alice *Messenger // client instance of Messenger
	bob   *Messenger // server instance of Messenger

	aliceKey *ecdsa.PrivateKey // private key for the alice instance of Messenger
	bobKey   *ecdsa.PrivateKey // private key for the bob instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerProfilePictureHandlerSuite) SetupSuite() {
	s.logger = tt.MustCreateTestLogger()
}

func (s *MessengerProfilePictureHandlerSuite) setup() {
	var err error

	// Setup Waku things
	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	wakuLogger := s.logger.Named("Waku")
	shh := waku.New(&config, wakuLogger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	// Generate private keys for Alice and Bob
	s.aliceKey, err = crypto.GenerateKey()
	s.Require().NoError(err)
	s.bobKey, err = crypto.GenerateKey()
	s.Require().NoError(err)

	// Generate Alice Messenger
	aliceLogger := s.logger.Named("Alice messenger")
	s.alice, err = newMessengerWithKey(s.shh, s.aliceKey, aliceLogger, []Option{})
	s.Require().NoError(err)
	s.logger.Debug("alice messenger created")

	// Generate Bob Messenger
	bobLogger := s.logger.Named("Bob messenger")
	s.bob, err = newMessengerWithKey(s.shh, s.bobKey, bobLogger, []Option{})
	s.Require().NoError(err)
	s.logger.Debug("bob messenger created")

	// Setup MultiAccount for Alice Messenger
	s.logger.Debug("alice setupMultiAccount before")
	s.setupMultiAccount(s.alice)
	s.logger.Debug("alice setupMultiAccount after")
}

func (s *MessengerProfilePictureHandlerSuite) setupTest() {
	s.logger.Debug("setupTest fired")
	s.setup()
	s.logger.Debug("setupTest completed")
}

func (s *MessengerProfilePictureHandlerSuite) SetupTest() {
	s.logger.Debug("SetupTest fired")
	s.setup()
	s.logger.Debug("SetupTest completed")
}

func (s *MessengerProfilePictureHandlerSuite) tearDown() {
	// Shutdown messengers
	s.NoError(s.alice.Shutdown())
	s.alice = nil
	s.NoError(s.bob.Shutdown())
	s.bob = nil
	_ = s.logger.Sync()
	//time.Sleep(2 * time.Second)
}

func (s *MessengerProfilePictureHandlerSuite) tearDownTest() {
	s.logger.Debug("tearDownTest fired")
	s.tearDown()
	s.logger.Debug("tearDownTest completed")
}

func (s *MessengerProfilePictureHandlerSuite) TearDownTest() {
	s.logger.Debug("TearDownTest fired")
	s.tearDown()
	s.logger.Debug("TearDownTest completed")
}

func (s *MessengerProfilePictureHandlerSuite) generateKeyUID(publicKey *ecdsa.PublicKey) string {
	return types.EncodeHex(crypto.FromECDSAPub(publicKey))
}

func (s *MessengerProfilePictureHandlerSuite) setupMultiAccount(m *Messenger) {
	keyUID := s.generateKeyUID(&m.identity.PublicKey)
	m.account = &multiaccounts.Account{KeyUID: keyUID}

	err := m.multiAccounts.SaveAccount(multiaccounts.Account{Name: "string", KeyUID: keyUID})
	s.NoError(err)
}

func (s *MessengerProfilePictureHandlerSuite) generateAndStoreIdentityImages(m *Messenger) []images.IdentityImage {
	keyUID := s.generateKeyUID(&m.identity.PublicKey)
	iis := images.SampleIdentityImages()
	s.Require().NoError(m.multiAccounts.StoreIdentityImages(keyUID, iis, false))

	return iis
}

func (s *MessengerProfilePictureHandlerSuite) TestChatIdentity() {
	iis := s.generateAndStoreIdentityImages(s.alice)
	ci, err := s.alice.createChatIdentity(privateChat)
	s.Require().NoError(err)
	s.Require().Exactly(len(iis), len(ci.Images))
}

func (s *MessengerProfilePictureHandlerSuite) TestEncryptDecryptIdentityImagesWithContactPubKeys() {
	smPayload := "hello small image"
	lgPayload := "hello large image"

	ci := protobuf.ChatIdentity{
		Clock: uint64(time.Now().Unix()),
		Images: map[string]*protobuf.IdentityImage{
			"small": {
				Payload: []byte(smPayload),
			},
			"large": {
				Payload: []byte(lgPayload),
			},
		},
	}

	// Make contact keys and Contacts, set the Contacts to added
	contactKeys := make([]*ecdsa.PrivateKey, 10)
	for i := range contactKeys {
		contactKey, err := crypto.GenerateKey()
		s.Require().NoError(err)
		contactKeys[i] = contactKey

		contact, err := BuildContactFromPublicKey(&contactKey.PublicKey)
		s.Require().NoError(err)

		contact.ContactRequestLocalState = ContactRequestStateSent

		s.alice.allContacts.Store(contact.ID, contact)
	}

	// Test EncryptIdentityImagesWithContactPubKeys
	err := EncryptIdentityImagesWithContactPubKeys(ci.Images, s.alice)
	s.Require().NoError(err)

	for _, ii := range ci.Images {
		s.Require().Equal(s.alice.allContacts.Len(), len(ii.EncryptionKeys))
	}
	s.Require().NotEqual([]byte(smPayload), ci.Images["small"].Payload)
	s.Require().NotEqual([]byte(lgPayload), ci.Images["large"].Payload)
	s.Require().True(ci.Images["small"].Encrypted)
	s.Require().True(ci.Images["large"].Encrypted)

	// Test DecryptIdentityImagesWithIdentityPrivateKey
	err = DecryptIdentityImagesWithIdentityPrivateKey(ci.Images, contactKeys[2], &s.alice.identity.PublicKey)
	s.Require().NoError(err)

	s.Require().Equal(smPayload, string(ci.Images["small"].Payload))
	s.Require().Equal(lgPayload, string(ci.Images["large"].Payload))
	s.Require().False(ci.Images["small"].Encrypted)
	s.Require().False(ci.Images["large"].Encrypted)

	// RESET Messenger identity, Contacts and IdentityImage.EncryptionKeys
	s.alice.allContacts = new(contactMap)
	ci.Images["small"].EncryptionKeys = nil
	ci.Images["large"].EncryptionKeys = nil

	// Test EncryptIdentityImagesWithContactPubKeys with no contacts
	err = EncryptIdentityImagesWithContactPubKeys(ci.Images, s.alice)
	s.Require().NoError(err)

	for _, ii := range ci.Images {
		s.Require().Equal(0, len(ii.EncryptionKeys))
	}
	s.Require().NotEqual([]byte(smPayload), ci.Images["small"].Payload)
	s.Require().NotEqual([]byte(lgPayload), ci.Images["large"].Payload)
	s.Require().True(ci.Images["small"].Encrypted)
	s.Require().True(ci.Images["large"].Encrypted)

	// Test DecryptIdentityImagesWithIdentityPrivateKey with no valid identity
	err = DecryptIdentityImagesWithIdentityPrivateKey(ci.Images, contactKeys[2], &s.alice.identity.PublicKey)
	s.Require().NoError(err)

	s.Require().NotEqual([]byte(smPayload), ci.Images["small"].Payload)
	s.Require().NotEqual([]byte(lgPayload), ci.Images["large"].Payload)
	s.Require().True(ci.Images["small"].Encrypted)
	s.Require().True(ci.Images["large"].Encrypted)
}

func (s *MessengerProfilePictureHandlerSuite) TestPictureInPrivateChatOneSided() {
	s.setupTest()
	err := s.bob.settings.SaveSettingField(settings.ProfilePicturesVisibility, settings.ProfilePicturesShowToEveryone)
	s.Require().NoError(err)

	err = s.alice.settings.SaveSettingField(settings.ProfilePicturesVisibility, settings.ProfilePicturesShowToEveryone)
	s.Require().NoError(err)

	bChat := CreateOneToOneChat(s.generateKeyUID(&s.aliceKey.PublicKey), &s.aliceKey.PublicKey, s.alice.transport)
	err = s.bob.SaveChat(bChat)
	s.Require().NoError(err)

	_, err = s.bob.Join(bChat)
	s.Require().NoError(err)

	// Alice sends a message to the public chat
	message := buildTestMessage(*bChat)
	response, err := s.bob.SendChatMessage(context.Background(), message)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 2 * time.Second
	}

	err = tt.RetryWithBackOff(func() error {

		response, err = s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		s.Require().NotNil(response)

		contacts := response.Contacts
		s.logger.Debug("RetryWithBackOff contact data", zap.Any("contacts", contacts))

		if len(contacts) > 0 && len(contacts[0].Images) > 0 {
			s.logger.Debug("", zap.Any("contacts", contacts))
			return nil
		}

		return errors.New("no new contacts with images received")
	}, options)
}

func (s *MessengerProfilePictureHandlerSuite) TestE2eSendingReceivingProfilePicture() {
	profilePicShowSettings := map[string]settings.ProfilePicturesShowToType{
		"ShowToContactsOnly": settings.ProfilePicturesShowToContactsOnly,
		"ShowToEveryone":     settings.ProfilePicturesShowToEveryone,
		"ShowToNone":         settings.ProfilePicturesShowToNone,
	}

	profilePicViewSettings := map[string]settings.ProfilePicturesVisibilityType{
		"ViewFromContactsOnly": settings.ProfilePicturesVisibilityContactsOnly,
		"ViewFromEveryone":     settings.ProfilePicturesVisibilityEveryone,
		"ViewFromNone":         settings.ProfilePicturesVisibilityNone,
	}

	isContactFor := map[string][]bool{
		"alice": {true, false},
		"bob":   {true, false},
	}

	chatContexts := []chatContext{
		publicChat,
		privateChat,
	}

	// TODO see if possible to push each test scenario into a go routine
	for _, cc := range chatContexts {
		for sn, ss := range profilePicShowSettings {
			for vn, vs := range profilePicViewSettings {
				for _, ac := range isContactFor["alice"] {
					for _, bc := range isContactFor["bob"] {
						s.logger.Debug("top of the loop")
						s.setupTest()
						s.logger.Info("testing with criteria:",
							zap.String("chat context type", string(cc)),
							zap.String("profile picture Show Settings", sn),
							zap.String("profile picture View Settings", vn),
							zap.Bool("bob in Alice's Contacts", ac),
							zap.Bool("alice in Bob's Contacts", bc),
						)

						expectPicture, err := resultExpected(ss, vs, ac, bc)
						s.logger.Debug("expect to receive a profile pic?",
							zap.Bool("result", expectPicture),
							zap.Error(err))

						// Setting up Bob
						s.logger.Debug("Setting up test criteria for Bob")

						s.logger.Debug("Save bob profile-pictures-visibility setting before")
						err = s.bob.settings.SaveSettingField(settings.ProfilePicturesVisibility, vs)
						s.Require().NoError(err)
						s.logger.Debug("Save bob profile-pictures-visibility setting after")

						s.logger.Debug("bob add contact before")
						if bc {
							s.logger.Debug("bob has contact to add")
							_, err = s.bob.AddContact(context.Background(), &requests.AddContact{ID: types.Hex2Bytes(s.generateKeyUID(&s.alice.identity.PublicKey))})
							s.Require().NoError(err)
							s.logger.Debug("bob add contact after")
						}
						s.logger.Debug("bob add contact after after")

						// Create Bob's chats
						switch cc {
						case publicChat:
							s.logger.Debug("making publicChats for bob")

							// Bob opens up the public chat and joins it
							bChat := CreatePublicChat("status", s.alice.transport)
							err = s.bob.SaveChat(bChat)
							s.Require().NoError(err)

							_, err = s.bob.Join(bChat)
							s.Require().NoError(err)
						case privateChat:
							s.logger.Debug("making privateChats for bob")

							s.logger.Debug("Bob making one to one chat with alice")
							bChat := CreateOneToOneChat(s.generateKeyUID(&s.aliceKey.PublicKey), &s.aliceKey.PublicKey, s.alice.transport)
							s.logger.Debug("Bob saving one to one chat with alice")
							err = s.bob.SaveChat(bChat)
							s.Require().NoError(err)
							s.logger.Debug("Bob saved one to one chat with alice")

							s.logger.Debug("Bob joining one to one chat with alice")
							_, err = s.bob.Join(bChat)
							s.Require().NoError(err)
							s.logger.Debug("Bob joined one to one chat with alice")
						default:
							s.Failf("unexpected chat context type", "%s", string(cc))
						}

						// Setting up Alice
						s.logger.Debug("Setting up test criteria for Alice")

						s.logger.Debug("Save alice profile-pictures-show-to setting before")
						err = s.alice.settings.SaveSettingField(settings.ProfilePicturesShowTo, ss)
						s.Require().NoError(err)
						s.logger.Debug("Save alice profile-pictures-show-to setting after")

						s.logger.Debug("alice add contact before")
						if ac {
							s.logger.Debug("alice has contact to add")
							_, err = s.alice.AddContact(context.Background(), &requests.AddContact{ID: types.Hex2Bytes(s.generateKeyUID(&s.bob.identity.PublicKey))})
							s.Require().NoError(err)
							s.logger.Debug("alice add contact after")
						}
						s.logger.Debug("alice add contact after after")

						s.logger.Debug("generateAndStoreIdentityImages before")
						iis := s.generateAndStoreIdentityImages(s.alice)
						s.logger.Debug("generateAndStoreIdentityImages after")

						s.logger.Debug("Before making chat for alice")
						// Create chats
						var aChat *Chat
						switch cc {
						case publicChat:
							s.logger.Debug("making publicChats for alice")

							// Alice opens creates a public chat
							aChat = CreatePublicChat("status", s.alice.transport)
							err = s.alice.SaveChat(aChat)
							s.Require().NoError(err)

						case privateChat:
							s.logger.Debug("making privateChats for alice")

							s.logger.Debug("Alice making one to one chat with bob")
							aChat = CreateOneToOneChat(s.generateKeyUID(&s.bobKey.PublicKey), &s.bobKey.PublicKey, s.bob.transport)
							s.logger.Debug("Alice saving one to one chat with bob")
							err = s.alice.SaveChat(aChat)
							s.Require().NoError(err)
							s.logger.Debug("Alice saved one to one chat with bob")

							s.logger.Debug("Alice joining one to one chat with bob")
							_, err = s.alice.Join(aChat)
							s.Require().NoError(err)
							s.logger.Debug("Alice joined one to one chat with bob")

							s.logger.Debug("alice before manually triggering publishContactCode")
							err = s.alice.publishContactCode()
							s.logger.Debug("alice after manually triggering publishContactCode",
								zap.Error(err))
							s.Require().NoError(err)
						default:
							s.Failf("unexpected chat context type", "%s", string(cc))
						}

						s.logger.Debug("Build and send a chat from alice")

						// Alice sends a message to the public chat
						message := buildTestMessage(*aChat)
						response, err := s.alice.SendChatMessage(context.Background(), message)
						s.Require().NoError(err)
						s.Require().NotNil(response)

						// Poll bob to see if he got the chatIdentity
						// Retrieve ChatIdentity
						var contacts []*Contact

						s.logger.Debug("Checking Bob to see if he got the chatIdentity")
						options := func(b *backoff.ExponentialBackOff) {
							b.MaxElapsedTime = 2 * time.Second
						}
						err = tt.RetryWithBackOff(func() error {

							response, err = s.bob.RetrieveAll()
							if err != nil {
								return err
							}

							contacts = response.Contacts
							s.logger.Debug("RetryWithBackOff contact data", zap.Any("contacts", contacts))

							if len(contacts) > 0 && len(contacts[0].Images) > 0 {
								s.logger.Debug("", zap.Any("contacts", contacts))
								return nil
							}

							return errors.New("no new contacts with images received")
						}, options)

						s.logger.Debug("Finished RetryWithBackOff got Bob",
							zap.Any("contacts", contacts),
							zap.Error(err))

						if expectPicture {
							s.logger.Debug("expecting a contact with images")
							s.Require().NoError(err)
							s.Require().NotNil(contacts)
						} else {
							s.logger.Debug("expecting no contacts with images")
							s.Require().EqualError(err, "no new contacts with images received")
							s.logger.Info("Completed testing with criteria:",
								zap.String("chat context type", string(cc)),
								zap.String("profile picture Show Settings", sn),
								zap.String("profile picture View Settings", vn),
								zap.Bool("bob in Alice's Contacts", ac),
								zap.Bool("alice in Bob's Contacts", bc),
							)
							s.tearDownTest()
							continue
						}

						s.logger.Debug("checking alice's contact")

						// Check if alice's contact data with profile picture is there
						var contact *Contact
						for _, c := range contacts {
							if c.ID == s.generateKeyUID(&s.alice.identity.PublicKey) {
								contact = c
							}
						}
						s.Require().NotNil(contact)

						s.logger.Debug("checked alice's contact info all good")

						// Check that Bob now has Alice's profile picture(s)
						switch cc {
						case publicChat:
							// In public chat context we only need the images.SmallDimName, but also may have the large
							s.Require().GreaterOrEqual(len(contact.Images), 1)

							// Check if the result matches expectation
							smImg, ok := contact.Images[images.SmallDimName]
							s.Require().True(ok, "contact images must contain the images.SmallDimName")

							for _, ii := range iis {
								if ii.Name == images.SmallDimName {
									s.Require().Equal(ii.Payload, smImg.Payload)
								}
							}
						case privateChat:
							s.Require().Equal(len(contact.Images), 2)
							s.logger.Info("private chat chat images", zap.Any("iis", iis))

							// Check if the result matches expectation
							smImg, ok := contact.Images[images.SmallDimName]
							s.Require().True(ok, "contact images must contain the images.SmallDimName")

							lgImg, ok := contact.Images[images.LargeDimName]
							s.Require().True(ok, "contact images must contain the images.LargeDimName")

							for _, ii := range iis {
								switch ii.Name {
								case images.SmallDimName:
									s.Require().Equal(ii.Payload, smImg.Payload)
								case images.LargeDimName:
									s.Require().Equal(ii.Payload, lgImg.Payload)
								}
							}

						}

						s.logger.Info("Completed testing with criteria:",
							zap.String("chat context type", string(cc)),
							zap.String("profile picture Show Settings", sn),
							zap.String("profile picture View Settings", vn),
							zap.Bool("bob in Alice's Contacts", ac),
							zap.Bool("alice in Bob's Contacts", bc),
						)
						s.tearDownTest()
					}
				}
			}
		}
	}

	s.setupTest()
}

func resultExpected(ss settings.ProfilePicturesShowToType, vs settings.ProfilePicturesVisibilityType, ac, bc bool) (bool, error) {
	switch ss {
	case settings.ProfilePicturesShowToContactsOnly:
		if ac {
			return resultExpectedVS(vs, bc)
		}
		return false, nil
	case settings.ProfilePicturesShowToEveryone:
		return resultExpectedVS(vs, bc)
	case settings.ProfilePicturesShowToNone:
		return false, nil
	default:
		return false, errors.New("unknown ProfilePicturesShowToType")
	}
}

func resultExpectedVS(vs settings.ProfilePicturesVisibilityType, bc bool) (bool, error) {
	switch vs {
	case settings.ProfilePicturesVisibilityContactsOnly:
		return true, nil
	case settings.ProfilePicturesVisibilityEveryone:
		return true, nil
	case settings.ProfilePicturesVisibilityNone:
		// If we are contacts, we save the image regardless
		return bc, nil
	default:
		return false, errors.New("unknown ProfilePicturesVisibilityType")
	}
}
