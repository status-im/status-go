package protocol

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/golang/protobuf/proto"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	enstypes "github.com/status-im/status-go/eth-node/types/ens"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/encryption/sharedsecret"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/identity/identicon"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	transport "github.com/status-im/status-go/protocol/transport/whisper"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

const PubKeyStringLength = 132

var (
	ErrChatIDEmpty    = errors.New("chat ID is empty")
	ErrNotImplemented = errors.New("not implemented")

	errChatNotFound = errors.New("chat not found")
)

// Messenger is a entity managing chats and messages.
// It acts as a bridge between the application and encryption
// layers.
// It needs to expose an interface to manage installations
// because installations are managed by the user.
// Similarly, it needs to expose an interface to manage
// mailservers because they can also be managed by the user.
type Messenger struct {
	node        types.Node
	identity    *ecdsa.PrivateKey
	persistence *sqlitePersistence
	transport   *transport.WhisperServiceTransport
	encryptor   *encryption.Protocol
	processor   *messageProcessor
	logger      *zap.Logger

	featureFlags               featureFlags
	messagesPersistenceEnabled bool
	shutdownTasks              []func() error
}

type RawResponse struct {
	Filter   *transport.Filter           `json:"filter"`
	Messages []*v1protocol.StatusMessage `json:"messages"`
}

type MessengerResponse struct {
	Chats    []*Chat    `json:"chats,omitEmpty"`
	Messages []*Message `json:"messages,omitEmpty"`
	Contacts []*Contact `json:"contacts,omitEmpty"`
	// Raw unprocessed messages
	RawMessages []*RawResponse `json:"rawMessages,omitEmpty"`
}

func (m *MessengerResponse) IsEmpty() bool {
	return len(m.Chats) == 0 && len(m.Messages) == 0 && len(m.Contacts) == 0 && len(m.RawMessages) == 0
}

type featureFlags struct {
	// datasync indicates whether direct messages should be sent exclusively
	// using datasync, breaking change for non-v1 clients. Public messages
	// are not impacted
	datasync bool
}

type dbConfig struct {
	dbPath string
	dbKey  string
}

type config struct {
	onNewInstallationsHandler func([]*multidevice.Installation)
	// This needs to be exposed until we move here mailserver logic
	// as otherwise the client is not notified of a new filter and
	// won't be pulling messages from mailservers until it reloads the chats/filters
	onNegotiatedFilters func([]*transport.Filter)
	// DEPRECATED: no need to expose it
	onSendContactCodeHandler func(*encryption.ProtocolMessageSpec)

	// Config for the envelopes monitor
	envelopesMonitorConfig *transport.EnvelopesMonitorConfig

	messagesPersistenceEnabled bool
	featureFlags               featureFlags

	// A path to a database or a database instance is required.
	// The database instance has a higher priority.
	dbConfig dbConfig
	db       *sql.DB

	logger *zap.Logger
}

type Option func(*config) error

func WithOnNewInstallationsHandler(h func([]*multidevice.Installation)) Option {
	return func(c *config) error {
		c.onNewInstallationsHandler = h
		return nil
	}
}

func WithOnNegotiatedFilters(h func([]*transport.Filter)) Option {
	return func(c *config) error {
		c.onNegotiatedFilters = h
		return nil
	}
}

func WithCustomLogger(logger *zap.Logger) Option {
	return func(c *config) error {
		c.logger = logger
		return nil
	}
}

func WithMessagesPersistenceEnabled() Option {
	return func(c *config) error {
		c.messagesPersistenceEnabled = true
		return nil
	}
}

func WithDatabaseConfig(dbPath, dbKey string) Option {
	return func(c *config) error {
		c.dbConfig = dbConfig{dbPath: dbPath, dbKey: dbKey}
		return nil
	}
}

func WithDatabase(db *sql.DB) Option {
	return func(c *config) error {
		c.db = db
		return nil
	}
}

func WithDatasync() func(c *config) error {
	return func(c *config) error {
		c.featureFlags.datasync = true
		return nil
	}
}

func WithEnvelopesMonitorConfig(emc *transport.EnvelopesMonitorConfig) Option {
	return func(c *config) error {
		c.envelopesMonitorConfig = emc
		return nil
	}
}

