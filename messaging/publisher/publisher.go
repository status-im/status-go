package publisher

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/golang/protobuf/proto"
	"github.com/status-im/status-go/messaging/chat"
	chatDB "github.com/status-im/status-go/messaging/chat/db"
	"github.com/status-im/status-go/messaging/chat/multidevice"
	"github.com/status-im/status-go/messaging/chat/protobuf"
	"github.com/status-im/status-go/messaging/chat/sharedsecret"
	"github.com/status-im/status-go/messaging/filter"
	"github.com/status-im/status-go/services/shhext/whisperutils"
	"github.com/status-im/status-go/signal"
	whisper "github.com/status-im/whisper/whisperv6"
	"golang.org/x/crypto/sha3"
	"os"
	"path/filepath"
	"time"
)

const (
	tickerInterval = 120
	// How often we should publish a contact code in seconds
	publishInterval  = 21600
	maxInstallations = 3
)

var (
	errProtocolNotInitialized = errors.New("protocol is not initialized")
	// ErrPFSNotEnabled is returned when an endpoint PFS only is called but
	// PFS is disabled
	ErrPFSNotEnabled = errors.New("pfs not enabled")
	errNoKeySelected = errors.New("no key selected")
)

type Publisher struct {
	whisper     *whisper.Whisper
	online      func() bool
	whisperAPI  *whisper.PublicWhisperAPI
	protocol    *chat.ProtocolService
	persistence Persistence
	log         log.Logger
	filter      *filter.Service
	config      *Config
	quit        chan struct{}
	ticker      *time.Ticker
}

type Config struct {
	PfsEnabled     bool
	DataDir        string
	InstallationID string
}

func New(config *Config, w *whisper.Whisper) *Publisher {
	return &Publisher{
		config:     config,
		whisper:    w,
		whisperAPI: whisper.NewPublicWhisperAPI(w),
		log:        log.New("package", "status-go/services/publisher.Publisher"),
	}
}

// InitProtocolWithPassword creates an instance of ProtocolService given an address and password used to generate an encryption key.
func (p *Publisher) InitProtocolWithPassword(address string, password string) error {
	digest := sha3.Sum256([]byte(password))
	encKey := fmt.Sprintf("%x", digest)
	return p.initProtocol(address, encKey, password)
}

// InitProtocolWithEncyptionKey creates an instance of ProtocolService given an address and encryption key.
func (p *Publisher) InitProtocolWithEncyptionKey(address string, encKey string) error {
	return p.initProtocol(address, encKey, "")
}

func (p *Publisher) initProtocol(address, encKey, password string) error {
	if !p.config.PfsEnabled {
		return nil
	}

	if err := os.MkdirAll(filepath.Clean(p.config.DataDir), os.ModePerm); err != nil {
		return err
	}
	v0Path := filepath.Join(p.config.DataDir, fmt.Sprintf("%x.db", address))
	v1Path := filepath.Join(p.config.DataDir, fmt.Sprintf("%p.db", p.config.InstallationID))
	v2Path := filepath.Join(p.config.DataDir, fmt.Sprintf("%p.v2.db", p.config.InstallationID))
	v3Path := filepath.Join(p.config.DataDir, fmt.Sprintf("%p.v3.db", p.config.InstallationID))
	v4Path := filepath.Join(p.config.DataDir, fmt.Sprintf("%p.v4.db", p.config.InstallationID))

	if password != "" {
		if err := chatDB.MigrateDBFile(v0Path, v1Path, "ON", password); err != nil {
			return err
		}

		if err := chatDB.MigrateDBFile(v1Path, v2Path, password, encKey); err != nil {
			// Remove db file as created with a blank password and never used,
			// and there's no need to rekey in this case
			os.Remove(v1Path)
			os.Remove(v2Path)
		}
	}

	if err := chatDB.MigrateDBKeyKdfIterations(v2Path, v3Path, encKey); err != nil {
		os.Remove(v2Path)
		os.Remove(v3Path)
	}

	// Fix IOS not encrypting database
	if err := chatDB.EncryptDatabase(v3Path, v4Path, encKey); err != nil {
		os.Remove(v3Path)
		os.Remove(v4Path)
	}

	// Desktop was passing a network dependent directory, which meant that
	// if running on testnet it would not access the right db. This copies
	// the db from mainnet to the root location.
	networkDependentPath := filepath.Join(p.config.DataDir, "ethereum", "mainnet_rpc", fmt.Sprintf("%p.v4.db", p.config.InstallationID))
	if _, err := os.Stat(networkDependentPath); err == nil {
		if err := os.Rename(networkDependentPath, v4Path); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	persistence, err := chat.NewSQLLitePersistence(v4Path, encKey)
	if err != nil {
		return err
	}

	addedBundlesHandler := func(addedBundles []*multidevice.Installation) {
		handler := SignalHandler{}
		for _, bundle := range addedBundles {
			handler.BundleAdded(bundle.Identity, bundle.ID)
		}
	}

	// Initialize persistence
	p.persistence = NewSQLLitePersistence(persistence.DB)

	// Initialize sharedsecret
	sharedSecretService := sharedsecret.NewService(persistence.GetSharedSecretStorage())
	// Initialize filter
	filterService := filter.New(p.whisper, filter.NewSQLLitePersistence(persistence.DB), sharedSecretService)
	p.filter = filterService

	// Initialize multidevice
	multideviceConfig := &multidevice.Config{
		InstallationID:   p.config.InstallationID,
		ProtocolVersion:  chat.ProtocolVersion,
		MaxInstallations: maxInstallations,
	}
	multideviceService := multidevice.New(multideviceConfig, persistence.GetMultideviceStorage())

	p.protocol = chat.NewProtocolService(
		chat.NewEncryptionService(
			persistence,
			chat.DefaultEncryptionServiceConfig(p.config.InstallationID)),
		sharedSecretService,
		multideviceService,
		addedBundlesHandler,
		p.onNewSharedSecretHandler)

	return nil
}

func (p *Publisher) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, bundle *protobuf.Bundle) ([]*multidevice.Installation, error) {
	if p.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	return p.protocol.ProcessPublicBundle(myIdentityKey, bundle)
}

