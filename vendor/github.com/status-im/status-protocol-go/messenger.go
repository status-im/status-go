package statusproto

import (
	"context"
	"crypto/ecdsa"
	"log"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	whisper "github.com/status-im/whisper/whisperv6"

	"github.com/status-im/status-protocol-go/encryption"
	"github.com/status-im/status-protocol-go/encryption/multidevice"
	"github.com/status-im/status-protocol-go/encryption/sharedsecret"
	migrations "github.com/status-im/status-protocol-go/internal/sqlite"
	"github.com/status-im/status-protocol-go/sqlite"
	transport "github.com/status-im/status-protocol-go/transport/whisper"
	"github.com/status-im/status-protocol-go/transport/whisper/filter"
	protocol "github.com/status-im/status-protocol-go/v1"
)

var (
	ErrChatIDEmpty    = errors.New("chat ID is empty")
	ErrNotImplemented = errors.New("not implemented")
)

// Messenger is a entity managing chats and messages.
// It acts as a bridge between the application and encryption
// layers.
// It needs to expose an interface to manage installations
// because installations are managed by the user.
// Similarly, it needs to expose an interface to manage
// mailservers because they can also be managed by the user.
type Messenger struct {
	identity    *ecdsa.PrivateKey
	persistence persistence
	adapter     *whisperAdapter
	encryptor   *encryption.Protocol

	ownMessages map[string][]*protocol.Message
}

type config struct {
	onNewInstallationsHandler func([]*multidevice.Installation)
	onNewSharedSecretHandler  func([]*sharedsecret.Secret)
	onSendContactCodeHandler  func(*encryption.ProtocolMessageSpec)

	publicChatNames []string
	publicKeys      []*ecdsa.PublicKey
	secrets         []filter.NegotiatedSecret
}

type Option func(*config) error

func WithOnNewInstallationsHandler(h func([]*multidevice.Installation)) func(c *config) error {
	return func(c *config) error {
		c.onNewInstallationsHandler = h
		return nil
	}
}

func WithOnNewSharedSecret(h func([]*sharedsecret.Secret)) func(c *config) error {
	return func(c *config) error {
		c.onNewSharedSecretHandler = h
		return nil
	}
}

func WithChats(
	publicChatNames []string,
	publicKeys []*ecdsa.PublicKey,
	secrets []filter.NegotiatedSecret,
) func(c *config) error {
	return func(c *config) error {
		c.publicChatNames = publicChatNames
		c.publicKeys = publicKeys
		c.secrets = secrets
		return nil
	}
}

