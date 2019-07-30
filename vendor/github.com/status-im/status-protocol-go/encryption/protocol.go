package encryption

import (
	"bytes"
	"crypto/ecdsa"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	"github.com/status-im/status-protocol-go/encryption/multidevice"
	"github.com/status-im/status-protocol-go/encryption/publisher"
	"github.com/status-im/status-protocol-go/encryption/sharedsecret"
)

//go:generate protoc --go_out=. ./protocol_message.proto

const (
	protocolVersion                = 1
	sharedSecretNegotiationVersion = 1
	partitionedTopicMinVersion     = 1
	defaultMinVersion              = 0
)

type PartitionTopicMode int

const (
	PartitionTopicNoSupport PartitionTopicMode = iota
	PartitionTopicV1
)

type ProtocolMessageSpec struct {
	Message *ProtocolMessage
	// Installations is the targeted devices
	Installations []*multidevice.Installation
	// SharedSecret is a shared secret established among the installations
	SharedSecret []byte
	// Public means that the spec contains a public wrapped message
	Public bool
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

func (p *ProtocolMessageSpec) PartitionedTopicMode() PartitionTopicMode {
	if p.MinVersion() >= partitionedTopicMinVersion {
		return PartitionTopicV1
	}
	return PartitionTopicNoSupport
}

type Protocol struct {
	encryptor   *encryptor
	secret      *sharedsecret.SharedSecret
	multidevice *multidevice.Multidevice
	publisher   *publisher.Publisher

	onAddedBundlesHandler    func([]*multidevice.Installation)
	onNewSharedSecretHandler func([]*sharedsecret.Secret)
	onSendContactCodeHandler func(*ProtocolMessageSpec)

	logger *zap.Logger
}

var (
	// ErrNoPayload means that there was no payload found in the received protocol message.
	ErrNoPayload = errors.New("no payload")
)

// New creates a new ProtocolService instance
func New(
	db *sql.DB,
	installationID string,
	addedBundlesHandler func([]*multidevice.Installation),
	onNewSharedSecretHandler func([]*sharedsecret.Secret),
	onSendContactCodeHandler func(*ProtocolMessageSpec),
	logger *zap.Logger,
) *Protocol {
	return NewWithEncryptorConfig(
		db,
		installationID,
		defaultEncryptorConfig(installationID, logger),
		addedBundlesHandler,
		onNewSharedSecretHandler,
		onSendContactCodeHandler,
		logger,
	)
}

// DB and migrations are shared between encryption package
// and its sub-packages.
func NewWithEncryptorConfig(
	db *sql.DB,
	installationID string,
	encryptorConfig encryptorConfig,
	addedBundlesHandler func([]*multidevice.Installation),
	onNewSharedSecretHandler func([]*sharedsecret.Secret),
	onSendContactCodeHandler func(*ProtocolMessageSpec),
	logger *zap.Logger,
) *Protocol {
	return &Protocol{
		encryptor: newEncryptor(db, encryptorConfig),
		secret:    sharedsecret.New(db, logger),
		multidevice: multidevice.New(db, &multidevice.Config{
			MaxInstallations: 3,
			ProtocolVersion:  protocolVersion,
			InstallationID:   installationID,
		}),
		publisher:                publisher.New(db, logger),
		onAddedBundlesHandler:    addedBundlesHandler,
		onNewSharedSecretHandler: onNewSharedSecretHandler,
		onSendContactCodeHandler: onSendContactCodeHandler,
		logger:                   logger.With(zap.Namespace("Protocol")),
	}
}

func (p *Protocol) Start(myIdentity *ecdsa.PrivateKey) error {
	// Propagate currently cached shared secrets.
	secrets, err := p.secret.All()
	if err != nil {
		return errors.Wrap(err, "failed to get all secrets")
	}
	p.onNewSharedSecretHandler(secrets)

	// Handle Publisher system messages.
	publisherCh := p.publisher.Start()

	go func() {
		for range publisherCh {
			messageSpec, err := p.buildContactCodeMessage(myIdentity)
			if err != nil {
				p.logger.Error("failed to build contact code message",
					zap.String("site", "Start"),
					zap.Error(err))
				continue
			}

			p.onSendContactCodeHandler(messageSpec)
		}
	}()

	return nil
}

func (p *Protocol) addBundle(myIdentityKey *ecdsa.PrivateKey, msg *ProtocolMessage, sendSingle bool) error {
	logger := p.logger.With(zap.String("site", "addBundle"))

	// Get a bundle
	installations, err := p.multidevice.GetOurActiveInstallations(&myIdentityKey.PublicKey)
	if err != nil {
		return err
	}

	logger.Info("adding bundle to the message",
		zap.Any("installations", installations),
	)

	bundle, err := p.encryptor.CreateBundle(myIdentityKey, installations)
	if err != nil {
		return err
	}

	if sendSingle {
		// DEPRECATED: This is only for backward compatibility, remove once not
		// an issue anymore
		msg.Bundle = bundle
	} else {
		msg.Bundles = []*Bundle{bundle}
	}

	return nil
}

// BuildPublicMessage marshals a public chat message given the user identity private key and a payload
func (p *Protocol) BuildPublicMessage(myIdentityKey *ecdsa.PrivateKey, payload []byte) (*ProtocolMessageSpec, error) {
	// Build message not encrypted
	message := &ProtocolMessage{
		InstallationId: p.encryptor.config.InstallationID,
		PublicMessage:  payload,
	}

	err := p.addBundle(myIdentityKey, message, false)
	if err != nil {
		return nil, err
	}

	return &ProtocolMessageSpec{Message: message, Public: true}, nil
}

// buildContactCodeMessage creates a contact code message. It's a public message
// without any data but it carries bundle information.
func (p *Protocol) buildContactCodeMessage(myIdentityKey *ecdsa.PrivateKey) (*ProtocolMessageSpec, error) {
	return p.BuildPublicMessage(myIdentityKey, nil)
}

// BuildDirectMessage returns a 1:1 chat message and optionally a negotiated topic given the user identity private key, the recipient's public key, and a payload
func (p *Protocol) BuildDirectMessage(myIdentityKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, payload []byte) (*ProtocolMessageSpec, error) {
	logger := p.logger.With(
		zap.String("site", "BuildDirectMessage"),
		zap.Binary("public-key", crypto.FromECDSAPub(publicKey)),
	)

	logger.Debug("building direct message")

	// Get recipients installations.
	activeInstallations, err := p.multidevice.GetActiveInstallations(publicKey)
	if err != nil {
		return nil, err
	}

	// Encrypt payload
	directMessage, installations, err := p.encryptor.EncryptPayload(publicKey, myIdentityKey, activeInstallations, payload)
	if err != nil {
		return nil, err
	}

	// Build message
	message := &ProtocolMessage{
		InstallationId: p.encryptor.config.InstallationID,
		DirectMessage:  directMessage,
	}

	err = p.addBundle(myIdentityKey, message, true)
	if err != nil {
		return nil, err
	}

	// Check who we are sending the message to, and see if we have a shared secret
	// across devices
	var installationIDs []string
	for installationID := range message.GetDirectMessage() {
		if installationID != noInstallationID {
			installationIDs = append(installationIDs, installationID)
		}
	}

	sharedSecret, agreed, err := p.secret.Agreed(myIdentityKey, p.encryptor.config.InstallationID, publicKey, installationIDs)
	if err != nil {
		return nil, err
	}

	logger.Debug("shared secret agreement",
		zap.Bool("has-shared-secret", sharedSecret != nil),
		zap.Bool("agreed", agreed))

	// Call handler
	if sharedSecret != nil {
		p.onNewSharedSecretHandler([]*sharedsecret.Secret{sharedSecret})
	}

	spec := &ProtocolMessageSpec{
		Message:       message,
		Installations: installations,
	}
	if agreed {
		spec.SharedSecret = sharedSecret.Key
	}
	return spec, nil
}

// BuildDHMessage builds a message with DH encryption so that it can be decrypted by any other device.
func (p *Protocol) BuildDHMessage(myIdentityKey *ecdsa.PrivateKey, destination *ecdsa.PublicKey, payload []byte) (*ProtocolMessageSpec, error) {
	// Encrypt payload
	encryptionResponse, err := p.encryptor.EncryptPayloadWithDH(destination, payload)
	if err != nil {
		return nil, err
	}

	// Build message
	message := &ProtocolMessage{
		InstallationId: p.encryptor.config.InstallationID,
		DirectMessage:  encryptionResponse,
	}

	err = p.addBundle(myIdentityKey, message, true)
	if err != nil {
		return nil, err
	}

	return &ProtocolMessageSpec{Message: message}, nil
}

// ProcessPublicBundle processes a received X3DH bundle.
func (p *Protocol) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, bundle *Bundle) ([]*multidevice.Installation, error) {
	logger := p.logger.With(zap.String("site", "ProcessPublicBundle"))

	logger.Debug("processing public bundle")

	if err := p.encryptor.ProcessPublicBundle(myIdentityKey, bundle); err != nil {
		return nil, err
	}

	installations, enabled, err := p.recoverInstallationsFromBundle(myIdentityKey, bundle)
	if err != nil {
		return nil, err
	}

	logger.Debug("recovered installations",
		zap.Int("installations", len(installations)),
		zap.Bool("enabled", enabled))

	// TODO(adam): why do we add installations using identity obtained from GetIdentity()
	// instead of the output of crypto.CompressPubkey()? I tried the second option
	// and the unit tests TestTopic and TestMaxDevices fail.
	identityFromBundle := bundle.GetIdentity()
	theirIdentity, err := ExtractIdentity(bundle)
	if err != nil {
		logger.Panic("unrecoverable error extracting identity", zap.Error(err))
	}
	compressedIdentity := crypto.CompressPubkey(theirIdentity)
	if !bytes.Equal(identityFromBundle, compressedIdentity) {
		logger.Panic("identity from bundle and compressed are not equal")
	}

	return p.multidevice.AddInstallations(bundle.GetIdentity(), bundle.GetTimestamp(), installations, enabled)
}

