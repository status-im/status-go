package protocol

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"sync"
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
	node                       types.Node
	identity                   *ecdsa.PrivateKey
	persistence                *sqlitePersistence
	transport                  *transport.WhisperServiceTransport
	encryptor                  *encryption.Protocol
	processor                  *messageProcessor
	logger                     *zap.Logger
	featureFlags               featureFlags
	messagesPersistenceEnabled bool
	shutdownTasks              []func() error
	systemMessagesTranslations map[protobuf.MembershipUpdateEvent_EventType]string
	allChats                   map[string]*Chat
	allContacts                map[string]*Contact
	mutex                      sync.Mutex
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

	// systemMessagesTranslations holds translations for system-messages
	systemMessagesTranslations map[protobuf.MembershipUpdateEvent_EventType]string
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

func WithSystemMessagesTranslations(t map[protobuf.MembershipUpdateEvent_EventType]string) Option {
	return func(c *config) error {
		c.systemMessagesTranslations = t
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
			slogger.Debug("received a SendContactCode request")

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

	if c.systemMessagesTranslations == nil {
		c.systemMessagesTranslations = defaultSystemMessagesTranslations
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
		systemMessagesTranslations: c.systemMessagesTranslations,
		allChats:                   make(map[string]*Chat),
		allContacts:                make(map[string]*Contact),
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
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Seed the for color generation
	rand.Seed(time.Now().Unix())

	logger := m.logger.With(zap.String("site", "Init"))

	var (
		publicChatIDs []string
		publicKeys    []*ecdsa.PublicKey
	)

	// Get chat IDs and public keys from the existing chats.
	// TODO: Get only active chats by the query.
	chats, err := m.persistence.Chats()
	if err != nil {
		return err
	}
	for _, chat := range chats {
		m.allChats[chat.ID] = chat
		if !chat.Active {
			continue
		}
		switch chat.ChatType {
		case ChatTypePublic:
			publicChatIDs = append(publicChatIDs, chat.ID)
		case ChatTypeOneToOne:
			pk, err := chat.PublicKey()
			if err != nil {
				return err
			}
			publicKeys = append(publicKeys, pk)
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
	contacts, err := m.persistence.Contacts()
	if err != nil {
		return err
	}
	for _, contact := range contacts {
		m.allContacts[contact.ID] = contact
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
		pk, err := chat.PublicKey()
		if err != nil {
			return err
		}

		return m.transport.JoinPrivate(pk)
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

// This is not accurate, it should not leave transport on removal of chat/group
// only once there is no more: Group chat with that member, one-to-one chat, contact added by us
func (m *Messenger) Leave(chat Chat) error {
	switch chat.ChatType {
	case ChatTypeOneToOne:
		pk, err := chat.PublicKey()
		if err != nil {
			return err
		}
		return m.transport.LeavePrivate(pk)
	case ChatTypePrivateGroupChat:
		members, err := chat.MembersAsPublicKeys()
		if err != nil {
			return err
		}
		return m.transport.LeaveGroup(members)
	case ChatTypePublic:
		return m.transport.LeavePublic(chat.Name)
	default:
		return errors.New("chat is neither public nor private")
	}
}

func (m *Messenger) CreateGroupChatWithMembers(ctx context.Context, name string, members []string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse
	logger := m.logger.With(zap.String("site", "CreateGroupChatWithMembers"))
	logger.Info("Creating group chat", zap.String("name", name), zap.Any("members", members))
	chat := createGroupChat()
	group, err := v1protocol.NewGroupWithCreator(name, m.identity)
	if err != nil {
		return nil, err
	}
	chat.updateChatFromProtocolGroup(group)

	// Add members
	event := v1protocol.NewMembersAddedEvent(members, group.NextClockValue())
	event.ChatID = chat.ID
	err = event.Sign(m.identity)
	if err != nil {
		return nil, err
	}

	err = group.ProcessEvent(event)
	if err != nil {
		return nil, err
	}
	recipients, err := stringSliceToPublicKeys(group.Members(), true)
	if err != nil {
		return nil, err
	}

	if _, err := m.propagateMembershipUpdates(ctx, group, recipients, nil); err != nil {
		return nil, err
	}
	chat.updateChatFromProtocolGroup(group)

	response.Chats = []*Chat{&chat}
	response.Messages = buildSystemMessages(chat.MembershipUpdates, m.systemMessagesTranslations)
	return &response, m.saveChat(&chat)
}

func (m *Messenger) RemoveMemberFromGroupChat(ctx context.Context, chatID string, member string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse
	logger := m.logger.With(zap.String("site", "RemoveMemberFromGroupChat"))
	logger.Info("Removing member form group chat", zap.String("chatID", chatID), zap.String("member", member))
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("can't find chat")
	}

	group, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return nil, err
	}

	// We save the initial recipients as we want to send updates to also
	// the members kicked out
	oldRecipients, err := stringSliceToPublicKeys(group.Members(), true)
	if err != nil {
		return nil, err
	}

	// Remove member
	event := v1protocol.NewMemberRemovedEvent(member, group.NextClockValue())
	event.ChatID = chat.ID
	err = event.Sign(m.identity)
	if err != nil {
		return nil, err
	}

	err = group.ProcessEvent(event)
	if err != nil {
		return nil, err
	}
	if _, err := m.propagateMembershipUpdates(ctx, group, oldRecipients, nil); err != nil {
		return nil, err
	}
	chat.updateChatFromProtocolGroup(group)
	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages(chat.MembershipUpdates, m.systemMessagesTranslations)
	return &response, m.saveChat(chat)
}

func (m *Messenger) AddMembersToGroupChat(ctx context.Context, chatID string, members []string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse
	logger := m.logger.With(zap.String("site", "AddMembersFromGroupChat"))
	logger.Info("Adding members form group chat", zap.String("chatID", chatID), zap.Any("members", members))
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("can't find chat")
	}

	group, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return nil, err
	}

	// Add members
	event := v1protocol.NewMembersAddedEvent(members, group.NextClockValue())
	event.ChatID = chat.ID
	err = event.Sign(m.identity)
	if err != nil {
		return nil, err
	}

	err = group.ProcessEvent(event)
	if err != nil {
		return nil, err
	}

	recipients, err := stringSliceToPublicKeys(group.Members(), true)
	if err != nil {
		return nil, err
	}

	if _, err := m.propagateMembershipUpdates(ctx, group, recipients, nil); err != nil {
		return nil, err
	}
	chat.updateChatFromProtocolGroup(group)

	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)

	return &response, m.saveChat(chat)
}

func (m *Messenger) AddAdminsToGroupChat(ctx context.Context, chatID string, members []string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse
	logger := m.logger.With(zap.String("site", "AddAdminsToGroupChat"))
	logger.Info("Add admins to group chat", zap.String("chatID", chatID), zap.Any("members", members))

	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("can't find chat")
	}

	group, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return nil, err
	}

	// Add members
	event := v1protocol.NewAdminsAddedEvent(members, group.NextClockValue())
	event.ChatID = chat.ID
	err = event.Sign(m.identity)
	if err != nil {
		return nil, err
	}

	err = group.ProcessEvent(event)
	if err != nil {
		return nil, err
	}

	recipients, err := stringSliceToPublicKeys(group.Members(), true)
	if err != nil {
		return nil, err
	}

	if _, err := m.propagateMembershipUpdates(ctx, group, recipients, nil); err != nil {
		return nil, err
	}
	chat.updateChatFromProtocolGroup(group)

	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)

	return &response, m.saveChat(chat)
}

