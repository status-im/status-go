package chat

import (
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/shhext/chat/multidevice"
	"github.com/status-im/status-go/services/shhext/chat/protobuf"
	"github.com/status-im/status-go/services/shhext/chat/sharedsecret"
)

const ProtocolVersion = 1
const sharedSecretNegotiationVersion = 1
const partitionedTopicMinVersion = 1

type PartitionTopic int

const (
	PARTITION_TOPIC_NO_SUPPORT PartitionTopic = iota
	PARTITION_TOPIC_V1         PartitionTopic = iota
)

type ProtocolService struct {
	log                      log.Logger
	encryption               *EncryptionService
	secret                   *sharedsecret.Service
	multidevice              *multidevice.Service
	addedBundlesHandler      func([]*multidevice.IdentityAndID)
	onNewSharedSecretHandler func([]*sharedsecret.Secret)
	Enabled                  bool
}

var ErrNotProtocolMessage = errors.New("Not a protocol message")

// NewProtocolService creates a new ProtocolService instance
func NewProtocolService(encryption *EncryptionService, secret *sharedsecret.Service, multidevice *multidevice.Service, addedBundlesHandler func([]*multidevice.IdentityAndID), onNewSharedSecretHandler func([]*sharedsecret.Secret)) *ProtocolService {
	return &ProtocolService{
		log:                      log.New("package", "status-go/services/sshext.chat"),
		encryption:               encryption,
		secret:                   secret,
		multidevice:              multidevice,
		addedBundlesHandler:      addedBundlesHandler,
		onNewSharedSecretHandler: onNewSharedSecretHandler,
	}
}

func (p *ProtocolService) addBundle(myIdentityKey *ecdsa.PrivateKey, msg *protobuf.ProtocolMessage, sendSingle bool) (*protobuf.ProtocolMessage, error) {

	// Get a bundle
	installations, err := p.multidevice.GetOurActiveInstallations(&myIdentityKey.PublicKey)
	if err != nil {
		return nil, err
	}

	bundle, err := p.encryption.CreateBundle(myIdentityKey, installations)
	if err != nil {
		p.log.Error("encryption-service", "error creating bundle", err)
		return nil, err
	}

	if sendSingle {
		// DEPRECATED: This is only for backward compatibility, remove once not
		// an issue anymore
		msg.Bundle = bundle
	} else {
		msg.Bundles = []*protobuf.Bundle{bundle}
	}

	return msg, nil
}

// BuildPublicMessage marshals a public chat message given the user identity private key and a payload
func (p *ProtocolService) BuildPublicMessage(myIdentityKey *ecdsa.PrivateKey, payload []byte) (*protobuf.ProtocolMessage, error) {
	// Build message not encrypted
	protocolMessage := &protobuf.ProtocolMessage{
		InstallationId: p.encryption.config.InstallationID,
		PublicMessage:  payload,
	}

	return p.addBundle(myIdentityKey, protocolMessage, false)
}

type ProtocolMessageSpec struct {
	Message *protobuf.ProtocolMessage
	// Installations is the targeted devices
	Installations []*multidevice.Installation
	// SharedSecret is a shared secret established among the installations
	SharedSecret []byte
}

func (p *ProtocolMessageSpec) MinVersion() uint32 {

	if len(p.Installations) == 0 {
		return 0
	}

	version := p.Installations[0].Version

	for _, installation := range p.Installations[1:] {
		if installation.Version < version {
			version = installation.Version
		}
	}
	return version
}

func (p *ProtocolMessageSpec) PartitionedTopic() PartitionTopic {
	if p.MinVersion() >= partitionedTopicMinVersion {
		return PARTITION_TOPIC_V1
	}
	return PARTITION_TOPIC_NO_SUPPORT
}

// BuildDirectMessage returns a 1:1 chat message and optionally a negotiated topic given the user identity private key, the recipient's public key, and a payload
func (p *ProtocolService) BuildDirectMessage(myIdentityKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, payload []byte) (*ProtocolMessageSpec, error) {
	activeInstallations, err := p.multidevice.GetActiveInstallations(publicKey)
	if err != nil {
		return nil, err
	}

	// Encrypt payload
	encryptionResponse, installations, err := p.encryption.EncryptPayload(publicKey, myIdentityKey, activeInstallations, payload)
	if err != nil {
		p.log.Error("encryption-service", "error encrypting payload", err)
		return nil, err
	}

	// Build message
	protocolMessage := &protobuf.ProtocolMessage{
		InstallationId: p.encryption.config.InstallationID,
		DirectMessage:  encryptionResponse,
	}

	msg, err := p.addBundle(myIdentityKey, protocolMessage, true)
	if err != nil {
		return nil, err
	}

	// Check who we are sending the message to, and see if we have a shared secret
	// across devices
	var installationIDs []string
	var sharedSecret *sharedsecret.Secret
	var agreed bool
	for installationID := range protocolMessage.GetDirectMessage() {
		if installationID != noInstallationID {
			installationIDs = append(installationIDs, installationID)
		}
	}

	sharedSecret, agreed, err = p.secret.Send(myIdentityKey, p.encryption.config.InstallationID, publicKey, installationIDs)
	if err != nil {
		return nil, err
	}

	// Call handler
	if sharedSecret != nil {
		p.onNewSharedSecretHandler([]*sharedsecret.Secret{sharedSecret})
	}
	response := &ProtocolMessageSpec{
		Message:       msg,
		Installations: installations,
	}

	if agreed {
		response.SharedSecret = sharedSecret.Key
	}
	return response, nil
}