// recoverInstallationsFromBundle extracts installations from the bundle.
// It returns extracted installations and true if the installations
// are ours, i.e. the bundle was created by our identity key.
func (p *Protocol) recoverInstallationsFromBundle(myIdentityKey *ecdsa.PrivateKey, bundle *Bundle) ([]*multidevice.Installation, bool, error) {
	var installations []*multidevice.Installation

	theirIdentity, err := ExtractIdentity(bundle)
	if err != nil {
		return nil, false, err
	}

	logger := p.logger.With(zap.String("site", "recoverInstallationsFromBundle"))

	myIdentityStr := fmt.Sprintf("0x%x", crypto.FromECDSAPub(&myIdentityKey.PublicKey))
	theirIdentityStr := fmt.Sprintf("0x%x", crypto.FromECDSAPub(theirIdentity))
	// Any device from other peers will be considered enabled, ours needs to
	// be explicitly enabled.
	enabled := theirIdentityStr != myIdentityStr
	signedPreKeys := bundle.GetSignedPreKeys()

	for installationID, signedPreKey := range signedPreKeys {
		logger.Info("recovered installation", zap.String("installation-id", installationID))
		if installationID != p.multidevice.InstallationID() {
			installations = append(installations, &multidevice.Installation{
				Identity: theirIdentityStr,
				ID:       installationID,
				Version:  signedPreKey.GetProtocolVersion(),
			})
		}
	}

	return installations, enabled, nil
}

