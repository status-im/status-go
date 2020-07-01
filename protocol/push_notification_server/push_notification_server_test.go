package protocol

import (
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/protobuf"
)

func TestPushNotificationServerValidateRegistration(t *testing.T) {
	accessToken := "b6ae4fde-bb65-11ea-b3de-0242ac130004"
	installationID := "c6ae4fde-bb65-11ea-b3de-0242ac130004"
	identity, err := crypto.GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Identity: identity,
	}

	server := Server{config: config}

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	sharedKey, err := server.generateSharedKey(&key.PublicKey)
	require.NoError(t, err)

	// Empty payload
	require.Equal(t, ErrEmptyPushNotificationOptionsPayload, server.ValidateRegistration(nil, &key.PublicKey, nil))

	// Empty key
	require.Equal(t, ErrEmptyPushNotificationOptionsPublicKey, server.ValidateRegistration(nil, nil, []byte("payload")))

	// Invalid cyphertext length
	require.Equal(t, ErrInvalidCiphertextLength, server.ValidateRegistration(nil, &key.PublicKey, []byte("too short")))

	// Invalid cyphertext length
	require.Equal(t, ErrInvalidCiphertextLength, server.ValidateRegistration(nil, &key.PublicKey, []byte("too short")))

	// Invalid ciphertext
	require.Error(t, ErrInvalidCiphertextLength, server.ValidateRegistration(nil, &key.PublicKey, []byte("not too short but invalid")))

	// Different key ciphertext
	cyphertext, err := encrypt([]byte("plaintext"), make([]byte, 32), rand.Reader)
	require.NoError(t, err)
	require.Error(t, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

	// Right cyphertext but non unmarshable payload
	cyphertext, err = encrypt([]byte("plaintext"), sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Equal(t, ErrCouldNotUnmarshalPushNotificationOptions, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

	// Missing installationID
	payload, err := proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken: accessToken,
		Version:     1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Equal(t, ErrMalformedPushNotificationOptionsInstallationID, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

	// Malformed installationID
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: "abc",
		Version:        1,
	})
	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Equal(t, ErrMalformedPushNotificationOptionsInstallationID, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

	// Version set to 0
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Equal(t, ErrInvalidPushNotificationOptionsVersion, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

	// Version lower than previous one
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Equal(t, ErrInvalidPushNotificationOptionsVersion, server.ValidateRegistration(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
		Version:        2}, &key.PublicKey, cyphertext))

	// Unregistering message
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		InstallationId: installationID,
		Unregister:     true,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Nil(t, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

	// Missing access token
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		InstallationId: installationID,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Equal(t, ErrMalformedPushNotificationOptionsAccessToken, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

	// Invalid access token
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    "bc",
		InstallationId: installationID,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Equal(t, ErrMalformedPushNotificationOptionsAccessToken, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

	// Missing device token
	payload, err = proto.Marshal(&protobuf.PushNotificationOptions{
		AccessToken:    accessToken,
		InstallationId: installationID,
		Version:        1,
	})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Equal(t, ErrMalformedPushNotificationOptionsDeviceToken, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

}
