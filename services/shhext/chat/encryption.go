package chat

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	ecrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/log"
	dr "github.com/status-im/doubleratchet"

	"github.com/status-im/status-go/services/shhext/chat/crypto"
)

var ErrKeyNotFound = errors.New("Key not found")

type EncryptionService struct {
	log         log.Logger
	persistence PersistenceServiceInterface
}

func NewEncryptionService(p PersistenceServiceInterface) *EncryptionService {
	return &EncryptionService{
		log:         log.New("package", "status-go/services/sshext.chat"),
		persistence: p,
	}
}

func (s *EncryptionService) keyFromX3DH(theirPublicKey *ecdsa.PublicKey, myIdentityKey *ecdsa.PrivateKey) ([]byte, []byte, *ecdsa.PublicKey, error) {

	bundle, err := s.persistence.GetPublicBundle(theirPublicKey)

	if err != nil {
		return nil, nil, nil, err
	}

	if bundle == nil {
		return nil, nil, nil, nil
	}

	key, ephemeralKey, err := PerformActiveX3DH(bundle, myIdentityKey)
	if err != nil {
		return nil, nil, nil, err
	}

	return key, bundle.GetSignedPreKey(), ephemeralKey, nil
}

func (s *EncryptionService) CreateBundle(privateKey *ecdsa.PrivateKey) (*Bundle, error) {
	bundle, err := s.persistence.GetAnyPrivateBundle()
	if err != nil {
		return nil, err
	}

	if bundle != nil {
		return bundle, nil
	}

	// needs transaction/mutex to avoid creating multiple bundles
	// although not a problem
	bundleContainer, err := NewBundleContainer(privateKey)
	if err != nil {
		return nil, err
	}

	err = s.persistence.AddPrivateBundle(bundleContainer)
	if err != nil {
		return nil, err
	}

	return bundleContainer.GetBundle(), nil
}

func (s *EncryptionService) DecryptSymmetricPayload(src *ecdsa.PublicKey, ephemeralKey *ecdsa.PublicKey, payload []byte) ([]byte, error) {

	symmetricKey, err := s.persistence.GetSymmetricKey(src, ephemeralKey)
	if err != nil {
		return nil, err
	}

	if symmetricKey == nil {
		return nil, ErrKeyNotFound
	}

	return crypto.DecryptSymmetric(symmetricKey, payload)
}

// Decrypt message sent with a DH key exchange, throw away the key after decryption
func (s *EncryptionService) DecryptWithDH(myIdentityKey *ecdsa.PrivateKey, theirEphemeralKey *ecdsa.PublicKey, payload []byte) ([]byte, error) {
	key, err := PerformDH(
		ecies.ImportECDSA(myIdentityKey),
		ecies.ImportECDSAPublic(theirEphemeralKey),
	)
	if err != nil {
		return nil, err
	}

	return crypto.DecryptSymmetric(key, payload)

}

// Decrypt message sent with a X3DH key exchange, store the key for future exchanges
func (s *EncryptionService) DecryptWithX3DH(myIdentityKey *ecdsa.PrivateKey, theirIdentityKey *ecdsa.PublicKey, theirEphemeralKey *ecdsa.PublicKey, ourBundleID []byte, payload []byte) ([]byte, error) {
	myBundle, err := s.persistence.GetPrivateBundle(ourBundleID)
	if err != nil {
		return nil, err
	}

	signedPreKey, err := ecrypto.ToECDSA(myBundle.GetPrivateSignedPreKey())
	if err != nil {
		return nil, err
	}

	key, err := PerformPassiveX3DH(
		theirIdentityKey,
		signedPreKey,
		theirEphemeralKey,
		myIdentityKey,
	)
	if err != nil {
		return nil, err
	}

	// We encrypt the payload
	encryptedPayload, err := crypto.DecryptSymmetric(key, payload)
	if err != nil {
		return nil, err
	}

	// And we store the key for later use
	err = s.persistence.AddSymmetricKey(
		theirIdentityKey,
		theirEphemeralKey,
		key)

	if err != nil {
		return nil, err
	}
	return encryptedPayload, nil
}

const (
	EncryptionTypeDH   = "dh"
	EncryptionTypeSym  = "sym"
	EncryptionTypeX3DH = "x3dh"
)

type EncryptionResponse struct {
	EphemeralKey     *ecdsa.PublicKey
	EncryptionType   string
	EncryptedPayload []byte
	BundleID         []byte
}

