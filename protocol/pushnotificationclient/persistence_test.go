package pushnotificationclient

import (
	"crypto/ecdsa"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
)

const (
	testAccessToken = "token"
	installationID1 = "installation-id-1"
	installationID2 = "installation-id-2"
	installationID3 = "installation-id-3"
)

func TestSQLitePersistenceSuite(t *testing.T) {
	suite.Run(t, new(SQLitePersistenceSuite))
}

type SQLitePersistenceSuite struct {
	suite.Suite
	tmpFile     *os.File
	persistence *Persistence
}

func (s *SQLitePersistenceSuite) SetupTest() {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)
	s.tmpFile = tmpFile

	database, err := sqlite.Open(s.tmpFile.Name(), "", sqlite.ReducedKDFIterationsNumber)
	s.Require().NoError(err)
	s.persistence = NewPersistence(database)
}

func (s *SQLitePersistenceSuite) TearDownTest() {
	_ = os.Remove(s.tmpFile.Name())
}

func (s *SQLitePersistenceSuite) TestSaveAndRetrieveServer() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	server := &PushNotificationServer{
		PublicKey:    &key.PublicKey,
		Registered:   true,
		RegisteredAt: 1,
		AccessToken:  testAccessToken,
	}

	s.Require().NoError(s.persistence.UpsertServer(server))

	retrievedServers, err := s.persistence.GetServers()
	s.Require().NoError(err)

	s.Require().Len(retrievedServers, 1)
	s.Require().True(retrievedServers[0].Registered)
	s.Require().Equal(int64(1), retrievedServers[0].RegisteredAt)
	s.Require().True(common.IsPubKeyEqual(retrievedServers[0].PublicKey, &key.PublicKey))
	s.Require().Equal(testAccessToken, retrievedServers[0].AccessToken)

	server.Registered = false
	server.RegisteredAt = 2

	s.Require().NoError(s.persistence.UpsertServer(server))

	retrievedServers, err = s.persistence.GetServers()
	s.Require().NoError(err)

	s.Require().Len(retrievedServers, 1)
	s.Require().False(retrievedServers[0].Registered)
	s.Require().Equal(int64(2), retrievedServers[0].RegisteredAt)
	s.Require().True(common.IsPubKeyEqual(retrievedServers[0].PublicKey, &key.PublicKey))
}

func (s *SQLitePersistenceSuite) TestSaveAndRetrieveInfo() {
	key1, err := crypto.GenerateKey()
	s.Require().NoError(err)
	key2, err := crypto.GenerateKey()
	s.Require().NoError(err)
	serverKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	infos := []*PushNotificationInfo{
		{
			PublicKey:       &key1.PublicKey,
			ServerPublicKey: &serverKey.PublicKey,
			RetrievedAt:     1,
			Version:         1,
			AccessToken:     testAccessToken,
			InstallationID:  installationID1,
		},
		{
			PublicKey:       &key1.PublicKey,
			ServerPublicKey: &serverKey.PublicKey,
			RetrievedAt:     1,
			Version:         1,
			AccessToken:     testAccessToken,
			InstallationID:  installationID2,
		},
		{
			PublicKey:       &key1.PublicKey,
			ServerPublicKey: &serverKey.PublicKey,
			RetrievedAt:     1,
			Version:         1,
			AccessToken:     testAccessToken,
			InstallationID:  installationID3,
		},
		{
			PublicKey:       &key2.PublicKey,
			ServerPublicKey: &serverKey.PublicKey,
			RetrievedAt:     1,
			Version:         1,
			AccessToken:     testAccessToken,
			InstallationID:  installationID1,
		},
		{
			PublicKey:       &key2.PublicKey,
			ServerPublicKey: &serverKey.PublicKey,
			RetrievedAt:     1,
			Version:         1,
			AccessToken:     testAccessToken,
			InstallationID:  installationID2,
		},
		{
			PublicKey:       &key2.PublicKey,
			ServerPublicKey: &serverKey.PublicKey,
			RetrievedAt:     1,
			Version:         1,
			AccessToken:     testAccessToken,
			InstallationID:  installationID3,
		},
	}

	s.Require().NoError(s.persistence.SavePushNotificationInfo(infos))

	retrievedInfos, err := s.persistence.GetPushNotificationInfo(&key1.PublicKey, []string{installationID1, installationID2})
	s.Require().NoError(err)

	s.Require().Len(retrievedInfos, 2)
}

func (s *SQLitePersistenceSuite) TestSaveAndRetrieveInfoWithVersion() {
	installationID := "installation-id-1"
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	serverKey1, err := crypto.GenerateKey()
	s.Require().NoError(err)
	serverKey2, err := crypto.GenerateKey()
	s.Require().NoError(err)

	infos := []*PushNotificationInfo{
		{
			PublicKey:       &key.PublicKey,
			ServerPublicKey: &serverKey1.PublicKey,
			RetrievedAt:     1,
			Version:         1,
			AccessToken:     testAccessToken,
			InstallationID:  installationID,
		},
		{
			PublicKey:       &key.PublicKey,
			ServerPublicKey: &serverKey2.PublicKey,
			RetrievedAt:     1,
			Version:         1,
			AccessToken:     testAccessToken,
			InstallationID:  installationID,
		},
	}

	s.Require().NoError(s.persistence.SavePushNotificationInfo(infos))

	retrievedInfos, err := s.persistence.GetPushNotificationInfo(&key.PublicKey, []string{installationID})
	s.Require().NoError(err)

	// We should retrieve both
	s.Require().Len(retrievedInfos, 2)
	s.Require().Equal(uint64(1), retrievedInfos[0].Version)

	// Bump version
	infos[0].Version = 2

	s.Require().NoError(s.persistence.SavePushNotificationInfo(infos))

	retrievedInfos, err = s.persistence.GetPushNotificationInfo(&key.PublicKey, []string{installationID})
	s.Require().NoError(err)

	// Only one should be retrieved now
	s.Require().Len(retrievedInfos, 1)
	s.Require().Equal(uint64(2), retrievedInfos[0].Version)

	// Lower version
	infos[0].Version = 1

	s.Require().NoError(s.persistence.SavePushNotificationInfo(infos))

	retrievedInfos, err = s.persistence.GetPushNotificationInfo(&key.PublicKey, []string{installationID})
	s.Require().NoError(err)

	s.Require().Len(retrievedInfos, 1)
	s.Require().Equal(uint64(2), retrievedInfos[0].Version)
}

