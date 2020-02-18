package protocol

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
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
	"github.com/status-im/status-go/protocol/transport"
	wakutransp "github.com/status-im/status-go/protocol/transport/waku"
	shhtransp "github.com/status-im/status-go/protocol/transport/whisper"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

const PubKeyStringLength = 132

const transactionSentTxt = "Transaction sent"

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
	node                       types.Node
	identity                   *ecdsa.PrivateKey
	persistence                *sqlitePersistence
	transport                  transport.Transport
	encryptor                  *encryption.Protocol
	processor                  *messageProcessor
	handler                    *MessageHandler
	logger                     *zap.Logger
	verifyTransactionClient    EthClient
	featureFlags               featureFlags
	messagesPersistenceEnabled bool
	shutdownTasks              []func() error
	systemMessagesTranslations map[protobuf.MembershipUpdateEvent_EventType]string
	allChats                   map[string]*Chat
	allContacts                map[string]*Contact
	allInstallations           map[string]*multidevice.Installation
	modifiedInstallations      map[string]bool
	installationID             string

	mutex sync.Mutex
}

type RawResponse struct {
	Filter   *transport.Filter           `json:"filter"`
	Messages []*v1protocol.StatusMessage `json:"messages"`
}

type MessengerResponse struct {
	Chats         []*Chat                     `json:"chats,omitempty"`
	Messages      []*Message                  `json:"messages,omitempty"`
	Contacts      []*Contact                  `json:"contacts,omitempty"`
	Installations []*multidevice.Installation `json:"installations,omitempty"`
	// Raw unprocessed messages
	RawMessages []*RawResponse `json:"rawMessages,omitempty"`
}

func (m *MessengerResponse) IsEmpty() bool {
	return len(m.Chats) == 0 && len(m.Messages) == 0 && len(m.Contacts) == 0 && len(m.RawMessages) == 0 && len(m.Installations) == 0
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

	verifyTransactionClient EthClient

	logger *zap.Logger
}

type Option func(*config) error

// WithSystemMessagesTranslations is required for Group Chats which are currently disabled.
// nolint: unused
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