func NewMessenger(
	identity *ecdsa.PrivateKey,
	node types.Node,
	installationID string,
	opts ...Option,
) (*Messenger, error) {
	var messenger *Messenger

	shh, err := node.GetWhisper(nil)
	if err != nil {
		return nil, err
	}

	c := config{}

	for _, opt := range opts {
		if err := opt(&c); err != nil {
			return nil, err
		}
	}

	logger := c.logger
	if c.logger == nil {
		var err error
		if logger, err = zap.NewDevelopment(); err != nil {
			return nil, errors.Wrap(err, "failed to create a logger")
		}
	}

	// Set default config fields.
	if c.onNewInstallationsHandler == nil {
		c.onNewInstallationsHandler = func(installations []*multidevice.Installation) {
			sugar := logger.Sugar().With("site", "onNewInstallationsHandler")
			for _, installation := range installations {
				sugar.Infow(
					"received a new installation",
					"identity", installation.Identity,
					"id", installation.ID)
			}
		}
	}
	onNewSharedSecretHandler := func(secrets []*sharedsecret.Secret) {
		filters, err := messenger.handleSharedSecrets(secrets)
		if err != nil {
			slogger := logger.With(zap.String("site", "onNewSharedSecretHandler"))
			slogger.Warn("failed to process secrets", zap.Error(err))
		}

		if c.onNegotiatedFilters != nil {
			c.onNegotiatedFilters(filters)
		}
	}
	if c.onSendContactCodeHandler == nil {
		c.onSendContactCodeHandler = func(messageSpec *encryption.ProtocolMessageSpec) {
			slogger := logger.With(zap.String("site", "onSendContactCodeHandler"))
			slogger.Info("received a SendContactCode request")

			newMessage, err := messageSpecToWhisper(messageSpec)
			if err != nil {
				slogger.Warn("failed to convert spec to Whisper message", zap.Error(err))
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			chatName := transport.ContactCodeTopic(&messenger.identity.PublicKey)
			_, err = messenger.transport.SendPublic(ctx, newMessage, chatName)
			if err != nil {
				slogger.Warn("failed to send a contact code", zap.Error(err))
			}
		}
	}

	// Configure the database.
	database := c.db
	if c.db == nil && c.dbConfig == (dbConfig{}) {
		return nil, errors.New("database instance or database path needs to be provided")
	}
	if c.db == nil {
		logger.Info("opening a database", zap.String("dbPath", c.dbConfig.dbPath))
		var err error
		database, err = sqlite.Open(c.dbConfig.dbPath, c.dbConfig.dbKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize database from the db config")
		}
	}

	// Apply migrations for all components.
	err = sqlite.Migrate(database)
	if err != nil {
		return nil, errors.Wrap(err, "failed to apply migrations")
	}

	// Initialize transport layer.
	t, err := transport.NewWhisperServiceTransport(
		shh,
		identity,
		database,
		nil,
		c.envelopesMonitorConfig,
		logger,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a WhisperServiceTransport")
	}

	// Initialize encryption layer.
	encryptionProtocol := encryption.New(
		database,
		installationID,
		c.onNewInstallationsHandler,
		onNewSharedSecretHandler,
		c.onSendContactCodeHandler,
		logger,
	)

	processor, err := newMessageProcessor(
		identity,
		database,
		encryptionProtocol,
		t,
		newPersistentMessageHandler(&sqlitePersistence{db: database}),
		logger,
		c.featureFlags,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create messageProcessor")
	}

	messenger = &Messenger{
		node:                       node,
		identity:                   identity,
		persistence:                &sqlitePersistence{db: database},
		transport:                  t,
		encryptor:                  encryptionProtocol,
		processor:                  processor,
		featureFlags:               c.featureFlags,
		messagesPersistenceEnabled: c.messagesPersistenceEnabled,
		shutdownTasks: []func() error{
			database.Close,
			t.Reset,
			t.Stop,
			func() error { processor.Stop(); return nil },
			// Currently this often fails, seems like it's safe to ignore them
			// https://github.com/uber-go/zap/issues/328
			func() error { _ = logger.Sync; return nil },
		},
		logger: logger,
	}

	// Start all services immediately.
	// TODO: consider removing identity as an argument to Start().
	if err := encryptionProtocol.Start(identity); err != nil {
		return nil, err
	}

	logger.Debug("messages persistence", zap.Bool("enabled", c.messagesPersistenceEnabled))

	return messenger, nil
}

// Init analyzes chats and contacts in order to setup filters
// which are responsible for retrieving messages.
func (m *Messenger) Init() error {
	logger := m.logger.With(zap.String("site", "Init"))

	var (
		publicChatIDs []string
		publicKeys    []*ecdsa.PublicKey
	)

	// Get chat IDs and public keys from the existing chats.
	// TODO: Get only active chats by the query.
	chats, err := m.Chats()
	if err != nil {
		return err
	}
	for _, chat := range chats {
		if !chat.Active {
			continue
		}
		switch chat.ChatType {
		case ChatTypePublic:
			publicChatIDs = append(publicChatIDs, chat.ID)
		case ChatTypeOneToOne:
			publicKeys = append(publicKeys, chat.PublicKey)
		case ChatTypePrivateGroupChat:
			for _, member := range chat.Members {
				publicKey, err := member.PublicKey()
				if err != nil {
					return errors.Wrapf(err, "invalid public key for member %s in chat %s", member.ID, chat.Name)
				}
				publicKeys = append(publicKeys, publicKey)
			}
		default:
			return errors.New("invalid chat type")
		}
	}

	// Get chat IDs and public keys from the contacts.
	contacts, err := m.Contacts()
	if err != nil {
		return err
	}
	for _, contact := range contacts {
		// We only need filters for contacts added by us and not blocked.
		if !contact.IsAdded() || contact.IsBlocked() {
			continue
		}
		publicKey, err := contact.PublicKey()
		if err != nil {
			logger.Error("failed to get contact's public key", zap.Error(err))
			continue
		}
		publicKeys = append(publicKeys, publicKey)
	}

	_, err = m.transport.InitFilters(publicChatIDs, publicKeys)
	return err
}

// Shutdown takes care of ensuring a clean shutdown of Messenger
func (m *Messenger) Shutdown() (err error) {
	for _, task := range m.shutdownTasks {
		if tErr := task(); tErr != nil {
			if err == nil {
				// First error appeared.
				err = tErr
			} else {
				// We return all errors. They will be concatenated in the order of occurrence,
				// however, they will also be returned as a single error.
				err = errors.Wrap(err, tErr.Error())
			}
		}
	}
	return
}

func (m *Messenger) handleSharedSecrets(secrets []*sharedsecret.Secret) ([]*transport.Filter, error) {
	logger := m.logger.With(zap.String("site", "handleSharedSecrets"))
	var result []*transport.Filter
	for _, secret := range secrets {
		logger.Debug("received shared secret", zap.Binary("identity", crypto.FromECDSAPub(secret.Identity)))
		fSecret := types.NegotiatedSecret{
			PublicKey: secret.Identity,
			Key:       secret.Key,
		}
		filter, err := m.transport.ProcessNegotiatedSecret(fSecret)
		if err != nil {
			return nil, err
		}
		result = append(result, filter)
	}
	return result, nil
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

// NOT IMPLEMENTED
func (m *Messenger) SelectMailserver(id string) error {
	return ErrNotImplemented
}

// NOT IMPLEMENTED
func (m *Messenger) AddMailserver(enode string) error {
	return ErrNotImplemented
}

// NOT IMPLEMENTED
func (m *Messenger) RemoveMailserver(id string) error {
	return ErrNotImplemented
}

// NOT IMPLEMENTED
func (m *Messenger) Mailservers() ([]string, error) {
	return nil, ErrNotImplemented
}

func (m *Messenger) Join(chat Chat) error {
	switch chat.ChatType {
	case ChatTypeOneToOne:
		return m.transport.JoinPrivate(chat.PublicKey)
	case ChatTypePrivateGroupChat:
		members, err := chat.MembersAsPublicKeys()
		if err != nil {
			return err
		}
		return m.transport.JoinGroup(members)
	case ChatTypePublic:
		return m.transport.JoinPublic(chat.Name)
	default:
		return errors.New("chat is neither public nor private")
	}
}

func (m *Messenger) Leave(chat Chat) error {
	if chat.PublicKey != nil {
		return m.transport.LeavePrivate(chat.PublicKey)
	} else if chat.Name != "" {
		return m.transport.LeavePublic(chat.Name)
	}
	return errors.New("chat is neither public nor private")
}

// TODO: consider moving to a ChatManager ???
func (m *Messenger) CreateGroupChat(name string) (*Chat, error) {
	chat := createGroupChat()
	group, err := v1protocol.NewGroupWithCreator(name, m.identity)
	if err != nil {
		return nil, err
	}
	chat.updateChatFromProtocolGroup(group)
	return &chat, nil
}

func (m *Messenger) AddMembersToChat(ctx context.Context, chat *Chat, members []*ecdsa.PublicKey) error {
	group, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return err
	}
	encodedMembers := make([]string, len(members))
	for idx, member := range members {
		encodedMembers[idx] = types.EncodeHex(crypto.FromECDSAPub(member))
	}
	event := v1protocol.NewMembersAddedEvent(encodedMembers, group.NextClockValue())
	err = group.ProcessEvent(&m.identity.PublicKey, event)
	if err != nil {
		return err
	}
	if err := m.propagateMembershipUpdates(ctx, group); err != nil {
		return err
	}
	chat.updateChatFromProtocolGroup(group)
	return m.SaveChat(*chat)
}

func (m *Messenger) ConfirmJoiningGroup(ctx context.Context, chat *Chat) error {
	group, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return err
	}
	event := v1protocol.NewMemberJoinedEvent(
		types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
		group.NextClockValue(),
	)
	err = group.ProcessEvent(&m.identity.PublicKey, event)
	if err != nil {
		return err
	}
	if err := m.propagateMembershipUpdates(ctx, group); err != nil {
		return err
	}
	chat.updateChatFromProtocolGroup(group)
	return m.SaveChat(*chat)
}