// BuildDHMessage builds a message with DH encryption so that it can be decrypted by any other device.
func (p *ProtocolService) BuildDHMessage(myIdentityKey *ecdsa.PrivateKey, destination *ecdsa.PublicKey, payload []byte) (*ProtocolMessageSpec, error) {
	// Encrypt payload
	encryptionResponse, err := p.encryption.EncryptPayloadWithDH(destination, payload)
	if err != nil {
		p.log.Error("encryption-service", "error encrypting payload", err)
		return nil, err
	}

	// Build message
	protocolMessage := &protobuf.ProtocolMessage{
		InstallationId: p.encryption.config.InstallationID,
		DirectMessage:  encryptionResponse,
	}

	msg, err := p.addBundle(myIdentityKey, protocolMessage, true)
	if err != nil {
		return nil, err
	}

	return &ProtocolMessageSpec{Message: msg}, nil
}

// ProcessPublicBundle processes a received X3DH bundle.
func (p *ProtocolService) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, bundle *protobuf.Bundle) ([]*multidevice.IdentityAndID, error) {
	if err := p.encryption.ProcessPublicBundle(myIdentityKey, bundle); err != nil {
		return nil, err
	}

	theirIdentityKey, err := ExtractIdentity(bundle)
	if err != nil {
		return nil, err
	}

	return p.multidevice.ProcessPublicBundle(myIdentityKey, theirIdentityKey, bundle)
}

// GetBundle retrieves or creates a X3DH bundle, given a private identity key.
func (p *ProtocolService) GetBundle(myIdentityKey *ecdsa.PrivateKey) (*protobuf.Bundle, error) {
	installations, err := p.multidevice.GetOurActiveInstallations(&myIdentityKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return p.encryption.CreateBundle(myIdentityKey, installations)
}

// EnableInstallation enables an installation for multi-device sync.
func (p *ProtocolService) EnableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return p.multidevice.EnableInstallation(myIdentityKey, installationID)
}

// DisableInstallation disables an installation for multi-device sync.
func (p *ProtocolService) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return p.multidevice.DisableInstallation(myIdentityKey, installationID)
}

// GetPublicBundle retrieves a public bundle given an identity
func (p *ProtocolService) GetPublicBundle(theirIdentityKey *ecdsa.PublicKey) (*protobuf.Bundle, error) {
	installations, err := p.multidevice.GetActiveInstallations(theirIdentityKey)
	if err != nil {
		return nil, err
	}
	return p.encryption.GetPublicBundle(theirIdentityKey, installations)
}

// ConfirmMessagesProcessed confirms and deletes message keys for the given messages
func (p *ProtocolService) ConfirmMessagesProcessed(messageIDs [][]byte) error {
	return p.encryption.ConfirmMessagesProcessed(messageIDs)
}

// HandleMessage unmarshals a message and processes it, decrypting it if it is a 1:1 message.
func (p *ProtocolService) HandleMessage(myIdentityKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, protocolMessage *protobuf.ProtocolMessage, messageID []byte) ([]byte, error) {
	if p.encryption == nil {
		return nil, errors.New("encryption service not initialized")
	}

	// Process bundle, deprecated, here for backward compatibility
	if bundle := protocolMessage.GetBundle(); bundle != nil {
		// Should we stop processing if the bundle cannot be verified?
		addedBundles, err := p.ProcessPublicBundle(myIdentityKey, bundle)
		if err != nil {
			return nil, err
		}

		p.addedBundlesHandler(addedBundles)
	}

	// Process bundles
	for _, bundle := range protocolMessage.GetBundles() {
		// Should we stop processing if the bundle cannot be verified?
		addedBundles, err := p.ProcessPublicBundle(myIdentityKey, bundle)
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

		var bundles []*protobuf.Bundle
		p.log.Info("Checking version")
		// Handle protocol negotiation for compatible clients
		p.log.Info("bundle", "bundles", protocolMessage)
		bundles = append(protocolMessage.GetBundles(), protocolMessage.GetBundle())
		version := getProtocolVersion(bundles, protocolMessage.GetInstallationId())
		if version >= sharedSecretNegotiationVersion {
			p.log.Info("Version greater than 1 negotianting")
			sharedSecret, err := p.secret.Receive(myIdentityKey, theirPublicKey, protocolMessage.GetInstallationId())
			if err != nil {
				return nil, err
			}

			p.onNewSharedSecretHandler([]*sharedsecret.Secret{sharedSecret})

		}
		return message, nil
	}

	// Return error
	return nil, errors.New("no payload")
}

func getProtocolVersion(bundles []*protobuf.Bundle, installationID string) uint32 {
	if installationID == "" {
		return 0
	}

	for _, bundle := range bundles {
		if bundle != nil {
			signedPreKeys := bundle.GetSignedPreKeys()
			if signedPreKeys == nil {
				continue
			}

			signedPreKey := signedPreKeys[installationID]
			if signedPreKey == nil {
				return 0
			}

			return signedPreKey.GetProtocolVersion()
		}
	}

	return 0
}
