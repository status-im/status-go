package chat

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"

	ecrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/log"
	dr "github.com/status-im/doubleratchet"

	"sync"
	"time"

	"github.com/status-im/status-go/services/shhext/chat/crypto"
)

var ErrSessionNotFound = errors.New("session not found")
var ErrDeviceNotFound = errors.New("device not found")

// If we have no bundles, we use a constant so that the message can reach any device.
const noInstallationID = "none"

// EncryptionService defines a service that is responsible for the encryption aspect of the protocol.
type EncryptionService struct {
	log         log.Logger
	persistence PersistenceService
	config      EncryptionServiceConfig
	mutex       sync.Mutex
}

type EncryptionServiceConfig struct {
	InstallationID string
	// Max number of installations we keep synchronized.
	MaxInstallations int
	// How many consecutive messages can be skipped in the receiving chain.
	MaxSkip int
	// Any message with seqNo <= currentSeq - maxKeep will be deleted.
	MaxKeep int
	// How many keys do we store in total per session.
	MaxMessageKeysPerSession int
	// How long before we refresh the interval in milliseconds
	BundleRefreshInterval int64
}

type IdentityAndIDPair [2]string

// DefaultEncryptionServiceConfig returns the default values used by the encryption service
func DefaultEncryptionServiceConfig(installationID string) EncryptionServiceConfig {
	return EncryptionServiceConfig{
		MaxInstallations:         3,
		MaxSkip:                  1000,
		MaxKeep:                  3000,
		MaxMessageKeysPerSession: 2000,
		BundleRefreshInterval:    24 * 60 * 60 * 1000,
		InstallationID:           installationID,
	}
}

// NewEncryptionService creates a new EncryptionService instance.
func NewEncryptionService(p PersistenceService, config EncryptionServiceConfig) *EncryptionService {
	logger := log.New("package", "status-go/services/sshext.chat")
	logger.Info("Initialized encryption service", "installationID", config.InstallationID)
	return &EncryptionService{
		log:         logger,
		persistence: p,
		config:      config,
		mutex:       sync.Mutex{},
	}
}

func (s *EncryptionService) keyFromActiveX3DH(theirIdentityKey []byte, theirSignedPreKey []byte, myIdentityKey *ecdsa.PrivateKey) ([]byte, *ecdsa.PublicKey, error) {
	sharedKey, ephemeralPubKey, err := PerformActiveX3DH(theirIdentityKey, theirSignedPreKey, myIdentityKey)
	if err != nil {
		return nil, nil, err
	}

	return sharedKey, ephemeralPubKey, nil
}

func (s *EncryptionService) getDRSession(id []byte) (dr.Session, error) {
	sessionStorage := s.persistence.GetSessionStorage()
	return dr.Load(
		id,
		sessionStorage,
		dr.WithKeysStorage(s.persistence.GetKeysStorage()),
		dr.WithMaxSkip(s.config.MaxSkip),
		dr.WithMaxKeep(s.config.MaxKeep),
		dr.WithMaxMessageKeysPerSession(s.config.MaxMessageKeysPerSession),
		dr.WithCrypto(crypto.EthereumCrypto{}),
	)

}

// CreateBundle retrieves or creates an X3DH bundle given a private key
func (s *EncryptionService) CreateBundle(privateKey *ecdsa.PrivateKey) (*Bundle, error) {
	ourIdentityKeyC := ecrypto.CompressPubkey(&privateKey.PublicKey)

	installationIDs, err := s.persistence.GetActiveInstallations(s.config.MaxInstallations-1, ourIdentityKeyC)
	if err != nil {
		return nil, err
	}

	installationIDs = append(installationIDs, s.config.InstallationID)

	bundleContainer, err := s.persistence.GetAnyPrivateBundle(ourIdentityKeyC, installationIDs)
	if err != nil {
		return nil, err
	}

	// If the bundle has expired we create a new one
	if bundleContainer != nil && bundleContainer.GetBundle().Timestamp < time.Now().Add(-1*time.Duration(s.config.BundleRefreshInterval)*time.Millisecond).UnixNano() {
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
	bundleContainer, err = NewBundleContainer(privateKey, s.config.InstallationID)
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

func (s *EncryptionService) EnableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	myIdentityKeyC := ecrypto.CompressPubkey(myIdentityKey)
	return s.persistence.EnableInstallation(myIdentityKeyC, installationID)
}

func (s *EncryptionService) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	myIdentityKeyC := ecrypto.CompressPubkey(myIdentityKey)
	return s.persistence.DisableInstallation(myIdentityKeyC, installationID)
}

