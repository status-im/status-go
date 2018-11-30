package chat

import (
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/golang/protobuf/proto"
)

type ProtocolService struct {
	log        log.Logger
	encryption *EncryptionService
	Enabled    bool
}

type HandleMessageResponse struct {
	AddedBundles []IdentityAndIDPair
	Message      []byte
}

// NewProtocolService creates a new ProtocolService instance
func NewProtocolService(encryption *EncryptionService) *ProtocolService {
	return &ProtocolService{
		log:        log.New("package", "status-go/services/sshext.chat"),
		encryption: encryption,
	}
}

func (p *ProtocolService) addBundleAndMarshal(myIdentityKey *ecdsa.PrivateKey, msg *ProtocolMessage) ([]byte, error) {
	// Get a bundle
	bundle, err := p.encryption.CreateBundle(myIdentityKey)
	if err != nil {
		p.log.Error("encryption-service", "error creating bundle", err)
		return nil, err
	}

	msg.Bundle = bundle

	// marshal for sending to wire
	marshaledMessage, err := proto.Marshal(msg)
	if err != nil {
		p.log.Error("encryption-service", "error marshaling message", err)
		return nil, err
	}

	return marshaledMessage, nil
}

// BuildPublicMessage marshals a public chat message given the user identity private key and a payload
func (p *ProtocolService) BuildPublicMessage(myIdentityKey *ecdsa.PrivateKey, payload []byte) ([]byte, error) {
	// Build message not encrypted
	protocolMessage := &ProtocolMessage{
		InstallationId: p.encryption.config.InstallationID,
		PublicMessage:  payload,
	}

	return p.addBundleAndMarshal(myIdentityKey, protocolMessage)
}

// BuildDirectMessage marshals a 1:1 chat message given the user identity private key, the recipient's public key, and a payload
func (p *ProtocolService) BuildDirectMessage(myIdentityKey *ecdsa.PrivateKey, payload []byte, theirPublicKeys ...*ecdsa.PublicKey) (map[*ecdsa.PublicKey][]byte, error) {
	response := make(map[*ecdsa.PublicKey][]byte)
	for _, publicKey := range theirPublicKeys {
		// Encrypt payload
		encryptionResponse, err := p.encryption.EncryptPayload(publicKey, myIdentityKey, payload)
		if err != nil {
			p.log.Error("encryption-service", "error encrypting payload", err)
			return nil, err
		}

		// Build message
		protocolMessage := &ProtocolMessage{
			InstallationId: p.encryption.config.InstallationID,
			DirectMessage:  encryptionResponse,
		}

		payload, err := p.addBundleAndMarshal(myIdentityKey, protocolMessage)
		if err != nil {
			return nil, err
		}

		if len(payload) != 0 {
			response[publicKey] = payload
		}
	}
	return response, nil
}

// BuildPairingMessage sends a message to our own devices using DH so that it can be decrypted by any other device.
func (p *ProtocolService) BuildPairingMessage(myIdentityKey *ecdsa.PrivateKey, payload []byte) ([]byte, error) {
	// Encrypt payload
	encryptionResponse, err := p.encryption.EncryptPayloadWithDH(&myIdentityKey.PublicKey, payload)
	if err != nil {
		p.log.Error("encryption-service", "error encrypting payload", err)
		return nil, err
	}

	// Build message
	protocolMessage := &ProtocolMessage{
		InstallationId: p.encryption.config.InstallationID,
		DirectMessage:  encryptionResponse,
	}

	return p.addBundleAndMarshal(myIdentityKey, protocolMessage)
}

// ProcessPublicBundle processes a received X3DH bundle.
func (p *ProtocolService) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, bundle *Bundle) ([]IdentityAndIDPair, error) {
	return p.encryption.ProcessPublicBundle(myIdentityKey, bundle)
}

// GetBundle retrieves or creates a X3DH bundle, given a private identity key.
func (p *ProtocolService) GetBundle(myIdentityKey *ecdsa.PrivateKey) (*Bundle, error) {
	return p.encryption.CreateBundle(myIdentityKey)
}

// EnableInstallation enables an installation for multi-device sync.
func (p *ProtocolService) EnableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return p.encryption.EnableInstallation(myIdentityKey, installationID)
}

// DisableInstallation disables an installation for multi-device sync.
func (p *ProtocolService) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return p.encryption.DisableInstallation(myIdentityKey, installationID)
}

// HandleMessage unmarshals a message and processes it, decrypting it if it is a 1:1 message.
func (p *ProtocolService) HandleMessage(myIdentityKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, payload []byte) (*HandleMessageResponse, error) {
	if p.encryption == nil {
		return nil, errors.New("encryption service not initialized")
	}

	response := &HandleMessageResponse{}

	// Unmarshal message
	protocolMessage := &ProtocolMessage{}

	if err := proto.Unmarshal(payload, protocolMessage); err != nil {
		return nil, err
	}

	// Process bundle
	if bundle := protocolMessage.GetBundle(); bundle != nil {
		// Should we stop processing if the bundle cannot be verified?
		addedBundles, err := p.encryption.ProcessPublicBundle(myIdentityKey, bundle)
		if err != nil {
			return nil, err
		}
		response.AddedBundles = addedBundles
	}

	// Check if it's a public message
	if publicMessage := protocolMessage.GetPublicMessage(); publicMessage != nil {
		response.Message = publicMessage
		// Nothing to do, as already in cleartext
		return response, nil
	}

	// Decrypt message
	if directMessage := protocolMessage.GetDirectMessage(); directMessage != nil {
		message, err := p.encryption.DecryptPayload(myIdentityKey, theirPublicKey, protocolMessage.GetInstallationId(), directMessage)
		if err != nil {
			return response, err
		}
		response.Message = message

		return response, nil
	}

	// Return error
	return response, errors.New("no payload")
}