func (m *Messenger) ConfirmJoiningGroup(ctx context.Context, chatID string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("can't find chat")
	}

	err := m.Join(*chat)
	if err != nil {
		return nil, err
	}

	group, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return nil, err
	}
	event := v1protocol.NewMemberJoinedEvent(
		group.NextClockValue(),
	)
	event.ChatID = chat.ID
	err = event.Sign(m.identity)
	if err != nil {
		return nil, err
	}

	err = group.ProcessEvent(event)
	if err != nil {
		return nil, err
	}

	recipients, err := stringSliceToPublicKeys(group.Members(), true)
	if err != nil {
		return nil, err
	}

	if _, err := m.propagateMembershipUpdates(ctx, group, recipients, nil); err != nil {
		return nil, err
	}
	chat.updateChatFromProtocolGroup(group)

	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)

	return &response, m.saveChat(chat)
}

func (m *Messenger) LeaveGroupChat(ctx context.Context, chatID string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("can't find chat")
	}

	err := m.Leave(*chat)
	if err != nil {
		return nil, err
	}

	group, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return nil, err
	}
	event := v1protocol.NewMemberRemovedEvent(
		types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
		group.NextClockValue(),
	)
	event.ChatID = chat.ID
	err = event.Sign(m.identity)
	if err != nil {
		return nil, err
	}

	err = group.ProcessEvent(event)
	if err != nil {
		return nil, err
	}

	recipients, err := stringSliceToPublicKeys(group.Members(), true)
	if err != nil {
		return nil, err
	}

	if _, err := m.propagateMembershipUpdates(ctx, group, recipients, nil); err != nil {
		return nil, err
	}
	chat.updateChatFromProtocolGroup(group)
	chat.Active = false

	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	return &response, m.saveChat(chat)
}