func (p *Publisher) GetBundle(myIdentityKey *ecdsa.PrivateKey) (*protobuf.Bundle, error) {
	if p.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	return p.protocol.GetBundle(myIdentityKey)
}

// EnableInstallation enables an installation for multi-device sync.
func (p *Publisher) EnableInstallation(installationID string) error {
	if p.protocol == nil {
		return errProtocolNotInitialized
	}

	privateKeyID := p.whisper.SelectedKeyPairID()
	if privateKeyID == "" {
		return errNoKeySelected
	}

	privateKey, err := p.whisper.GetPrivateKey(privateKeyID)
	if err != nil {
		return err
	}

	return p.protocol.EnableInstallation(&privateKey.PublicKey, installationID)
}

// DisableInstallation disables an installation for multi-device sync.
func (p *Publisher) DisableInstallation(installationID string) error {
	if p.protocol == nil {
		return errProtocolNotInitialized
	}

	privateKeyID := p.whisper.SelectedKeyPairID()
	if privateKeyID == "" {
		return errNoKeySelected
	}

	privateKey, err := p.whisper.GetPrivateKey(privateKeyID)
	if err != nil {
		return err
	}

	return p.protocol.DisableInstallation(&privateKey.PublicKey, installationID)
}

// GetOurInstallations returns all the installations available given an identity
func (p *Publisher) GetOurInstallations() ([]*multidevice.Installation, error) {
	if p.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	privateKeyID := p.whisper.SelectedKeyPairID()
	if privateKeyID == "" {
		return nil, errNoKeySelected
	}

	privateKey, err := p.whisper.GetPrivateKey(privateKeyID)
	if err != nil {
		return nil, err
	}

	return p.protocol.GetOurInstallations(&privateKey.PublicKey)
}

// SetInstallationMetadata sets the metadata for our own installation
func (p *Publisher) SetInstallationMetadata(installationID string, data *multidevice.InstallationMetadata) error {
	if p.protocol == nil {
		return errProtocolNotInitialized
	}

	privateKeyID := p.whisper.SelectedKeyPairID()
	if privateKeyID == "" {
		return errNoKeySelected
	}

	privateKey, err := p.whisper.GetPrivateKey(privateKeyID)
	if err != nil {
		return err
	}

	return p.protocol.SetInstallationMetadata(&privateKey.PublicKey, installationID, data)
}

func (p *Publisher) GetPublicBundle(identityKey *ecdsa.PublicKey) (*protobuf.Bundle, error) {
	if p.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	return p.protocol.GetPublicBundle(identityKey)
}

func (p *Publisher) Start(online func() bool, startTicker bool) error {
	p.online = online
	if startTicker {
		p.startTicker()
	}
	return nil
}

func (p *Publisher) Stop() error {
	if p.filter != nil {
		if err := p.filter.Stop(); err != nil {
			log.Error("Failed to stop filter service with error", "err", err)
		}
	}

	return nil
}

func (p *Publisher) getNegotiatedChat(identity *ecdsa.PublicKey) *filter.Chat {
	return p.filter.GetNegotiated(identity)
}

