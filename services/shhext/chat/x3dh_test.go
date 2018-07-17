package chat

import (
	"testing"

	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/stretchr/testify/assert"
)

const (
	alicePrivateKey   = "00000000000000000000000000000000"
	aliceEphemeralKey = "11111111111111111111111111111111"
	bobPrivateKey     = "22222222222222222222222222222222"
	bobSignedPreKey   = "33333333333333333333333333333333"

	jsonBundle          = "{\"identity\":\"ApCZnbv0MDS/+x3VPqwetMM6TqHE9Iulhc/eODCEDwVV\",\"signedPreKey\":\"AjxyrdtP3wmvlPDJTX/pKjhqfnDPih2FkWOGuyU1x7Gx\",\"signature\":\"P1ax7dSQmCjr/UfPFB8dxk0FowfSP7R7KV8F/WuimwtnvJkz3yT+oNDdlbm4ddDjOFjwTVDscPK2qbraTkkg9gA=\"}"
	jsonBundleContainer = "{\"bundle\":{\"identity\":\"ApCZnbv0MDS/+x3VPqwetMM6TqHE9Iulhc/eODCEDwVV\",\"signedPreKey\":\"AjxyrdtP3wmvlPDJTX/pKjhqfnDPih2FkWOGuyU1x7Gx\",\"signature\":\"P1ax7dSQmCjr/UfPFB8dxk0FowfSP7R7KV8F/WuimwtnvJkz3yT+oNDdlbm4ddDjOFjwTVDscPK2qbraTkkg9gA=\"},\"privateSignedPreKey\":\"MzAzMDMwMzAzMDMwMzAzMDMwMzAzMDMwMzAzMDMwMzAzMDMwMzAzMDMwMzAzMDMwMzAzMDMwMzAzMDMwMzAzMA==\"}"
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

	bundle := Bundle{
		Identity:     crypto.CompressPubkey(&privateKey.PublicKey),
		SignedPreKey: compressedPreKey,
		Signature:    signature,
	}

	return &bundle, nil
}

func TestNewBundleContainer(t *testing.T) {
	privateKey, err := crypto.ToECDSA([]byte(alicePrivateKey))

	assert.Nil(t, err, "Private key should be generated without errors")

	bundleContainer, err := NewBundleContainer(privateKey)
	assert.Nil(t, err, "Bundle container should be created successfully")

	bundle := bundleContainer.Bundle

	assert.Nil(t, err, "Bundle should be generated without errors")

	recoveredPublicKey, err := crypto.SigToPub(
		crypto.Keccak256(bundle.GetSignedPreKey()),
		bundle.Signature,
	)

	assert.Nil(t, err, "Public key should be recovered from the bundle successfully")

	assert.Equalf(
		t,
		&privateKey.PublicKey,
		recoveredPublicKey,
		"The correct public key should be recovered",
	)
}

func TestToJSON(t *testing.T) {
	privateKey, err := crypto.ToECDSA([]byte(alicePrivateKey))
	assert.Nil(t, err, "Key should be generated without errors")

	encodedKey := []byte(hex.EncodeToString(crypto.FromECDSA(privateKey)))

	bundle, err := bobBundle()
	assert.Nil(t, err, "Test bundle should be generated without errors")

	bundleContainer := BundleContainer{
		Bundle:              bundle,
		PrivateSignedPreKey: encodedKey,
	}

	actualJSONBundleContainer, err := bundleContainer.ToJSON()

	assert.Nil(t, err, "no error should be reported")

	assert.Equalf(
		t,
		jsonBundleContainer,
		actualJSONBundleContainer,
		"The correct bundle should be generated",
	)
}

func TestFromJSON(t *testing.T) {

	expectedBundle, err := bobBundle()
	assert.Nil(t, err, "Test bundle should be generated without errors")

	actualBundle, err := FromJSON(jsonBundle)

	assert.Nil(t, err, "Bundle should be unmarshaled without errors")

	assert.Equalf(
		t,
		expectedBundle,
		actualBundle,
		"The correct bundle should be generated",
	)
}

// Alice wants to send a message to Bob
func TestX3dhActive(t *testing.T) {

	bobIdentityKey, err := crypto.ToECDSA([]byte(bobPrivateKey))
	assert.Nil(t, err, "Bundle identity key should be generated without errors")

	bobSignedPreKey, err := crypto.ToECDSA([]byte(bobSignedPreKey))
	assert.Nil(t, err, "Bundle signed pre key should be generated without errors")

	aliceIdentityKey, err := crypto.ToECDSA([]byte(alicePrivateKey))
	assert.Nil(t, err, "private key should be generated without errors")

	aliceEphemeralKey, err := crypto.ToECDSA([]byte(aliceEphemeralKey))
	assert.Nil(t, err, "ephemeral key should be generated without errors")

	x3dh, err := x3dhActive(
		ecies.ImportECDSA(aliceIdentityKey),
		ecies.ImportECDSAPublic(&bobSignedPreKey.PublicKey),
		ecies.ImportECDSA(aliceEphemeralKey),
		ecies.ImportECDSAPublic(&bobIdentityKey.PublicKey),
	)

	assert.Nil(t, err, "Shared key should be generated without errors")
	assert.Equalf(t, sharedKey, x3dh, "Should generate the correct key")
}

// Bob receives a message from Alice
func TestPerformX3DHPassive(t *testing.T) {

	alicePrivateKey, err := crypto.ToECDSA([]byte(alicePrivateKey))
	assert.Nil(t, err, "Private key should be generated without errors")

	bobSignedPreKey, err := crypto.ToECDSA([]byte(bobSignedPreKey))
	assert.Nil(t, err, "Private key should be generated without errors")

	aliceEphemeralKey, err := crypto.ToECDSA([]byte(aliceEphemeralKey))
	assert.Nil(t, err, "ephemeral key should be generated without errors")

	bobPrivateKey, err := crypto.ToECDSA([]byte(bobPrivateKey))
	assert.Nil(t, err, "Private key should be generated without errors")

	x3dh, err := PerformPassiveX3DH(
		&alicePrivateKey.PublicKey,
		bobSignedPreKey,
		&aliceEphemeralKey.PublicKey,
		bobPrivateKey,
	)

	assert.Nil(t, err, "Shared key should be generated without errors")
	assert.Equalf(t, sharedKey, x3dh, "Should generate the correct key")
}

func TestPerformActiveX3DH(t *testing.T) {
	bundle, err := bobBundle()

	assert.Nil(t, err, "Test bundle should be generated without errors")

	privateKey, err := crypto.ToECDSA([]byte(bobPrivateKey))

	assert.Nil(t, err, "Private key should be imported without errors")

	actualSharedSecret, actualEphemeralKey, err := PerformActiveX3DH(bundle, privateKey)

	assert.Nil(t, err, "no error should be reported")
	assert.NotNil(t, actualEphemeralKey, "An ephemeral key-pair should be generated")
	assert.NotNil(t, actualSharedSecret, "A shared key should be generated")
}
