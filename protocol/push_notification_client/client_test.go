package push_notification_client

import (
	"bytes"
	"crypto/ecdsa"
	"io/ioutil"
	"math/rand"
	"os"

	"testing"

	"github.com/google/uuid"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/stretchr/testify/suite"
)

type ClientSuite struct {
	suite.Suite
	tmpFile        *os.File
	persistence    *Persistence
	identity       *ecdsa.PrivateKey
	installationID string
	client         *Client
}

func TestClientSuite(t *testing.T) {
	s := new(ClientSuite)
	s.installationID = "c6ae4fde-bb65-11ea-b3de-0242ac130004"

	suite.Run(t, s)
}

func (s *ClientSuite) SetupTest() {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)
	s.tmpFile = tmpFile

	database, err := sqlite.Open(s.tmpFile.Name(), "")
	s.Require().NoError(err)
	s.persistence = NewPersistence(database)

	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.identity = identity

	config := &Config{
		Identity:                   identity,
		Logger:                     tt.MustCreateTestLogger(),
		RemoteNotificationsEnabled: true,
		InstallationID:             s.installationID,
	}

	s.client = New(s.persistence, config, nil)
}

func (s *ClientSuite) TestBuildPushNotificationRegisterMessage() {
	myDeviceToken := "device-token"
	mutedChatList := []string{"a", "b"}

	// build chat lish hashes
	var mutedChatListHashes [][]byte
	for _, chatID := range mutedChatList {
		mutedChatListHashes = append(mutedChatListHashes, common.Shake256([]byte(chatID)))
	}

	contactKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactIDs := []*ecdsa.PublicKey{&contactKey.PublicKey}

	// Set random generator for uuid
	var seed int64 = 1
	uuid.SetRand(rand.New(rand.NewSource(seed)))

	// Get token
	expectedUUID := uuid.New().String()

	// Reset random generator
	uuid.SetRand(rand.New(rand.NewSource(seed)))

	s.client.deviceToken = myDeviceToken
	// Set reader
	s.client.reader = bytes.NewReader([]byte(expectedUUID))

	options := &protobuf.PushNotificationRegistration{
		Version:         1,
		AccessToken:     expectedUUID,
		Token:           myDeviceToken,
		InstallationId:  s.installationID,
		Enabled:         true,
		BlockedChatList: mutedChatListHashes,
	}

	actualMessage, err := s.client.buildPushNotificationRegistrationMessage(contactIDs, mutedChatList)
	s.Require().NoError(err)

	s.Require().Equal(options, actualMessage)
}

func (s *ClientSuite) TestBuildPushNotificationRegisterMessageAllowFromContactsOnly() {
	myDeviceToken := "device-token"
	mutedChatList := []string{"a", "b"}

	// build chat lish hashes
	var mutedChatListHashes [][]byte
	for _, chatID := range mutedChatList {
		mutedChatListHashes = append(mutedChatListHashes, common.Shake256([]byte(chatID)))
	}

	contactKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactIDs := []*ecdsa.PublicKey{&contactKey.PublicKey}

	// Set random generator for uuid
	var seed int64 = 1
	uuid.SetRand(rand.New(rand.NewSource(seed)))

	// Get token
	expectedUUID := uuid.New().String()

	// set up reader
	reader := bytes.NewReader([]byte(expectedUUID))

	sharedKey, err := ecies.ImportECDSA(s.identity).GenerateShared(
		ecies.ImportECDSAPublic(&contactKey.PublicKey),
		accessTokenKeyLength,
		accessTokenKeyLength,
	)
	s.Require().NoError(err)
	// build encrypted token
	encryptedToken, err := encryptAccessToken([]byte(expectedUUID), sharedKey, reader)
	s.Require().NoError(err)

	// Reset random generator
	uuid.SetRand(rand.New(rand.NewSource(seed)))

	s.client.config.allowFromContactsOnly = true
	s.client.deviceToken = myDeviceToken
	// Set reader
	s.client.reader = bytes.NewReader([]byte(expectedUUID))

	options := &protobuf.PushNotificationRegistration{
		Version:               1,
		AccessToken:           expectedUUID,
		Token:                 myDeviceToken,
		InstallationId:        s.installationID,
		AllowFromContactsOnly: true,
		Enabled:               true,
		BlockedChatList:       mutedChatListHashes,
		AllowedUserList:       [][]byte{encryptedToken},
	}

	actualMessage, err := s.client.buildPushNotificationRegistrationMessage(contactIDs, mutedChatList)
	s.Require().NoError(err)

	s.Require().Equal(options, actualMessage)
}

func (s *ClientSuite) TestNotifyOnMessageID() {
	messageID := []byte("message-id")
	chatID := "chat-id"
	installationID1 := "1"
	installationID2 := "2"

	s.Require().NoError(s.client.NotifyOnMessageID(chatID, messageID))

	key1, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// First time, should notify
	response, err := s.client.shouldNotifyOn(&key1.PublicKey, installationID1, messageID)
	s.Require().NoError(err)
	s.Require().True(response)

	// Save notification
	s.Require().NoError(s.client.notifiedOn(&key1.PublicKey, installationID1, messageID))

	// Second time, should not notify
	response, err = s.client.shouldNotifyOn(&key1.PublicKey, installationID1, messageID)
	s.Require().NoError(err)
	s.Require().False(response)

	// Different installationID
	response, err = s.client.shouldNotifyOn(&key1.PublicKey, installationID2, messageID)
	s.Require().NoError(err)
	s.Require().True(response)

	key2, err := crypto.GenerateKey()
	s.Require().NoError(err)
	// different key, should notify
	response, err = s.client.shouldNotifyOn(&key2.PublicKey, installationID1, messageID)
	s.Require().NoError(err)
	s.Require().True(response)

	// non tracked message id
	response, err = s.client.shouldNotifyOn(&key1.PublicKey, installationID1, []byte("not-existing"))
	s.Require().NoError(err)
	s.Require().False(response)
}

func (s *ClientSuite) TestShouldRefreshToken() {
	key1, err := crypto.GenerateKey()
	s.Require().NoError(err)
	key2, err := crypto.GenerateKey()
	s.Require().NoError(err)
	key3, err := crypto.GenerateKey()
	s.Require().NoError(err)
	key4, err := crypto.GenerateKey()
	s.Require().NoError(err)

	// Contacts are added
	s.Require().False(s.client.shouldRefreshToken([]*ecdsa.PublicKey{&key1.PublicKey, &key2.PublicKey}, []*ecdsa.PublicKey{&key1.PublicKey, &key2.PublicKey, &key3.PublicKey, &key4.PublicKey}))

	// everything the same
	s.Require().False(s.client.shouldRefreshToken([]*ecdsa.PublicKey{&key1.PublicKey, &key2.PublicKey}, []*ecdsa.PublicKey{&key2.PublicKey, &key1.PublicKey}))

	// A contact is removed
	s.Require().True(s.client.shouldRefreshToken([]*ecdsa.PublicKey{&key1.PublicKey, &key2.PublicKey}, []*ecdsa.PublicKey{&key2.PublicKey}))

}