func (p *Publisher) LoadFilters(chats []*filter.Chat) ([]*filter.Chat, error) {
	return p.filter.Init(chats)
}

func (p *Publisher) LoadFilter(chat *filter.Chat) ([]*filter.Chat, error) {
	return p.filter.Load(chat)
}

func (p *Publisher) RemoveFilters(chats []*filter.Chat) error {
	return p.filter.Remove(chats)
}

func (p *Publisher) onNewSharedSecretHandler(sharedSecrets []*sharedsecret.Secret) {
	var filters []*signal.Filter
	for _, sharedSecret := range sharedSecrets {
		chat, err := p.filter.ProcessNegotiatedSecret(sharedSecret)
		if err != nil {
			log.Error("Failed to process negotiated secret", "err", err)
			return
		}

		filter := &signal.Filter{
			ChatID:   chat.ChatID,
			SymKeyID: chat.SymKeyID,
			Listen:   chat.Listen,
			FilterID: chat.FilterID,
			Identity: chat.Identity,
			Topic:    chat.Topic,
		}

		filters = append(filters, filter)

	}
	if len(filters) != 0 {
		handler := SignalHandler{}
		handler.WhisperFilterAdded(filters)
	}

}

func (p *Publisher) ProcessMessage(msg *whisper.Message, msgID []byte) error {
	if !p.config.PfsEnabled {
		return nil
	}

	privateKeyID := p.whisper.SelectedKeyPairID()
	if privateKeyID == "" {
		return errNoKeySelected
	}

	privateKey, err := p.whisper.GetPrivateKey(privateKeyID)
	if err != nil {
		return err
	}

	publicKey, err := crypto.UnmarshalPubkey(msg.Sig)
	if err != nil {
		return err
	}

	// Unmarshal message
	protocolMessage := &protobuf.ProtocolMessage{}

	if err := proto.Unmarshal(msg.Payload, protocolMessage); err != nil {
		p.log.Debug("Not a protocol message", "err", err)
		return nil
	}

	response, err := p.protocol.HandleMessage(privateKey, publicKey, protocolMessage, msgID)

	switch err {
	case nil:
		// Set the decrypted payload
		msg.Payload = response
	case chat.ErrDeviceNotFound:
		// Notify that someone tried to contact us using an invalid bundle
		if privateKey.PublicKey != *publicKey {
			p.log.Warn("Device not found, sending signal", "err", err)
			keyString := fmt.Sprintf("0x%x", crypto.FromECDSAPub(publicKey))
			handler := SignalHandler{}
			handler.DecryptMessageFailed(keyString)
		}
	default:
		// Log and pass to the client, even if failed to decrypt
		p.log.Error("Failed handling message with error", "err", err)
	}

	return nil
}

// CreateDirectMessage creates a 1:1 chat message
func (p *Publisher) CreateDirectMessage(signature string, destination hexutil.Bytes, DH bool, payload []byte) (*whisper.NewMessage, error) {
	if !p.config.PfsEnabled {
		return nil, ErrPFSNotEnabled
	}

	privateKey, err := p.whisper.GetPrivateKey(signature)
	if err != nil {
		return nil, err
	}

	publicKey, err := crypto.UnmarshalPubkey(destination)
	if err != nil {
		return nil, err
	}

	var msgSpec *chat.ProtocolMessageSpec

	if DH {
		p.log.Debug("Building dh message")
		msgSpec, err = p.protocol.BuildDHMessage(privateKey, publicKey, payload)
	} else {
		p.log.Debug("Building direct message")
		msgSpec, err = p.protocol.BuildDirectMessage(privateKey, publicKey, payload)
	}
	if err != nil {
		return nil, err
	}

	whisperMessage, err := p.directMessageToWhisper(privateKey, publicKey, destination, signature, msgSpec)
	if err != nil {
		p.log.Error("sshext-service", "error building whisper message", err)
		return nil, err
	}

	return whisperMessage, nil
}