// ProcessPublicBundle persists a bundle and returns a list of tuples identity/installationID
func (s *EncryptionService) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, b *Bundle) ([]IdentityAndIDPair, error) {
	// Make sure the bundle belongs to who signed it
	identity, err := ExtractIdentity(b)
	if err != nil {
		return nil, err
	}
	signedPreKeys := b.GetSignedPreKeys()
	var response []IdentityAndIDPair
	var installationIDs []string
	myIdentityStr := fmt.Sprintf("0x%x", ecrypto.FromECDSAPub(&myIdentityKey.PublicKey))

	// Any device from other peers will be considered enabled, ours needs to
	// be explicitly enabled
	fromOurIdentity := identity != myIdentityStr

	for installationID := range signedPreKeys {
		if installationID != s.config.InstallationID {
			installationIDs = append(installationIDs, installationID)
			response = append(response, IdentityAndIDPair{identity, installationID})
		}
	}

	if err = s.persistence.AddInstallations(b.GetIdentity(), b.GetTimestamp(), installationIDs, fromOurIdentity); err != nil {
		return nil, err
	}

	if err = s.persistence.AddPublicBundle(b); err != nil {
		return nil, err
	}

	return response, nil
}

// DecryptPayload decrypts the payload of a DirectMessageProtocol, given an identity private key and the sender's public key
func (s *EncryptionService) DecryptPayload(myIdentityKey *ecdsa.PrivateKey, theirIdentityKey *ecdsa.PublicKey, theirInstallationID string, msgs map[string]*DirectMessageProtocol) ([]byte, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	msg := msgs[s.config.InstallationID]
	if msg == nil {
		msg = msgs[noInstallationID]
	}

	// We should not be sending a signal if it's coming from us, as we receive our own messages
	if msg == nil && *theirIdentityKey != myIdentityKey.PublicKey {
		return nil, ErrDeviceNotFound
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
		err = s.persistence.AddRatchetInfo(symmetricKey, theirIdentityKeyC, bundleID, nil, theirInstallationID)
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

		drInfo, err := s.persistence.GetRatchetInfo(drHeader.GetId(), theirIdentityKeyC, theirInstallationID)
		if err != nil {
			s.log.Error("Could not get ratchet info", "err", err)
			return nil, err
		}

		// We mark the exchange as successful so we stop sending x3dh header
		if err = s.persistence.RatchetInfoConfirmed(drHeader.GetId(), theirIdentityKeyC, theirInstallationID); err != nil {
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

func (s *EncryptionService) createNewSession(drInfo *RatchetInfo, sk [32]byte, keyPair crypto.DHPair) (dr.Session, error) {
	var err error
	var session dr.Session

	if drInfo.PrivateKey != nil {
		session, err = dr.New(
			drInfo.ID,
			sk,
			keyPair,
			s.persistence.GetSessionStorage(),
			dr.WithKeysStorage(s.persistence.GetKeysStorage()),
			dr.WithMaxSkip(s.config.MaxSkip),
			dr.WithMaxKeep(s.config.MaxKeep),
			dr.WithMaxMessageKeysPerSession(s.config.MaxMessageKeysPerSession),
			dr.WithCrypto(crypto.EthereumCrypto{}))
	} else {
		session, err = dr.NewWithRemoteKey(
			drInfo.ID,
			sk,
			keyPair.PubKey,
			s.persistence.GetSessionStorage(),
			dr.WithKeysStorage(s.persistence.GetKeysStorage()),
			dr.WithMaxSkip(s.config.MaxSkip),
			dr.WithMaxKeep(s.config.MaxKeep),
			dr.WithMaxMessageKeysPerSession(s.config.MaxMessageKeysPerSession),
			dr.WithCrypto(crypto.EthereumCrypto{}))
	}

	return session, err
}

func (s *EncryptionService) encryptUsingDR(theirIdentityKey *ecdsa.PublicKey, drInfo *RatchetInfo, payload []byte) ([]byte, *DRHeader, error) {
	var err error

	var session dr.Session
	var sk, publicKey, privateKey [32]byte
	copy(sk[:], drInfo.Sk)
	copy(publicKey[:], drInfo.PublicKey[:32])
	copy(privateKey[:], drInfo.PrivateKey[:])

	keyPair := crypto.DHPair{
		PrvKey: privateKey,
		PubKey: publicKey,
	}

	// Load session from store first
	session, err = s.getDRSession(drInfo.ID)

	if err != nil {
		return nil, nil, err
	}

	// Create a new one
	if session == nil {
		session, err = s.createNewSession(drInfo, sk, keyPair)
		if err != nil {
			return nil, nil, err
		}
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
	var sk, publicKey, privateKey [32]byte
	copy(sk[:], drInfo.Sk)
	copy(publicKey[:], drInfo.PublicKey[:32])
	copy(privateKey[:], drInfo.PrivateKey[:])

	keyPair := crypto.DHPair{
		PrvKey: privateKey,
		PubKey: publicKey,
	}

	session, err = s.getDRSession(drInfo.ID)
	if err != nil {
		return nil, err
	}

	if session == nil {
		session, err = s.createNewSession(drInfo, sk, keyPair)
		if err != nil {
			return nil, err
		}
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

func (s *EncryptionService) EncryptPayloadWithDH(theirIdentityKey *ecdsa.PublicKey, payload []byte) (map[string]*DirectMessageProtocol, error) {
	response := make(map[string]*DirectMessageProtocol)
	dmp, err := s.encryptWithDH(theirIdentityKey, payload)
	if err != nil {
		return nil, err
	}

	response[noInstallationID] = dmp
	return response, nil
}

// GetPublicBundle returns the active installations bundles for a given user
func (s *EncryptionService) GetPublicBundle(theirIdentityKey *ecdsa.PublicKey) (*Bundle, error) {
	theirIdentityKeyC := ecrypto.CompressPubkey(theirIdentityKey)

	installationIDs, err := s.persistence.GetActiveInstallations(s.config.MaxInstallations, theirIdentityKeyC)
	if err != nil {
		return nil, err
	}

	return s.persistence.GetPublicBundle(theirIdentityKey, installationIDs)
}

// EncryptPayload returns a new DirectMessageProtocol with a given payload encrypted, given a recipient's public key and the sender private identity key
// TODO: refactor this
// nolint: gocyclo
func (s *EncryptionService) EncryptPayload(theirIdentityKey *ecdsa.PublicKey, myIdentityKey *ecdsa.PrivateKey, payload []byte) (map[string]*DirectMessageProtocol, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	theirIdentityKeyC := ecrypto.CompressPubkey(theirIdentityKey)

	// Get their latest bundle
	theirBundle, err := s.GetPublicBundle(theirIdentityKey)
	if err != nil {
		return nil, err
	}

	// We don't have any, send a message with DH
	if theirBundle == nil && !bytes.Equal(theirIdentityKeyC, ecrypto.CompressPubkey(&myIdentityKey.PublicKey)) {
		return s.EncryptPayloadWithDH(theirIdentityKey, payload)
	}

	response := make(map[string]*DirectMessageProtocol)

	for installationID, signedPreKeyContainer := range theirBundle.GetSignedPreKeys() {
		if s.config.InstallationID == installationID {
			continue
		}

		theirSignedPreKey := signedPreKeyContainer.GetSignedPreKey()
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
					Key: drInfo.EphemeralKey,
					Id:  drInfo.BundleID,
				}
			}

			response[drInfo.InstallationID] = &dmp
			continue
		}

		sharedKey, ourEphemeralKey, err := s.keyFromActiveX3DH(theirIdentityKeyC, theirSignedPreKey, myIdentityKey)
		if err != nil {
			return nil, err
		}
		theirIdentityKeyC := ecrypto.CompressPubkey(theirIdentityKey)
		ourEphemeralKeyC := ecrypto.CompressPubkey(ourEphemeralKey)

		err = s.persistence.AddRatchetInfo(sharedKey, theirIdentityKeyC, theirSignedPreKey, ourEphemeralKeyC, installationID)
		if err != nil {
			return nil, err
		}

		x3dhHeader := &X3DHHeader{
			Key: ourEphemeralKeyC,
			Id:  theirSignedPreKey,
		}

		drInfo, err = s.persistence.GetRatchetInfo(theirSignedPreKey, theirIdentityKeyC, installationID)
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

	return response, nil
}
