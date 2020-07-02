package push_notification_server

import (
	"crypto/ecdsa"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
)

func TestServerSuite(t *testing.T) {
	s := new(ServerSuite)
	s.accessToken = "b6ae4fde-bb65-11ea-b3de-0242ac130004"
	s.installationID = "c6ae4fde-bb65-11ea-b3de-0242ac130004"

	suite.Run(t, s)
}

type ServerSuite struct {
	suite.Suite
	tmpFile        *os.File
	persistence    Persistence
	accessToken    string
	installationID string
	identity       *ecdsa.PrivateKey
	key            *ecdsa.PrivateKey
	sharedKey      []byte
	server         *Server
}

func (s *ServerSuite) SetupTest() {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)
	s.tmpFile = tmpFile

	database, err := sqlite.Open(s.tmpFile.Name(), "")
	s.Require().NoError(err)
	s.persistence = NewSQLitePersistence(database)

	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.identity = identity

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.key = key

	config := &Config{
		Identity: identity,
	}

	s.server = New(config, s.persistence)

	sharedKey, err := s.server.generateSharedKey(&s.key.PublicKey)
	s.Require().NoError(err)
	s.sharedKey = sharedKey

}

func (s *ServerSuite) TestPushNotificationServerValidateRegistration() {

	// Empty payload
	_, err := s.server.ValidateRegistration(&s.key.PublicKey, nil)
	s.Require().Equal(ErrEmptyPushNotificationRegistrationPayload, err)

	// Empty key
	_, err = s.server.ValidateRegistration(nil, []byte("payload"))
	s.Require().Equal(ErrEmptyPushNotificationRegistrationPublicKey, err)

	// Invalid cyphertext length
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, []byte("too short"))
	s.Require().Equal(ErrInvalidCiphertextLength, err)

	// Invalid cyphertext length
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, []byte("too short"))
	s.Require().Equal(ErrInvalidCiphertextLength, err)

	// Invalid ciphertext
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, []byte("not too short but invalid"))
	s.Require().Error(ErrInvalidCiphertextLength, err)

	// Different key ciphertext
	cyphertext, err := encrypt([]byte("plaintext"), make([]byte, 32), rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Error(err)

	// Right cyphertext but non unmarshable payload
	cyphertext, err = encrypt([]byte("plaintext"), s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrCouldNotUnmarshalPushNotificationRegistration, err)

	// Missing installationID
	payload, err := proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken: s.accessToken,
		Version:     1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationInstallationID, err)

	// Malformed installationID
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: "abc",
		Version:        1,
	})
	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationInstallationID, err)

	// Version set to 0
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrInvalidPushNotificationRegistrationVersion, err)

	// Version lower than previous one
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)

	// Setup persistence
	s.Require().NoError(s.persistence.SavePushNotificationRegistration(&s.key.PublicKey, &protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
		Version:        2}))

	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrInvalidPushNotificationRegistrationVersion, err)

	// Cleanup persistence
	s.Require().NoError(s.persistence.DeletePushNotificationRegistration(&s.key.PublicKey, s.installationID))

	// Unregistering message
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		InstallationId: s.installationID,
		Unregister:     true,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Nil(err)

	// Missing access token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationAccessToken, err)

	// Invalid access token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    "bc",
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationAccessToken, err)

	// Missing device token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationDeviceToken, err)

	// Successful
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		Token:          "abc",
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.ValidateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NoError(err)
}

