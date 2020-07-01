package push_notification_client

import (
	"bytes"
	"crypto/ecdsa"
	"math/rand"

	"testing"

	"github.com/google/uuid"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/stretchr/testify/require"
)

func TestBuildPushNotificationRegisterMessage(t *testing.T) {
	myDeviceToken := "device-token"
	myInstallationID := "installationID"
	mutedChatList := []string{"a", "b"}

	// build chat lish hashes
	var mutedChatListHashes [][]byte
	for _, chatID := range mutedChatList {
		mutedChatListHashes = append(mutedChatListHashes, shake256(chatID))
	}

	identity, err := crypto.GenerateKey()
	contactKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	contactIDs := []*ecdsa.PublicKey{&contactKey.PublicKey}

	// Set random generator for uuid
	var seed int64 = 1
	uuid.SetRand(rand.New(rand.NewSource(seed)))

	// Get token
	expectedUUID := uuid.New().String()

	// set up reader
	reader := bytes.NewReader([]byte(expectedUUID))

	sharedKey, err := ecies.ImportECDSA(identity).GenerateShared(
		ecies.ImportECDSAPublic(&contactKey.PublicKey),
		accessTokenKeyLength,
		accessTokenKeyLength,
	)
	require.NoError(t, err)
	// build encrypted token
	encryptedToken, err := encryptAccessToken([]byte(expectedUUID), sharedKey, reader)
	require.NoError(t, err)

	// Reset random generator
	uuid.SetRand(rand.New(rand.NewSource(seed)))

	config := &Config{
		Identity:                   identity,
		RemoteNotificationsEnabled: true,
		MutedChatIDs:               mutedChatList,
		ContactIDs:                 contactIDs,
		InstallationID:             myInstallationID,
	}

	client := &Client{}
	client.config = config
	client.DeviceToken = myDeviceToken
	// Set reader
	client.reader = bytes.NewReader([]byte(expectedUUID))

	options := &protobuf.PushNotificationOptions{
		Version:         1,
		AccessToken:     expectedUUID,
		Token:           myDeviceToken,
		InstallationId:  myInstallationID,
		Enabled:         true,
		BlockedChatList: mutedChatListHashes,
		AllowedUserList: [][]byte{encryptedToken},
	}

	actualMessage, err := client.buildPushNotificationOptionsMessage()
	require.NoError(t, err)

	require.Equal(t, options, actualMessage)
}
