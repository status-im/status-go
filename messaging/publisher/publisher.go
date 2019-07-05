package publisher

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/messaging/chat"
	"github.com/status-im/status-go/messaging/chat/protobuf"
	"github.com/status-im/status-go/messaging/filter"
	"github.com/status-im/status-go/messaging/multidevice"
	"github.com/status-im/status-go/messaging/sharedsecret"

	"github.com/status-im/status-go/services/shhext/whisperutils"

	"github.com/golang/protobuf/proto"
	whisper "github.com/status-im/whisper/whisperv6"
)

const (
	tickerInterval = 120
	// How often we should publish a contact code in seconds
	publishInterval = 21600
	// How often we should check for new messages
	pollIntervalMs = 300
)

var (
	errProtocolNotInitialized = errors.New("protocol is not initialized")
	// ErrPFSNotEnabled is returned when an endpoint PFS only is called but
	// PFS is disabled.
	ErrPFSNotEnabled = errors.New("pfs not enabled")
	errNoKeySelected = errors.New("no key selected")
	// ErrNoProtocolMessage means that a message was not a protocol message,
	// that is it could not be unmarshaled.
	ErrNoProtocolMessage = errors.New("not a protocol message")
)

type Publisher struct {
	config      Config
	whisper     *whisper.Whisper
	online      func() bool
	whisperAPI  *whisper.PublicWhisperAPI
	protocol    *chat.ProtocolService
	persistence Persistence
	log         log.Logger
	filter      *filter.Service
	quit        chan struct{}
	ticker      *time.Ticker
}

type Config struct {
	PFSEnabled bool
}

func New(w *whisper.Whisper, c Config) *Publisher {
	return &Publisher{
		config:     c,
		whisper:    w,
		whisperAPI: whisper.NewPublicWhisperAPI(w),
		log:        log.New("package", "status-go/messaging/publisher.Publisher"),
	}
}

func (p *Publisher) Init(db *sql.DB, protocol *chat.ProtocolService, onNewMessagesHandler func([]*filter.Messages)) {

	filterService := filter.New(p.whisper, filter.NewSQLLitePersistence(db), protocol.GetSharedSecretService(), onNewMessagesHandler)

	p.persistence = NewSQLLitePersistence(db)
	p.protocol = protocol
	p.filter = filterService
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
	if p.protocol == nil {
		return errProtocolNotInitialized
	}

	p.online = online
	if startTicker {
		p.startTicker()
	}
	go p.filter.Start(pollIntervalMs * time.Millisecond)
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
func (p *Publisher) ProcessNegotiatedSecret(secrets []*sharedsecret.Secret) {
	for _, secret := range secrets {
		_, err := p.filter.ProcessNegotiatedSecret(secret)
		if err != nil {
			log.Error("could not process negotiated filter", "err", err)
		}
	}
}

func (p *Publisher) ProcessMessage(msg *whisper.Message, msgID []byte) error {
	if !p.config.PFSEnabled {
		return ErrPFSNotEnabled
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
		return ErrNoProtocolMessage
	}

	response, err := p.protocol.HandleMessage(privateKey, publicKey, protocolMessage, msgID)
	if err == nil {
		msg.Payload = response
	}
	return err
}

// CreateDirectMessage creates a 1:1 chat message
func (p *Publisher) CreateDirectMessage(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, DH bool, payload []byte) (*whisper.NewMessage, error) {
	if !p.config.PFSEnabled {
		return nil, ErrPFSNotEnabled
	}

	var (
		msgSpec *chat.ProtocolMessageSpec
		err     error
	)

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

	whisperMessage, err := p.directMessageToWhisper(privateKey, publicKey, msgSpec)
	if err != nil {
		p.log.Error("sshext-service", "error building whisper message", err)
		return nil, err
	}

	return whisperMessage, nil
}

func (p *Publisher) directMessageToWhisper(myPrivateKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, spec *chat.ProtocolMessageSpec) (*whisper.NewMessage, error) {
	// marshal for sending to wire
	marshaledMessage, err := proto.Marshal(spec.Message)
	if err != nil {
		p.log.Error("encryption-service", "error marshaling message", err)
		return nil, err
	}

	// We rely on the fact that a deterministic ID is created for the same keys.
	sigID, err := p.whisper.AddKeyPair(myPrivateKey)
	if err != nil {
		p.log.Error("failed to add key pair in order to get signature ID", "err", err)
		return nil, err
	}

	destination := hexutil.Bytes(crypto.FromECDSAPub(theirPublicKey))

	whisperMessage := whisperutils.DefaultWhisperMessage()
	whisperMessage.Payload = marshaledMessage
	whisperMessage.Sig = sigID

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
func (p *Publisher) CreatePublicMessage(privateKey *ecdsa.PrivateKey, chatID string, payload []byte, wrap bool) (*whisper.NewMessage, error) {
	if !p.config.PFSEnabled {
		return nil, ErrPFSNotEnabled
	}

	filter := p.filter.GetByID(chatID)
	if filter == nil {
		return nil, errors.New("not subscribed to chat")
	}

	sigID, err := p.whisper.AddKeyPair(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get a signature ID: %v", err)
	}

	p.log.Info("signature ID", sigID)

	// Enrich with transport layer info
	whisperMessage := whisperutils.DefaultWhisperMessage()
	whisperMessage.Sig = sigID
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
	if !p.config.PFSEnabled {
		return nil, nil
	}

	if p.persistence == nil {
		p.log.Info("not initialized, skipping")
		return nil, nil
	}

	lastPublished, err := p.persistence.Get()
	if err != nil {
		p.log.Error("could not fetch config from db", "err", err)
		return nil, err
	}

	now := time.Now().Unix()

	if now-lastPublished < publishInterval {
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

	message, err := p.CreatePublicMessage(privateKey, filter.ContactCodeTopic(identity), nil, true)
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