func (m *Messenger) propagateMembershipUpdates(ctx context.Context, group *v1protocol.Group) error {
	events := make([]v1protocol.MembershipUpdateEvent, len(group.Updates()))
	for idx, event := range group.Updates() {
		events[idx] = event.MembershipUpdateEvent
	}
	update := v1protocol.MembershipUpdate{
		ChatID: group.ChatID(),
		Events: events,
	}
	if err := update.Sign(m.identity); err != nil {
		return err
	}
	recipients, err := stringSliceToPublicKeys(group.Members(), true)
	if err != nil {
		return err
	}
	// Filter out my key from the recipients
	n := 0
	for _, recipient := range recipients {
		if !isPubKeyEqual(recipient, &m.identity.PublicKey) {
			recipients[n] = recipient
			n++
		}
	}
	recipients = recipients[:n]
	// Finally send membership updates to all recipients.
	_, err = m.processor.SendMembershipUpdate(
		ctx,
		recipients,
		group.ChatID(),
		[]v1protocol.MembershipUpdate{update},
		group.NextClockValue(),
	)
	return err
}

func (m *Messenger) SaveChat(chat Chat) error {
	return m.persistence.SaveChat(chat)
}

func (m *Messenger) Chats() ([]*Chat, error) {
	return m.persistence.Chats()
}

