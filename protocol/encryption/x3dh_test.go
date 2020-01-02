package encryption

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/crypto/ecies"
)

const (
	alicePrivateKey   = "00000000000000000000000000000000"
	aliceEphemeralKey = "11111111111111111111111111111111"
	bobPrivateKey     = "22222222222222222222222222222222"
	bobSignedPreKey   = "33333333333333333333333333333333"
)

var sharedKey = []byte{0xa4, 0xe9, 0x23, 0xd0, 0xaf, 0x8f, 0xe7, 0x8a, 0x5, 0x63, 0x63, 0xbe, 0x20, 0xe7, 0x1c, 0xa, 0x58, 0xe5, 0x69, 0xea, 0x8f, 0xc1, 0xf7, 0x92, 0x89, 0xec, 0xa1, 0xd, 0x9f, 0x68, 0x13, 0x3a}

func bobBundle() (*Bundle, error) {
	privateKey, err := crypto.ToECDSA([]byte(bobPrivateKey))
	if err != nil {
		return nil, err
	}

	signedPreKey, err := crypto.ToECDSA([]byte(bobSignedPreKey))
	if err != nil {
		return nil, err
	}

	compressedPreKey := crypto.CompressPubkey(&signedPreKey.PublicKey)

	signature, err := crypto.Sign(crypto.Keccak256(compressedPreKey), privateKey)
	if err != nil {
		return nil, err
	}

	signedPreKeys := make(map[string]*SignedPreKey)
	signedPreKeys[bobInstallationID] = &SignedPreKey{SignedPreKey: compressedPreKey}

	bundle := Bundle{
		Identity:      crypto.CompressPubkey(&privateKey.PublicKey),
		SignedPreKeys: signedPreKeys,
		Signature:     signature,
	}

	return &bundle, nil
}

func TestNewBundleContainer(t *testing.T) {
	privateKey, err := crypto.ToECDSA([]byte(alicePrivateKey))
	require.NoError(t, err, "Private key should be generated without errors")

	bundleContainer, err := NewBundleContainer(privateKey, bobInstallationID)
	require.NoError(t, err, "Bundle container should be created successfully")

	err = SignBundle(privateKey, bundleContainer)
	require.NoError(t, err, "Bundle container should be signed successfully")

	require.NoError(t, err, "Bundle container should be created successfully")

	bundle := bundleContainer.Bundle
	require.NotNil(t, bundle, "Bundle should be generated without errors")

	signatureMaterial := append([]byte(bobInstallationID), bundle.GetSignedPreKeys()[bobInstallationID].GetSignedPreKey()...)
	signatureMaterial = append(signatureMaterial, []byte("0")...)
	signatureMaterial = append(signatureMaterial, []byte(fmt.Sprint(bundle.GetTimestamp()))...)
	recoveredPublicKey, err := crypto.SigToPub(
		crypto.Keccak256(signatureMaterial),
		bundle.Signature,
	)

	require.NoError(t, err, "Public key should be recovered from the bundle successfully")

	require.Equal(
		t,
		privateKey.PublicKey,
		*recoveredPublicKey,
		"The correct public key should be recovered",
	)
}

func TestSignBundle(t *testing.T) {
	privateKey, err := crypto.ToECDSA([]byte(alicePrivateKey))
	require.NoError(t, err, "Private key should be generated without errors")

	bundleContainer1, err := NewBundleContainer(privateKey, "1")
	require.NoError(t, err, "Bundle container should be created successfully")

	bundle1 := bundleContainer1.Bundle
	require.NotNil(t, bundle1, "Bundle should be generated without errors")

	// We add a signed pre key
	signedPreKeys := bundle1.GetSignedPreKeys()
	signedPreKeys["2"] = &SignedPreKey{SignedPreKey: []byte("key")}

	err = SignBundle(privateKey, bundleContainer1)
	require.NoError(t, err)

	signatureMaterial := append([]byte("1"), bundle1.GetSignedPreKeys()["1"].GetSignedPreKey()...)
	signatureMaterial = append(signatureMaterial, []byte("0")...)
	signatureMaterial = append(signatureMaterial, []byte("2")...)
	signatureMaterial = append(signatureMaterial, []byte("key")...)
	signatureMaterial = append(signatureMaterial, []byte("0")...)
	signatureMaterial = append(signatureMaterial, []byte(fmt.Sprint(bundle1.GetTimestamp()))...)

	recoveredPublicKey, err := crypto.SigToPub(
		crypto.Keccak256(signatureMaterial),
		bundleContainer1.GetBundle().Signature,
	)

	require.NoError(t, err, "Public key should be recovered from the bundle successfully")

	require.Equal(
		t,
		privateKey.PublicKey,
		*recoveredPublicKey,
		"The correct public key should be recovered",
	)
}