func WithVerifyTransactionClient(client EthClient) Option {
	return func(c *config) error {
		c.verifyTransactionClient = client
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

	onNewInstallationsHandler := func(installations []*multidevice.Installation) {

		for _, installation := range installations {
			if installation.Identity == contactIDFromPublicKey(&messenger.identity.PublicKey) {
				if _, ok := messenger.allInstallations[installation.ID]; !ok {
					messenger.allInstallations[installation.ID] = installation
					messenger.modifiedInstallations[installation.ID] = true
				}
			}
		}
	}
	// Set default config fields.
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
	err := sqlite.Migrate(database)
	if err != nil {
		return nil, errors.Wrap(err, "failed to apply migrations")
	}

	// Initialize transport layer.
	var transp transport.Transport
	if shh, err := node.GetWhisper(nil); err == nil && shh != nil {
		transp, err = shhtransp.NewTransport(
			shh,
			identity,
			database,
			nil,
			c.envelopesMonitorConfig,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create Transport")
		}
	} else {
		logger.Info("failed to find Whisper service; trying Waku", zap.Error(err))
		waku, err := node.GetWaku(nil)
		if err != nil || waku == nil {
			return nil, errors.Wrap(err, "failed to find Whisper and Waku services")
		}
		transp, err = wakutransp.NewTransport(
			waku,
			identity,
			database,
			nil,
			c.envelopesMonitorConfig,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create  Transport")
		}
	}

	// Initialize encryption layer.
	encryptionProtocol := encryption.New(
		database,
		installationID,
		onNewInstallationsHandler,
		onNewSharedSecretHandler,
		c.onSendContactCodeHandler,
		logger,
	)

	processor, err := newMessageProcessor(
		identity,
		database,
		encryptionProtocol,
		transp,
		logger,
		c.featureFlags,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create messageProcessor")
	}

	handler := newMessageHandler(identity, logger, &sqlitePersistence{db: database})

	messenger = &Messenger{
		node:                       node,
		identity:                   identity,
		persistence:                &sqlitePersistence{db: database},
		transport:                  transp,
		encryptor:                  encryptionProtocol,
		processor:                  processor,
		handler:                    handler,
		featureFlags:               c.featureFlags,
		systemMessagesTranslations: c.systemMessagesTranslations,
		allChats:                   make(map[string]*Chat),
		allContacts:                make(map[string]*Contact),
		allInstallations:           make(map[string]*multidevice.Installation),
		installationID:             installationID,
		modifiedInstallations:      make(map[string]bool),
		messagesPersistenceEnabled: c.messagesPersistenceEnabled,
		verifyTransactionClient:    c.verifyTransactionClient,
		shutdownTasks: []func() error{
			database.Close,
			transp.ResetFilters,
			transp.Stop,
			func() error { processor.Stop(); return nil },
			// Currently this often fails, seems like it's safe to ignore them
			// https://github.com/uber-go/zap/issues/328
			func() error { _ = logger.Sync; return nil },
		},
		logger: logger,
	}

	logger.Debug("messages persistence", zap.Bool("enabled", c.messagesPersistenceEnabled))

	return messenger, nil
}

func (m *Messenger) Start() error {
	return m.encryptor.Start(m.identity)
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
		if err := chat.Validate(); err != nil {
			logger.Warn("failed to validate chat", zap.Error(err))
			continue
		}

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

	installations, err := m.encryptor.GetOurInstallations(&m.identity.PublicKey)
	if err != nil {
		return err
	}

	for _, installation := range installations {
		m.allInstallations[installation.ID] = installation
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
	m.mutex.Lock()
	defer m.mutex.Unlock()

	installation, ok := m.allInstallations[id]
	if !ok {
		return errors.New("no installation found")
	}

	err := m.encryptor.EnableInstallation(&m.identity.PublicKey, id)
	if err != nil {
		return err
	}
	installation.Enabled = true
	m.allInstallations[id] = installation
	return nil
}

func (m *Messenger) DisableInstallation(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	installation, ok := m.allInstallations[id]
	if !ok {
		return errors.New("no installation found")
	}

	err := m.encryptor.DisableInstallation(&m.identity.PublicKey, id)
	if err != nil {
		return err
	}
	installation.Enabled = false
	m.allInstallations[id] = installation
	return nil
}

func (m *Messenger) Installations() []*multidevice.Installation {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	installations := make([]*multidevice.Installation, len(m.allInstallations))

	var i = 0
	for _, installation := range m.allInstallations {
		installations[i] = installation
		i++
	}
	return installations
}

func (m *Messenger) setInstallationMetadata(id string, data *multidevice.InstallationMetadata) error {
	installation, ok := m.allInstallations[id]
	if !ok {
		return errors.New("no installation found")
	}

	installation.InstallationMetadata = data
	return m.encryptor.SetInstallationMetadata(&m.identity.PublicKey, id, data)
}

func (m *Messenger) SetInstallationMetadata(id string, data *multidevice.InstallationMetadata) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.setInstallationMetadata(id, data)
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
		return m.transport.JoinPublic(chat.ID)
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
	chat := CreateGroupChat(m.getTimesource())
	group, err := v1protocol.NewGroupWithCreator(name, m.identity)
	if err != nil {
		return nil, err
	}
	chat.updateChatFromProtocolGroup(group)

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	// Add members
	event := v1protocol.NewMembersAddedEvent(members, clock)
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

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	m.allChats[chat.ID] = &chat

	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:          recipients,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	chat.updateChatFromProtocolGroup(group)

	response.Chats = []*Chat{&chat}
	response.Messages = buildSystemMessages(chat.MembershipUpdates, m.systemMessagesTranslations)
	err = m.persistence.SaveMessagesLegacy(response.Messages)
	if err != nil {
		return nil, err
	}

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

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	// Remove member
	event := v1protocol.NewMemberRemovedEvent(member, clock)
	event.ChatID = chat.ID
	err = event.Sign(m.identity)
	if err != nil {
		return nil, err
	}

	err = group.ProcessEvent(event)
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:          oldRecipients,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	chat.updateChatFromProtocolGroup(group)
	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages(chat.MembershipUpdates, m.systemMessagesTranslations)
	err = m.persistence.SaveMessagesLegacy(response.Messages)
	if err != nil {
		return nil, err
	}

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

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	// Add members
	event := v1protocol.NewMembersAddedEvent(members, clock)
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

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:          recipients,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	chat.updateChatFromProtocolGroup(group)

	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	err = m.persistence.SaveMessagesLegacy(response.Messages)
	if err != nil {
		return nil, err
	}

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

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	// Add members
	event := v1protocol.NewAdminsAddedEvent(members, clock)
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

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:          recipients,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	chat.updateChatFromProtocolGroup(group)

	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	err = m.persistence.SaveMessagesLegacy(response.Messages)
	if err != nil {
		return nil, err
	}

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
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	event := v1protocol.NewMemberJoinedEvent(
		clock,
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

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:          recipients,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	chat.updateChatFromProtocolGroup(group)

	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	err = m.persistence.SaveMessagesLegacy(response.Messages)
	if err != nil {
		return nil, err
	}

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
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	event := v1protocol.NewMemberRemovedEvent(
		contactIDFromPublicKey(&m.identity.PublicKey),
		clock,
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

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:          recipients,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	chat.updateChatFromProtocolGroup(group)
	chat.Active = false

	response.Chats = []*Chat{chat}
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	err = m.persistence.SaveMessagesLegacy(response.Messages)
	if err != nil {
		return nil, err
	}

	return &response, m.saveChat(chat)
}

func (m *Messenger) saveChat(chat *Chat) error {
	_, ok := m.allChats[chat.ID]
	// Sync chat if it's a new active public chat
	if !ok && chat.Active && chat.Public() {
		if err := m.syncPublicChat(context.Background(), chat); err != nil {
			return err
		}
	}

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
	return m.saveChat(chat)
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

func (m *Messenger) isNewContact(contact *Contact) bool {
	previousContact, ok := m.allContacts[contact.ID]
	return contact.IsAdded() && (!ok || !previousContact.IsAdded())
}

func (m *Messenger) saveContact(contact *Contact) error {
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

	if m.isNewContact(contact) {
		err := m.syncContact(context.Background(), contact)
		if err != nil {
			return err
		}
	}

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	m.allContacts[contact.ID] = contact
	return nil

}
func (m *Messenger) SaveContact(contact *Contact) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.saveContact(contact)
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

// GetContactByID assumes pubKey includes 0x prefix
func (m *Messenger) GetContactByID(pubKey string) (*Contact, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	contact, ok := m.allContacts[pubKey]
	if !ok {
		return nil, errors.New("no contact found")
	}
	return contact, nil
}

// ReSendChatMessage pulls a message from the database and sends it again
func (m *Messenger) ReSendChatMessage(ctx context.Context, messageID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	message, err := m.persistence.RawMessageByID(messageID)
	if err != nil {
		return err
	}

	chat, ok := m.allChats[message.LocalChatID]
	if !ok {
		return errors.New("chat not found")
	}

	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID: chat.ID,
		Payload:     message.Payload,
		MessageType: message.MessageType,
		Recipients:  message.Recipients,
	})
	return err
}

func (m *Messenger) hasPairedDevices() bool {
	var count int
	for _, i := range m.allInstallations {
		if i.Enabled {
			count++
		}
	}
	return count > 1
}

// sendToPairedDevices will check if we have any paired devices and send to them if necessary
func (m *Messenger) sendToPairedDevices(ctx context.Context, payload []byte, messageType protobuf.ApplicationMetadataMessage_Type) error {
	hasPairedDevices := m.hasPairedDevices()
	// We send a message to any paired device
	if hasPairedDevices {
		_, err := m.processor.SendPrivateRaw(ctx, &m.identity.PublicKey, payload, messageType)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Messenger) dispatchPairInstallationMessage(ctx context.Context, spec *RawMessage) ([]byte, error) {
	var err error
	var id []byte

	id, err = m.processor.SendPairInstallation(ctx, &m.identity.PublicKey, spec.Payload, spec.MessageType)

	if err != nil {
		return nil, err
	}
	spec.ID = types.EncodeHex(id)
	spec.SendCount++
	err = m.persistence.SaveRawMessage(spec)
	if err != nil {
		return nil, err
	}

	return id, nil
}

func (m *Messenger) dispatchMessage(ctx context.Context, spec *RawMessage) ([]byte, error) {
	var err error
	var id []byte
	logger := m.logger.With(zap.String("site", "dispatchMessage"), zap.String("chatID", spec.LocalChatID))
	chat, ok := m.allChats[spec.LocalChatID]
	if !ok {
		return nil, errors.New("no chat found")
	}

	switch chat.ChatType {
	case ChatTypeOneToOne:
		publicKey, err := chat.PublicKey()
		if err != nil {
			return nil, err
		}
		if !isPubKeyEqual(publicKey, &m.identity.PublicKey) {
			id, err = m.processor.SendPrivateRaw(ctx, publicKey, spec.Payload, spec.MessageType)

			if err != nil {
				return nil, err
			}
		}

		err = m.sendToPairedDevices(ctx, spec.Payload, spec.MessageType)

		if err != nil {
			return nil, err
		}

	case ChatTypePublic:
		logger.Debug("sending public message", zap.String("chatName", chat.Name))
		id, err = m.processor.SendPublicRaw(ctx, chat.ID, spec.Payload, spec.MessageType)
		if err != nil {
			return nil, err
		}

	case ChatTypePrivateGroupChat:
		logger.Debug("sending group message", zap.String("chatName", chat.Name))
		if spec.Recipients == nil {
			spec.Recipients, err = chat.MembersAsPublicKeys()
			if err != nil {
				return nil, err
			}
		}
		hasPairedDevices := m.hasPairedDevices()

		if !hasPairedDevices {
			// Filter out my key from the recipients
			n := 0
			for _, recipient := range spec.Recipients {
				if !isPubKeyEqual(recipient, &m.identity.PublicKey) {
					spec.Recipients[n] = recipient
					n++
				}
			}
			spec.Recipients = spec.Recipients[:n]
		}

		// We always wrap in group information
		id, err = m.processor.SendGroupRaw(ctx, spec.Recipients, spec.Payload, protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("chat type not supported")
	}
	spec.ID = types.EncodeHex(id)
	spec.SendCount++
	err = m.persistence.SaveRawMessage(spec)
	if err != nil {
		return nil, err
	}

	return id, nil
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

	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.getTimesource())
	if err != nil {
		return nil, err
	}

	var encodedMessage []byte
	switch chat.ChatType {
	case ChatTypeOneToOne:
		logger.Debug("sending private message")
		message.MessageType = protobuf.ChatMessage_ONE_TO_ONE
		encodedMessage, err = proto.Marshal(message)
		if err != nil {
			return nil, err
		}
	case ChatTypePublic:
		logger.Debug("sending public message", zap.String("chatName", chat.Name))
		message.MessageType = protobuf.ChatMessage_PUBLIC_GROUP
		encodedMessage, err = proto.Marshal(message)
		if err != nil {
			return nil, err
		}
	case ChatTypePrivateGroupChat:
		message.MessageType = protobuf.ChatMessage_PRIVATE_GROUP
		logger.Debug("sending group message", zap.String("chatName", chat.Name))

		group, err := newProtocolGroupFromChat(chat)
		if err != nil {
			return nil, err
		}

		encodedMessage, err = m.processor.EncodeMembershipUpdate(group, &message.ChatMessage)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("chat type not supported")
	}

	id, err := m.dispatchMessage(ctx, &RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
	})
	if err != nil {
		return nil, err
	}

	message.ID = types.EncodeHex(id)
	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.getTimesource())
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, m.saveChat(chat)
}

// Send contact updates to all contacts added by us
func (m *Messenger) SendContactUpdates(ctx context.Context, ensName, profileImage string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	myID := contactIDFromPublicKey(&m.identity.PublicKey)

	if _, err := m.sendContactUpdate(ctx, myID, ensName, profileImage); err != nil {
		return err
	}

	// TODO: This should not be sending paired messages, as we do it above
	for _, contact := range m.allContacts {
		if contact.IsAdded() {
			if _, err := m.sendContactUpdate(ctx, contact.ID, ensName, profileImage); err != nil {
				return err
			}
		}
	}
	return nil
}

// SendContactUpdate sends a contact update to a user and adds the user to contacts
func (m *Messenger) SendContactUpdate(ctx context.Context, chatID, ensName, profileImage string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.sendContactUpdate(ctx, chatID, ensName, profileImage)
}

func (m *Messenger) sendContactUpdate(ctx context.Context, chatID, ensName, profileImage string) (*MessengerResponse, error) {
	var response MessengerResponse

	contact, ok := m.allContacts[chatID]
	if !ok {
		pubkeyBytes, err := types.DecodeHex(chatID)
		if err != nil {
			return nil, err
		}

		publicKey, err := crypto.UnmarshalPubkey(pubkeyBytes)
		if err != nil {
			return nil, err
		}

		contact, err = buildContact(publicKey)
		if err != nil {
			return nil, err
		}
	}

	chat, ok := m.allChats[chatID]
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return nil, err
		}
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats[chat.ID] = chat
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	contactUpdate := &protobuf.ContactUpdate{
		Clock:        clock,
		EnsName:      ensName,
		ProfileImage: profileImage}
	encodedMessage, err := proto.Marshal(contactUpdate)
	if err != nil {
		return nil, err
	}

	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_CONTACT_UPDATE,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	if !contact.IsAdded() && contact.ID != contactIDFromPublicKey(&m.identity.PublicKey) {
		contact.SystemTags = append(contact.SystemTags, contactAdded)
	}

	response.Contacts = []*Contact{contact}
	response.Chats = []*Chat{chat}

	chat.LastClockValue = clock
	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}
	return &response, m.saveContact(contact)
}