func (m *Messenger) propagateMembershipUpdates(ctx context.Context, group *v1protocol.Group, recipients []*ecdsa.PublicKey, chatMessage *protobuf.ChatMessage) ([]byte, error) {
	hasPairedDevices, err := m.hasPairedDevices()
	if err != nil {
		return nil, err
	}

	if !hasPairedDevices {
		// Filter out my key from the recipients
		n := 0
		for _, recipient := range recipients {
			if !isPubKeyEqual(recipient, &m.identity.PublicKey) {
				recipients[n] = recipient
				n++
			}
		}
		recipients = recipients[:n]
	}
	// Finally send membership updates to all recipients.
	return m.processor.SendMembershipUpdate(
		ctx,
		recipients,
		group,
		chatMessage,
	)
}

func (m *Messenger) saveChat(chat *Chat) error {
	err := m.persistence.SaveChat(*chat)
	if err != nil {
		return err
	}
	m.allChats[chat.ID] = chat

	return nil
}

func (m *Messenger) saveChats(chats []*Chat) error {
	err := m.persistence.SaveChats(chats)
	if err != nil {
		return err
	}
	for _, chat := range chats {
		m.allChats[chat.ID] = chat
	}

	return nil

}

func (m *Messenger) SaveChat(chat *Chat) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	err := m.saveChat(chat)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) Chats() []*Chat {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var chats []*Chat

	for _, c := range m.allChats {
		chats = append(chats, c)
	}

	return chats
}

func (m *Messenger) DeleteChat(chatID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := m.persistence.DeleteChat(chatID)
	if err != nil {
		return err
	}
	delete(m.allChats, chatID)

	return nil
}

func (m *Messenger) SaveContact(contact *Contact) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

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

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	m.allContacts[contact.ID] = contact
	return nil
}

func (m *Messenger) BlockContact(contact *Contact) ([]*Chat, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	chats, err := m.persistence.BlockContact(contact)
	if err != nil {
		return nil, err
	}
	m.allContacts[contact.ID] = contact
	for _, chat := range chats {
		m.allChats[chat.ID] = chat
	}
	delete(m.allChats, contact.ID)
	return chats, nil
}

func (m *Messenger) Contacts() []*Contact {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	var contacts []*Contact
	for _, contact := range m.allContacts {
		contacts = append(contacts, contact)
	}

	return contacts
}

func timestampInMs() uint64 {
	return uint64(time.Now().UnixNano() / int64(time.Millisecond))
}

