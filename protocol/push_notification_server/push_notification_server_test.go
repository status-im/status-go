package protocol

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	//nodecrypto "github.com/status-im/status-go/eth-node/crypto"
	//"github.com/status-im/status-go/protocol/protobuf"
)

func TestPushNotificationServerValidateRegistration(t *testing.T) {
	identity, err := crypto.GenerateKey()
	require.NoError(t, err)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	config := &Config{
		Identity: identity,
	}

	server := Server{config: config}

	// Empty payload
	require.Equal(t, ErrEmptyPushNotificationRegisterPayload, server.ValidateRegistration(nil, &key.PublicKey, nil))

	// Empty key
	require.Equal(t, ErrEmptyPushNotificationRegisterPublicKey, server.ValidateRegistration(nil, nil, []byte("payload")))

	/*
		// Invalid signature
		signature, err := nodecrypto.SignBytes([]byte("a"), key)

		require.Equal(t, ErrInvalidPushNotificationRegisterVersion, server.ValidateRegistration(nil, &protobuf.PushNotificationRegister{
			Payload:   []byte("btahtasht"),
			Signature: signature,
		}))*/

}
