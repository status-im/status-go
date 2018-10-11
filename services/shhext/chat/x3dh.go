package chat

import (
	"crypto/ecdsa"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/golang/protobuf/proto"
)

const (
	// Shared secret key length
	sskLen = 16
)

// ToBase64 returns a Base64 encoding representation of the protobuf Bundle message
func (bundle *Bundle) ToBase64() (string, error) {
	marshaledMessage, err := proto.Marshal(bundle)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(marshaledMessage), nil
}

// FromBase64 unmarshals a Bundle from a Base64 encoding representation of the protobuf Bundle message
func FromBase64(str string) (*Bundle, error) {
	bundle := &Bundle{}

	decodedBundle, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}

	if err := proto.Unmarshal(decodedBundle, bundle); err != nil {
		return nil, err
	}

	return bundle, nil
}

func buildSignatureMaterial(signedPreKeys *map[string]*SignedPreKey) []byte {
	var keys []string
	for k := range *signedPreKeys {
		keys = append(keys, k)
	}
	var signatureMaterial []byte

	sort.Strings(keys)

	for _, installationID := range keys {
		signedPreKey := (*signedPreKeys)[installationID]
		signatureMaterial = append(signatureMaterial, []byte(installationID)...)
		signatureMaterial = append(signatureMaterial, signedPreKey.SignedPreKey...)
		signatureMaterial = append(signatureMaterial, []byte(fmt.Sprint(signedPreKey.Version))...)
	}
	return signatureMaterial

}

func SignBundle(identity *ecdsa.PrivateKey, bundleContainer *BundleContainer) error {
	signedPreKeys := bundleContainer.GetBundle().GetSignedPreKeys()
	signatureMaterial := buildSignatureMaterial(&signedPreKeys)

	signature, err := crypto.Sign(crypto.Keccak256(signatureMaterial), identity)
	if err != nil {
		return err
	}
	bundleContainer.Bundle.Signature = signature
	return nil
}

// NewBundleContainer creates a new BundleContainer from an identity private key
func NewBundleContainer(identity *ecdsa.PrivateKey, installationID string) (*BundleContainer, error) {
	preKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	compressedPreKey := crypto.CompressPubkey(&preKey.PublicKey)
	compressedIdentityKey := crypto.CompressPubkey(&identity.PublicKey)

	encodedPreKey := crypto.FromECDSA(preKey)
	signedPreKeys := make(map[string]*SignedPreKey)
	signedPreKeys[installationID] = &SignedPreKey{SignedPreKey: compressedPreKey}

	bundle := Bundle{
		Identity:      compressedIdentityKey,
		SignedPreKeys: signedPreKeys,
	}

	return &BundleContainer{
		Bundle:              &bundle,
		PrivateSignedPreKey: encodedPreKey,
	}, nil
}

// VerifyBundle checks that a bundle is valid
func VerifyBundle(bundle *Bundle) error {
	_, err := ExtractIdentity(bundle)
	return err
}

// ExtractIdentity extracts the identity key from a given bundle
func ExtractIdentity(bundle *Bundle) (string, error) {
	bundleIdentityKey, err := crypto.DecompressPubkey(bundle.GetIdentity())
	if err != nil {
		return "", err
	}

	signedPreKeys := bundle.GetSignedPreKeys()
	signatureMaterial := buildSignatureMaterial(&signedPreKeys)

	recoveredKey, err := crypto.SigToPub(
		crypto.Keccak256(signatureMaterial),
		bundle.GetSignature(),
	)
	if err != nil {
		return "", err
	}

	if crypto.PubkeyToAddress(*recoveredKey) != crypto.PubkeyToAddress(*bundleIdentityKey) {
		return "", errors.New("identity key and signature mismatch")
	}

	return fmt.Sprintf("0x%x", crypto.FromECDSAPub(recoveredKey)), nil
}

// PerformDH generates a shared key given a private and a public key
func PerformDH(privateKey *ecies.PrivateKey, publicKey *ecies.PublicKey) ([]byte, error) {
	return privateKey.GenerateShared(
		publicKey,
		sskLen,
		sskLen,
	)
}