func (m *Messenger) DeleteChat(chatID string) error {
	return m.persistence.DeleteChat(chatID)
}

func (m *Messenger) chatByID(id string) (*Chat, error) {
	chats, err := m.persistence.Chats()
	if err != nil {
		return nil, err
	}
	for _, c := range chats {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, errChatNotFound
}

func (m *Messenger) SaveContact(contact Contact) error {
	identicon, err := identicon.GenerateBase64(contact.ID)
	if err != nil {
		return err
	}

	contact.Identicon = identicon

	name, err := alias.GenerateFromPublicKeyString(contact.ID)
	if err != nil {
		return err
	}

	contact.Alias = name

	return m.persistence.SaveContact(contact, nil)
}

func (m *Messenger) BlockContact(contact Contact) ([]*Chat, error) {
	return m.persistence.BlockContact(contact)
}

func (m *Messenger) Contacts() ([]*Contact, error) {
	return m.persistence.Contacts()
}

func timestampInMs() uint64 {
	return uint64(time.Now().UnixNano() / int64(time.Millisecond))
}

// ReSendChatMessage pulls a message from the database and sends it again
func (m *Messenger) ReSendChatMessage(ctx context.Context, messageID string) (*MessengerResponse, error) {
	logger := m.logger.With(zap.String("site", "ReSendChatMessage"))
	var response MessengerResponse
	message, err := m.persistence.MessageByID(messageID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, errors.New("message not found")
	}
	if message.RawPayload == nil {
		return nil, errors.New("message payload not found, can't resend message")
	}

	chat, err := m.chatByID(message.LocalChatID)
	if err != nil {
		return nil, err
	}

	switch chat.ChatType {
	case ChatTypeOneToOne:
		publicKey := crypto.FromECDSAPub(chat.PublicKey)
		logger.Debug("re-sending private message", zap.Binary("publicKey", publicKey))
		id, err := m.processor.SendPrivateRaw(ctx, chat.PublicKey, message.RawPayload)
		if err != nil {
			return nil, err
		}
		message.ID = "0x" + hex.EncodeToString(id)
		err = m.sendToPairedDevices(ctx, message.RawPayload)
		if err != nil {
			return nil, err
		}

	case ChatTypePublic:
		logger.Debug("re-sending public message", zap.String("chatName", chat.Name))
		id, err := m.processor.SendPublicRaw(ctx, chat.ID, message.RawPayload)
		if err != nil {
			return nil, err
		}
		message.ID = "0x" + hex.EncodeToString(id)
	case ChatTypePrivateGroupChat:
		logger.Debug("re-sending group message", zap.String("chatName", chat.Name))
		recipients, err := chat.MembersAsPublicKeys()
		if err != nil {
			return nil, err
		}

		n := 0
		for _, item := range recipients {
			if !isPubKeyEqual(item, &m.identity.PublicKey) {
				recipients[n] = item
				n++
			}
		}
		id, err := m.processor.SendGroupRaw(ctx, recipients[:n], message.RawPayload)
		if err != nil {
			return nil, err
		}

		message.ID = "0x" + hex.EncodeToString(id)

		err = m.sendToPairedDevices(ctx, message.RawPayload)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("chat type not supported")
	}

	response.Messages = []*Message{message}
	response.Chats = []*Chat{chat}
	return &response, nil
}

// sendToPairedDevices will check if we have any paired devices and send to them if necessary
func (m *Messenger) sendToPairedDevices(ctx context.Context, payload []byte) error {
	activeInstallations, err := m.encryptor.GetOurActiveInstallations(&m.identity.PublicKey)
	if err != nil {
		return err
	}
	// We send a message to any paired device
	if len(activeInstallations) > 1 {
		_, err := m.processor.SendPrivateRaw(ctx, &m.identity.PublicKey, payload)
		if err != nil {
			return err
		}
	}
	return nil
}

// SendChatMessage takes a minimal message and sends it based on the corresponding chat
func (m *Messenger) SendChatMessage(ctx context.Context, message *Message) (*MessengerResponse, error) {
	logger := m.logger.With(zap.String("site", "Send"), zap.String("chatID", message.ChatId))
	var response MessengerResponse

	// A valid added chat is required.
	chat, err := m.chatByID(message.ChatId)
	if err != nil {
		return nil, err
	}

	clock := chat.LastClockValue
	timestamp := timestampInMs()
	if clock == 0 || clock < timestamp {
		clock = timestamp
	} else {
		clock = clock + 1
	}

	message.LocalChatID = chat.ID
	message.Clock = clock
	message.Timestamp = timestamp
	message.From = "0x" + hex.EncodeToString(crypto.FromECDSAPub(&m.identity.PublicKey))
	message.SigPubKey = &m.identity.PublicKey
	message.WhisperTimestamp = timestamp
	message.Seen = true
	message.OutgoingStatus = OutgoingStatusSending

	identicon, err := identicon.GenerateBase64(message.From)
	if err != nil {
		return nil, err
	}

	message.Identicon = identicon

	alias, err := alias.GenerateFromPublicKeyString(message.From)
	if err != nil {
		return nil, err
	}

	message.Alias = alias

	switch chat.ChatType {
	case ChatTypeOneToOne:
		publicKey := crypto.FromECDSAPub(chat.PublicKey)
		logger.Debug("sending private message", zap.Binary("publicKey", publicKey))
		message.MessageType = protobuf.ChatMessage_ONE_TO_ONE
		encodedMessage, err := proto.Marshal(message)
		if err != nil {
			return nil, err
		}
		message.RawPayload = encodedMessage

		id, err := m.processor.SendPrivateRaw(ctx, chat.PublicKey, encodedMessage)
		if err != nil {
			return nil, err
		}
		message.ID = "0x" + hex.EncodeToString(id)

		err = m.sendToPairedDevices(ctx, encodedMessage)
		if err != nil {
			return nil, err
		}

	case ChatTypePublic:
		logger.Debug("sending public message", zap.String("chatName", chat.Name))
		message.MessageType = protobuf.ChatMessage_PUBLIC_GROUP
		encodedMessage, err := proto.Marshal(message)
		if err != nil {
			return nil, err
		}
		message.RawPayload = encodedMessage

		id, err := m.processor.SendPublicRaw(ctx, chat.ID, encodedMessage)
		if err != nil {
			return nil, err
		}
		message.ID = "0x" + hex.EncodeToString(id)
	case ChatTypePrivateGroupChat:
		logger.Debug("sending public message", zap.String("chatName", chat.Name))
		message.MessageType = protobuf.ChatMessage_PRIVATE_GROUP
		encodedMessage, err := proto.Marshal(message)
		if err != nil {
			return nil, err
		}
		message.RawPayload = encodedMessage

		logger.Debug("sending group message", zap.String("chatName", chat.Name))
		recipients, err := chat.MembersAsPublicKeys()
		if err != nil {
			return nil, err
		}

		n := 0
		for _, item := range recipients {
			if !isPubKeyEqual(item, &m.identity.PublicKey) {
				recipients[n] = item
				n++
			}
		}
		id, err := m.processor.SendGroupRaw(ctx, recipients[:n], encodedMessage)
		if err != nil {
			return nil, err
		}

		message.ID = "0x" + hex.EncodeToString(id)

		err = m.sendToPairedDevices(ctx, encodedMessage)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("chat type not supported")
	}

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	chat.LastClockValue = clock
	chat.LastMessage = jsonMessage
	chat.Timestamp = int64(timestamp)
	if err := m.SaveChat(*chat); err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, nil
}

// SendRaw takes encoded data, encrypts it and sends through the wire.
// DEPRECATED
func (m *Messenger) SendRaw(ctx context.Context, chat Chat, data []byte) ([]byte, error) {
	if chat.PublicKey != nil {
		return m.processor.SendPrivateRaw(ctx, chat.PublicKey, data)
	} else if chat.Name != "" {
		return m.processor.SendPublicRaw(ctx, chat.Name, data)
	}
	return nil, errors.New("chat is neither public nor private")
}

// RetrieveAll retrieves messages from all filters, processes them and returns a
// MessengerResponse to the client
func (m *Messenger) RetrieveAll() (*MessengerResponse, error) {
	chatWithMessages, err := m.transport.RetrieveRawAll()
	if err != nil {
		return nil, err
	}

	return m.handleRetrievedMessages(chatWithMessages)
}

func (m *Messenger) handleRetrievedMessages(chatWithMessages map[transport.Filter][]*types.Message) (*MessengerResponse, error) {
	response := &MessengerResponse{
		Chats:    []*Chat{},
		Messages: []*Message{},
	}
	allChats, err := m.persistence.Chats()
	if err != nil {
		return nil, err
	}

	postProcessor := newPostProcessor(m, postProcessorConfig{MatchChat: true})

	logger := m.logger.With(zap.String("site", "RetrieveAll"))
	rawMessages := make(map[transport.Filter][]*v1protocol.StatusMessage)

	// We should query this instead
	contacts, err := m.Contacts()
	if err != nil {
		return nil, err
	}

	blockedContacts := make(map[string]bool)
	for _, c := range contacts {
		if c.IsBlocked() {
			blockedContacts[c.ID] = true
		}
	}

	allContactsMap := make(map[string]*Contact)
	allChatsMap := make(map[string]*Chat)
	existingMessagesMap := make(map[string]bool)

	for chat, messages := range chatWithMessages {
		for _, shhMessage := range messages {
			// TODO: fix this to use an exported method.
			statusMessages, err := m.processor.handleMessages(shhMessage, true)
			if err != nil {
				logger.Info("failed to decode messages", zap.Error(err))
				continue
			}

			for _, msg := range statusMessages {
				// Check for messages from blocked users
				senderID := "0x" + hex.EncodeToString(crypto.FromECDSAPub(msg.SigPubKey()))
				if blockedContacts[senderID] {
					continue
				}
				// Don't process duplicates
				messageID := "0x" + hex.EncodeToString(msg.ID)
				if _, ok := existingMessagesMap[messageID]; ok {
					continue
				}
				existingMessagesMap[messageID] = true

				// Check against the database, this is probably a bit slow for
				// each message, but for now might do, we'll make it faster later
				existingMessage, err := m.persistence.MessageByID(messageID)
				if err != nil && err != errRecordNotFound {
					return nil, err
				}
				if existingMessage != nil {
					continue
				}

				publicKey := msg.SigPubKey()
				if publicKey == nil {
					return nil, errors.New("public key can't be nil")
				}

				var contact *Contact
				if c, ok := allContactsMap[senderID]; ok {
					contact = c
				} else {
					c, err := buildContact(publicKey)
					if err != nil {
						logger.Info("failed to build contact", zap.Error(err))
						continue
					}
					contact = c
					allContactsMap[senderID] = c
					response.Contacts = append(response.Contacts, c)
				}

				if msg.ParsedMessage != nil {
					if textMessage, ok := msg.ParsedMessage.(protobuf.ChatMessage); ok {
						receivedMessage := &Message{
							ID:               messageID,
							ChatMessage:      textMessage,
							From:             contact.ID,
							Alias:            contact.Alias,
							SigPubKey:        publicKey,
							Identicon:        contact.Identicon,
							WhisperTimestamp: uint64(msg.TransportMessage.Timestamp) * 1000,
						}
						receivedMessage.PrepareContent()

						chat, err := postProcessor.matchMessage(receivedMessage, allChats)
						if err != nil {
							logger.Warn("failed to match message", zap.String("receivedChatID", receivedMessage.ChatId), zap.Error(err))
							continue
						}

						// If deleted-at is greater, ignore message
						if chat.DeletedAtClockValue >= receivedMessage.Clock {
							continue
						}

						// Set the LocalChatID for the message
						receivedMessage.LocalChatID = chat.ID

						if c, ok := allChatsMap[chat.ID]; ok {
							chat = c
						}

						// Increase unviewed count
						if !isPubKeyEqual(receivedMessage.SigPubKey, &m.identity.PublicKey) {
							chat.UnviewedMessagesCount++
						} else {
							// Our own message, mark as sent
							receivedMessage.OutgoingStatus = OutgoingStatusSent
						}

						// Update chat timestamp
						chat.Timestamp = int64(timestampInMs())
						// Update last clock value
						if chat.LastClockValue <= receivedMessage.Clock {
							chat.LastClockValue = receivedMessage.Clock
							encodedLastMessage, err := json.Marshal(receivedMessage)
							if err != nil {
								return nil, err
							}
							chat.LastMessage = encodedLastMessage
						}

						// Set chat active
						chat.Active = true
						// Set in the map
						allChatsMap[chat.ID] = chat
						// Add to response
						response.Messages = append(response.Messages, receivedMessage)
					}
				} else {
					// RawMessage, not processed here, pass straight to the client
					rawMessages[chat] = append(rawMessages[chat], msg)
				}

			}
		}
	}

	err = m.persistence.SetContactsGeneratedData(response.Contacts, nil)
	if err != nil {
		return nil, err
	}

	for _, c := range allChatsMap {
		response.Chats = append(response.Chats, c)
	}

	m.persistence.SaveChats(response.Chats)
	m.SaveMessages(response.Messages)

	for filter, messages := range rawMessages {
		response.RawMessages = append(response.RawMessages, &RawResponse{Filter: &filter, Messages: messages})
	}
	return response, nil
}

func (m *Messenger) RequestHistoricMessages(
	ctx context.Context,
	peer []byte, // should be removed after mailserver logic is ported
	from, to uint32,
	cursor []byte,
) ([]byte, error) {
	return m.transport.SendMessagesRequest(ctx, peer, from, to, cursor)
}

// DEPRECATED
func (m *Messenger) LoadFilters(filters []*transport.Filter) ([]*transport.Filter, error) {
	return m.transport.LoadFilters(filters)
}

// DEPRECATED
func (m *Messenger) RemoveFilters(filters []*transport.Filter) error {
	return m.transport.RemoveFilters(filters)
}

// DEPRECATED
func (m *Messenger) ConfirmMessagesProcessed(messageIDs [][]byte) error {
	for _, id := range messageIDs {
		if err := m.encryptor.ConfirmMessageProcessed(id); err != nil {
			return err
		}
	}
	return nil
}

// DEPRECATED: required by status-react.
func (m *Messenger) MessageByID(id string) (*Message, error) {
	return m.persistence.MessageByID(id)
}

// DEPRECATED: required by status-react.
func (m *Messenger) MessagesExist(ids []string) (map[string]bool, error) {
	return m.persistence.MessagesExist(ids)
}

// DEPRECATED: required by status-react.
func (m *Messenger) MessageByChatID(chatID, cursor string, limit int) ([]*Message, string, error) {
	return m.persistence.MessageByChatID(chatID, cursor, limit)
}

// DEPRECATED: required by status-react.
func (m *Messenger) SaveMessages(messages []*Message) error {
	return m.persistence.SaveMessagesLegacy(messages)
}

// AddSystemMessages format an array of system-messages and saves them to the database
// It's needed until group chats are fully in status-go.
func (m *Messenger) AddSystemMessages(messages []*Message) ([]*Message, error) {
	timestamp := timestampInMs()

	for _, message := range messages {
		message.LocalChatID = message.ChatId
		message.Timestamp = timestamp
		message.WhisperTimestamp = timestamp
		message.Seen = true

		identicon, err := identicon.GenerateBase64(message.From)
		if err != nil {
			return nil, err
		}

		message.Identicon = identicon

		alias, err := alias.GenerateFromPublicKeyString(message.From)
		if err != nil {
			return nil, err
		}

		message.ID = "0x" + hex.EncodeToString(crypto.Keccak256([]byte(message.Text+message.From+strconv.FormatUint(message.Clock, 10))))
		message.Alias = alias
		message.ContentType = protobuf.ChatMessage_STATUS
		message.MessageType = protobuf.ChatMessage_SYSTEM_MESSAGE_PRIVATE_GROUP
		err = message.PrepareContent()
		if err != nil {
			return nil, err
		}
	}

	err := m.SaveMessages(messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

// DEPRECATED: required by status-react.
func (m *Messenger) DeleteMessage(id string) error {
	return m.persistence.DeleteMessage(id)
}

// DEPRECATED: required by status-react.
func (m *Messenger) DeleteMessagesByChatID(id string) error {
	return m.persistence.DeleteMessagesByChatID(id)
}

// DEPRECATED: required by status-react.
func (m *Messenger) MarkMessagesSeen(chatID string, ids []string) error {
	return m.persistence.MarkMessagesSeen(chatID, ids)
}

// DEPRECATED: required by status-react.
func (m *Messenger) UpdateMessageOutgoingStatus(id, newOutgoingStatus string) error {
	return m.persistence.UpdateMessageOutgoingStatus(id, newOutgoingStatus)
}

// postProcessor performs a set of actions on newly retrieved messages.
// If persist is true, it saves the messages into the database.
// If matchChat is true, it matches each messages against a Chat instance.
type postProcessor struct {
	myPublicKey *ecdsa.PublicKey
	persistence *sqlitePersistence
	logger      *zap.Logger

	config postProcessorConfig
}

type postProcessorConfig struct {
	MatchChat bool // match each messages to a chat; may result in a new chat creation
	Persist   bool // if true, all sent and received user messages will be persisted
	Parse     bool // if true, it will parse the content
}

func newPostProcessor(m *Messenger, config postProcessorConfig) *postProcessor {
	return &postProcessor{
		myPublicKey: &m.identity.PublicKey,
		persistence: m.persistence,
		logger:      m.logger,
		config:      config,
	}
}

func (p *postProcessor) matchMessages(messages []*Message) ([]*Message, error) {
	chats, err := p.persistence.Chats()
	if err != nil {
		return nil, err
	}

	result := make([]*Message, 0, len(messages))
	for _, message := range messages {
		chat, err := p.matchMessage(message, chats)
		if err != nil {
			p.logger.Error("failed to match a chat to a message", zap.Error(err))
			continue
		}
		message.LocalChatID = chat.ID
		result = append(result, message)
	}
	return result, nil
}

func (p *postProcessor) matchMessage(message *Message, chats []*Chat) (*Chat, error) {
	if message.SigPubKey == nil {
		p.logger.Error("public key can't be empty")
		return nil, errors.New("received a message with empty public key")
	}

	switch {
	case message.MessageType == protobuf.ChatMessage_PUBLIC_GROUP:
		// For public messages, all outgoing and incoming messages have the same chatID
		// equal to a public chat name.
		chatID := message.ChatId
		chat := findChatByID(chatID, chats)
		if chat == nil {
			return nil, errors.New("received a public message from non-existing chat")
		}
		return chat, nil
	case message.MessageType == protobuf.ChatMessage_ONE_TO_ONE && isPubKeyEqual(message.SigPubKey, p.myPublicKey):
		// It's a private message coming from us so we rely on Message.ChatId
		// If chat does not exist, it should be created to support multidevice synchronization.
		chatID := message.ChatId
		chat := findChatByID(chatID, chats)
		if chat == nil {
			if len(chatID) != PubKeyStringLength {
				return nil, errors.New("invalid pubkey length")
			}
			bytePubKey, err := hex.DecodeString(chatID[2:])
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode hex chatID")
			}

			pubKey, err := crypto.UnmarshalPubkey(bytePubKey)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode pubkey")
			}

			newChat := CreateOneToOneChat(chatID[:8], pubKey)
			if err := p.persistence.SaveChat(newChat); err != nil {
				return nil, errors.Wrap(err, "failed to save newly created chat")
			}
			chat = &newChat
		}
		return chat, nil
	case message.MessageType == protobuf.ChatMessage_ONE_TO_ONE:
		// It's an incoming private message. ChatID is calculated from the signature.
		// If a chat does not exist, a new one is created and saved.
		chatID := types.EncodeHex(crypto.FromECDSAPub(message.SigPubKey))
		chat := findChatByID(chatID, chats)
		if chat == nil {
			// TODO: this should be a three-word name used in the mobile client
			newChat := CreateOneToOneChat(chatID[:8], message.SigPubKey)
			if err := p.persistence.SaveChat(newChat); err != nil {
				return nil, errors.Wrap(err, "failed to save newly created chat")
			}
			chat = &newChat
		}
		return chat, nil
	case message.MessageType == protobuf.ChatMessage_PRIVATE_GROUP:
		// In the case of a group message, ChatID is the same for all messages belonging to a group.
		// It needs to be verified if the signature public key belongs to the chat.
		chatID := message.ChatId
		chat := findChatByID(chatID, chats)
		if chat == nil {
			return nil, errors.New("received group chat message for non-existing chat")
		}

		theirKeyHex := types.EncodeHex(crypto.FromECDSAPub(message.SigPubKey))
		myKeyHex := types.EncodeHex(crypto.FromECDSAPub(p.myPublicKey))
		var theyJoined bool
		var iJoined bool
		for _, member := range chat.Members {
			if member.ID == theirKeyHex && member.Joined {
				theyJoined = true
			}
		}
		for _, member := range chat.Members {
			if member.ID == myKeyHex && member.Joined {
				iJoined = true
			}
		}

		if theyJoined && iJoined {
			return chat, nil
		}

		return nil, errors.New("did not find a matching group chat")
	default:
		return nil, errors.New("can not match a chat because there is no valid case")
	}
}

// Identicon returns an identicon based on the input string
func Identicon(id string) (string, error) {
	return identicon.GenerateBase64(id)
}

// VerifyENSNames verifies that a registered ENS name matches the expected public key
func (m *Messenger) VerifyENSNames(rpcEndpoint, contractAddress string, ensDetails []enstypes.ENSDetails) (map[string]enstypes.ENSResponse, error) {
	verifier := m.node.NewENSVerifier(m.logger)

	ensResponse, err := verifier.CheckBatch(ensDetails, rpcEndpoint, contractAddress)
	if err != nil {
		return nil, err
	}

	// Update contacts
	var contacts []*Contact
	for _, details := range ensResponse {
		if details.Error == nil {
			contact, err := buildContact(details.PublicKey)
			if err != nil {
				return nil, err
			}
			contact.ENSVerified = details.Verified
			contact.ENSVerifiedAt = details.VerifiedAt
			contact.Name = details.Name

			contacts = append(contacts, contact)
		} else {
			m.logger.Warn("Failed to resolve ens name",
				zap.String("name", details.Name),
				zap.String("publicKey", details.PublicKeyString),
				zap.Error(details.Error),
			)
		}
	}

	if len(contacts) != 0 {
		err = m.persistence.SetContactsENSData(contacts)
		if err != nil {
			return nil, err
		}
	}

	return ensResponse, nil
}

// GenerateAlias name returns the generated name given a public key hex encoded prefixed with 0x
func GenerateAlias(id string) (string, error) {
	return alias.GenerateFromPublicKeyString(id)
}
