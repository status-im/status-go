package payload

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	mrand "math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func singleMessageTest(t *testing.T, symmetric bool) {
	message := createTestMsg(1)
	data := message.Payload

	var err error

	keyInfo := new(KeyInfo)
	keyInfo.PrivKey, err = crypto.GenerateKey()
	require.NoError(t, err)
	if symmetric {
		keyInfo.Kind = Symmetric
		keyInfo.SymKey, err = generateSymKey()
		require.NoError(t, err)
	} else {
		keyInfo.Kind = Asymmetric
		keyInfo.PubKey = keyInfo.PrivKey.PublicKey // We'll simulate 'sending' a message to ourselves
	}

	err = EncodeWakuMessage(message, keyInfo)
	require.NoError(t, err)

	var decryptedPayload []byte
	if symmetric {
		decryptedPayload, err = decryptSymmetric(message.Payload, keyInfo.SymKey)
		require.NoError(t, err)
	} else {
		decryptedPayload, err = decryptAsymmetric(message.Payload, keyInfo.PrivKey)
		require.NoError(t, err)
	}

	decodedPayload, err := validateAndParse(decryptedPayload)
	require.NoError(t, err)
	require.Equal(t, data, decodedPayload.Data)

	require.True(t, isMessageSigned(decryptedPayload[0]))
	require.Len(t, decodedPayload.Signature, signatureLength)
	require.Equal(t, keyInfo.PrivKey.PublicKey, *decodedPayload.PubKey)
}

func TestMessageEncryption(t *testing.T) {
	var symmetric bool
	for i := 0; i < 256; i++ {
		singleMessageTest(t, symmetric)
		symmetric = !symmetric
	}
}

func TestEncryptWithZeroKey(t *testing.T) {
	message := createTestMsg(1)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = Symmetric
	keyInfo.SymKey = make([]byte, aesKeyLength)

	err := EncodeWakuMessage(message, keyInfo)
	require.Error(t, err)
	require.EqualError(t, err, "couldn't encrypt using symmetric key: invalid key provided for symmetric encryption, size: 32")
}

func singlePaddingTest(t *testing.T, padSize int) {
	var err error

	keyInfo := new(KeyInfo)
	keyInfo.Kind = Symmetric
	keyInfo.SymKey, err = generateSymKey()
	require.NoError(t, err)

	p := Payload{
		Data:    []byte{0, 1, 2},
		Padding: make([]byte, padSize),
		Key:     keyInfo,
	}

	_, err = crand.Read(p.Padding) // nolint: gosec
	require.NoError(t, err)

	encodedPayload, err := p.Encode(1)
	require.NoError(t, err)

	decodedData, err := decryptSymmetric(encodedPayload, keyInfo.SymKey)
	require.NoError(t, err)

	decodedPayload, err := validateAndParse(decodedData)
	require.NoError(t, err)

	require.Equal(t, p.Padding, decodedPayload.Padding)
}

func TestPadding(t *testing.T) {
	for i := 1; i < 260; i++ {
		singlePaddingTest(t, i)
	}

	lim := 256 * 256
	for i := lim - 5; i < lim+2; i++ {
		singlePaddingTest(t, i)
	}

	for i := 0; i < 256; i++ {
		n := mrand.Intn(256*254) + 256 // nolint: gosec
		singlePaddingTest(t, n)
	}

	for i := 0; i < 256; i++ {
		n := mrand.Intn(256*1024) + 256*256 // nolint: gosec
		singlePaddingTest(t, n)
	}
}

func TestPaddingAppendedToSymMessagesWithSignature(t *testing.T) {
	pSrc, err := crypto.GenerateKey()
	require.NoError(t, err)

	p := Payload{
		Data: make([]byte, 246),
		Key: &KeyInfo{
			Kind:    Symmetric,
			SymKey:  make([]byte, aesKeyLength),
			PrivKey: pSrc,
		},
	}

	// Simulate a message with a payload just under 256 so that
	// payload + flag + signature > 256. Check that the result
	// is padded on the next 256 boundary.
	const payloadSizeFieldMinSize = 1
	rawMessage := make([]byte, flagsLength+payloadSizeFieldMinSize+len(p.Data))
	rawMessage, err = p.appendPadding(rawMessage)

	require.NoError(t, err)
	require.Equal(t, 512-signatureLength, len(rawMessage))
}

func TestAesNonce(t *testing.T) {
	key := hexutil.MustDecode("0x03ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd31")
	block, err := aes.NewCipher(key)
	require.NoError(t, err)

	aesgcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	require.Equal(t, aesgcm.NonceSize(), aesNonceLength)
}
