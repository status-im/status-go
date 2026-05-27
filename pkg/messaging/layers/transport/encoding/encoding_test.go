package encoding

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"fmt"
	mrand "math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// encodeDecode round-trips a payload through Encode and Decode, mirroring the
// behaviour of the former EncodeWakuMessage/DecodeWakuMessage helpers.
func encodeDecode(version uint32, data []byte, keyInfo *KeyInfo) ([]byte, *DecodedPayload, error) {
	p := Payload{Data: data, Key: keyInfo}
	encoded, err := p.Encode(version)
	if err != nil {
		return nil, nil, err
	}
	decoded, err := Decode(version, encoded, keyInfo)
	return encoded, decoded, err
}

func TestEncodeDecodePayload(t *testing.T) {
	data := []byte{0, 1, 2}
	version := uint32(0)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = None

	encodedPayload, decodedPayload, err := encodeDecode(version, data, keyInfo)
	require.NoError(t, err)
	require.Equal(t, data, encodedPayload)
	require.Equal(t, data, decodedPayload.Data)
}

func TestEncodeDecodeVersion0(t *testing.T) {
	data := []byte{0, 1, 2}

	keyInfo := new(KeyInfo)
	keyInfo.Kind = None

	_, decoded, err := encodeDecode(0, data, keyInfo)
	require.NoError(t, err)
	require.Equal(t, data, decoded.Data)
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
	data := []byte{0, 1, 2}

	keyInfo := new(KeyInfo)
	keyInfo.Kind = Symmetric

	var err error
	keyInfo.SymKey, err = generateSymKey()
	require.NoError(t, err)

	encoded, decoded, err := encodeDecode(1, data, keyInfo)
	require.NoError(t, err)
	require.NotEqual(t, data, encoded)
	require.NotNil(t, encoded)
	require.Equal(t, data, decoded.Data)
}

func TestEncodeDecodeVersion1Asymmetric(t *testing.T) {
	data := []byte{0, 1, 2}

	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = Asymmetric
	keyInfo.PubKey = privKey.PublicKey

	p := Payload{Data: data, Key: keyInfo}
	encoded, err := p.Encode(1)
	require.NoError(t, err)
	require.NotEqual(t, data, encoded)
	require.NotNil(t, encoded)

	keyInfo.PrivKey = privKey
	decoded, err := Decode(1, encoded, keyInfo)
	require.NoError(t, err)
	require.Equal(t, data, decoded.Data)
}

func TestEncodeDecodeIncorrectKey(t *testing.T) {
	data := []byte{0, 1, 2}

	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	symKey, err := generateSymKey()
	require.NoError(t, err)

	keyInfo := new(KeyInfo)
	keyInfo.Kind = Asymmetric
	keyInfo.SymKey = symKey

	p := Payload{Data: data, Key: keyInfo}
	_, err = p.Encode(1)
	require.Error(t, err)

	keyInfo.SymKey = nil
	keyInfo.PrivKey = privKey

	p = Payload{Data: data, Key: keyInfo}
	_, err = p.Encode(1)
	require.Error(t, err)
}

func TestEncodeUnsupportedVersion(t *testing.T) {
	keyInfo := new(KeyInfo)
	keyInfo.Kind = None

	p := Payload{Data: []byte{0, 1, 2}, Key: keyInfo}
	_, err := p.Encode(99)
	require.Error(t, err)
	require.EqualError(t, err, "unsupported wakumessage version")
}

func TestDecodeUnsupportedVersion(t *testing.T) {
	keyInfo := new(KeyInfo)
	keyInfo.Kind = None

	decodedPayload, err := Decode(99, []byte{0, 1, 2}, keyInfo)

	require.Nil(t, decodedPayload)
	require.Error(t, err)
	require.EqualError(t, err, "unsupported wakumessage version")
}

func singleMessageTest(t *testing.T, symmetric bool) {
	data := []byte{0, 1, 2}

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

	p := Payload{Data: data, Key: keyInfo}
	encoded, err := p.Encode(1)
	require.NoError(t, err)

	var decryptedPayload []byte
	if symmetric {
		decryptedPayload, err = decryptSymmetric(encoded, keyInfo.SymKey)
		require.NoError(t, err)
	} else {
		decryptedPayload, err = decryptAsymmetric(encoded, keyInfo.PrivKey)
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
	keyInfo := new(KeyInfo)
	keyInfo.Kind = Symmetric
	keyInfo.SymKey = make([]byte, aesKeyLength)

	p := Payload{Data: []byte{0, 1, 2}, Key: keyInfo}
	_, err := p.Encode(1)
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