func TestExtractIdentity(t *testing.T) {
	privateKey, err := crypto.ToECDSA([]byte(alicePrivateKey))
	require.NoError(t, err, "Private key should be generated without errors")

	bundleContainer, err := NewBundleContainer(privateKey, "1")
	require.NoError(t, err, "Bundle container should be created successfully")

	err = SignBundle(privateKey, bundleContainer)
	require.NoError(t, err, "Bundle container should be signed successfully")

	bundle := bundleContainer.Bundle
	require.NotNil(t, bundle, "Bundle should be generated without errors")

	recoveredPublicKey, err := ExtractIdentity(bundle)

	require.NoError(t, err, "Public key should be recovered from the bundle successfully")

	require.Equal(
		t,
		privateKey.PublicKey,
		*recoveredPublicKey,
		"The correct public key should be recovered",
	)
}

// Alice wants to send a message to Bob
func TestX3dhActive(t *testing.T) {
	bobIdentityKey, err := crypto.ToECDSA([]byte(bobPrivateKey))
	require.NoError(t, err, "Bundle identity key should be generated without errors")

	bobSignedPreKey, err := crypto.ToECDSA([]byte(bobSignedPreKey))
	require.NoError(t, err, "Bundle signed pre key should be generated without errors")

	aliceIdentityKey, err := crypto.ToECDSA([]byte(alicePrivateKey))
	require.NoError(t, err, "Private key should be generated without errors")

	aliceEphemeralKey, err := crypto.ToECDSA([]byte(aliceEphemeralKey))
	require.NoError(t, err, "Ephemeral key should be generated without errors")

	x3dh, err := x3dhActive(
		ecies.ImportECDSA(aliceIdentityKey),
		ecies.ImportECDSAPublic(&bobSignedPreKey.PublicKey),
		ecies.ImportECDSA(aliceEphemeralKey),
		ecies.ImportECDSAPublic(&bobIdentityKey.PublicKey),
	)
	require.NoError(t, err, "Shared key should be generated without errors")
	require.Equal(t, sharedKey, x3dh, "Should generate the correct key")
}

// Bob receives a message from Alice
func TestPerformPassiveX3DH(t *testing.T) {
	alicePrivateKey, err := crypto.ToECDSA([]byte(alicePrivateKey))
	require.NoError(t, err, "Private key should be generated without errors")

	bobSignedPreKey, err := crypto.ToECDSA([]byte(bobSignedPreKey))
	require.NoError(t, err, "Private key should be generated without errors")

	aliceEphemeralKey, err := crypto.ToECDSA([]byte(aliceEphemeralKey))
	require.NoError(t, err, "Ephemeral key should be generated without errors")

	bobPrivateKey, err := crypto.ToECDSA([]byte(bobPrivateKey))
	require.NoError(t, err, "Private key should be generated without errors")

	x3dh, err := PerformPassiveX3DH(
		&alicePrivateKey.PublicKey,
		bobSignedPreKey,
		&aliceEphemeralKey.PublicKey,
		bobPrivateKey,
	)
	require.NoError(t, err, "Shared key should be generated without errors")
	require.Equal(t, sharedKey, x3dh, "Should generate the correct key")
}

func TestPerformActiveX3DH(t *testing.T) {
	bundle, err := bobBundle()
	require.NoError(t, err, "Test bundle should be generated without errors")

	privateKey, err := crypto.ToECDSA([]byte(bobPrivateKey))
	require.NoError(t, err, "Private key should be imported without errors")

	signedPreKey := bundle.GetSignedPreKeys()[bobInstallationID].GetSignedPreKey()

	actualSharedSecret, actualEphemeralKey, err := PerformActiveX3DH(bundle.GetIdentity(), signedPreKey, privateKey)
	require.NoError(t, err, "No error should be reported")
	require.NotNil(t, actualEphemeralKey, "An ephemeral key-pair should be generated")
	require.NotNil(t, actualSharedSecret, "A shared key should be generated")
}