// SyncDevices sends all public chats and contacts to paired devices
func (m *Messenger) SyncDevices(ctx context.Context, ensName, photoPath string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	myID := contactIDFromPublicKey(&m.identity.PublicKey)

	if _, err := m.sendContactUpdate(ctx, myID, ensName, photoPath); err != nil {
		return err
	}

	for _, chat := range m.allChats {
		if chat.Public() && chat.Active {
			if err := m.syncPublicChat(ctx, chat); err != nil {
				return err
			}
		}
	}

	for _, contact := range m.allContacts {
		if contact.IsAdded() && contact.ID != myID {
			if err := m.syncContact(ctx, contact); err != nil {
				return err
			}
		}
	}

	return nil
}

// SendPairInstallation sends a pair installation message
func (m *Messenger) SendPairInstallation(ctx context.Context) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var err error
	var response MessengerResponse

	installation, ok := m.allInstallations[m.installationID]
	if !ok {
		return nil, errors.New("no installation found")
	}

	if installation.InstallationMetadata == nil {
		return nil, errors.New("no installation metadata")
	}

	chatID := contactIDFromPublicKey(&m.identity.PublicKey)

	chat, ok := m.allChats[chatID]
	if !ok {
		chat = OneToOneFromPublicKey(&m.identity.PublicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats[chat.ID] = chat
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	pairMessage := &protobuf.PairInstallation{
		Clock:          clock,
		Name:           installation.InstallationMetadata.Name,
		InstallationId: installation.ID,
		DeviceType:     installation.InstallationMetadata.DeviceType}
	encodedMessage, err := proto.Marshal(pairMessage)
	if err != nil {
		return nil, err
	}

	_, err = m.dispatchPairInstallationMessage(ctx, &RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_PAIR_INSTALLATION,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}

	chat.LastClockValue = clock
	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// syncPublicChat sync a public chat with paired devices
func (m *Messenger) syncPublicChat(ctx context.Context, publicChat *Chat) error {
	var err error
	if !m.hasPairedDevices() {
		return nil
	}
	chatID := contactIDFromPublicKey(&m.identity.PublicKey)

	chat, ok := m.allChats[chatID]
	if !ok {
		chat = OneToOneFromPublicKey(&m.identity.PublicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats[chat.ID] = chat
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	syncMessage := &protobuf.SyncInstallationPublicChat{
		Clock: clock,
		Id:    publicChat.ID,
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_PUBLIC_CHAT,
		ResendAutomatically: true,
	})
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

// syncContact sync as contact with paired devices
func (m *Messenger) syncContact(ctx context.Context, contact *Contact) error {
	var err error
	if !m.hasPairedDevices() {
		return nil
	}
	chatID := contactIDFromPublicKey(&m.identity.PublicKey)

	chat, ok := m.allChats[chatID]
	if !ok {
		chat = OneToOneFromPublicKey(&m.identity.PublicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats[chat.ID] = chat
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	syncMessage := &protobuf.SyncInstallationContact{
		Clock:        clock,
		Id:           contact.ID,
		EnsName:      contact.Name,
		ProfileImage: contact.Photo,
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_INSTALLATION_CONTACT,
		ResendAutomatically: true,
	})
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
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

type CurrentMessageState struct {
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
}

type ReceivedMessageState struct {
	// State on the message being processed
	CurrentMessageState *CurrentMessageState
	// AllChats in memory
	AllChats map[string]*Chat
	// List of chats modified
	ModifiedChats map[string]bool
	// All contacts in memory
	AllContacts map[string]*Contact
	// List of contacts modified
	ModifiedContacts map[string]bool
	// All installations in memory
	AllInstallations map[string]*multidevice.Installation
	// List of installations modified
	ModifiedInstallations map[string]bool
	// Map of existing messages
	ExistingMessagesMap map[string]bool
	// Response to the client
	Response *MessengerResponse
	// Timesource is a time source for clock values/timestamps.
	Timesource TimeSource
}

func (m *Messenger) handleRetrievedMessages(chatWithMessages map[transport.Filter][]*types.Message) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	messageState := &ReceivedMessageState{
		AllChats:              m.allChats,
		ModifiedChats:         make(map[string]bool),
		AllContacts:           m.allContacts,
		ModifiedContacts:      make(map[string]bool),
		AllInstallations:      m.allInstallations,
		ModifiedInstallations: m.modifiedInstallations,
		ExistingMessagesMap:   make(map[string]bool),
		Response:              &MessengerResponse{},
		Timesource:            m.getTimesource(),
	}

	logger := m.logger.With(zap.String("site", "RetrieveAll"))
	rawMessages := make(map[transport.Filter][]*v1protocol.StatusMessage)

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
				senderID := contactIDFromPublicKey(publicKey)
				if _, ok := messageState.AllContacts[senderID]; ok && messageState.AllContacts[senderID].IsBlocked() {
					continue
				}
				// Don't process duplicates
				messageID := types.EncodeHex(msg.ID)
				exists, err := m.handler.messageExists(messageID, messageState.ExistingMessagesMap)
				if err != nil {
					logger.Warn("failed to check message exists", zap.Error(err))
				}
				if exists {
					continue
				}

				var contact *Contact
				if c, ok := messageState.AllContacts[senderID]; ok {
					contact = c
				} else {
					c, err := buildContact(publicKey)
					if err != nil {
						logger.Info("failed to build contact", zap.Error(err))
						continue
					}
					contact = c
					messageState.AllContacts[senderID] = c
					messageState.ModifiedContacts[contact.ID] = true
				}
				messageState.CurrentMessageState = &CurrentMessageState{
					MessageID:        messageID,
					WhisperTimestamp: uint64(msg.TransportMessage.Timestamp) * 1000,
					Contact:          contact,
					PublicKey:        publicKey,
				}

				if msg.ParsedMessage != nil {
					logger.Debug("Handling parsed message")
					switch msg.ParsedMessage.(type) {
					case protobuf.MembershipUpdateMessage:
						logger.Debug("Handling MembershipUpdateMessage")

						rawMembershipUpdate := msg.ParsedMessage.(protobuf.MembershipUpdateMessage)

						err = m.handler.HandleMembershipUpdate(messageState, messageState.AllChats[rawMembershipUpdate.ChatId], rawMembershipUpdate, m.systemMessagesTranslations)
						if err != nil {
							logger.Warn("failed to handle MembershipUpdate", zap.Error(err))
							continue
						}

					case protobuf.ChatMessage:
						logger.Debug("Handling ChatMessage")
						messageState.CurrentMessageState.Message = msg.ParsedMessage.(protobuf.ChatMessage)
						err = m.handler.HandleChatMessage(messageState)
						if err != nil {
							logger.Warn("failed to handle ChatMessage", zap.Error(err))
							continue
						}
					case protobuf.PairInstallation:
						if !isPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}
						p := msg.ParsedMessage.(protobuf.PairInstallation)
						logger.Debug("Handling PairInstallation", zap.Any("message", p))
						err = m.handler.HandlePairInstallation(messageState, p)
						if err != nil {
							logger.Warn("failed to handle PairInstallation", zap.Error(err))
							continue
						}

					case protobuf.SyncInstallationContact:
						if !isPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.(protobuf.SyncInstallationContact)
						logger.Debug("Handling SyncInstallationContact", zap.Any("message", p))
						err = m.handler.HandleSyncInstallationContact(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncInstallationContact", zap.Error(err))
							continue
						}
					case protobuf.SyncInstallationPublicChat:
						if !isPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.(protobuf.SyncInstallationPublicChat)
						logger.Debug("Handling SyncInstallationPublicChat", zap.Any("message", p))
						err = m.handler.HandleSyncInstallationPublicChat(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncInstallationPublicChat", zap.Error(err))
							continue
						}
					case protobuf.RequestAddressForTransaction:
						command := msg.ParsedMessage.(protobuf.RequestAddressForTransaction)
						logger.Debug("Handling RequestAddressForTransaction", zap.Any("message", command))
						err = m.handler.HandleRequestAddressForTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle RequestAddressForTransaction", zap.Error(err))
							continue
						}
					case protobuf.SendTransaction:
						command := msg.ParsedMessage.(protobuf.SendTransaction)
						logger.Debug("Handling SendTransaction", zap.Any("message", command))
						err = m.handler.HandleSendTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle SendTransaction", zap.Error(err))
							continue
						}
					case protobuf.AcceptRequestAddressForTransaction:
						command := msg.ParsedMessage.(protobuf.AcceptRequestAddressForTransaction)
						logger.Debug("Handling AcceptRequestAddressForTransaction")
						err = m.handler.HandleAcceptRequestAddressForTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle AcceptRequestAddressForTransaction", zap.Error(err))
							continue
						}

					case protobuf.DeclineRequestAddressForTransaction:
						command := msg.ParsedMessage.(protobuf.DeclineRequestAddressForTransaction)
						logger.Debug("Handling DeclineRequestAddressForTransaction")
						err = m.handler.HandleDeclineRequestAddressForTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle DeclineRequestAddressForTransaction", zap.Error(err))
							continue
						}

					case protobuf.DeclineRequestTransaction:
						command := msg.ParsedMessage.(protobuf.DeclineRequestTransaction)
						logger.Debug("Handling DeclineRequestTransaction")
						err = m.handler.HandleDeclineRequestTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle DeclineRequestTransaction", zap.Error(err))
							continue
						}

					case protobuf.RequestTransaction:
						command := msg.ParsedMessage.(protobuf.RequestTransaction)
						logger.Debug("Handling RequestTransaction")
						err = m.handler.HandleRequestTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle RequestTransaction", zap.Error(err))
							continue
						}
					case protobuf.ContactUpdate:
						logger.Debug("Handling ContactUpdate")

						contactUpdate := msg.ParsedMessage.(protobuf.ContactUpdate)

						err = m.handler.HandleContactUpdate(messageState, contactUpdate)
						if err != nil {
							logger.Warn("failed to handle ContactUpdate", zap.Error(err))
							continue
						}

					default:
						// RawMessage, not processed here, pass straight to the client
						rawMessages[chat] = append(rawMessages[chat], msg)

					}
				} else {
					logger.Debug("Adding raw message", zap.Any("msg", msg))
					rawMessages[chat] = append(rawMessages[chat], msg)
				}
			}
		}
	}

	for id := range messageState.ModifiedChats {
		messageState.Response.Chats = append(messageState.Response.Chats, messageState.AllChats[id])
	}

	for id := range messageState.ModifiedContacts {
		messageState.Response.Contacts = append(messageState.Response.Contacts, messageState.AllContacts[id])
	}

	for id := range messageState.ModifiedInstallations {
		installation := messageState.AllInstallations[id]
		messageState.Response.Installations = append(messageState.Response.Installations, installation)
		if installation.InstallationMetadata != nil {
			err := m.setInstallationMetadata(id, installation.InstallationMetadata)
			if err != nil {
				return nil, err
			}
		}
	}

	var err error
	if len(messageState.Response.Chats) > 0 {
		err = m.saveChats(messageState.Response.Chats)
		if err != nil {
			return nil, err
		}
	}
	if len(messageState.Response.Messages) > 0 {
		err = m.SaveMessages(messageState.Response.Messages)
		if err != nil {
			return nil, err
		}
	}

	if len(messageState.Response.Contacts) > 0 {
		err = m.persistence.SaveContacts(messageState.Response.Contacts)
		if err != nil {
			return nil, err
		}
	}

	for filter, messages := range rawMessages {
		messageState.Response.RawMessages = append(messageState.Response.RawMessages, &RawResponse{Filter: &filter, Messages: messages})
	}

	// Reset installations
	m.modifiedInstallations = make(map[string]bool)

	return messageState.Response, nil
}