func (s *SQLitePersistenceSuite) TestNotifiedOnAndUpdateNotificationResponse() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	installationID := "installation-id"
	messageID := []byte("message-id")

	sentNotification := &SentNotification{
		PublicKey:      &key.PublicKey,
		InstallationID: installationID,
		MessageID:      messageID,
		LastTriedAt:    time.Now().Unix(),
	}

	s.Require().NoError(s.persistence.UpsertSentNotification(sentNotification))

	retrievedNotification, err := s.persistence.GetSentNotification(sentNotification.HashedPublicKey(), installationID, messageID)
	s.Require().NoError(err)
	s.Require().Equal(sentNotification, retrievedNotification)

	retriableNotifications, err := s.persistence.GetRetriablePushNotifications()
	s.Require().NoError(err)
	s.Require().Len(retriableNotifications, 0)

	response := &protobuf.PushNotificationReport{
		Success:        false,
		Error:          protobuf.PushNotificationReport_WRONG_TOKEN,
		PublicKey:      sentNotification.HashedPublicKey(),
		InstallationId: installationID,
	}

	s.Require().NoError(s.persistence.UpdateNotificationResponse(messageID, response))
	// This notification should be retriable
	retriableNotifications, err = s.persistence.GetRetriablePushNotifications()
	s.Require().NoError(err)
	s.Require().Len(retriableNotifications, 1)

	sentNotification.Error = protobuf.PushNotificationReport_WRONG_TOKEN

	retrievedNotification, err = s.persistence.GetSentNotification(sentNotification.HashedPublicKey(), installationID, messageID)
	s.Require().NoError(err)
	s.Require().Equal(sentNotification, retrievedNotification)

	// Update with a successful notification
	response = &protobuf.PushNotificationReport{
		Success:        true,
		PublicKey:      sentNotification.HashedPublicKey(),
		InstallationId: installationID,
	}

	s.Require().NoError(s.persistence.UpdateNotificationResponse(messageID, response))

	sentNotification.Success = true
	sentNotification.Error = protobuf.PushNotificationReport_UNKNOWN_ERROR_TYPE

	retrievedNotification, err = s.persistence.GetSentNotification(sentNotification.HashedPublicKey(), installationID, messageID)
	s.Require().NoError(err)
	s.Require().Equal(sentNotification, retrievedNotification)

	// This notification should not be retriable
	retriableNotifications, err = s.persistence.GetRetriablePushNotifications()
	s.Require().NoError(err)
	s.Require().Len(retriableNotifications, 0)

	// Update with a unsuccessful notification, it should be ignored
	response = &protobuf.PushNotificationReport{
		Success:        false,
		Error:          protobuf.PushNotificationReport_WRONG_TOKEN,
		PublicKey:      sentNotification.HashedPublicKey(),
		InstallationId: installationID,
	}

	s.Require().NoError(s.persistence.UpdateNotificationResponse(messageID, response))

	sentNotification.Success = true
	sentNotification.Error = protobuf.PushNotificationReport_UNKNOWN_ERROR_TYPE

	retrievedNotification, err = s.persistence.GetSentNotification(sentNotification.HashedPublicKey(), installationID, messageID)
	s.Require().NoError(err)
	s.Require().Equal(sentNotification, retrievedNotification)
}

func (s *SQLitePersistenceSuite) TestSaveAndRetrieveRegistration() {
	// Try with nil first
	retrievedRegistration, retrievedContactIDs, err := s.persistence.GetLastPushNotificationRegistration()
	s.Require().NoError(err)
	s.Require().Nil(retrievedRegistration)
	s.Require().Nil(retrievedContactIDs)

	// Save & retrieve registration
	registration := &protobuf.PushNotificationRegistration{
		AccessToken: "test",
		Version:     3,
	}

	key1, err := crypto.GenerateKey()
	s.Require().NoError(err)

	key2, err := crypto.GenerateKey()
	s.Require().NoError(err)

	key3, err := crypto.GenerateKey()
	s.Require().NoError(err)

	publicKeys := []*ecdsa.PublicKey{&key1.PublicKey, &key2.PublicKey}

	s.Require().NoError(s.persistence.SaveLastPushNotificationRegistration(registration, publicKeys))
	retrievedRegistration, retrievedContactIDs, err = s.persistence.GetLastPushNotificationRegistration()
	s.Require().NoError(err)
	s.Require().True(proto.Equal(registration, retrievedRegistration))
	s.Require().Equal(publicKeys, retrievedContactIDs)

	// Override and retrieve

	registration.Version = 5
	publicKeys = append(publicKeys, &key3.PublicKey)
	s.Require().NoError(s.persistence.SaveLastPushNotificationRegistration(registration, publicKeys))
	retrievedRegistration, retrievedContactIDs, err = s.persistence.GetLastPushNotificationRegistration()
	s.Require().NoError(err)
	s.Require().True(proto.Equal(registration, retrievedRegistration))
	s.Require().Equal(publicKeys, retrievedContactIDs)
}
