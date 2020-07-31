package pushnotificationserver

import (
	"crypto/ecdsa"
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
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
	grant          []byte
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
		Logger:   tt.MustCreateTestLogger(),
	}

	s.server = New(config, s.persistence, nil)

	sharedKey, err := s.server.generateSharedKey(&s.key.PublicKey)
	s.Require().NoError(err)
	s.sharedKey = sharedKey
	signatureMaterial := s.server.buildGrantSignatureMaterial(&s.key.PublicKey, &identity.PublicKey, s.accessToken)
	grant, err := crypto.Sign(signatureMaterial, s.key)
	s.Require().NoError(err)

	s.grant = grant

}

func (s *ServerSuite) TestPushNotificationServerValidateRegistration() {

	// Empty payload
	_, err := s.server.validateRegistration(&s.key.PublicKey, nil)
	s.Require().Equal(ErrEmptyPushNotificationRegistrationPayload, err)

	// Empty key
	_, err = s.server.validateRegistration(nil, []byte("payload"))
	s.Require().Equal(ErrEmptyPushNotificationRegistrationPublicKey, err)

	// Invalid cyphertext length
	_, err = s.server.validateRegistration(&s.key.PublicKey, []byte("too short"))
	s.Require().Equal(common.ErrInvalidCiphertextLength, err)

	// Invalid cyphertext length
	_, err = s.server.validateRegistration(&s.key.PublicKey, []byte("too short"))
	s.Require().Equal(common.ErrInvalidCiphertextLength, err)

	// Invalid ciphertext
	_, err = s.server.validateRegistration(&s.key.PublicKey, []byte("not too short but invalid"))
	s.Require().Error(common.ErrInvalidCiphertextLength, err)

	// Different key ciphertext
	cyphertext, err := common.Encrypt([]byte("plaintext"), make([]byte, 32), rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Error(err)

	// Right cyphertext but non unmarshable payload
	cyphertext, err = common.Encrypt([]byte("plaintext"), s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrCouldNotUnmarshalPushNotificationRegistration, err)

	// Missing installationID
	payload, err := proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken: s.accessToken,
		Grant:       s.grant,
		TokenType:   protobuf.PushNotificationRegistration_APN_TOKEN,
		Version:     1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationInstallationID, err)

	// Malformed installationID
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		Grant:          s.grant,
		InstallationId: "abc",
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationInstallationID, err)

	// Version set to 0
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		Grant:          s.grant,
		InstallationId: s.installationID,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrInvalidPushNotificationRegistrationVersion, err)

	// Version lower than previous one
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		Grant:          s.grant,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)

	// Setup persistence
	s.Require().NoError(s.persistence.SavePushNotificationRegistration(common.HashPublicKey(&s.key.PublicKey), &protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		Grant:          s.grant,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		InstallationId: s.installationID,
		Version:        2}))

	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrInvalidPushNotificationRegistrationVersion, err)

	// Cleanup persistence
	s.Require().NoError(s.persistence.DeletePushNotificationRegistration(common.HashPublicKey(&s.key.PublicKey), s.installationID))

	// Unregistering message
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		InstallationId: s.installationID,
		Unregister:     true,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Nil(err)

	// Missing access token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		InstallationId: s.installationID,
		Grant:          s.grant,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationAccessToken, err)

	// Invalid access token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    "bc",
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		Grant:          s.grant,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationAccessToken, err)

	// Missing device token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		Grant:          s.grant,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationDeviceToken, err)

	// Missing  grant
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		DeviceToken:    "device-token",
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationGrant, err)

	// Invalid  grant
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		DeviceToken:    "device-token",
		Grant:          crypto.Keccak256([]byte("invalid")),
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationRegistrationGrant, err)

	// Missing  token type
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		DeviceToken:    "device-token",
		Grant:          s.grant,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().Equal(ErrUnknownPushNotificationRegistrationTokenType, err)

	// Successful
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		DeviceToken:    "abc",
		AccessToken:    s.accessToken,
		Grant:          s.grant,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = s.server.validateRegistration(&s.key.PublicKey, cyphertext)
	s.Require().NoError(err)
}