func (m *Messenger) RequestHistoricMessages(
	ctx context.Context,
	peer []byte, // should be removed after mailserver logic is ported
	from, to uint32,
	cursor []byte,
) ([]byte, error) {
	return m.transport.SendMessagesRequest(ctx, peer, from, to, cursor)
}

func (m *Messenger) LoadFilters(filters []*transport.Filter) ([]*transport.Filter, error) {
	return m.transport.LoadFilters(filters)
}

func (m *Messenger) RemoveFilters(filters []*transport.Filter) error {
	return m.transport.RemoveFilters(filters)
}

func (m *Messenger) ConfirmMessagesProcessed(messageIDs [][]byte) error {
	for _, id := range messageIDs {
		if err := m.encryptor.ConfirmMessageProcessed(id); err != nil {
			return err
		}
	}
	return nil
}

func (m *Messenger) MessageByID(id string) (*Message, error) {
	return m.persistence.MessageByID(id)
}

func (m *Messenger) MessagesExist(ids []string) (map[string]bool, error) {
	return m.persistence.MessagesExist(ids)
}

func (m *Messenger) MessageByChatID(chatID, cursor string, limit int) ([]*Message, string, error) {
	return m.persistence.MessageByChatID(chatID, cursor, limit)
}

func (m *Messenger) SaveMessages(messages []*Message) error {
	return m.persistence.SaveMessagesLegacy(messages)
}

