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
	"github.com/status-im/status-go/services/shhext/chat"
	chatDB "github.com/status-im/status-go/services/shhext/chat/db"
	"github.com/status-im/status-go/services/shhext/chat/multidevice"
	"github.com/status-im/status-go/services/shhext/chat/protobuf"
	"github.com/status-im/status-go/services/shhext/chat/sharedsecret"
	"github.com/status-im/status-go/services/shhext/dedup"
	"github.com/status-im/status-go/services/shhext/filter"
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
	errProtocolNotInitialized = errors.New("procotol is not initialized")
	// ErrPFSNotEnabled is returned when an endpoint PFS only is called but
	// PFS is disabled
	ErrPFSNotEnabled = errors.New("pfs not enabled")
)

type Service struct {
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

func New(config *Config, w *whisper.Whisper) *Service {
	return &Service{
		config:     config,
		whisper:    w,
		whisperAPI: whisper.NewPublicWhisperAPI(w),
		log:        log.New("package", "status-go/services/publisher.Service"),
	}
}

// InitProtocolWithPassword creates an instance of ProtocolService given an address and password used to generate an encryption key.
func (s *Service) InitProtocolWithPassword(address string, password string) error {
	digest := sha3.Sum256([]byte(password))
	encKey := fmt.Sprintf("%x", digest)
	return s.initProtocol(address, encKey, password)
}

// InitProtocolWithEncyptionKey creates an instance of ProtocolService given an address and encryption key.
func (s *Service) InitProtocolWithEncyptionKey(address string, encKey string) error {
	return s.initProtocol(address, encKey, "")
}

func (s *Service) initProtocol(address, encKey, password string) error {
	if !s.config.PfsEnabled {
		return nil
	}

	if err := os.MkdirAll(filepath.Clean(s.config.DataDir), os.ModePerm); err != nil {
		return err
	}
	v0Path := filepath.Join(s.config.DataDir, fmt.Sprintf("%x.db", address))
	v1Path := filepath.Join(s.config.DataDir, fmt.Sprintf("%s.db", s.config.InstallationID))
	v2Path := filepath.Join(s.config.DataDir, fmt.Sprintf("%s.v2.db", s.config.InstallationID))
	v3Path := filepath.Join(s.config.DataDir, fmt.Sprintf("%s.v3.db", s.config.InstallationID))
	v4Path := filepath.Join(s.config.DataDir, fmt.Sprintf("%s.v4.db", s.config.InstallationID))

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
	networkDependentPath := filepath.Join(s.config.DataDir, "ethereum", "mainnet_rpc", fmt.Sprintf("%s.v4.db", s.config.InstallationID))
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

	addedBundlesHandler := func(addedBundles []*multidevice.IdentityAndID) {
		handler := SignalHandler{}
		for _, bundle := range addedBundles {
			handler.BundleAdded(bundle.Identity, bundle.ID)
		}
	}

	// Initialize persistence

	s.persistence = NewSQLLitePersistence(persistence.DB)

	// Initialize sharedsecret
	sharedSecretService := sharedsecret.NewService(persistence.GetSharedSecretStorage())
	// Initialize filter
	filterService := filter.New(s.whisper, sharedSecretService)
	s.filter = filterService

	// Initialize multidevice
	multideviceConfig := &multidevice.Config{
		InstallationID:   s.config.InstallationID,
		ProtocolVersion:  chat.ProtocolVersion,
		MaxInstallations: maxInstallations,
	}
	multideviceService := multidevice.New(multideviceConfig, persistence.GetMultideviceStorage())

	s.protocol = chat.NewProtocolService(
		chat.NewEncryptionService(
			persistence,
			chat.DefaultEncryptionServiceConfig(s.config.InstallationID)),
		sharedSecretService,
		multideviceService,
		addedBundlesHandler,
		s.onNewSharedSecretHandler)

	return nil
}

func (s *Service) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, bundle *protobuf.Bundle) ([]*multidevice.IdentityAndID, error) {
	if s.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	return s.protocol.ProcessPublicBundle(myIdentityKey, bundle)
}

func (s *Service) GetBundle(myIdentityKey *ecdsa.PrivateKey) (*protobuf.Bundle, error) {
	if s.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	return s.protocol.GetBundle(myIdentityKey)
}

// EnableInstallation enables an installation for multi-device sync.
func (s *Service) EnableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	if s.protocol == nil {
		return errProtocolNotInitialized
	}

	return s.protocol.EnableInstallation(myIdentityKey, installationID)
}

