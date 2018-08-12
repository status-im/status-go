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
		MessageType: &ProtocolMessage_PublicMessage{
			payload,
		},
	}

	return p.addBundleAndMarshal(myIdentityKey, protocolMessage)
}

// BuildDirectMessage marshals a 1:1 chat message given the user identity private key, the recipient's public key, and a payload
func (p *ProtocolService) BuildDirectMessage(myIdentityKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, payload []byte) ([]byte, error) {
	// Encrypt payload
	encryptionResponse, err := p.encryption.EncryptPayload(theirPublicKey, myIdentityKey, payload)
	if err != nil {
		p.log.Error("encryption-service", "error encrypting payload", err)
		return nil, err
	}

	// Build message
	protocolMessage := &ProtocolMessage{
		MessageType: &ProtocolMessage_DirectMessage{encryptionResponse},
	}

	return p.addBundleAndMarshal(myIdentityKey, protocolMessage)
}

// ProcessPublicBundle processes a received X3DH bundle
func (p *ProtocolService) ProcessPublicBundle(bundle *Bundle) error {
	return p.encryption.ProcessPublicBundle(bundle)
}

// GetBundle retrieves or creates a X3DH bundle, given a private identity key
func (p *ProtocolService) GetBundle(myIdentityKey *ecdsa.PrivateKey) (*Bundle, error) {
	return p.encryption.CreateBundle(myIdentityKey)
}

// HandleMessage unmarshals a message and processes it, decrypting it if it is a 1:1 message
func (p *ProtocolService) HandleMessage(myIdentityKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, payload []byte) ([]byte, error) {
	// Unmarshal message
	protocolMessage := &ProtocolMessage{}

	if err := proto.Unmarshal(payload, protocolMessage); err != nil {
		return nil, err
	}

	// Process bundle
	if bundle := protocolMessage.GetBundle(); bundle != nil {
		// Should we stop processing if the bundle cannot be verified?
		err := p.encryption.ProcessPublicBundle(bundle)
		if err != nil {
			return nil, err
		}
	}

	// Check if it's a public message
	if publicMessage := protocolMessage.GetPublicMessage(); publicMessage != nil {
		// Nothing to do, as already in cleartext
		return publicMessage, nil
	}

	// Decrypt message
	if directMessage := protocolMessage.GetDirectMessage(); directMessage != nil {
		return p.encryption.DecryptPayload(myIdentityKey, theirPublicKey, directMessage)
	}

	// Return error
	return nil, errors.New("no payload")
}