// GetBundle retrieves or creates a X3DH bundle, given a private identity key.
func (p *Protocol) GetBundle(myIdentityKey *ecdsa.PrivateKey) (*Bundle, error) {
	installations, err := p.multidevice.GetOurActiveInstallations(&myIdentityKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return p.encryptor.CreateBundle(myIdentityKey, installations)
}

// EnableInstallation enables an installation for multi-device sync.
func (p *Protocol) EnableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return p.multidevice.EnableInstallation(myIdentityKey, installationID)
}

// DisableInstallation disables an installation for multi-device sync.
func (p *Protocol) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return p.multidevice.DisableInstallation(myIdentityKey, installationID)
}

// GetOurInstallations returns all the installations available given an identity
func (p *Protocol) GetOurInstallations(myIdentityKey *ecdsa.PublicKey) ([]*multidevice.Installation, error) {
	return p.multidevice.GetOurInstallations(myIdentityKey)
}

// SetInstallationMetadata sets the metadata for our own installation
func (p *Protocol) SetInstallationMetadata(myIdentityKey *ecdsa.PublicKey, installationID string, data *multidevice.InstallationMetadata) error {
	return p.multidevice.SetInstallationMetadata(myIdentityKey, installationID, data)
}

// GetPublicBundle retrieves a public bundle given an identity
func (p *Protocol) GetPublicBundle(theirIdentityKey *ecdsa.PublicKey) (*Bundle, error) {
	installations, err := p.multidevice.GetActiveInstallations(theirIdentityKey)
	if err != nil {
		return nil, err
	}
	return p.encryptor.GetPublicBundle(theirIdentityKey, installations)
}

