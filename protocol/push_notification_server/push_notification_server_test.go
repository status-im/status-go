package push_notification_server

import (
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
	suite.Run(t, new(ServerSuite))
}

type ServerSuite struct {
	suite.Suite
	tmpFile     *os.File
	persistence Persistence
}

func (s *ServerSuite) SetupTest() {
	tmpFile, err := ioutil.TempFile("", "")
	s.Require().NoError(err)
	s.tmpFile = tmpFile

	database, err := sqlite.Open(s.tmpFile.Name(), "")
	s.Require().NoError(err)
	s.persistence = NewSQLitePersistence(database)
}

func (s *ServerSuite) TestPushNotificationServerValidateRegistration() {
	accessToken := "b6ae4fde-bb65-11ea-b3de-0242ac130004"
	installationID := "c6ae4fde-bb65-11ea-b3de-0242ac130004"
	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	config := &Config{
		Identity: identity,
	}

	server := New(config, s.persistence)

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	sharedKey, err := server.generateSharedKey(&key.PublicKey)
	s.Require().NoError(err)

	// Empty payload
	_, err = server.ValidateRegistration(&key.PublicKey, nil)
	s.Require().Equal(ErrEmptyPushNotificationOptionsPayload, err)

	// Empty key
	_, err = server.ValidateRegistration(nil, []byte("payload"))
	s.Require().Equal(ErrEmptyPushNotificationOptionsPublicKey, err)

	// Invalid cyphertext length
	_, err = server.ValidateRegistration(&key.PublicKey, []byte("too short"))
	s.Require().Equal(ErrInvalidCiphertextLength, err)

	// Invalid cyphertext length
	_, err = server.ValidateRegistration(&key.PublicKey, []byte("too short"))
	s.Require().Equal(ErrInvalidCiphertextLength, err)

	// Invalid ciphertext
	_, err = server.ValidateRegistration(&key.PublicKey, []byte("not too short but invalid"))
	s.Require().Error(ErrInvalidCiphertextLength, err)

	// Different key ciphertext
	cyphertext, err := encrypt([]byte("plaintext"), make([]byte, 32), rand.Reader)
	s.Require().NoError(err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Error(err)

	// Right cyphertext but non unmarshable payload
	cyphertext, err = encrypt([]byte("plaintext"), sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Equal(ErrCouldNotUnmarshalPushNotificationOptions, err)

	// Missing installationID
	payload, err := proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken: accessToken,
		Version:     1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationOptionsInstallationID, err)

	// Malformed installationID
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: "abc",
		Version:        1,
	})
	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationOptionsInstallationID, err)

	// Version set to 0
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Equal(ErrInvalidPushNotificationOptionsVersion, err)

	// Version lower than previous one
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	s.Require().NoError(err)

	// Setup mock
	s.Require().NoError(s.persistence.SavePushNotificationOptions(&key.PublicKey, &protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
		Version:        2}))

	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Equal(ErrInvalidPushNotificationOptionsVersion, err)

	// Cleanup mock
	s.Require().NoError(s.persistence.DeletePushNotificationOptions(&key.PublicKey, installationID))

	// Unregistering message
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		InstallationId: installationID,
		Unregister:     true,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Nil(err)

	// Missing access token
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		InstallationId: installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationOptionsAccessToken, err)

	// Invalid access token
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    "bc",
		InstallationId: installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationOptionsAccessToken, err)

	// Missing device token
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
		Version:        1,
	})
	s.Require().NoError(err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	s.Require().NoError(err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	s.Require().Equal(ErrMalformedPushNotificationOptionsDeviceToken, err)
}