func (s *EncryptionService) ProcessPublicBundle(b *Bundle) error {
	// Make sure the bundle belongs to who signed it
	err := VerifyBundle(b)
	if err != nil {
		return err
	}

	return s.persistence.AddPublicBundle(b)
}

func buildX3DHHeader(e *EncryptionResponse) *X3DHHeader {
	ephemeralKey := ecrypto.CompressPubkey(e.EphemeralKey)
	message := &X3DHHeader{}
	switch e.EncryptionType {
	case EncryptionTypeDH:
		message.EphemeralKey = &X3DHHeader_DhKey{
			ephemeralKey,
		}
	case EncryptionTypeX3DH:
		message.EphemeralKey = &X3DHHeader_BundleKey{
			ephemeralKey,
		}
		message.BundleId = e.BundleID
		m, _ := dr.DefaultCrypto{}.GenerateDH()
		fmt.Printf("%x\n", m)
	case EncryptionTypeSym:
		message.EphemeralKey = &X3DHHeader_SymKey{
			ephemeralKey,
		}

	}

	return message
}

func (p *EncryptionService) DecryptPayload(myIdentityKey *ecdsa.PrivateKey, theirIdentityKey *ecdsa.PublicKey, msg *DirectMessageProtocol) ([]byte, error) {
	payload := msg.GetPayload()
	header := msg.GetX3DHHeader()
	// Try Sym Key
	symKeyID := header.GetSymKey()
	if symKeyID != nil {
		decompressedKey, err := ecrypto.DecompressPubkey(symKeyID)
		if err != nil {
			return nil, err
		}
		return p.DecryptSymmetricPayload(theirIdentityKey, decompressedKey, payload)
	}

	// Try X3DH
	x3dhKey := header.GetBundleKey()
	bundleID := header.GetBundleId()
	if x3dhKey != nil {
		decompressedKey, err := ecrypto.DecompressPubkey(x3dhKey)
		if err != nil {
			return nil, err
		}
		return p.DecryptWithX3DH(myIdentityKey, theirIdentityKey, decompressedKey, bundleID, payload)

	}

	// Try DH
	dhKey := header.GetDhKey()
	if dhKey != nil {
		decompressedKey, err := ecrypto.DecompressPubkey(dhKey)
		if err != nil {
			return nil, err
		}
		return p.DecryptWithDH(myIdentityKey, decompressedKey, payload)

	}

	return nil, errors.New("No key specified")
}

func (s *EncryptionService) EncryptPayload(theirIdentityKey *ecdsa.PublicKey, myIdentityKey *ecdsa.PrivateKey, payload []byte) (*DirectMessageProtocol, error) {
	var symmetricKey []byte
	// The ephemeral key used to encrypt the payload
	var ourEphemeralKey *ecdsa.PublicKey
	// The bundle used
	var bundleID []byte

	encryptionType := EncryptionTypeSym

	// This should be in a transaction or similar

	// Check if we have already a key established
	symmetricKey, ourEphemeralKey, err := s.persistence.GetAnySymmetricKey(theirIdentityKey)
	if err != nil {
		return nil, err
	}

	// If not there try with a bundle and store the key
	if symmetricKey == nil {
		encryptionType = EncryptionTypeX3DH
		symmetricKey, bundleID, ourEphemeralKey, err = s.keyFromX3DH(theirIdentityKey, myIdentityKey)
		if err != nil {
			return nil, err
		}

		if ourEphemeralKey != nil {
			err = s.persistence.AddSymmetricKey(theirIdentityKey, ourEphemeralKey, symmetricKey)
			if err != nil {
				return nil, err
			}
		}
	}
	if err != nil {
		return nil, err
	}

	// keys from DH should not be re-used, so we don't store them
	if symmetricKey == nil {
		encryptionType = EncryptionTypeDH
		symmetricKey, ourEphemeralKey, err = PerformActiveDH(theirIdentityKey)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}

	encryptedPayload, err := crypto.EncryptSymmetric(symmetricKey, payload)
	if err != nil {
		return nil, err
	}

	encryptionResponse := &EncryptionResponse{
		EncryptedPayload: encryptedPayload,
		EphemeralKey:     ourEphemeralKey,
		EncryptionType:   encryptionType,
		BundleID:         bundleID,
	}

	return &DirectMessageProtocol{
		Encryption: &DirectMessageProtocol_X3DHHeader{
			buildX3DHHeader(encryptionResponse),
		},
		Payload: encryptedPayload,
	}, nil
}
