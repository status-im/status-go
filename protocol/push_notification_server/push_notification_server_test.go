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
	require.Equal(t, ErrEmptyPushNotificationPreferencesPayload, server.ValidateRegistration(nil, &key.PublicKey, nil))

	// Empty key
	require.Equal(t, ErrEmptyPushNotificationPreferencesPublicKey, server.ValidateRegistration(nil, nil, []byte("payload")))

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
	require.Equal(t, ErrCouldNotUnmarshalPushNotificationPreferences, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))

	// Version set to 0
	payload, err := proto.Marshal(&protobuf.PushNotificationPreferences{})
	require.NoError(t, err)

	cyphertext, err = encrypt(payload, sharedKey, rand.Reader)
	require.NoError(t, err)
	require.Equal(t, ErrInvalidPushNotificationPreferencesVersion, server.ValidateRegistration(nil, &key.PublicKey, cyphertext))
}