func NewMessenger(
	identity *ecdsa.PrivateKey,
	server transport.Server,
	shh *whisper.Whisper,
	dataDir string,
	dbKey string,
	installationID string,
	opts ...Option,
) (*Messenger, error) {
	var messenger *Messenger

	// Set default config fields.
	c := config{
		onNewInstallationsHandler: func(installations []*multidevice.Installation) {
			for _, installation := range installations {
				log.Printf(
					"[onNewInstallationsHandler] received a new installation %s from %s",
					installation.Identity, installation.ID,
				)
			}
		},
		onNewSharedSecretHandler: func(secrets []*sharedsecret.Secret) {
			if err := messenger.handleSharedSecrets(secrets); err != nil {
				log.Printf("[onNewSharedSecretHandler] failed to process secrets: %v", err)
			}
		},
		onSendContactCodeHandler: func(messageSpec *encryption.ProtocolMessageSpec) {
			log.Printf("[onSendContactCodeHandler] received a SendContactCode request")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := messenger.adapter.SendContactCode(ctx, messageSpec)
			if err != nil {
				log.Printf("[onSendContactCodeHandler] failed to send a contact code: %v", err)
			}
		},
	}

	for _, opt := range opts {
		if err := opt(&c); err != nil {
			return nil, err
		}
	}

	t, err := transport.NewWhisperServiceTransport(
		server,
		shh,
		identity,
		dataDir,
		dbKey,
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a WhisperServiceTransport")
	}

	if _, err := t.Init(c.publicChatNames, c.publicKeys, c.secrets); err != nil {
		return nil, errors.Wrap(err, "failed to initialize WhisperServiceTransport")
	}

	encryptionProtocol, err := encryption.New(
		dataDir,
		dbKey,
		installationID,
		c.onNewInstallationsHandler,
		c.onNewSharedSecretHandler,
		c.onSendContactCodeHandler,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create the encryption layer")
	}
	// TODO: consider removing identity as an argument to Start().
	encryptionProtocol.Start(identity)

	messagesDB, err := sqlite.Open(filepath.Join(dataDir, "messages.sql"), dbKey, sqlite.MigrationConfig{
		AssetNames: migrations.AssetNames(),
		AssetGetter: func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize messages db")
	}

	messenger = &Messenger{
		identity:    identity,
		persistence: &sqlitePersistence{db: messagesDB},
		adapter:     newWhisperAdapter(identity, t, encryptionProtocol),
		encryptor:   encryptionProtocol,
		ownMessages: make(map[string][]*protocol.Message),
	}

	return messenger, nil
}

func (m *Messenger) handleSharedSecrets(secrets []*sharedsecret.Secret) error {
	return m.adapter.handleSharedSecrets(secrets)
}

func (m *Messenger) EnableInstallation(id string) error {
	return m.encryptor.EnableInstallation(&m.identity.PublicKey, id)
}

func (m *Messenger) DisableInstallation(id string) error {
	return m.encryptor.DisableInstallation(&m.identity.PublicKey, id)
}

func (m *Messenger) Installations() ([]*multidevice.Installation, error) {
	return m.encryptor.GetOurInstallations(&m.identity.PublicKey)
}

func (m *Messenger) SetInstallationMetadata(id string, data *multidevice.InstallationMetadata) error {
	return m.encryptor.SetInstallationMetadata(&m.identity.PublicKey, id, data)
}

// NOT_IMPLEMENTED
func (m *Messenger) SelectMailserver(id string) error {
	return ErrNotImplemented
}

// NOT_IMPLEMENTED
func (m *Messenger) AddMailserver(enode string) error {
	return ErrNotImplemented
}

// NOT_IMPLEMENTED
func (m *Messenger) RemoveMailserver(id string) error {
	return ErrNotImplemented
}

// NOT_IMPLEMENTED
func (m *Messenger) Mailservers() ([]string, error) {
	return nil, ErrNotImplemented
}

func (m *Messenger) Join(chat Chat) error {
	if chat.PublicKey() != nil {
		return m.adapter.JoinPrivate(chat.PublicKey())
	} else if chat.PublicName() != "" {
		return m.adapter.JoinPublic(chat.PublicName())
	}
	return errors.New("chat is neither public nor private")
}

func (m *Messenger) Leave(chat Chat) error {
	if chat.PublicKey() != nil {
		return m.adapter.LeavePrivate(chat.PublicKey())
	} else if chat.PublicName() != "" {
		return m.adapter.LeavePublic(chat.PublicName())
	}
	return errors.New("chat is neither public nor private")
}

func (m *Messenger) Send(ctx context.Context, chat Chat, data []byte) ([]byte, error) {
	chatID := chat.ID()
	if chatID == "" {
		return nil, ErrChatIDEmpty
	}

	clock, err := m.persistence.LastMessageClock(chat.ID())
	if err != nil {
		return nil, err
	}

	if chat.PublicKey() != nil {
		hash, message, err := m.adapter.SendPrivate(ctx, chat.PublicKey(), chat.ID(), data, clock)
		if err != nil {
			return nil, err
		}

		// Save our message because it won't be received from the transport layer.
		message.ID = hash // a Message need ID to be properly stored in the db
		message.SigPubKey = &m.identity.PublicKey
		_, err = m.persistence.SaveMessages(chat.ID(), []*protocol.Message{message})
		if err != nil {
			return nil, err
		}

		// Cache it to be returned in Retrieve().
		m.ownMessages[chatID] = append(m.ownMessages[chatID], message)

		return hash, nil
	} else if chat.PublicName() != "" {
		return m.adapter.SendPublic(ctx, chat.PublicName(), chat.ID(), data, clock)
	}
	return nil, errors.New("chat is neither public nor private")
}

type RetrieveConfig struct {
	From        time.Time
	To          time.Time
	latest      bool
	last24Hours bool
}

var (
	RetrieveLatest  = RetrieveConfig{latest: true}
	RetrieveLastDay = RetrieveConfig{latest: true, last24Hours: true}
)

func (m *Messenger) Retrieve(ctx context.Context, chat Chat, c RetrieveConfig) (messages []*protocol.Message, err error) {
	var latest []*protocol.Message

	if chat.PublicKey() != nil {
		latest, err = m.adapter.RetrievePrivateMessages(chat.PublicKey())
		// Return any own messages for this chat as well.
		if ownMessages, ok := m.ownMessages[chat.ID()]; ok {
			latest = append(latest, ownMessages...)
			delete(m.ownMessages, chat.ID())
		}
	} else if chat.PublicName() != "" {
		latest, err = m.adapter.RetrievePublicMessages(chat.PublicName())
	} else {
		return nil, errors.New("chat is neither public nor private")
	}

	if err != nil {
		err = errors.Wrap(err, "failed to retrieve messages")
		return
	}

	_, err = m.persistence.SaveMessages(chat.ID(), latest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save latest messages")
	}

	for _, message := range latest {
		if err := m.encryptor.ConfirmMessageProcessed(message.ID); err != nil {
			return nil, errors.Wrap(err, "failed to confirm message being processed")
		}
	}

	return m.retrieveSaved(ctx, chat, c, latest)
}

func (m *Messenger) retrieveSaved(ctx context.Context, chat Chat, c RetrieveConfig, latest []*protocol.Message) (messages []*protocol.Message, err error) {
	if !c.latest {
		return m.persistence.Messages(chat.ID(), c.From, c.To)
	}

	if c.last24Hours {
		to := time.Now()
		from := to.Add(-time.Hour * 24)
		return m.persistence.Messages(chat.ID(), from, to)
	}

	return latest, nil
}

// LEGACY
func (m *Messenger) RetrieveAllRaw() (map[filter.Chat][]*whisper.Message, error) {
	return m.adapter.RetrieveAllRaw()
}

// LEGACY
func (m *Messenger) ConfirmMessagesProcessed(messageIDs [][]byte) error {
	for _, id := range messageIDs {
		if err := m.encryptor.ConfirmMessageProcessed(id); err != nil {
			return err
		}
	}
	return nil
}