func (s *Service) GetPublicBundle(identityKey *ecdsa.PublicKey) (*protobuf.Bundle, error) {
	if s.protocol == nil {
		return nil, errProtocolNotInitialized
	}

	return s.protocol.GetPublicBundle(identityKey)
}

// DisableInstallation disables an installation for multi-device sync.
func (s *Service) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	if s.protocol == nil {
		return errProtocolNotInitialized
	}

	return s.protocol.DisableInstallation(myIdentityKey, installationID)
}

func (s *Service) Start(online func() bool, startTicker bool) error {
	s.online = online
	if startTicker {
		s.startTicker()
	}
	return nil
}

func (s *Service) Stop() error {
	if s.filter != nil {
		if err := s.filter.Stop(); err != nil {
			log.Error("Failed to stop filter service with error", "err", err)
		}
	}

	return nil
}

func (s *Service) getNegotiatedChat(identity *ecdsa.PublicKey) *filter.Chat {
	return s.filter.GetNegotiated(identity)
}

func (s *Service) LoadFilters(chats []*filter.Chat) ([]*filter.Chat, error) {
	return s.filter.Init(chats)
}

func (s *Service) LoadFilter(chat *filter.Chat) ([]*filter.Chat, error) {
	return s.filter.Load(chat)
}

func (s *Service) RemoveFilters(chats []*filter.Chat) error {
	return s.filter.Remove(chats)
}