func (s *ServerSuite) TestPushNotificationHandleRegistration() {
	// Empty payload
	response := s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, nil)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Empty key
	response = s.server.buildPushNotificationRegistrationResponse(nil, []byte("payload"))
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Invalid cyphertext length
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, []byte("too short"))
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Invalid cyphertext length
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, []byte("too short"))
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Invalid ciphertext
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, []byte("not too short but invalid"))
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Different key ciphertext
	cyphertext, err := common.Encrypt([]byte("plaintext"), make([]byte, 32), rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Right cyphertext but non unmarshable payload
	cyphertext, err = common.Encrypt([]byte("plaintext"), s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Missing installationID
	payload, err := proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken: s.accessToken,
		Grant:       s.grant,
		Version:     1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Malformed installationID
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		InstallationId: "abc",
		Grant:          s.grant,
		Version:        1,
	})
	s.Require().NoError(err)
	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Version set to 0
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		Grant:          s.grant,
		InstallationId: s.installationID,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_VERSION_MISMATCH)

	// Version lower than previous one
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		Grant:          s.grant,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)

	// Setup persistence
	s.Require().NoError(s.persistence.SavePushNotificationRegistration(common.HashPublicKey(&s.key.PublicKey), &protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		Grant:          s.grant,
		InstallationId: s.installationID,
		Version:        2}))

	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_VERSION_MISMATCH)

	// Cleanup persistence
	s.Require().NoError(s.persistence.DeletePushNotificationRegistration(common.HashPublicKey(&s.key.PublicKey), s.installationID))

	// Missing access token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		InstallationId: s.installationID,
		Grant:          s.grant,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Invalid access token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    "bc",
		Grant:          s.grant,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Missing device token
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		AccessToken:    s.accessToken,
		Grant:          s.grant,
		InstallationId: s.installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().False(response.Success)
	s.Require().Equal(response.Error, protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE)

	// Successful
	registration := &protobuf.PushNotificationRegistration{
		DeviceToken:    "abc",
		AccessToken:    s.accessToken,
		Grant:          s.grant,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		InstallationId: s.installationID,
		Version:        1,
	}
	payload, err = proto.Marshal(registration)
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().True(response.Success)

	// Pull from the db
	retrievedRegistration, err := s.persistence.GetPushNotificationRegistrationByPublicKeyAndInstallationID(common.HashPublicKey(&s.key.PublicKey), s.installationID)
	s.Require().NoError(err)
	s.Require().NotNil(retrievedRegistration)
	s.Require().True(proto.Equal(retrievedRegistration, registration))

	// Unregistering message
	payload, err = proto.Marshal(&protobuf.PushNotificationRegistration{
		DeviceToken:    "token",
		InstallationId: s.installationID,
		Unregister:     true,
		Version:        2,
	})
	s.Require().NoError(err)

	cyphertext, err = common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response = s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().True(response.Success)
	s.Require().Equal(common.Shake256(cyphertext), response.RequestId)

	// Check is gone from the db
	retrievedRegistration, err = s.persistence.GetPushNotificationRegistrationByPublicKeyAndInstallationID(common.HashPublicKey(&s.key.PublicKey), s.installationID)
	s.Require().NoError(err)
	s.Require().Nil(retrievedRegistration)
	// Check version is mantained
	version, err := s.persistence.GetPushNotificationRegistrationVersion(common.HashPublicKey(&s.key.PublicKey), s.installationID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(2), version)
}

func (s *ServerSuite) TestbuildPushNotificationQueryResponseNoFiltering() {
	hashedPublicKey := common.HashPublicKey(&s.key.PublicKey)
	// Successful
	registration := &protobuf.PushNotificationRegistration{
		DeviceToken:    "abc",
		AccessToken:    s.accessToken,
		Grant:          s.grant,
		TokenType:      protobuf.PushNotificationRegistration_APN_TOKEN,
		InstallationId: s.installationID,
		Version:        1,
	}
	payload, err := proto.Marshal(registration)
	s.Require().NoError(err)

	cyphertext, err := common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response := s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().True(response.Success)

	query := &protobuf.PushNotificationQuery{
		PublicKeys: [][]byte{[]byte("non-existing"), hashedPublicKey},
	}

	queryResponse := s.server.buildPushNotificationQueryResponse(query)
	s.Require().NotNil(queryResponse)
	s.Require().True(queryResponse.Success)
	s.Require().Len(queryResponse.Info, 1)
	s.Require().Equal(s.accessToken, queryResponse.Info[0].AccessToken)
	s.Require().Equal(hashedPublicKey, queryResponse.Info[0].PublicKey)
	s.Require().Equal(s.installationID, queryResponse.Info[0].InstallationId)
	s.Require().Nil(queryResponse.Info[0].AllowedKeyList)
}

func (s *ServerSuite) TestbuildPushNotificationQueryResponseWithFiltering() {
	hashedPublicKey := common.HashPublicKey(&s.key.PublicKey)
	allowedKeyList := [][]byte{[]byte("a")}
	// Successful

	registration := &protobuf.PushNotificationRegistration{
		DeviceToken:           "abc",
		AccessToken:           s.accessToken,
		Grant:                 s.grant,
		TokenType:             protobuf.PushNotificationRegistration_APN_TOKEN,
		InstallationId:        s.installationID,
		AllowFromContactsOnly: true,
		AllowedKeyList:        allowedKeyList,
		Version:               1,
	}
	payload, err := proto.Marshal(registration)
	s.Require().NoError(err)

	cyphertext, err := common.Encrypt(payload, s.sharedKey, rand.Reader)
	s.Require().NoError(err)
	response := s.server.buildPushNotificationRegistrationResponse(&s.key.PublicKey, cyphertext)
	s.Require().NotNil(response)
	s.Require().True(response.Success)

	query := &protobuf.PushNotificationQuery{
		PublicKeys: [][]byte{[]byte("non-existing"), hashedPublicKey},
	}

	queryResponse := s.server.buildPushNotificationQueryResponse(query)
	s.Require().NotNil(queryResponse)
	s.Require().True(queryResponse.Success)
	s.Require().Len(queryResponse.Info, 1)
	s.Require().Equal(hashedPublicKey, queryResponse.Info[0].PublicKey)
	s.Require().Equal(s.installationID, queryResponse.Info[0].InstallationId)
	s.Require().Equal(allowedKeyList, queryResponse.Info[0].AllowedKeyList)
}
