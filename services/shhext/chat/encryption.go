package chat

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	ecrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/log"
	dr "github.com/status-im/doubleratchet"

	"sync"
	"time"

	"github.com/status-im/status-go/services/shhext/chat/crypto"
)

// EncryptionService defines a service that is responsible for the encryption aspect of the protocol

var ErrSessionNotFound = errors.New("Session not found")

type EncryptionService struct {
	log            log.Logger
	persistence    PersistenceServiceInterface
	installationID string
	mutex          *sync.Mutex
}

// NewEncryptionService creates a new EncryptionService instance
func NewEncryptionService(p PersistenceServiceInterface, installationID string) *EncryptionService {
	logger := log.New("package", "status-go/services/sshext.chat")
	logger.Info("Initialized encryption service", "installationID", installationID)
	return &EncryptionService{
		log:            logger,
		persistence:    p,
		installationID: installationID,
		mutex:          &sync.Mutex{},
	}
}

func (s *EncryptionService) keyFromActiveX3DH(theirIdentityKey []byte, theirSignedPreKey []byte, myIdentityKey *ecdsa.PrivateKey) ([]byte, *ecdsa.PublicKey, error) {
	sharedKey, ephemeralPubKey, err := PerformActiveX3DH(theirIdentityKey, theirSignedPreKey, myIdentityKey)
	if err != nil {
		return nil, nil, err
	}

	return sharedKey, ephemeralPubKey, nil
}

// CreateBundle retrieves or creates an X3DH bundle given a private key
func (s *EncryptionService) CreateBundle(privateKey *ecdsa.PrivateKey) (*Bundle, error) {

	ourIdentityKeyC := ecrypto.CompressPubkey(&privateKey.PublicKey)
	bundleContainer, err := s.persistence.GetAnyPrivateBundle(ourIdentityKeyC)
	if err != nil {
		return nil, err
	}

	// If the bundle has expired we create a new one
	if bundleContainer != nil && bundleContainer.Timestamp < time.Now().AddDate(0, 0, -14).UnixNano() {
		// Mark sessions has expired
		if err := s.persistence.MarkBundleExpired(bundleContainer.GetBundle().GetIdentity()); err != nil {
			return nil, err
		}

	} else if bundleContainer != nil {
		err = SignBundle(privateKey, bundleContainer)
		if err != nil {
			return nil, err
		}
		return bundleContainer.GetBundle(), nil
	}

	// needs transaction/mutex to avoid creating multiple bundles
	// although not a problem
	bundleContainer, err = NewBundleContainer(privateKey, s.installationID)
	if err != nil {
		return nil, err
	}

	if err = s.persistence.AddPrivateBundle(bundleContainer); err != nil {
		return nil, err
	}

	return s.CreateBundle(privateKey)
}

// DecryptWithDH decrypts message sent with a DH key exchange, and throws away the key after decryption
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

// keyFromPassiveX3DH decrypts message sent with a X3DH key exchange, storing the key for future exchanges
func (s *EncryptionService) keyFromPassiveX3DH(myIdentityKey *ecdsa.PrivateKey, theirIdentityKey *ecdsa.PublicKey, theirEphemeralKey *ecdsa.PublicKey, ourBundleID []byte) ([]byte, error) {
	bundlePrivateKey, err := s.persistence.GetPrivateKeyBundle(ourBundleID)
	if err != nil {
		s.log.Error("Could not get private bundle", "err", err)
		return nil, err
	}

	if bundlePrivateKey == nil {
		return nil, ErrSessionNotFound
	}

	signedPreKey, err := ecrypto.ToECDSA(bundlePrivateKey)
	if err != nil {
		s.log.Error("Could not convert to ecdsa", "err", err)
		return nil, err
	}

	key, err := PerformPassiveX3DH(
		theirIdentityKey,
		signedPreKey,
		theirEphemeralKey,
		myIdentityKey,
	)
	if err != nil {
		s.log.Error("Could not perform passive x3dh", "err", err)
		return nil, err
	}
	return key, nil
}

// ProcessPublicBundle persists a bundle
func (s *EncryptionService) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, b *Bundle) error {

	// Make sure the bundle belongs to who signed it
	err := VerifyBundle(b)
	if err != nil {
		return err
	}
	return s.persistence.AddPublicBundle(b)

	// If it's our bundle but different device id merge the bundles
	//crypto.FromECDSAPub(recoveredKey)

}