// ReSendChatMessage pulls a message from the database and sends it again
func (m *Messenger) ReSendChatMessage(ctx context.Context, messageID string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

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

	chat, ok := m.allChats[message.LocalChatID]
	if !ok {
		return nil, errors.New("chat not found")
	}

	switch chat.ChatType {
	case ChatTypeOneToOne:
		publicKey, err := chat.PublicKey()
		if err != nil {
			return nil, err
		}
		logger.Debug("re-sending private message")
		id, err := m.processor.SendPrivateRaw(ctx, publicKey, message.RawPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)
		if err != nil {
			return nil, err
		}
		message.ID = "0x" + hex.EncodeToString(id)
		err = m.sendToPairedDevices(ctx, message.RawPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)

		if err != nil {
			return nil, err
		}

	case ChatTypePublic:
		logger.Debug("re-sending public message", zap.String("chatName", chat.Name))
		id, err := m.processor.SendPublicRaw(ctx, chat.ID, message.RawPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)

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
		id, err := m.processor.SendGroupRaw(ctx, recipients[:n], message.RawPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)

		if err != nil {
			return nil, err
		}

		message.ID = "0x" + hex.EncodeToString(id)

		err = m.sendToPairedDevices(ctx, message.RawPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)

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

func (m *Messenger) hasPairedDevices() (bool, error) {
	activeInstallations, err := m.encryptor.GetOurActiveInstallations(&m.identity.PublicKey)
	if err != nil {
		return false, err
	}
	return len(activeInstallations) > 1, nil
}

// sendToPairedDevices will check if we have any paired devices and send to them if necessary
func (m *Messenger) sendToPairedDevices(ctx context.Context, payload []byte, messageType protobuf.ApplicationMetadataMessage_Type) error {
	hasPairedDevices, err := m.hasPairedDevices()
	if err != nil {
		return err
	}
	// We send a message to any paired device
	if hasPairedDevices {
		_, err := m.processor.SendPrivateRaw(ctx, &m.identity.PublicKey, payload, messageType)
		if err != nil {
			return err
		}
	}
	return nil
}

// SendChatMessage takes a minimal message and sends it based on the corresponding chat
func (m *Messenger) SendChatMessage(ctx context.Context, message *Message) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	logger := m.logger.With(zap.String("site", "Send"), zap.String("chatID", message.ChatId))
	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats[message.ChatId]
	if !ok {
		return nil, errors.New("Chat not found")
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
		publicKey, err := chat.PublicKey()
		if err != nil {
			return nil, err
		}
		logger.Debug("sending private message")
		message.MessageType = protobuf.ChatMessage_ONE_TO_ONE
		encodedMessage, err := proto.Marshal(message)
		if err != nil {
			return nil, err
		}
		message.RawPayload = encodedMessage

		id, err := m.processor.SendPrivateRaw(ctx, publicKey, encodedMessage, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)

		if err != nil {
			return nil, err
		}
		message.ID = "0x" + hex.EncodeToString(id)

		err = m.sendToPairedDevices(ctx, encodedMessage, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)

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

		id, err := m.processor.SendPublicRaw(ctx, chat.ID, encodedMessage, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)

		if err != nil {
			return nil, err
		}
		message.ID = "0x" + hex.EncodeToString(id)
	case ChatTypePrivateGroupChat:
		logger.Debug("sending public message", zap.String("chatName", chat.Name))
		message.MessageType = protobuf.ChatMessage_PRIVATE_GROUP
		logger.Debug("sending group message", zap.String("chatName", chat.Name))
		recipients, err := chat.MembersAsPublicKeys()
		if err != nil {
			return nil, err
		}

		group, err := newProtocolGroupFromChat(chat)
		if err != nil {
			return nil, err
		}

		encodedMessage, err := m.propagateMembershipUpdates(ctx, group, recipients, &message.ChatMessage)
		if err != nil {
			return nil, err
		}

		id := v1protocol.MessageID(&m.identity.PublicKey, encodedMessage)
		message.ID = "0x" + hex.EncodeToString(id)
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

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, m.saveChat(chat)
}

// SendRaw takes encoded data, encrypts it and sends through the wire.
// DEPRECATED
func (m *Messenger) SendRaw(ctx context.Context, chat Chat, data []byte) ([]byte, error) {
	publicKey, err := chat.PublicKey()
	if err != nil {
		return nil, err
	}
	if publicKey != nil {
		return m.processor.SendPrivateRaw(ctx, publicKey, data, protobuf.ApplicationMetadataMessage_UNKNOWN)
	} else if chat.Name != "" {
		return m.processor.SendPublicRaw(ctx, chat.Name, data, protobuf.ApplicationMetadataMessage_UNKNOWN)
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

type ReceivedMessageState struct {
	// Message is the protobuf message received
	Message protobuf.ChatMessage
	// MessageID is the ID of the message
	MessageID string
	// WhisperTimestamp is the whisper timestamp of the message
	WhisperTimestamp uint64
	// Contact is the contact associated with the author of the message
	Contact *Contact
	// PublicKey is the public key of the author of the message
	PublicKey *ecdsa.PublicKey
	// List of chats modified
	ModifiedChats map[string]bool
	PostProcessor *postProcessor
}

func (m *Messenger) handleChatMessage(state *ReceivedMessageState) (*Message, error) {
	logger := m.logger.With(zap.String("site", "handleChatMessage"))
	if err := ValidateReceivedChatMessage(&state.Message); err != nil {
		logger.Warn("failed to validate message", zap.Error(err))
		return nil, err
	}
	receivedMessage := &Message{
		ID:               state.MessageID,
		ChatMessage:      state.Message,
		From:             state.Contact.ID,
		Alias:            state.Contact.Alias,
		SigPubKey:        state.PublicKey,
		Identicon:        state.Contact.Identicon,
		WhisperTimestamp: state.WhisperTimestamp,
	}
	receivedMessage.PrepareContent()
	chat, err := state.PostProcessor.matchMessage(receivedMessage, m.allChats)
	if err != nil {
		return nil, err
	}

	// If deleted-at is greater, ignore message
	if chat.DeletedAtClockValue >= receivedMessage.Clock {
		return nil, nil
	}

	// Set the LocalChatID for the message
	receivedMessage.LocalChatID = chat.ID

	if c, ok := m.allChats[chat.ID]; ok {
		chat = c
	}

	// Set the LocalChatID for the message
	receivedMessage.LocalChatID = chat.ID

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
	// Set in the modified maps chat
	state.ModifiedChats[chat.ID] = true
	m.allChats[chat.ID] = chat

	return receivedMessage, nil
}

func (m *Messenger) messageExists(messageID string, existingMessagesMap map[string]bool) (bool, error) {
	if _, ok := existingMessagesMap[messageID]; ok {
		return true, nil
	}

	existingMessagesMap[messageID] = true

	// Check against the database, this is probably a bit slow for
	// each message, but for now might do, we'll make it faster later
	existingMessage, err := m.persistence.MessageByID(messageID)
	if err != nil && err != errRecordNotFound {
		return false, err
	}
	if existingMessage != nil {
		return true, nil
	}
	return false, nil
}

func (m *Messenger) handleRetrievedMessages(chatWithMessages map[transport.Filter][]*types.Message) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	response := &MessengerResponse{
		Chats:    []*Chat{},
		Messages: []*Message{},
	}

	postProcessor := newPostProcessor(m, postProcessorConfig{MatchChat: true})

	logger := m.logger.With(zap.String("site", "RetrieveAll"))
	rawMessages := make(map[transport.Filter][]*v1protocol.StatusMessage)

	modifiedChats := make(map[string]bool)
	modifiedContacts := make(map[string]bool)
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
				publicKey := msg.SigPubKey()

				// Check for messages from blocked users
				senderID := types.EncodeHex(crypto.FromECDSAPub(publicKey))
				if _, ok := m.allContacts[senderID]; ok && m.allContacts[senderID].IsBlocked() {
					continue
				}
				// Don't process duplicates
				messageID := types.EncodeHex(msg.ID)
				exists, err := m.messageExists(messageID, existingMessagesMap)
				if err != nil {
					logger.Warn("failed to check message exists", zap.Error(err))
				}
				if exists {
					continue
				}

				var contact *Contact
				if c, ok := m.allContacts[senderID]; ok {
					contact = c
				} else {
					c, err := buildContact(publicKey)
					if err != nil {
						logger.Info("failed to build contact", zap.Error(err))
						continue
					}
					contact = c
					m.allContacts[senderID] = c
					modifiedContacts[contact.ID] = true
				}
				messageState := &ReceivedMessageState{
					MessageID:        messageID,
					WhisperTimestamp: uint64(msg.TransportMessage.Timestamp) * 1000,
					Contact:          contact,
					PublicKey:        publicKey,
					ModifiedChats:    modifiedChats,
					PostProcessor:    postProcessor,
				}

				if msg.ParsedMessage != nil {
					switch msg.ParsedMessage.(type) {
					case protobuf.MembershipUpdateMessage:

						rawMembershipUpdate := msg.ParsedMessage.(protobuf.MembershipUpdateMessage)
						membershipUpdate, err := v1protocol.MembershipUpdateMessageFromProtobuf(&rawMembershipUpdate)
						if err != nil {
							logger.Warn("failed to process membership update", zap.Error(err))
							continue

						}

						chat, systemMessages, err := HandleMembershipUpdate(m.allChats[membershipUpdate.ChatID], membershipUpdate, types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)), m.systemMessagesTranslations)
						if err != nil {
							logger.Warn("failed to process membership update", zap.Error(err))
							continue
						}

						for _, message := range systemMessages {
							messageID := message.ID
							exists, err := m.messageExists(messageID, existingMessagesMap)
							if err != nil {
								logger.Warn("failed to check message exists", zap.Error(err))
							}
							if exists {
								continue
							}
							response.Messages = append(response.Messages, message)
						}

						// Store in chats map as it might be a new one
						m.allChats[chat.ID] = chat
						// Set in the map
						modifiedChats[chat.ID] = true

						if rawMembershipUpdate.Message != nil {
							messageState.Message = *rawMembershipUpdate.Message
							receivedMessage, err := m.handleChatMessage(messageState)
							if err != nil {
								logger.Warn("failed to process message", zap.Error(err))
								continue
							}
							// Add to response
							if receivedMessage != nil {
								response.Messages = append(response.Messages, receivedMessage)
							}

						}

					case protobuf.ChatMessage:
						messageState.Message = msg.ParsedMessage.(protobuf.ChatMessage)
						receivedMessage, err := m.handleChatMessage(messageState)
						if err != nil {
							logger.Warn("failed to process message", zap.Error(err))
							continue
						}
						// Add to response
						if receivedMessage != nil {
							response.Messages = append(response.Messages, receivedMessage)
						}
					default:
						// RawMessage, not processed here, pass straight to the client
						rawMessages[chat] = append(rawMessages[chat], msg)

					}
				} else {
					rawMessages[chat] = append(rawMessages[chat], msg)
				}
			}
		}
	}

	for id := range modifiedChats {
		response.Chats = append(response.Chats, m.allChats[id])
	}

	for id := range modifiedContacts {
		response.Contacts = append(response.Contacts, m.allContacts[id])
	}

	var err error
	if len(response.Chats) > 0 {
		err = m.saveChats(response.Chats)
		if err != nil {
			return nil, err
		}
	}
	if len(response.Messages) > 0 {
		err = m.SaveMessages(response.Messages)
		if err != nil {
			return nil, err
		}
	}

	if len(response.Contacts) > 0 {
		err = m.persistence.SaveContacts(response.Contacts)
		if err != nil {
			return nil, err
		}
	}

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
	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := m.persistence.MarkMessagesSeen(chatID, ids)
	if err != nil {
		return err
	}
	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return err
	}
	m.allChats[chatID] = chat
	return nil
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