func (p *Publisher) directMessageToWhisper(myPrivateKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, destination hexutil.Bytes, signature string, spec *chat.ProtocolMessageSpec) (*whisper.NewMessage, error) {
	// marshal for sending to wire
	marshaledMessage, err := proto.Marshal(spec.Message)
	if err != nil {
		p.log.Error("encryption-service", "error marshaling message", err)
		return nil, err
	}

	whisperMessage := whisperutils.DefaultWhisperMessage()
	whisperMessage.Payload = marshaledMessage
	whisperMessage.Sig = signature

	if spec.SharedSecret != nil {
		chat := p.getNegotiatedChat(theirPublicKey)
		if chat != nil {
			p.log.Debug("Sending on negotiated topic", "public-key", destination)
			whisperMessage.SymKeyID = chat.SymKeyID
			whisperMessage.Topic = chat.Topic
			whisperMessage.PublicKey = nil
			return &whisperMessage, nil
		}
	} else if spec.PartitionedTopic() == chat.PartitionTopicV1 {
		p.log.Debug("Sending on partitioned topic", "public-key", destination)
		// Create filter on demand
		if _, err := p.filter.LoadPartitioned(myPrivateKey, theirPublicKey, false); err != nil {
			return nil, err
		}
		t := filter.PublicKeyToPartitionedTopicBytes(theirPublicKey)
		whisperMessage.Topic = whisper.BytesToTopic(t)
		whisperMessage.PublicKey = destination
		return &whisperMessage, nil
	}

	p.log.Debug("Sending on old discovery topic", "public-key", destination)
	whisperMessage.Topic = whisperutils.DiscoveryTopicBytes
	whisperMessage.PublicKey = destination

	return &whisperMessage, nil
}

// CreatePublicMessage sends a public chat message to the underlying transport
func (p *Publisher) CreatePublicMessage(signature string, chatID string, payload []byte, wrap bool) (*whisper.NewMessage, error) {
	if !p.config.PfsEnabled {
		return nil, ErrPFSNotEnabled
	}

	filter := p.filter.GetByID(chatID)
	if filter == nil {
		return nil, errors.New("not subscribed to chat")
	}
	p.log.Info("SIG", signature)

	// Enrich with transport layer info
	whisperMessage := whisperutils.DefaultWhisperMessage()
	whisperMessage.Sig = signature
	whisperMessage.Topic = whisperutils.ToTopic(chatID)
	whisperMessage.SymKeyID = filter.SymKeyID

	if wrap {
		privateKeyID := p.whisper.SelectedKeyPairID()
		if privateKeyID == "" {
			return nil, errNoKeySelected
		}

		privateKey, err := p.whisper.GetPrivateKey(privateKeyID)
		if err != nil {
			return nil, err
		}

		message, err := p.protocol.BuildPublicMessage(privateKey, payload)
		if err != nil {
			return nil, err
		}
		marshaledMessage, err := proto.Marshal(message)
		if err != nil {
			p.log.Error("encryption-service", "error marshaling message", err)
			return nil, err
		}
		whisperMessage.Payload = marshaledMessage

	} else {
		whisperMessage.Payload = payload
	}

	return &whisperMessage, nil
}

func (p *Publisher) ConfirmMessagesProcessed(ids [][]byte) error {
	return p.protocol.ConfirmMessagesProcessed(ids)
}

func (p *Publisher) startTicker() {
	p.ticker = time.NewTicker(tickerInterval * time.Second)
	p.quit = make(chan struct{})
	go func() {
		for {
			select {
			case <-p.ticker.C:
				_, err := p.sendContactCode()
				if err != nil {
					p.log.Error("could not execute tick", "err", err)
				}
			case <-p.quit:
				p.ticker.Stop()
				return
			}
		}
	}()
}

func (p *Publisher) sendContactCode() (*whisper.NewMessage, error) {
	p.log.Info("publishing bundle")
	if !p.config.PfsEnabled {
		return nil, nil
	}

	lastPublished, err := p.persistence.Get()
	if err != nil {
		p.log.Error("could not fetch config from db", "err", err)
		return nil, err
	}

	now := time.Now().Unix()

	if now-lastPublished < publishInterval {
		fmt.Println("NOTHING")
		p.log.Debug("nothing to do")
		return nil, nil
	}

	if !p.online() {
		p.log.Debug("not connected")
		return nil, nil
	}

	privateKeyID := p.whisper.SelectedKeyPairID()
	if privateKeyID == "" {
		return nil, errNoKeySelected
	}

	privateKey, err := p.whisper.GetPrivateKey(privateKeyID)
	if err != nil {
		return nil, err
	}

	identity := fmt.Sprintf("%x", crypto.FromECDSAPub(&privateKey.PublicKey))

	message, err := p.CreatePublicMessage("0x"+identity, filter.ContactCodeTopic(identity), nil, true)
	if err != nil {
		p.log.Error("could not build contact code", "identity", identity, "err", err)
		return nil, err
	}

	_, err = p.whisperAPI.Post(context.TODO(), *message)
	if err != nil {
		p.log.Error("could not publish contact code on whisper", "identity", identity, "err", err)
		return nil, err
	}

	err = p.persistence.Set(now)
	if err != nil {
		p.log.Error("could not set last published", "err", err)
		return nil, err
	}

	return message, nil
}
