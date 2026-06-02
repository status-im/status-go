package rfc26

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

func generateSymKey() ([]byte, error) {
	key, err := generateSecureRandomData(aesKeyLength)
	if err != nil {
		return nil, err
	} else if !validateDataIntegrity(key, aesKeyLength) {
		return nil, fmt.Errorf("error in generating symkey: crypto/rand failed to generate random data")
	}

	return key, nil
}

func TestEncodeDecodeSymmetric(t *testing.T) {
	data := []byte{0, 1, 2}

	symKey, err := generateSymKey()
	require.NoError(t, err)

	encoded, err := Encode(data, symKey, nil, nil)
	require.NoError(t, err)
	require.NotEqual(t, data, encoded)
	require.NotNil(t, encoded)

	decoded, err := Decode(encoded, &KeyInfo{Kind: Symmetric, SymKey: symKey})
	require.NoError(t, err)
	require.Equal(t, data, decoded.Data)
	require.Nil(t, decoded.PubKey)
}

func TestEncodeDecodeSymmetricSigned(t *testing.T) {
	data := []byte{0, 1, 2}

	symKey, err := generateSymKey()
	require.NoError(t, err)

	sigKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	encoded, err := Encode(data, symKey, nil, sigKey)
	require.NoError(t, err)

	decoded, err := Decode(encoded, &KeyInfo{Kind: Symmetric, SymKey: symKey})
	require.NoError(t, err)
	require.Equal(t, data, decoded.Data)
	require.NotNil(t, decoded.PubKey)
	require.Equal(t, sigKey.PublicKey, *decoded.PubKey)
}

func TestEncodeDecodeAsymmetric(t *testing.T) {
	data := []byte{0, 1, 2}

	recipient, err := crypto.GenerateKey()
	require.NoError(t, err)

	sigKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	encoded, err := Encode(data, nil, &recipient.PublicKey, sigKey)
	require.NoError(t, err)
	require.NotEqual(t, data, encoded)
	require.NotNil(t, encoded)

	decoded, err := Decode(encoded, &KeyInfo{Kind: Asymmetric, PrivKey: recipient})
	require.NoError(t, err)
	require.Equal(t, data, decoded.Data)
	require.Equal(t, sigKey.PublicKey, *decoded.PubKey)
}

func TestEncodeRequiresExactlyOneKey(t *testing.T) {
	data := []byte{0, 1, 2}

	// Both nil.
	_, err := Encode(data, nil, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one")

	// Both set.
	symKey, err := generateSymKey()
	require.NoError(t, err)
	recipient, err := crypto.GenerateKey()
	require.NoError(t, err)

	_, err = Encode(data, symKey, &recipient.PublicKey, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one")
}

func TestDecodeRequiresKey(t *testing.T) {
	_, err := Decode([]byte{0, 1, 2}, &KeyInfo{Kind: Symmetric})
	require.Error(t, err)

	_, err = Decode([]byte{0, 1, 2}, &KeyInfo{Kind: Asymmetric})
	require.Error(t, err)

	_, err = Decode([]byte{0, 1, 2}, &KeyInfo{Kind: None})
	require.Error(t, err)
}

func singleMessageTest(t *testing.T, symmetric bool) {
	data := []byte{0, 1, 2}

	// The signer is also the recipient: we simulate 'sending' to ourselves.
	sigKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	var symKey []byte
	var encoded []byte
	if symmetric {
		symKey, err = generateSymKey()
		require.NoError(t, err)
		encoded, err = Encode(data, symKey, nil, sigKey)
	} else {
		encoded, err = Encode(data, nil, &sigKey.PublicKey, sigKey)
	}
	require.NoError(t, err)

	var decryptedPayload []byte
	if symmetric {
		decryptedPayload, err = decryptSymmetric(encoded, symKey)
	} else {
		decryptedPayload, err = decryptAsymmetric(encoded, sigKey)
	}
	require.NoError(t, err)

	decodedPayload, err := validateAndParse(decryptedPayload)
	require.NoError(t, err)
	require.Equal(t, data, decodedPayload.Data)

	require.True(t, isMessageSigned(decryptedPayload[0]))
	require.Len(t, decodedPayload.Signature, signatureLength)
	require.Equal(t, sigKey.PublicKey, *decodedPayload.PubKey)
}

func TestMessageEncryption(t *testing.T) {
	var symmetric bool
	for i := 0; i < 256; i++ {
		singleMessageTest(t, symmetric)
		symmetric = !symmetric
	}
}

func TestEncryptWithZeroKey(t *testing.T) {
	_, err := Encode([]byte{0, 1, 2}, make([]byte, aesKeyLength), nil, nil)
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

	encodedPayload, err := p.encode()
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