func (m *Messenger) DeleteMessage(id string) error {
	return m.persistence.DeleteMessage(id)
}

func (m *Messenger) DeleteMessagesByChatID(id string) error {
	return m.persistence.DeleteMessagesByChatID(id)
}

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

func (m *Messenger) MarkAllRead(chatID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	chat, ok := m.allChats[chatID]
	if !ok {
		return errors.New("chat not found")
	}

	err := m.persistence.MarkAllRead(chatID)
	if err != nil {
		return err
	}

	chat.UnviewedMessagesCount = 0
	m.allChats[chat.ID] = chat
	return nil
}

func (m *Messenger) UpdateMessageOutgoingStatus(id, newOutgoingStatus string) error {
	return m.persistence.UpdateMessageOutgoingStatus(id, newOutgoingStatus)
}

// Identicon returns an identicon based on the input string
func Identicon(id string) (string, error) {
	return identicon.GenerateBase64(id)
}

// VerifyENSNames verifies that a registered ENS name matches the expected public key
func (m *Messenger) VerifyENSNames(ctx context.Context, rpcEndpoint, contractAddress string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Debug("verifying ENS Names", zap.String("endpoint", rpcEndpoint))
	verifier := m.node.NewENSVerifier(m.logger)

	var response MessengerResponse

	var ensDetails []enstypes.ENSDetails

	now := m.getTimesource().GetCurrentTime()
	for _, contact := range m.allContacts {
		if shouldENSBeVerified(contact, now) {
			ensDetails = append(ensDetails, enstypes.ENSDetails{
				PublicKeyString: contact.ID[2:],
				Name:            contact.Name,
			})
		}
	}

	ensResponse, err := verifier.CheckBatch(ensDetails, rpcEndpoint, contractAddress)
	if err != nil {
		return nil, err
	}

	for _, details := range ensResponse {
		contact, ok := m.allContacts["0x"+details.PublicKeyString]
		if !ok {
			return nil, errors.New("contact must be existing")
		}

		m.logger.Debug("verifying ENS Name", zap.Any("details", details), zap.Any("contact", contact))

		contact.ENSVerifiedAt = uint64(details.VerifiedAt)

		if details.Error == nil {
			contact.ENSVerified = details.Verified
			m.allContacts[contact.ID] = contact
		} else {
			m.logger.Warn("Failed to resolve ens name",
				zap.String("name", details.Name),
				zap.String("publicKey", details.PublicKeyString),
				zap.Error(details.Error),
			)
		}
		response.Contacts = append(response.Contacts, contact)
	}

	if len(response.Contacts) != 0 {
		err = m.persistence.SaveContacts(response.Contacts)
		if err != nil {
			return nil, err
		}
	}

	return &response, nil
}