func (s *Service) onNewSharedSecretHandler(sharedSecrets []*sharedsecret.Secret) {
	var filters []*signal.Filter
	for _, sharedSecret := range sharedSecrets {
		chat, err := s.filter.ProcessNegotiatedSecret(sharedSecret)
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

func (s *Service) ProcessMessage(dedupMessage dedup.DeduplicateMessage) error {
	if !s.config.PfsEnabled {
		return nil
	}
	msg := dedupMessage.Message

	privateKeyID := s.whisper.SelectedKeyPairID()
	if privateKeyID == "" {
		return errors.New("no key selected")
	}

	privateKey, err := s.whisper.GetPrivateKey(privateKeyID)
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
		s.log.Debug("Not a protocol message", "err", err)
		return nil
	}

	response, err := s.protocol.HandleMessage(privateKey, publicKey, protocolMessage, dedupMessage.DedupID)

	switch err {
	case nil:
		// Set the decrypted payload
		msg.Payload = response
	case chat.ErrDeviceNotFound:
		// Notify that someone tried to contact us using an invalid bundle
		if privateKey.PublicKey != *publicKey {
			s.log.Warn("Device not found, sending signal", "err", err)
			keyString := fmt.Sprintf("0x%x", crypto.FromECDSAPub(publicKey))
			handler := SignalHandler{}
			handler.DecryptMessageFailed(keyString)
		}
	default:
		// Log and pass to the client, even if failed to decrypt
		s.log.Error("Failed handling message with error", "err", err)
	}

	return nil
}

// CreateDirectMessage creates a 1:1 chat message
func (s *Service) CreateDirectMessage(signature string, destination hexutil.Bytes, DH bool, payload []byte) (*whisper.NewMessage, error) {
	if !s.config.PfsEnabled {
		return nil, ErrPFSNotEnabled
	}

	privateKey, err := s.whisper.GetPrivateKey(signature)
	if err != nil {
		return nil, err
	}

	publicKey, err := crypto.UnmarshalPubkey(destination)
	if err != nil {
		return nil, err
	}

	var msgSpec *chat.ProtocolMessageSpec

	if DH {
		s.log.Debug("Building dh message")
		msgSpec, err = s.protocol.BuildDHMessage(privateKey, publicKey, payload)
	} else {
		s.log.Debug("Building direct message")
		msgSpec, err = s.protocol.BuildDirectMessage(privateKey, publicKey, payload)
	}
	if err != nil {
		return nil, err
	}

	whisperMessage, err := s.directMessageToWhisper(privateKey, publicKey, destination, signature, msgSpec)
	if err != nil {
		s.log.Error("sshext-service", "error building whisper message", err)
		return nil, err
	}

	return whisperMessage, nil
}

func (s *Service) directMessageToWhisper(myPrivateKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, destination hexutil.Bytes, signature string, spec *chat.ProtocolMessageSpec) (*whisper.NewMessage, error) {
	// marshal for sending to wire
	marshaledMessage, err := proto.Marshal(spec.Message)
	if err != nil {
		s.log.Error("encryption-service", "error marshaling message", err)
		return nil, err
	}

	whisperMessage := whisperutils.DefaultWhisperMessage()
	whisperMessage.Payload = marshaledMessage
	whisperMessage.Sig = signature

	if spec.SharedSecret != nil {
		chat := s.getNegotiatedChat(theirPublicKey)
		if chat != nil {
			s.log.Debug("Sending on negotiated topic")
			whisperMessage.SymKeyID = chat.SymKeyID
			whisperMessage.Topic = chat.Topic
			whisperMessage.PublicKey = nil
			return &whisperMessage, nil
		}
	} else if spec.PartitionedTopic() == chat.PartitionTopicV1 {
		s.log.Debug("Sending on partitioned topic")
		// Create filter on demand
		if _, err := s.filter.LoadPartitioned(myPrivateKey, theirPublicKey, false); err != nil {
			return nil, err
		}
		t := filter.PublicKeyToPartitionedTopicBytes(theirPublicKey)
		whisperMessage.Topic = whisper.BytesToTopic(t)
		whisperMessage.PublicKey = destination
		return &whisperMessage, nil
	}

	s.log.Debug("Sending on old discovery topic")
	whisperMessage.Topic = whisperutils.DiscoveryTopicBytes
	whisperMessage.PublicKey = destination

	return &whisperMessage, nil
}

// CreatePublicMessage sends a public chat message to the underlying transport
func (s *Service) CreatePublicMessage(signature string, chatID string, payload []byte, wrap bool) (*whisper.NewMessage, error) {
	if !s.config.PfsEnabled {
		return nil, ErrPFSNotEnabled
	}

	filter := s.filter.GetByID(chatID)
	if filter == nil {
		return nil, errors.New("not subscribed to chat")
	}
	s.log.Info("SIG", signature)

	// Enrich with transport layer info
	whisperMessage := whisperutils.DefaultWhisperMessage()
	whisperMessage.Sig = signature
	whisperMessage.Topic = whisperutils.ToTopic(chatID)
	whisperMessage.SymKeyID = filter.SymKeyID

	if wrap {
		privateKeyID := s.whisper.SelectedKeyPairID()
		if privateKeyID == "" {
			return nil, errors.New("no key selected")
		}

		privateKey, err := s.whisper.GetPrivateKey(privateKeyID)
		if err != nil {
			return nil, err
		}

		message, err := s.protocol.BuildPublicMessage(privateKey, payload)
		if err != nil {
			return nil, err
		}
		marshaledMessage, err := proto.Marshal(message)
		if err != nil {
			s.log.Error("encryption-service", "error marshaling message", err)
			return nil, err
		}
		whisperMessage.Payload = marshaledMessage

	} else {
		whisperMessage.Payload = payload
	}

	return &whisperMessage, nil
}

func (s *Service) ConfirmMessagesProcessed(ids [][]byte) error {
	return s.protocol.ConfirmMessagesProcessed(ids)
}

func (s *Service) startTicker() {
	s.ticker = time.NewTicker(tickerInterval * time.Second)
	s.quit = make(chan struct{})
	go func() {
		for {
			select {
			case <-s.ticker.C:
				_, err := s.sendContactCode()
				if err != nil {
					s.log.Error("could not execute tick", "err", err)
				}
			case <-s.quit:
				s.ticker.Stop()
				return
			}
		}
	}()
}

func (s *Service) sendContactCode() (*whisper.NewMessage, error) {
	s.log.Info("publishing bundle")
	if !s.config.PfsEnabled {
		return nil, nil
	}

	lastPublished, err := s.persistence.Get()
	if err != nil {
		s.log.Error("could not fetch config from db", "err", err)
		return nil, err
	}

	now := time.Now().Unix()

	if now-lastPublished < publishInterval {
		fmt.Println("NOTHING")
		s.log.Debug("nothing to do")
		return nil, nil
	}

	if !s.online() {
		s.log.Debug("not connected")
		return nil, nil
	}

	privateKeyID := s.whisper.SelectedKeyPairID()
	if privateKeyID == "" {
		return nil, errors.New("no key selected")
	}

	privateKey, err := s.whisper.GetPrivateKey(privateKeyID)
	if err != nil {
		return nil, err
	}

	identity := fmt.Sprintf("%x", crypto.FromECDSAPub(&privateKey.PublicKey))

	message, err := s.CreatePublicMessage("0x"+identity, filter.ContactCodeTopic(identity), nil, true)
	if err != nil {
		s.log.Error("could not build contact code", "identity", identity, "err", err)
		return nil, err
	}

	_, err = s.whisperAPI.Post(context.TODO(), *message)
	if err != nil {
		s.log.Error("could not publish contact code on whisper", "identity", identity, "err", err)
		return nil, err
	}

	err = s.persistence.Set(now)
	if err != nil {
		s.log.Error("could not set last published", "err", err)
		return nil, err
	}

	return message, nil
}
