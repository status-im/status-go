package chat

import (
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/shhext/chat/topic"
)

const protocolCurrentVersion = 1
const topicNegotiationVersion = 1

type ProtocolService struct {
	log                 log.Logger
	encryption          *EncryptionService
	topic               *topic.Service
	addedBundlesHandler func([]IdentityAndIDPair)
	onNewTopicHandler   func([]*topic.Secret)
	Enabled             bool
}

var ErrNotProtocolMessage = errors.New("Not a protocol message")

// NewProtocolService creates a new ProtocolService instance
func NewProtocolService(encryption *EncryptionService, topic *topic.Service, addedBundlesHandler func([]IdentityAndIDPair), onNewTopicHandler func([]*topic.Secret)) *ProtocolService {
	return &ProtocolService{
		log:                 log.New("package", "status-go/services/sshext.chat"),
		encryption:          encryption,
		topic:               topic,
		addedBundlesHandler: addedBundlesHandler,
		onNewTopicHandler:   onNewTopicHandler,
	}
}

func (p *ProtocolService) addBundle(myIdentityKey *ecdsa.PrivateKey, msg *ProtocolMessage, sendSingle bool) (*ProtocolMessage, error) {
	// Get a bundle
	bundle, err := p.encryption.CreateBundle(myIdentityKey)
	if err != nil {
		p.log.Error("encryption-service", "error creating bundle", err)
		return nil, err
	}

	if sendSingle {
		// DEPRECATED: This is only for backward compatibility, remove once not
		// an issue anymore
		msg.Bundle = bundle
	} else {
		msg.Bundles = []*Bundle{bundle}
	}

	return msg, nil
}

// BuildPublicMessage marshals a public chat message given the user identity private key and a payload
func (p *ProtocolService) BuildPublicMessage(myIdentityKey *ecdsa.PrivateKey, payload []byte) (*ProtocolMessage, error) {
	// Build message not encrypted
	protocolMessage := &ProtocolMessage{
		InstallationId: p.encryption.config.InstallationID,
		PublicMessage:  payload,
		Version:        protocolCurrentVersion,
	}

	return p.addBundle(myIdentityKey, protocolMessage, false)
}

// BuildDirectMessage returns a 1:1 chat message and optionally a negotiated topic given the user identity private key, the recipient's public key, and a payload
func (p *ProtocolService) BuildDirectMessage(myIdentityKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, payload []byte) (*ProtocolMessage, []byte, error) {
	// Encrypt payload
	encryptionResponse, err := p.encryption.EncryptPayload(publicKey, myIdentityKey, payload)
	if err != nil {
		p.log.Error("encryption-service", "error encrypting payload", err)
		return nil, nil, err
	}

	// Build message
	protocolMessage := &ProtocolMessage{
		InstallationId: p.encryption.config.InstallationID,
		DirectMessage:  encryptionResponse,
		Version:        protocolCurrentVersion,
	}

	msg, err := p.addBundle(myIdentityKey, protocolMessage, true)
	if err != nil {
		return nil, nil, err
	}

	// Check who we are sending the message to, and see if we have a shared secret
	// across devices
	var installationIDs []string
	var sharedSecret *topic.Secret
	var agreed bool
	for installationID := range protocolMessage.GetDirectMessage() {
		if installationID != noInstallationID {
			installationIDs = append(installationIDs, installationID)
		}
	}
	if len(installationIDs) != 0 {
		sharedSecret, agreed, err = p.topic.Send(myIdentityKey, p.encryption.config.InstallationID, publicKey, installationIDs)
		if err != nil {
			return nil, nil, err
		}
	}

	// Call handler
	if sharedSecret != nil {
		p.onNewTopicHandler([]*topic.Secret{sharedSecret})
	}

	if agreed {
		return msg, sharedSecret.Key, nil
	} else {
		return msg, nil, nil
	}
}

// BuildDHMessage builds a message with DH encryption so that it can be decrypted by any other device.
func (p *ProtocolService) BuildDHMessage(myIdentityKey *ecdsa.PrivateKey, destination *ecdsa.PublicKey, payload []byte) (*ProtocolMessage, []byte, error) {
	// Encrypt payload
	encryptionResponse, err := p.encryption.EncryptPayloadWithDH(destination, payload)
	if err != nil {
		p.log.Error("encryption-service", "error encrypting payload", err)
		return nil, nil, err
	}

	// Build message
	protocolMessage := &ProtocolMessage{
		InstallationId: p.encryption.config.InstallationID,
		DirectMessage:  encryptionResponse,
		Version:        protocolCurrentVersion,
	}

	msg, err := p.addBundle(myIdentityKey, protocolMessage, true)
	if err != nil {
		return nil, nil, err
	}

	return msg, nil, nil
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

// GetPublicBundle retrieves a public bundle given an identity
func (p *ProtocolService) GetPublicBundle(theirIdentityKey *ecdsa.PublicKey) (*Bundle, error) {
	return p.encryption.GetPublicBundle(theirIdentityKey)
}

// ConfirmMessagesProcessed confirms and deletes message keys for the given messages
func (p *ProtocolService) ConfirmMessagesProcessed(messageIDs [][]byte) error {
	return p.encryption.ConfirmMessagesProcessed(messageIDs)
}

// HandleMessage unmarshals a message and processes it, decrypting it if it is a 1:1 message.
func (p *ProtocolService) HandleMessage(myIdentityKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, protocolMessage *ProtocolMessage, messageID []byte) ([]byte, error) {
	if p.encryption == nil {
		return nil, errors.New("encryption service not initialized")
	}

	// Process bundle, deprecated, here for backward compatibility
	if bundle := protocolMessage.GetBundle(); bundle != nil {
		// Should we stop processing if the bundle cannot be verified?
		addedBundles, err := p.encryption.ProcessPublicBundle(myIdentityKey, bundle)
		if err != nil {
			return nil, err
		}

		p.addedBundlesHandler(addedBundles)
	}

	// Process bundles
	for _, bundle := range protocolMessage.GetBundles() {
		// Should we stop processing if the bundle cannot be verified?
		addedBundles, err := p.encryption.ProcessPublicBundle(myIdentityKey, bundle)
		if err != nil {
			return nil, err
		}

		p.addedBundlesHandler(addedBundles)
	}

	// Check if it's a public message
	if publicMessage := protocolMessage.GetPublicMessage(); publicMessage != nil {
		// Nothing to do, as already in cleartext
		return publicMessage, nil
	}

	// Decrypt message
	if directMessage := protocolMessage.GetDirectMessage(); directMessage != nil {
		message, err := p.encryption.DecryptPayload(myIdentityKey, theirPublicKey, protocolMessage.GetInstallationId(), directMessage, messageID)
		if err != nil {
			return nil, err
		}

		p.log.Info("Checking version")
		// Handle protocol negotiation for compatible clients
		if protocolMessage.Version >= topicNegotiationVersion {
			p.log.Info("Version greater than 1 negotianting")
			sharedSecret, err := p.topic.Receive(myIdentityKey, theirPublicKey, protocolMessage.GetInstallationId())
			if err != nil {
				return nil, err
			}

			p.onNewTopicHandler([]*topic.Secret{sharedSecret})

		}
		return message, nil
	}

	// Return error
	return nil, errors.New("no payload")
}
