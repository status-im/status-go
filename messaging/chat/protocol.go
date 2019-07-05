package chat

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/messaging/chat/protobuf"
	"github.com/status-im/status-go/messaging/multidevice"
	"github.com/status-im/status-go/messaging/sharedsecret"
)

const ProtocolVersion = 1
const sharedSecretNegotiationVersion = 1
const partitionedTopicMinVersion = 1
const defaultMinVersion = 0

type PartitionTopic int

const (
	PartitionTopicNoSupport PartitionTopic = iota
	PartitionTopicV1
)

type ProtocolService struct {
	log                      log.Logger
	encryption               *EncryptionService
	secret                   *sharedsecret.Service
	multidevice              *multidevice.Service
	addedBundlesHandler      func([]*multidevice.Installation)
	onNewSharedSecretHandler func([]*sharedsecret.Secret)
	Enabled                  bool
}

var (
	ErrNotProtocolMessage = errors.New("not a protocol message")
	ErrNoPayload          = errors.New("no payload")
)

// NewProtocolService creates a new ProtocolService instance
func NewProtocolService(encryption *EncryptionService, secret *sharedsecret.Service, multidevice *multidevice.Service, addedBundlesHandler func([]*multidevice.Installation), onNewSharedSecretHandler func([]*sharedsecret.Secret)) *ProtocolService {
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
		return defaultMinVersion
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
		return PartitionTopicV1
	}
	return PartitionTopicNoSupport
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
func (p *ProtocolService) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, bundle *protobuf.Bundle) ([]*multidevice.Installation, error) {
	p.log.Debug("Processing bundle", "bundle", bundle)

	if err := p.encryption.ProcessPublicBundle(myIdentityKey, bundle); err != nil {
		return nil, err
	}

	installations, fromOurs, err := p.recoverInstallationsFromBundle(myIdentityKey, bundle)
	if err != nil {
		return nil, err
	}

	// TODO(adam): why do we add installations using identity obtained from GetIdentity()
	// instead of the output of crypto.CompressPubkey()? I tried the second option
	// and the unit tests TestTopic and TestMaxDevices fail.
	return p.multidevice.AddInstallations(bundle.GetIdentity(), bundle.GetTimestamp(), installations, fromOurs)
}

// recoverInstallationsFromBundle extracts installations from the bundle.
// It returns extracted installations and true if the installations
// are ours, i.e. the bundle was created by our identity key.
func (p *ProtocolService) recoverInstallationsFromBundle(myIdentityKey *ecdsa.PrivateKey, bundle *protobuf.Bundle) ([]*multidevice.Installation, bool, error) {
	var installations []*multidevice.Installation

	theirIdentity, err := ExtractIdentity(bundle)
	if err != nil {
		return nil, false, err
	}

	myIdentityStr := fmt.Sprintf("0x%x", crypto.FromECDSAPub(&myIdentityKey.PublicKey))
	theirIdentityStr := fmt.Sprintf("0x%x", crypto.FromECDSAPub(theirIdentity))
	// Any device from other peers will be considered enabled, ours needs to
	// be explicitly enabled
	fromOurIdentity := theirIdentityStr != myIdentityStr
	signedPreKeys := bundle.GetSignedPreKeys()

	for installationID, signedPreKey := range signedPreKeys {
		if installationID != p.multidevice.InstallationID() {
			installations = append(installations, &multidevice.Installation{
				Identity: theirIdentityStr,
				ID:       installationID,
				Version:  signedPreKey.GetProtocolVersion(),
			})
		}
	}

	return installations, fromOurIdentity, nil
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

// GetOurInstallations returns all the installations available given an identity
func (p *ProtocolService) GetOurInstallations(myIdentityKey *ecdsa.PublicKey) ([]*multidevice.Installation, error) {
	return p.multidevice.GetOurInstallations(myIdentityKey)
}

// SetInstallationMetadata sets the metadata for our own installation
func (p *ProtocolService) SetInstallationMetadata(myIdentityKey *ecdsa.PublicKey, installationID string, data *multidevice.InstallationMetadata) error {
	return p.multidevice.SetInstallationMetadata(myIdentityKey, installationID, data)
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

func (p *ProtocolService) GetSharedSecretService() *sharedsecret.Service {
	return p.secret
}

// HandleMessage unmarshals a message and processes it, decrypting it if it is a 1:1 message.
func (p *ProtocolService) HandleMessage(myIdentityKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, protocolMessage *protobuf.ProtocolMessage, messageID []byte) ([]byte, error) {
	p.log.Debug("Received message from", "public-key", theirPublicKey)
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
		p.log.Debug("Public message, nothing to do")
		// Nothing to do, as already in cleartext
		return publicMessage, nil
	}

	// Decrypt message
	if directMessage := protocolMessage.GetDirectMessage(); directMessage != nil {
		p.log.Debug("Processing direct message")
		message, err := p.encryption.DecryptPayload(myIdentityKey, theirPublicKey, protocolMessage.GetInstallationId(), directMessage, messageID)
		if err != nil {
			return nil, err
		}

		// Handle protocol negotiation for compatible clients
		bundles := append(protocolMessage.GetBundles(), protocolMessage.GetBundle())
		version := getProtocolVersion(bundles, protocolMessage.GetInstallationId())
		p.log.Debug("Message version is", "version", version)
		if version >= sharedSecretNegotiationVersion {
			p.log.Debug("Negotiating shared secret")
			sharedSecret, err := p.secret.Receive(myIdentityKey, theirPublicKey, protocolMessage.GetInstallationId())
			if err != nil {
				return nil, err
			}

			p.onNewSharedSecretHandler([]*sharedsecret.Secret{sharedSecret})

		}
		return message, nil
	}

	// Return error
	return nil, ErrNoPayload
}

func getProtocolVersion(bundles []*protobuf.Bundle, installationID string) uint32 {
	if installationID == "" {
		return defaultMinVersion
	}

	for _, bundle := range bundles {
		if bundle != nil {
			signedPreKeys := bundle.GetSignedPreKeys()
			if signedPreKeys == nil {
				continue
			}

			signedPreKey := signedPreKeys[installationID]
			if signedPreKey == nil {
				return defaultMinVersion
			}

			return signedPreKey.GetProtocolVersion()
		}
	}

	return defaultMinVersion
}
