package chat

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	ecrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/log"
	dr "github.com/status-im/doubleratchet"

	"github.com/status-im/status-go/services/shhext/chat/crypto"
	"github.com/status-im/status-go/services/shhext/chat/multidevice"
	"github.com/status-im/status-go/services/shhext/chat/protobuf"
)

var ErrSessionNotFound = errors.New("session not found")
var ErrDeviceNotFound = errors.New("device not found")

// If we have no bundles, we use a constant so that the message can reach any device.
const noInstallationID = "none"

type ConfirmationData struct {
	header *dr.MessageHeader
	drInfo *RatchetInfo
}

// EncryptionService defines a service that is responsible for the encryption aspect of the protocol.
type EncryptionService struct {
	log         log.Logger
	persistence PersistenceService
	config      EncryptionServiceConfig
	messageIDs  map[string]*ConfirmationData
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
		messageIDs:  make(map[string]*ConfirmationData),
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

func confirmationIDString(id []byte) string {
	return hex.EncodeToString(id)
}

// ConfirmMessagesProcessed confirms and deletes message keys for the given messages
func (s *EncryptionService) ConfirmMessagesProcessed(messageIDs [][]byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, idByte := range messageIDs {
		id := confirmationIDString(idByte)
		confirmationData, ok := s.messageIDs[id]
		if !ok {
			s.log.Debug("Could not confirm message", "messageID", id)
			continue
		}

		// Load session from store first
		session, err := s.getDRSession(confirmationData.drInfo.ID)
		if err != nil {
			return err
		}

		if err := session.DeleteMk(confirmationData.header.DH, confirmationData.header.N); err != nil {
			return err
		}
	}
	return nil
}

// CreateBundle retrieves or creates an X3DH bundle given a private key
func (s *EncryptionService) CreateBundle(privateKey *ecdsa.PrivateKey, installations []*multidevice.Installation) (*protobuf.Bundle, error) {
	ourIdentityKeyC := ecrypto.CompressPubkey(&privateKey.PublicKey)

	bundleContainer, err := s.persistence.GetAnyPrivateBundle(ourIdentityKeyC, installations)
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

	return s.CreateBundle(privateKey, installations)
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
func (s *EncryptionService) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, b *protobuf.Bundle) error {
	return s.persistence.AddPublicBundle(b)
}

// DecryptPayload decrypts the payload of a DirectMessageProtocol, given an identity private key and the sender's public key
func (s *EncryptionService) DecryptPayload(myIdentityKey *ecdsa.PrivateKey, theirIdentityKey *ecdsa.PublicKey, theirInstallationID string, msgs map[string]*protobuf.DirectMessageProtocol, messageID []byte) ([]byte, error) {
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

		// Add installations with a timestamp of 0, as we don't have bundle informations
		//if err = s.persistence.AddInstallations(theirIdentityKeyC, 0, []*Installation{{ID: theirInstallationID, Version: 0}}, true); err != nil {
		//		return nil, err
		//	}

		// We mark the exchange as successful so we stop sending x3dh header
		if err = s.persistence.RatchetInfoConfirmed(drHeader.GetId(), theirIdentityKeyC, theirInstallationID); err != nil {
			s.log.Error("Could not confirm ratchet info", "err", err)
			return nil, err
		}

		if drInfo == nil {
			s.log.Error("Could not find a session")
			return nil, ErrSessionNotFound
		}

		confirmationData := &ConfirmationData{
			header: &drMessage.Header,
			drInfo: drInfo,
		}
		s.messageIDs[confirmationIDString(messageID)] = confirmationData

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

func (s *EncryptionService) encryptUsingDR(theirIdentityKey *ecdsa.PublicKey, drInfo *RatchetInfo, payload []byte) ([]byte, *protobuf.DRHeader, error) {
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

	header := &protobuf.DRHeader{
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

func (s *EncryptionService) encryptWithDH(theirIdentityKey *ecdsa.PublicKey, payload []byte) (*protobuf.DirectMessageProtocol, error) {
	symmetricKey, ourEphemeralKey, err := PerformActiveDH(theirIdentityKey)
	if err != nil {
		return nil, err
	}

	encryptedPayload, err := crypto.EncryptSymmetric(symmetricKey, payload)
	if err != nil {
		return nil, err
	}

	return &protobuf.DirectMessageProtocol{
		DHHeader: &protobuf.DHHeader{
			Key: ecrypto.CompressPubkey(ourEphemeralKey),
		},
		Payload: encryptedPayload,
	}, nil
}

func (s *EncryptionService) EncryptPayloadWithDH(theirIdentityKey *ecdsa.PublicKey, payload []byte) (map[string]*protobuf.DirectMessageProtocol, error) {
	response := make(map[string]*protobuf.DirectMessageProtocol)
	dmp, err := s.encryptWithDH(theirIdentityKey, payload)
	if err != nil {
		return nil, err
	}

	response[noInstallationID] = dmp
	return response, nil
}

// GetPublicBundle returns the active installations bundles for a given user
func (s *EncryptionService) GetPublicBundle(theirIdentityKey *ecdsa.PublicKey, installations []*multidevice.Installation) (*protobuf.Bundle, error) {
	return s.persistence.GetPublicBundle(theirIdentityKey, installations)
}

// EncryptPayload returns a new DirectMessageProtocol with a given payload encrypted, given a recipient's public key and the sender private identity key
// TODO: refactor this
// nolint: gocyclo
func (s *EncryptionService) EncryptPayload(theirIdentityKey *ecdsa.PublicKey, myIdentityKey *ecdsa.PrivateKey, installations []*multidevice.Installation, payload []byte) (map[string]*protobuf.DirectMessageProtocol, []*multidevice.Installation, error) {
	// Which installations we are sending the message to
	var targetedInstallations []*multidevice.Installation

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.log.Debug("Sending message", "theirKey", theirIdentityKey)

	theirIdentityKeyC := ecrypto.CompressPubkey(theirIdentityKey)

	// We don't have any, send a message with DH
	if installations == nil && !bytes.Equal(theirIdentityKeyC, ecrypto.CompressPubkey(&myIdentityKey.PublicKey)) {
		encryptedPayload, err := s.EncryptPayloadWithDH(theirIdentityKey, payload)
		return encryptedPayload, targetedInstallations, err
	}

	response := make(map[string]*protobuf.DirectMessageProtocol)

	for _, installation := range installations {
		installationID := installation.ID
		s.log.Debug("Processing installation", "installationID", installationID)
		if s.config.InstallationID == installationID {
			continue
		}
		bundle, err := s.persistence.GetPublicBundle(theirIdentityKey, []*multidevice.Installation{installation})
		if err != nil {
			return nil, nil, err
		}

		// See if a session is there already
		drInfo, err := s.persistence.GetAnyRatchetInfo(theirIdentityKeyC, installationID)
		if err != nil {
			return nil, nil, err
		}

		targetedInstallations = append(targetedInstallations, installation)

		if drInfo != nil {
			s.log.Debug("Found DR info", "installationID", installationID)
			encryptedPayload, drHeader, err := s.encryptUsingDR(theirIdentityKey, drInfo, payload)
			if err != nil {
				return nil, nil, err
			}

			dmp := protobuf.DirectMessageProtocol{
				Payload:  encryptedPayload,
				DRHeader: drHeader,
			}

			if drInfo.EphemeralKey != nil {
				dmp.X3DHHeader = &protobuf.X3DHHeader{
					Key: drInfo.EphemeralKey,
					Id:  drInfo.BundleID,
				}
			}

			response[drInfo.InstallationID] = &dmp
			continue
		}

		theirSignedPreKeyContainer := bundle.GetSignedPreKeys()[installationID]

		// This should not be nil at this point
		if theirSignedPreKeyContainer == nil {
			s.log.Warn("Could not find either a ratchet info or a bundle for installationId", "installationID", installationID)
			continue

		}
		s.log.Debug("DR info not found, using bundle", "installationID", installationID)

		theirSignedPreKey := theirSignedPreKeyContainer.GetSignedPreKey()

		sharedKey, ourEphemeralKey, err := s.keyFromActiveX3DH(theirIdentityKeyC, theirSignedPreKey, myIdentityKey)
		if err != nil {
			return nil, nil, err
		}
		theirIdentityKeyC := ecrypto.CompressPubkey(theirIdentityKey)
		ourEphemeralKeyC := ecrypto.CompressPubkey(ourEphemeralKey)

		err = s.persistence.AddRatchetInfo(sharedKey, theirIdentityKeyC, theirSignedPreKey, ourEphemeralKeyC, installationID)
		if err != nil {
			return nil, nil, err
		}

		x3dhHeader := &protobuf.X3DHHeader{
			Key: ourEphemeralKeyC,
			Id:  theirSignedPreKey,
		}

		drInfo, err = s.persistence.GetRatchetInfo(theirSignedPreKey, theirIdentityKeyC, installationID)
		if err != nil {
			return nil, nil, err
		}

		if drInfo != nil {
			encryptedPayload, drHeader, err := s.encryptUsingDR(theirIdentityKey, drInfo, payload)
			if err != nil {
				return nil, nil, err
			}

			dmp := &protobuf.DirectMessageProtocol{
				Payload:    encryptedPayload,
				X3DHHeader: x3dhHeader,
				DRHeader:   drHeader,
			}

			response[drInfo.InstallationID] = dmp
		}
	}

	s.log.Debug("Built message", "theirKey", theirIdentityKey)

	return response, targetedInstallations, nil
}