// DecryptPayload decrypts the payload of a DirectMessageProtocol, given an identity private key and the sender's public key
func (s *EncryptionService) DecryptPayload(myIdentityKey *ecdsa.PrivateKey, theirIdentityKey *ecdsa.PublicKey, msgs *map[string]*DirectMessageProtocol) ([]byte, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	msg := (*msgs)[s.installationID]
	if msg == nil {
		msg = (*msgs)["none"]
	}

	if msg == nil {
		return nil, ErrSessionNotFound
	}
	payload := msg.GetPayload()

	if x3dhHeader := msg.GetX3DHHeader(); x3dhHeader != nil {
		bundleID := x3dhHeader.GetId()
		theirEphemeralKey, err := ecrypto.DecompressPubkey(x3dhHeader.GetKey())

		if err != nil {
			return nil, err
		}

		symmetricKey, err := s.keyFromPassiveX3DH(myIdentityKey, theirIdentityKey, theirEphemeralKey, bundleID)
		if err != nil {
			return nil, err
		}

		theirIdentityKeyC := ecrypto.CompressPubkey(theirIdentityKey)
		err = s.persistence.AddRatchetInfo(symmetricKey, theirIdentityKeyC, bundleID, nil, x3dhHeader.GetInstallationId())
		if err != nil {
			return nil, err
		}
	}

	if drHeader := msg.GetDRHeader(); drHeader != nil {
		var dh [32]byte
		copy(dh[:], drHeader.GetKey())

		drMessage := &dr.Message{
			Header: dr.MessageHeader{
				N:  drHeader.GetN(),
				PN: drHeader.GetPn(),
				DH: dh,
			},
			Ciphertext: msg.GetPayload(),
		}

		theirIdentityKeyC := ecrypto.CompressPubkey(theirIdentityKey)

		drInfo, err := s.persistence.GetRatchetInfo(drHeader.GetId(), theirIdentityKeyC)
		if err != nil {
			s.log.Error("Could not get ratchet info", "err", err)
			return nil, err
		}

		// We mark the exchange as successful so we stop sending x3dh header
		if err = s.persistence.RatchetInfoConfirmed(drHeader.GetId(), theirIdentityKeyC); err != nil {
			s.log.Error("Could not confirm ratchet info", "err", err)
			return nil, err
		}

		if drInfo == nil {
			s.log.Error("Could not find a session")
			return nil, ErrSessionNotFound
		}

		return s.decryptUsingDR(theirIdentityKey, drInfo, drMessage)
	}

	// Try DH
	if header := msg.GetDHHeader(); header != nil {
		decompressedKey, err := ecrypto.DecompressPubkey(header.GetKey())
		if err != nil {
			return nil, err
		}
		return s.DecryptWithDH(myIdentityKey, decompressedKey, payload)
	}

	return nil, errors.New("no key specified")
}

func (s *EncryptionService) encryptUsingDR(theirIdentityKey *ecdsa.PublicKey, drInfo *RatchetInfo, payload []byte) ([]byte, *DRHeader, error) {
	var err error

	var session dr.Session
	var sk [32]byte
	copy(sk[:], drInfo.Sk)

	var publicKey [32]byte
	copy(publicKey[:], drInfo.PublicKey[:32])

	var privateKey [32]byte
	copy(privateKey[:], drInfo.PrivateKey[:])

	keyPair := crypto.DHPair{
		PrvKey: privateKey,
		PubKey: publicKey,
	}

	sessionStorage := s.persistence.GetSessionStorage()
	// Load session from store first
	session, err = dr.Load(
		drInfo.ID,
		sessionStorage,
		dr.WithKeysStorage(s.persistence.GetKeysStorage()),
		dr.WithCrypto(crypto.EthereumCrypto{}),
	)
	if err != nil {
		return nil, nil, err
	}

	// Create a new one
	if session == nil {

		if drInfo.PrivateKey != nil {
			session, err = dr.New(
				drInfo.ID,
				sk,
				keyPair,
				sessionStorage,
				dr.WithKeysStorage(s.persistence.GetKeysStorage()),
				dr.WithCrypto(crypto.EthereumCrypto{}))
		} else {
			session, err = dr.NewWithRemoteKey(
				drInfo.ID,
				sk,
				publicKey,
				sessionStorage,
				dr.WithKeysStorage(s.persistence.GetKeysStorage()),
				dr.WithCrypto(crypto.EthereumCrypto{}))
		}
	}

	if err != nil {
		return nil, nil, err
	}

	response, err := session.RatchetEncrypt(payload, nil)
	if err != nil {
		return nil, nil, err
	}

	header := &DRHeader{
		Id:  drInfo.BundleID,
		Key: response.Header.DH[:],
		N:   response.Header.N,
		Pn:  response.Header.PN,
	}

	return response.Ciphertext, header, nil
}