// GenerateAlias name returns the generated name given a public key hex encoded prefixed with 0x
func GenerateAlias(id string) (string, error) {
	return alias.GenerateFromPublicKeyString(id)
}

func (m *Messenger) RequestTransaction(ctx context.Context, chatID, value, contract, address string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	message := &Message{}
	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
	if err != nil {
		return nil, err
	}

	message.MessageType = protobuf.ChatMessage_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	message.Text = "Request transaction"

	request := &protobuf.RequestTransaction{
		Clock:    message.Clock,
		Address:  address,
		Value:    value,
		Contract: contract,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	id, err := m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_REQUEST_TRANSACTION,
		ResendAutomatically: true,
	})

	message.CommandParameters = &CommandParameters{
		ID:           types.EncodeHex(id),
		Value:        value,
		Address:      address,
		Contract:     contract,
		CommandState: CommandStateRequestTransaction,
	}

	if err != nil {
		return nil, err
	}
	messageID := types.EncodeHex(id)

	message.ID = messageID
	message.CommandParameters.ID = messageID
	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, m.saveChat(chat)
}

func (m *Messenger) RequestAddressForTransaction(ctx context.Context, chatID, from, value, contract string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	message := &Message{}
	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
	if err != nil {
		return nil, err
	}

	message.MessageType = protobuf.ChatMessage_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	message.Text = "Request address for transaction"

	request := &protobuf.RequestAddressForTransaction{
		Clock:    message.Clock,
		Value:    value,
		Contract: contract,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}
	id, err := m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_REQUEST_ADDRESS_FOR_TRANSACTION,
		ResendAutomatically: true,
	})

	message.CommandParameters = &CommandParameters{
		ID:           types.EncodeHex(id),
		From:         from,
		Value:        value,
		Contract:     contract,
		CommandState: CommandStateRequestAddressForTransaction,
	}

	if err != nil {
		return nil, err
	}
	messageID := types.EncodeHex(id)

	message.ID = messageID
	message.CommandParameters.ID = messageID
	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, m.saveChat(chat)
}