func (s *ServerSuite) TestPushNotificationHandleRegistration() {
	// Empty payload
	response := s.server.HandlePushNotificationRegistration(&s.key.PublicKey, nil)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Empty key
	response = s.server.HandlePushNotificationRegistration(nil, []byte("payload"))
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Invalid cyphertext length
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, []byte("too short"))
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Invalid cyphertext length
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, []byte("too short"))
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Invalid ciphertext
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, []byte("not too short but invalid"))
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Different key ciphertext
	cyphertext, err := encrypt([]byte("plaintext"), make([]byte, 32), rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Right cyphertext but non unmarshable payload
	cyphertext, err = encrypt([]byte("plaintext"), s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Missing installationID
	payload, err := proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken: s.accessToken,
		Version:     1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Malformed installationID
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: "abc",
		Version:        1,
	})
	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Version set to 0
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_VERSION_MISMATCH)

	// Version lower than previous one
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)

	// Setup persistence
	s.Require().NoError(s.persistence.SavePushNotificationRegistration(&s.key.PublicKey, &protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
		Version:        2}))

	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_VERSION_MISMATCH)

	// Cleanup persistence
	s.Require().NoError(s.persistence.DeletePushNotificationRegistration(&s.key.PublicKey, s.installationID))

	// Missing access token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Invalid access token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    "bc",
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Missing device token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Successful
	registration := &protobuf.PushNotificationRegistration{
		Token:          "abc",
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
		Version:        1,
	}
	payload, err = proto.Marshal(registration)
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().True(response.Success)

	// Pull from the db
	retrievedRegistration, err := s.persistence.GetPushNotificationRegistrationByPublicKeyAndInstallationID(&s.key.PublicKey, s.installationID)
	s.Require().NoError(err)
	s.Require().NotNil(retrievedRegistration)
	s.Require().True(proto.Equal(retrievedRegistration, registration))

	// Unregistering message
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		Token:          "token",
		InstallationId: s.installationID,
		Unregister:     true,
		Version:        2,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().True(response.Success)

	// Check is gone from the db
	retrievedRegistration, err = s.persistence.GetPushNotificationRegistrationByPublicKeyAndInstallationID(&s.key.PublicKey, s.installationID)
	s.Require().NoError(err)
	s.Require().NotNil(retrievedRegistration)
	s.Require().Empty(retrievedRegistration.AccessToken)
	s.Require().Empty(retrievedRegistration.Token)
	s.Require().Equal(uint64(2), retrievedRegistration.Version)
	s.Require().Equal(s.installationID, retrievedRegistration.InstallationId)
	s.Require().Equal(shake256(cyphertext), response.RequestId)
}

func (s *ServerSuite) TestHandlePushNotificationQueryNoFiltering() {
	hashedPublicKey := hashPublicKey(&s.key.PublicKey)
	// Successful
	registration := &protobuf.PushNotificationRegistration{
		Token:          "abc",
		AccessToken:    s.accessToken,
		InstallationId: s.installationID,
		Version:        1,
	}
	payload, err := proto.Marshal(registration)
	s.Require().NoError(err)

	cyphertext, err := encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response := s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().True(response.Success)

	query := &protobuf.PushNotificationQuery{
		PublicKeys: [][]byte{[]byte("non-existing"), hashedPublicKey},
	}

	queryResponse := s.server.HandlePushNotificationQuery(query)
	s.Require().NotNil(queryResponse)
	s.Require().True(queryResponse.Success)
	s.Require().Len(queryResponse.Info, 1)
	s.Require().Equal(s.accessToken, queryResponse.Info[0].AccessToken)
	s.Require().Equal(hashedPublicKey, queryResponse.Info[0].PublicKey)
	s.Require().Equal(s.installationID, queryResponse.Info[0].InstallationId)
	s.Require().Nil(queryResponse.Info[0].AllowedUserList)
}

func (s *ServerSuite) TestHandlePushNotificationQueryWithFiltering() {
	hashedPublicKey := hashPublicKey(&s.key.PublicKey)
	allowedUserList := [][]byte{[]byte("a")}
	// Successful

	registration := &protobuf.PushNotificationRegistration{
		Token:           "abc",
		AccessToken:     s.accessToken,
		InstallationId:  s.installationID,
		AllowedUserList: allowedUserList,
		Version:         1,
	}
	payload, err := proto.Marshal(registration)
	s.Require().NoError(err)

	cyphertext, err := encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response := s.server.HandlePushNotificationRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().True(response.Success)

	query := &protobuf.PushNotificationQuery{
		PublicKeys: [][]byte{[]byte("non-existing"), hashedPublicKey},
	}

	queryResponse := s.server.HandlePushNotificationQuery(query)
	s.Require().NotNil(queryResponse)
	s.Require().True(queryResponse.Success)
	s.Require().Len(queryResponse.Info, 1)
	s.Require().Equal(hashedPublicKey, queryResponse.Info[0].PublicKey)
	s.Require().Equal(s.installationID, queryResponse.Info[0].InstallationId)
	s.Require().Equal(allowedUserList, queryResponse.Info[0].AllowedUserList)
}