func (s *EncryptionService) decryptUsingDR(theirIdentityKey *ecdsa.PublicKey, drInfo *RatchetInfo, payload *dr.Message) ([]byte, error) {
	var err error

	var session dr.Session
	var sk [32]byte
	copy(sk[:], drInfo.Sk)

	var publicKey [32]byte
	copy(publicKey[:], drInfo.PublicKey[:32])

	var privateKey [32]byte
	copy(privateKey[:], drInfo.PrivateKey[:])

	keyPair := crypto.DHPair{
		PrvKey: privateKey,
		PubKey: publicKey,
	}

	sessionStorage := s.persistence.GetSessionStorage()
	session, err = dr.Load(
		drInfo.ID,
		sessionStorage,
		dr.WithKeysStorage(s.persistence.GetKeysStorage()),
		dr.WithCrypto(crypto.EthereumCrypto{}),
	)

	if err != nil {
		return nil, err
	}

	if session == nil {
		if drInfo.PrivateKey != nil {
			session, err = dr.New(
				drInfo.ID,
				sk,
				keyPair,
				sessionStorage,
				dr.WithKeysStorage(s.persistence.GetKeysStorage()),
				dr.WithCrypto(crypto.EthereumCrypto{}))
		} else {
			session, err = dr.NewWithRemoteKey(
				drInfo.ID,
				sk,
				publicKey,
				sessionStorage,
				dr.WithKeysStorage(s.persistence.GetKeysStorage()),
				dr.WithCrypto(crypto.EthereumCrypto{}))
		}
	}

	if err != nil {
		return nil, err
	}

	plaintext, err := session.RatchetDecrypt(*payload, nil)

	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (s *EncryptionService) encryptWithDH(theirIdentityKey *ecdsa.PublicKey, payload []byte) (*DirectMessageProtocol, error) {
	symmetricKey, ourEphemeralKey, err := PerformActiveDH(theirIdentityKey)
	if err != nil {
		return nil, err
	}

	encryptedPayload, err := crypto.EncryptSymmetric(symmetricKey, payload)
	if err != nil {
		return nil, err
	}

	return &DirectMessageProtocol{
		DHHeader: &DHHeader{
			Key: ecrypto.CompressPubkey(ourEphemeralKey),
		},
		Payload: encryptedPayload,
	}, nil
}

// EncryptPayload returns a new DirectMessageProtocol with a given payload encrypted, given a recipient's public key and the sender private identity key
// TODO: refactor this
// nolint: gocyclo
func (s *EncryptionService) EncryptPayload(theirIdentityKey *ecdsa.PublicKey, myIdentityKey *ecdsa.PrivateKey, payload []byte) (*map[string]*DirectMessageProtocol, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	response := make(map[string]*DirectMessageProtocol)
	theirIdentityKeyC := ecrypto.CompressPubkey(theirIdentityKey)

	// Get their latest bundle
	theirBundle, err := s.persistence.GetPublicBundle(theirIdentityKey)
	if err != nil {
		return nil, err
	}

	// We don't have any, send a message with DH
	if theirBundle == nil && !bytes.Equal(theirIdentityKeyC, ecrypto.CompressPubkey(&myIdentityKey.PublicKey)) {
		dmp, err := s.encryptWithDH(theirIdentityKey, payload)
		if err != nil {
			return nil, err
		}

		response["none"] = dmp
		return &response, nil

	}

	for installationID, signedPreKeyContainer := range theirBundle.GetSignedPreKeys() {
		if s.installationID != installationID {
			signedPreKey := signedPreKeyContainer.GetSignedPreKey()
			// See if a session is there already
			drInfo, err := s.persistence.GetAnyRatchetInfo(theirIdentityKeyC, installationID)
			if err != nil {
				return nil, err
			}

			if drInfo != nil {
				encryptedPayload, drHeader, err := s.encryptUsingDR(theirIdentityKey, drInfo, payload)
				if err != nil {
					return nil, err
				}

				dmp := DirectMessageProtocol{
					Payload:  encryptedPayload,
					DRHeader: drHeader,
				}

				if drInfo.EphemeralKey != nil {
					dmp.X3DHHeader = &X3DHHeader{
						Key:            drInfo.EphemeralKey,
						Id:             drInfo.BundleID,
						InstallationId: s.installationID,
					}
				}

				response[drInfo.InstallationID] = &dmp

				return &response, nil
			}

			// check if a bundle is there
			theirBundle, err := s.persistence.GetPublicBundle(theirIdentityKey)
			if err != nil {
				return nil, err
			}

			if theirBundle != nil {
				sharedKey, ourEphemeralKey, err := s.keyFromActiveX3DH(theirIdentityKeyC, signedPreKey, myIdentityKey)
				if err != nil {
					return nil, err
				}
				theirIdentityKeyC := ecrypto.CompressPubkey(theirIdentityKey)
				ourEphemeralKeyC := ecrypto.CompressPubkey(ourEphemeralKey)

				err = s.persistence.AddRatchetInfo(sharedKey, theirIdentityKeyC, signedPreKey, ourEphemeralKeyC, installationID)
				if err != nil {
					return nil, err
				}

				x3dhHeader := &X3DHHeader{
					Key:            ourEphemeralKeyC,
					Id:             signedPreKey,
					InstallationId: s.installationID,
				}

				drInfo, err := s.persistence.GetAnyRatchetInfo(theirIdentityKeyC, installationID)
				if err != nil {
					return nil, err
				}

				if drInfo != nil {
					encryptedPayload, drHeader, err := s.encryptUsingDR(theirIdentityKey, drInfo, payload)
					if err != nil {
						return nil, err
					}

					dmp := &DirectMessageProtocol{
						Payload:    encryptedPayload,
						X3DHHeader: x3dhHeader,
						DRHeader:   drHeader,
					}

					response[drInfo.InstallationID] = dmp

				}
			}

		}
	}
	return &response, nil

}