func (m *Messenger) AcceptRequestAddressForTransaction(ctx context.Context, messageID, address string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	message, err := m.MessageByID(messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, errors.New("message not found")
	}

	chatID := message.LocalChatID

	// A valid added chat is required.
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Timestamp = timestamp
	message.Text = "Request address for transaction accepted"
	message.OutgoingStatus = OutgoingStatusSending

	// Hide previous message
	previousMessage, err := m.persistence.MessageByCommandID(chatID, messageID)
	if err != nil {
		return nil, err
	}

	if previousMessage == nil {
		return nil, errors.New("No previous message found")
	}

	err = m.persistence.HideMessage(previousMessage.ID)
	if err != nil {
		return nil, err
	}

	message.Replace = previousMessage.ID

	request := &protobuf.AcceptRequestAddressForTransaction{
		Clock:   message.Clock,
		Id:      messageID,
		Address: address,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	newMessageID, err := m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_ACCEPT_REQUEST_ADDRESS_FOR_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = types.EncodeHex(newMessageID)
	message.CommandParameters.Address = address
	message.CommandParameters.CommandState = CommandStateRequestAddressForTransactionAccepted

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, m.saveChat(chat)
}

func (m *Messenger) DeclineRequestTransaction(ctx context.Context, messageID string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	message, err := m.MessageByID(messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, errors.New("message not found")
	}

	chatID := message.LocalChatID

	// A valid added chat is required.
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Timestamp = timestamp
	message.Text = "Transaction request declined"
	message.OutgoingStatus = OutgoingStatusSending
	message.Replace = messageID

	err = m.persistence.HideMessage(messageID)
	if err != nil {
		return nil, err
	}

	request := &protobuf.DeclineRequestTransaction{
		Clock: message.Clock,
		Id:    messageID,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	newMessageID, err := m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = types.EncodeHex(newMessageID)
	message.CommandParameters.CommandState = CommandStateRequestTransactionDeclined

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, m.saveChat(chat)
}

func (m *Messenger) DeclineRequestAddressForTransaction(ctx context.Context, messageID string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	message, err := m.MessageByID(messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, errors.New("message not found")
	}

	chatID := message.LocalChatID

	// A valid added chat is required.
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Timestamp = timestamp
	message.Text = "Request address for transaction declined"
	message.OutgoingStatus = OutgoingStatusSending
	message.Replace = messageID

	err = m.persistence.HideMessage(messageID)
	if err != nil {
		return nil, err
	}

	request := &protobuf.DeclineRequestAddressForTransaction{
		Clock: message.Clock,
		Id:    messageID,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	newMessageID, err := m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_ADDRESS_FOR_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = types.EncodeHex(newMessageID)
	message.CommandParameters.CommandState = CommandStateRequestAddressForTransactionDeclined

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, m.saveChat(chat)
}

func (m *Messenger) AcceptRequestTransaction(ctx context.Context, transactionHash, messageID string, signature []byte) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	message, err := m.MessageByID(messageID)
	if err != nil {
		return nil, err
	}

	if message == nil {
		return nil, errors.New("message not found")
	}

	chatID := message.LocalChatID

	// A valid added chat is required.
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Timestamp = timestamp
	message.Text = transactionSentTxt
	message.OutgoingStatus = OutgoingStatusSending

	// Hide previous message
	previousMessage, err := m.persistence.MessageByCommandID(chatID, messageID)
	if err != nil && err != errRecordNotFound {
		return nil, err
	}

	if previousMessage != nil {
		err = m.persistence.HideMessage(previousMessage.ID)
		if err != nil {
			return nil, err
		}
		message.Replace = previousMessage.ID
	}

	err = m.persistence.HideMessage(messageID)
	if err != nil {
		return nil, err
	}

	request := &protobuf.SendTransaction{
		Clock:           message.Clock,
		Id:              messageID,
		TransactionHash: transactionHash,
		Signature:       signature,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	newMessageID, err := m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SEND_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = types.EncodeHex(newMessageID)
	message.CommandParameters.TransactionHash = transactionHash
	message.CommandParameters.Signature = signature
	message.CommandParameters.CommandState = CommandStateTransactionSent

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, m.saveChat(chat)
}

func (m *Messenger) SendTransaction(ctx context.Context, chatID, value, contract, transactionHash string, signature []byte) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, errors.New("Chat not found")
	}
	if chat.ChatType != ChatTypeOneToOne {
		return nil, errors.New("Need to be a one-to-one chat")
	}

	message := &Message{}
	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
	if err != nil {
		return nil, err
	}

	message.MessageType = protobuf.ChatMessage_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	message.LocalChatID = chatID

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Timestamp = timestamp
	message.Text = transactionSentTxt

	request := &protobuf.SendTransaction{
		Clock:           message.Clock,
		TransactionHash: transactionHash,
		Signature:       signature,
	}
	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	newMessageID, err := m.dispatchMessage(ctx, &RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SEND_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = types.EncodeHex(newMessageID)
	message.CommandParameters = &CommandParameters{
		TransactionHash: transactionHash,
		Value:           value,
		Contract:        contract,
		Signature:       signature,
		CommandState:    CommandStateTransactionSent,
	}

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessagesLegacy([]*Message{message})
	if err != nil {
		return nil, err
	}

	response.Chats = []*Chat{chat}
	response.Messages = []*Message{message}
	return &response, m.saveChat(chat)
}

func (m *Messenger) ValidateTransactions(ctx context.Context, addresses []types.Address) (*MessengerResponse, error) {
	if m.verifyTransactionClient == nil {
		return nil, nil
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	modifiedChats := make(map[string]bool)

	logger := m.logger.With(zap.String("site", "ValidateTransactions"))
	logger.Debug("Validating transactions")
	txs, err := m.persistence.TransactionsToValidate()
	if err != nil {
		logger.Error("Error pulling", zap.Error(err))
		return nil, err
	}
	logger.Debug("Txs", zap.Int("count", len(txs)), zap.Any("txs", txs))
	var response MessengerResponse
	validator := NewTransactionValidator(addresses, m.persistence, m.verifyTransactionClient, m.logger)
	responses, err := validator.ValidateTransactions(ctx)
	if err != nil {
		logger.Error("Error validating", zap.Error(err))
		return nil, err
	}
	for _, validationResult := range responses {
		var message *Message
		chatID := contactIDFromPublicKey(validationResult.Transaction.From)
		chat, ok := m.allChats[chatID]
		if !ok {
			chat = OneToOneFromPublicKey(validationResult.Transaction.From, m.transport)
		}
		if validationResult.Message != nil {
			message = validationResult.Message
		} else {
			message = &Message{}
			err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
			if err != nil {
				return nil, err
			}
		}

		message.MessageType = protobuf.ChatMessage_ONE_TO_ONE
		message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
		message.LocalChatID = chatID
		message.OutgoingStatus = ""

		clock, timestamp := chat.NextClockAndTimestamp(m.transport)
		message.Clock = clock
		message.Timestamp = timestamp
		message.WhisperTimestamp = timestamp
		message.Text = "Transaction received"

		message.ID = validationResult.Transaction.MessageID
		if message.CommandParameters == nil {
			message.CommandParameters = &CommandParameters{}
		} else {
			message.CommandParameters = validationResult.Message.CommandParameters
		}

		message.CommandParameters.Value = validationResult.Value
		message.CommandParameters.Contract = validationResult.Contract
		message.CommandParameters.Address = validationResult.Address
		message.CommandParameters.CommandState = CommandStateTransactionSent
		message.CommandParameters.TransactionHash = validationResult.Transaction.TransactionHash

		err = message.PrepareContent()
		if err != nil {
			return nil, err
		}

		err = chat.UpdateFromMessage(message, m.transport)
		if err != nil {
			return nil, err
		}

		if len(message.CommandParameters.ID) != 0 {
			// Hide previous message
			previousMessage, err := m.persistence.MessageByCommandID(chatID, message.CommandParameters.ID)
			if err != nil && err != errRecordNotFound {
				return nil, err
			}

			if previousMessage != nil {
				err = m.persistence.HideMessage(previousMessage.ID)
				if err != nil {
					return nil, err
				}
				message.Replace = previousMessage.ID
			}
		}

		response.Messages = append(response.Messages, message)
		m.allChats[chat.ID] = chat
		modifiedChats[chat.ID] = true

	}
	for id := range modifiedChats {
		response.Chats = append(response.Chats, m.allChats[id])
	}

	if len(response.Messages) > 0 {
		err = m.SaveMessages(response.Messages)
		if err != nil {
			return nil, err
		}
	}
	return &response, nil
}

func (m *Messenger) getTimesource() TimeSource {
	return m.transport
}

func (m *Messenger) Timesource() TimeSource {
	return m.getTimesource()
}
