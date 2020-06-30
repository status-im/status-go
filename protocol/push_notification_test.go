package protocol

import (
	"bytes"
	"crypto/ecdsa"
	"math/rand"

	"testing"

	"github.com/golang/protobuf/proto"
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

	tokenPair := &protobuf.PushNotificationTokenPair{
		Token:     encryptedToken,
		PublicKey: crypto.CompressPubkey(&contactKey.PublicKey),
	}

	// Reset random generator
	uuid.SetRand(rand.New(rand.NewSource(seed)))

	config := &PushNotificationConfig{
		Identity:                   identity,
		RemoteNotificationsEnabled: true,
		MutedChatIDs:               mutedChatList,
		ContactIDs:                 contactIDs,
		InstallationID:             myInstallationID,
	}

	service := &PushNotificationService{}
	service.config = config
	service.DeviceToken = myDeviceToken
	// Set reader
	service.reader = bytes.NewReader([]byte(expectedUUID))

	options := &protobuf.PushNotificationOptions{
		Token:           myDeviceToken,
		InstallationId:  myInstallationID,
		Enabled:         true,
		BlockedChatList: mutedChatListHashes,
		AllowedUserList: []*protobuf.PushNotificationTokenPair{tokenPair},
	}

	preferences := &protobuf.PushNotificationPreferences{
		Options:     []*protobuf.PushNotificationOptions{options},
		Version:     1,
		AccessToken: expectedUUID,
	}

	// Marshal message
	marshaledPreferences, err := proto.Marshal(preferences)
	require.NoError(t, err)

	expectedMessage := &protobuf.PushNotificationRegister{Payload: marshaledPreferences}
	actualMessage, err := service.buildPushNotificationRegisterMessage()
	require.NoError(t, err)

	require.Equal(t, expectedMessage, actualMessage)
}

func TestBuildPushNotificationRegisterMessageWithPrevious(t *testing.T) {
	deviceToken1 := "device-token-1"
	deviceToken2 := "device-token-2"
	installationID1 := "installationID-1"
	installationID2 := "installationID-2"

	contactKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	// build previous push notification options
	options2 := &protobuf.PushNotificationOptions{
		Token:          deviceToken2,
		InstallationId: installationID2,
		Enabled:        true,
		AllowedUserList: []*protobuf.PushNotificationTokenPair{{
			Token:     []byte{0x01},
			PublicKey: crypto.CompressPubkey(&contactKey.PublicKey),
		}},
	}

	preferences2 := &protobuf.PushNotificationPreferences{
		Options:     []*protobuf.PushNotificationOptions{options2},
		Version:     1,
		AccessToken: "some-token",
	}

	// Marshal message
	marshaledPreferences2, err := proto.Marshal(preferences2)
	require.NoError(t, err)

	lastPushNotificationRegister := &protobuf.PushNotificationRegister{Payload: marshaledPreferences2}

	mutedChatList := []string{"a", "b"}

	// build chat lish hashes
	var mutedChatListHashes [][]byte
	for _, chatID := range mutedChatList {
		mutedChatListHashes = append(mutedChatListHashes, shake256(chatID))
	}

	identity, err := crypto.GenerateKey()
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
	encryptedToken1, err := encryptAccessToken([]byte(expectedUUID), sharedKey, reader)
	require.NoError(t, err)

	encryptedToken2, err := encryptAccessToken([]byte(expectedUUID), sharedKey, reader)
	require.NoError(t, err)

	tokenPair1 := &protobuf.PushNotificationTokenPair{
		Token:     encryptedToken1,
		PublicKey: crypto.CompressPubkey(&contactKey.PublicKey),
	}

	tokenPair2 := &protobuf.PushNotificationTokenPair{
		Token:     encryptedToken2,
		PublicKey: crypto.CompressPubkey(&contactKey.PublicKey),
	}

	// Reset random generator
	uuid.SetRand(rand.New(rand.NewSource(seed)))

	config := &PushNotificationConfig{
		Identity:                   identity,
		RemoteNotificationsEnabled: true,
		MutedChatIDs:               mutedChatList,
		ContactIDs:                 contactIDs,
		InstallationID:             installationID1,
	}

	service := &PushNotificationService{}
	service.config = config
	service.DeviceToken = deviceToken1
	service.lastPushNotificationRegister = lastPushNotificationRegister
	// Set reader
	service.reader = bytes.NewReader([]byte(expectedUUID))

	options1 := &protobuf.PushNotificationOptions{
		Token:           deviceToken1,
		InstallationId:  installationID1,
		Enabled:         true,
		BlockedChatList: mutedChatListHashes,
		AllowedUserList: []*protobuf.PushNotificationTokenPair{tokenPair1},
	}
	options2.AllowedUserList = []*protobuf.PushNotificationTokenPair{tokenPair2}

	preferences := &protobuf.PushNotificationPreferences{
		Options:     []*protobuf.PushNotificationOptions{options1, options2},
		Version:     2,
		AccessToken: expectedUUID,
	}

	// Marshal message
	marshaledPreferences, err := proto.Marshal(preferences)
	require.NoError(t, err)

	expectedMessage := &protobuf.PushNotificationRegister{Payload: marshaledPreferences}
	actualMessage, err := service.buildPushNotificationRegisterMessage()
	require.NoError(t, err)

	require.Equal(t, expectedMessage, actualMessage)
}