// ConfirmMessageProcessed confirms and deletes message keys for the given messages
func (p *Protocol) ConfirmMessageProcessed(messageID []byte) error {
	return p.encryptor.ConfirmMessageProcessed(messageID)
}

// HandleMessage unmarshals a message and processes it, decrypting it if it is a 1:1 message.
func (p *Protocol) HandleMessage(
	myIdentityKey *ecdsa.PrivateKey,
	theirPublicKey *ecdsa.PublicKey,
	protocolMessage *ProtocolMessage,
	messageID []byte,
) ([]byte, error) {
	logger := p.logger.With(zap.String("site", "HandleMessage"))

	logger.Debug("received a protocol message", zap.Binary("sender-public-key", crypto.FromECDSAPub(theirPublicKey)))

	if p.encryptor == nil {
		return nil, errors.New("encryption service not initialized")
	}

	// Process bundle, deprecated, here for backward compatibility
	if bundle := protocolMessage.GetBundle(); bundle != nil {
		// Should we stop processing if the bundle cannot be verified?
		addedBundles, err := p.ProcessPublicBundle(myIdentityKey, bundle)
		if err != nil {
			return nil, err
		}

		p.onAddedBundlesHandler(addedBundles)
	}

	// Process bundles
	for _, bundle := range protocolMessage.GetBundles() {
		// Should we stop processing if the bundle cannot be verified?
		addedBundles, err := p.ProcessPublicBundle(myIdentityKey, bundle)
		if err != nil {
			return nil, err
		}

		p.onAddedBundlesHandler(addedBundles)
	}

	// Check if it's a public message
	if publicMessage := protocolMessage.GetPublicMessage(); publicMessage != nil {
		logger.Debug("received a public message in direct message")
		// Nothing to do, as already in cleartext
		return publicMessage, nil
	}

	// Decrypt message
	if directMessage := protocolMessage.GetDirectMessage(); directMessage != nil {
		logger.Debug("processing direct message")
		message, err := p.encryptor.DecryptPayload(
			myIdentityKey,
			theirPublicKey,
			protocolMessage.GetInstallationId(),
			directMessage,
			messageID,
		)
		if err != nil {
			return nil, err
		}

		// Handle protocol negotiation for compatible clients
		bundles := append(protocolMessage.GetBundles(), protocolMessage.GetBundle())
		version := getProtocolVersion(bundles, protocolMessage.GetInstallationId())
		logger.Debug("direct message version", zap.Uint32("version", version))
		if version >= sharedSecretNegotiationVersion {
			logger.Debug("negotiating shared secret",
				zap.Binary("public-key", crypto.FromECDSAPub(theirPublicKey)))
			sharedSecret, err := p.secret.Generate(myIdentityKey, theirPublicKey, protocolMessage.GetInstallationId())
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

func (p *Protocol) ShouldAdvertiseBundle(publicKey *ecdsa.PublicKey, time int64) (bool, error) {
	return p.publisher.ShouldAdvertiseBundle(publicKey, time)
}

func (p *Protocol) ConfirmBundleAdvertisement(publicKey *ecdsa.PublicKey, time int64) {
	p.publisher.SetLastAck(publicKey, time)
}

func (p *Protocol) BuildBundleAdvertiseMessage(myIdentityKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) (*ProtocolMessageSpec, error) {
	return p.BuildDHMessage(myIdentityKey, publicKey, nil)
}

func getProtocolVersion(bundles []*Bundle, installationID string) uint32 {
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