func getSharedSecret(dh1 []byte, dh2 []byte, dh3 []byte) []byte {
	secretInput := append(append(dh1, dh2...), dh3...)

	return crypto.Keccak256(secretInput)
}

// x3dhActive handles initiating an X3DH session
func x3dhActive(
	myIdentityKey *ecies.PrivateKey,
	theirSignedPreKey *ecies.PublicKey,
	myEphemeralKey *ecies.PrivateKey,
	theirIdentityKey *ecies.PublicKey,
) ([]byte, error) {
	var dh1, dh2, dh3 []byte
	var err error

	if dh1, err = PerformDH(myIdentityKey, theirSignedPreKey); err != nil {
		return nil, err
	}

	if dh2, err = PerformDH(myEphemeralKey, theirIdentityKey); err != nil {
		return nil, err
	}

	if dh3, err = PerformDH(myEphemeralKey, theirSignedPreKey); err != nil {
		return nil, err
	}

	return getSharedSecret(dh1, dh2, dh3), nil
}

// x3dhPassive handles the response to an initiated X3DH session
func x3dhPassive(
	theirIdentityKey *ecies.PublicKey,
	mySignedPreKey *ecies.PrivateKey,
	theirEphemeralKey *ecies.PublicKey,
	myIdentityKey *ecies.PrivateKey,
) ([]byte, error) {
	var dh1, dh2, dh3 []byte
	var err error

	if dh1, err = PerformDH(mySignedPreKey, theirIdentityKey); err != nil {
		return nil, err
	}

	if dh2, err = PerformDH(myIdentityKey, theirEphemeralKey); err != nil {
		return nil, err
	}

	if dh3, err = PerformDH(mySignedPreKey, theirEphemeralKey); err != nil {
		return nil, err
	}

	return getSharedSecret(dh1, dh2, dh3), nil
}

// PerformActiveDH performs a Diffie-Hellman exchange using a public key and a generated ephemeral key.
// Returns the key resulting from the DH exchange as well as the ephemeral public key.
func PerformActiveDH(publicKey *ecdsa.PublicKey) ([]byte, *ecdsa.PublicKey, error) {
	ephemeralKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	key, err := PerformDH(
		ecies.ImportECDSA(ephemeralKey),
		ecies.ImportECDSAPublic(publicKey),
	)
	if err != nil {
		return nil, nil, err
	}

	return key, &ephemeralKey.PublicKey, err
}

// PerformActiveX3DH takes someone else's bundle and calculates shared secret.
// Returns the shared secret and the ephemeral key used.
func PerformActiveX3DH(identity []byte, signedPreKey []byte, prv *ecdsa.PrivateKey) ([]byte, *ecdsa.PublicKey, error) {
	bundleIdentityKey, err := crypto.DecompressPubkey(identity)
	if err != nil {
		return nil, nil, err
	}

	bundleSignedPreKey, err := crypto.DecompressPubkey(signedPreKey)
	if err != nil {
		return nil, nil, err
	}

	ephemeralKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	sharedSecret, err := x3dhActive(
		ecies.ImportECDSA(prv),
		ecies.ImportECDSAPublic(bundleSignedPreKey),
		ecies.ImportECDSA(ephemeralKey),
		ecies.ImportECDSAPublic(bundleIdentityKey),
	)
	if err != nil {
		return nil, nil, err
	}

	return sharedSecret, &ephemeralKey.PublicKey, nil
}

// PerformPassiveX3DH handles the part of the protocol where
// our interlocutor used our bundle, with ID of the signedPreKey,
// we loaded our identity key and the correct signedPreKey and we perform X3DH
func PerformPassiveX3DH(theirIdentityKey *ecdsa.PublicKey, mySignedPreKey *ecdsa.PrivateKey, theirEphemeralKey *ecdsa.PublicKey, myPrivateKey *ecdsa.PrivateKey) ([]byte, error) {
	sharedSecret, err := x3dhPassive(
		ecies.ImportECDSAPublic(theirIdentityKey),
		ecies.ImportECDSA(mySignedPreKey),
		ecies.ImportECDSAPublic(theirEphemeralKey),
		ecies.ImportECDSA(myPrivateKey),
	)
	if err != nil {
		return nil, err
	}

	return sharedSecret, nil
}
