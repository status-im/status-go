package payload

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"google.golang.org/protobuf/proto"
)

func createTestMsg(version uint32) *pb.WakuMessage {
	message := new(pb.WakuMessage)
	message.Payload = []byte{0, 1, 2}
	message.Version = proto.Uint32(version)
	message.Timestamp = proto.Int64(123456)
	return message
}

func TestEncodeDecodePayload(t *testing.T) {
	data := []byte{0, 1, 2}
	version := uint32(0)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = None

	p := Payload{
		Data: data,
		Key:  keyInfo,
	}

	encodedPayload, err := p.Encode(version)
	require.NoError(t, err)
	require.Equal(t, data, encodedPayload)

	message := new(pb.WakuMessage)
	message.Payload = encodedPayload
	message.Version = proto.Uint32(version)
	message.Timestamp = proto.Int64(123456)

	decodedPayload, err := DecodePayload(message, keyInfo)
	require.NoError(t, err)
	require.Equal(t, data, decodedPayload.Data)
}

func TestEncodeDecodeVersion0(t *testing.T) {
	message := createTestMsg(0)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = None

	err := EncodeWakuMessage(message, keyInfo)
	require.NoError(t, err)

	err = DecodeWakuMessage(message, keyInfo)
	require.NoError(t, err)
}

func generateSymKey() ([]byte, error) {
	key, err := generateSecureRandomData(aesKeyLength)
	if err != nil {
		return nil, err
	} else if !validateDataIntegrity(key, aesKeyLength) {
		return nil, fmt.Errorf("error in generating symkey: crypto/rand failed to generate random data")
	}

	return key, nil
}

func TestEncodeDecodeVersion1Symmetric(t *testing.T) {
	message := createTestMsg(1)
	data := message.Payload

	keyInfo := new(KeyInfo)
	keyInfo.Kind = Symmetric

	var err error
	keyInfo.SymKey, err = generateSymKey()
	require.NoError(t, err)

	err = EncodeWakuMessage(message, keyInfo)
	require.NoError(t, err)
	require.NotEqual(t, data, message.Payload)
	require.NotNil(t, message.Payload)

	err = DecodeWakuMessage(message, keyInfo)
	require.NoError(t, err)
	require.Equal(t, data, message.Payload)
}

func TestEncodeDecodeVersion1Asymmetric(t *testing.T) {
	message := createTestMsg(1)
	data := message.Payload

	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = Asymmetric
	keyInfo.PubKey = privKey.PublicKey

	err = EncodeWakuMessage(message, keyInfo)
	require.NoError(t, err)
	require.NotEqual(t, data, message.Payload)
	require.NotNil(t, message.Payload)

	keyInfo.PrivKey = privKey
	err = DecodeWakuMessage(message, keyInfo)
	require.NoError(t, err)
	require.Equal(t, data, message.Payload)
}

func TestEncodeDecodeIncorrectKey(t *testing.T) {
	message := createTestMsg(1)

	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	symKey, err := generateSymKey()
	require.NoError(t, err)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = Asymmetric
	keyInfo.SymKey = symKey

	err = EncodeWakuMessage(message, keyInfo)
	require.Error(t, err)

	keyInfo.SymKey = nil
	keyInfo.PrivKey = privKey

	err = EncodeWakuMessage(message, keyInfo)
	require.Error(t, err)
}

func TestEncodeUnsupportedVersion(t *testing.T) {
	message := createTestMsg(99)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = None

	err := EncodeWakuMessage(message, keyInfo)
	require.Error(t, err)
	require.EqualError(t, err, "unsupported wakumessage version")
}

func TestDecodeUnsupportedVersion(t *testing.T) {
	message := createTestMsg(99)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = None

	decodedPayload, err := DecodePayload(message, keyInfo)

	require.Nil(t, decodedPayload)
	require.Error(t, err)
	require.EqualError(t, err, "unsupported wakumessage version")
}
