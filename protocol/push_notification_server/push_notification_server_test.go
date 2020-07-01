package protocol

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/protobuf"
)

type MockPersistence struct {
	pno *protobuf.PushNotificationOptions
}

func (p *MockPersistence) GetPushNotificationOptions(publicKey *ecdsa.PublicKey, installationID string) (*protobuf.PushNotificationOptions, error) {
	return p.pno, nil
}

func TestPushNotificationServerValidateRegistration(t *testing.T) {
	accessToken := "b6ae4fde-bb65-11ea-b3de-0242ac130004"
	installationID := "c6ae4fde-bb65-11ea-b3de-0242ac130004"
	identity, err := crypto.GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Identity: identity,
	}

	mockPersistence := &MockPersistence{}
	server := New(config, mockPersistence)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	sharedKey, err := server.generateSharedKey(&key.PublicKey)
	require.NoError(t, err)

	// Empty payload
	_, err = server.ValidateRegistration(&key.PublicKey, nil)
	require.Equal(t, ErrEmptyPushNotificationOptionsPayload, err)

	// Empty key
	_, err = server.ValidateRegistration(nil, []byte("payload"))
	require.Equal(t, ErrEmptyPushNotificationOptionsPublicKey, err)

	// Invalid cyphertext length
	_, err = server.ValidateRegistration(&key.PublicKey, []byte("too short"))
	require.Equal(t, ErrInvalidCiphertextLength, err)

	// Invalid cyphertext length
	_, err = server.ValidateRegistration(&key.PublicKey, []byte("too short"))
	require.Equal(t, ErrInvalidCiphertextLength, err)

	// Invalid ciphertext
	_, err = server.ValidateRegistration(&key.PublicKey, []byte("not too short but invalid"))
	require.Error(t, ErrInvalidCiphertextLength, err)

	// Different key ciphertext
	cyphertext, err := encrypt([]byte("plaintext"), make([]byte, 32), rand.Reader)
	require.NoError(t, err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Error(t, err)

	// Right cyphertext but non unmarshable payload
	cyphertext, err = encrypt([]byte("plaintext"), sharedKey, rand.Reader)
	require.NoError(t, err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Equal(t, ErrCouldNotUnmarshalPushNotificationOptions, err)

	// Missing installationID
	payload, err := proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken: accessToken,
		Version:     1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Equal(t, ErrMalformedPushNotificationOptionsInstallationID, err)

	// Malformed installationID
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: "abc",
		Version:        1,
	})
	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Equal(t, ErrMalformedPushNotificationOptionsInstallationID, err)

	// Version set to 0
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Equal(t, ErrInvalidPushNotificationOptionsVersion, err)

	// Version lower than previous one
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)

	// Setup mock
	mockPersistence.pno = &protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
		Version:        2}

	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Equal(t, ErrInvalidPushNotificationOptionsVersion, err)

	// Cleanup mock
	mockPersistence.pno = nil

	// Unregistering message
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		InstallationId: installationID,
		Unregister:     true,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Nil(t, err)

	// Missing access token
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		InstallationId: installationID,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Equal(t, ErrMalformedPushNotificationOptionsAccessToken, err)

	// Invalid access token
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    "bc",
		InstallationId: installationID,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Equal(t, ErrMalformedPushNotificationOptionsAccessToken, err)

	// Missing device token
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	_, err = server.ValidateRegistration(&key.PublicKey, cyphertext)
	require.Equal(t, ErrMalformedPushNotificationOptionsDeviceToken, err)
}