func (p *postProcessor) matchMessage(message *Message, chats map[string]*Chat) (*Chat, error) {
	if message.SigPubKey == nil {
		p.logger.Error("public key can't be empty")
		return nil, errors.New("received a message with empty public key")
	}

	switch {
	case message.MessageType == protobuf.ChatMessage_PUBLIC_GROUP:
		// For public messages, all outgoing and incoming messages have the same chatID
		// equal to a public chat name.
		chatID := message.ChatId
		chat := chats[chatID]
		if chat == nil {
			return nil, errors.New("received a public message from non-existing chat")
		}
		return chat, nil
	case message.MessageType == protobuf.ChatMessage_ONE_TO_ONE && isPubKeyEqual(message.SigPubKey, p.myPublicKey):
		// It's a private message coming from us so we rely on Message.ChatId
		// If chat does not exist, it should be created to support multidevice synchronization.
		chatID := message.ChatId
		chat := chats[chatID]
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
			chat = &newChat
		}
		return chat, nil
	case message.MessageType == protobuf.ChatMessage_ONE_TO_ONE:
		// It's an incoming private message. ChatID is calculated from the signature.
		// If a chat does not exist, a new one is created and saved.
		chatID := types.EncodeHex(crypto.FromECDSAPub(message.SigPubKey))
		chat := chats[chatID]
		if chat == nil {
			// TODO: this should be a three-word name used in the mobile client
			newChat := CreateOneToOneChat(chatID[:8], message.SigPubKey)
			chat = &newChat
		}
		return chat, nil
	case message.MessageType == protobuf.ChatMessage_PRIVATE_GROUP:
		// In the case of a group message, ChatID is the same for all messages belonging to a group.
		// It needs to be verified if the signature public key belongs to the chat.
		chatID := message.ChatId
		chat := chats[chatID]
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
	m.mutex.Lock()
	defer m.mutex.Unlock()
	verifier := m.node.NewENSVerifier(m.logger)

	ensResponse, err := verifier.CheckBatch(ensDetails, rpcEndpoint, contractAddress)
	if err != nil {
		return nil, err
	}

	// Update contacts
	var contacts []*Contact
	for _, details := range ensResponse {
		if details.Error == nil {
			contact, ok := m.allContacts[details.PublicKeyString]
			if !ok {
				contact, err = buildContact(details.PublicKey)
				if err != nil {
					return nil, err
				}
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
		err = m.persistence.SaveContacts(contacts)
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
