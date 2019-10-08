package statusproto

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-protocol-go/encryption"
	"github.com/status-im/status-protocol-go/encryption/multidevice"
	"github.com/status-im/status-protocol-go/encryption/sharedsecret"
	"github.com/status-im/status-protocol-go/ens"
	"github.com/status-im/status-protocol-go/identity/alias"
	"github.com/status-im/status-protocol-go/identity/identicon"
	"github.com/status-im/status-protocol-go/sqlite"
	transport "github.com/status-im/status-protocol-go/transport/whisper"
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	statusproto "github.com/status-im/status-protocol-go/types"
	protocol "github.com/status-im/status-protocol-go/v1"
)

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
	identity    *ecdsa.PrivateKey
	persistence *sqlitePersistence
	transport   *transport.WhisperServiceTransport
	encryptor   *encryption.Protocol
	processor   *messageProcessor
	logger      *zap.Logger

	ownMessages                []*protocol.Message
	featureFlags               featureFlags
	messagesPersistenceEnabled bool
	shutdownTasks              []func() error
}

type featureFlags struct {
	genericDiscoveryTopicEnabled bool
	// sendV1Messages indicates whether we should send
	// messages compatible only with V1 and later.
	// V1 messages adds additional wrapping
	// which contains a signature and payload.
	sendV1Messages bool

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

func WithGenericDiscoveryTopicSupport() Option {
	return func(c *config) error {
		c.featureFlags.genericDiscoveryTopicEnabled = true
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

func WithSendV1Messages() Option {
	return func(c *config) error {
		c.featureFlags.sendV1Messages = true
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
	shh whispertypes.Whisper,
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
	err := sqlite.Migrate(database)
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
		transport.SetGenericDiscoveryTopicSupport(c.featureFlags.genericDiscoveryTopicEnabled),
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
		fSecret := whispertypes.NegotiatedSecret{
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
	group, err := protocol.NewGroupWithCreator(name, m.identity)
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
		encodedMembers[idx] = hexutil.Encode(crypto.FromECDSAPub(member))
	}
	event := protocol.NewMembersAddedEvent(encodedMembers, group.NextClockValue())
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
	event := protocol.NewMemberJoinedEvent(
		statusproto.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
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

func (m *Messenger) propagateMembershipUpdates(ctx context.Context, group *protocol.Group) error {
	events := make([]protocol.MembershipUpdateEvent, len(group.Updates()))
	for idx, event := range group.Updates() {
		events[idx] = event.MembershipUpdateEvent
	}
	update := protocol.MembershipUpdate{
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
		[]protocol.MembershipUpdate{update},
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

func (m *Messenger) Send(ctx context.Context, chatID string, data []byte) ([][]byte, error) {
	logger := m.logger.With(zap.String("site", "Send"), zap.String("chatID", chatID))

	// A valid added chat is required.
	chat, err := m.chatByID(chatID)
	if err != nil {
		return nil, err
	}

	clock, err := m.persistence.LastMessageClock(chat.ID)
	if err != nil {
		return nil, err
	}

	logger.Debug("last message clock received", zap.Int64("clock", clock))

	switch chat.ChatType {
	case ChatTypeOneToOne:
		logger.Debug("sending private message", zap.Binary("publicKey", crypto.FromECDSAPub(chat.PublicKey)))
		id, message, err := m.processor.SendPrivate(ctx, chat.PublicKey, chat.ID, data, clock)
		if err != nil {
			return nil, err
		}
		if err := m.cacheOwnMessage(chatID, id, message); err != nil {
			return nil, err
		}
		return [][]byte{id}, nil
	case ChatTypePublic:
		logger.Debug("sending public message", zap.String("chatName", chat.Name))
		id, err := m.processor.SendPublic(ctx, chat.ID, data, clock)
		if err != nil {
			return nil, err
		}
		return [][]byte{id}, nil
	case ChatTypePrivateGroupChat:
		logger.Debug("sending group message", zap.String("chatName", chat.Name))
		recipients, err := chat.MembersAsPublicKeys()
		if err != nil {
			return nil, err
		}
		// Filter me out of recipients.
		n := 0
		for _, item := range recipients {
			if !isPubKeyEqual(item, &m.identity.PublicKey) {
				recipients[n] = item
				n++
			}
		}
		ids, messages, err := m.processor.SendGroup(ctx, recipients[:n], chat.ID, data, clock)
		if err != nil {
			return nil, err
		}
		for idx, message := range messages {
			if err := m.cacheOwnMessage(chatID, ids[idx], message); err != nil {
				return nil, err
			}
		}
		return ids, nil
	default:
		return nil, errors.New("chat is neither public nor private")
	}
}

func (m *Messenger) cacheOwnMessage(chatID string, id []byte, message *protocol.Message) error {
	// Save our message because it won't be received from the transport layer.
	message.ID = id // a Message need ID to be properly stored in the db
	message.SigPubKey = &m.identity.PublicKey
	message.ChatID = chatID

	if m.messagesPersistenceEnabled {
		_, err := m.persistence.SaveMessages([]*protocol.Message{message})
		if err != nil {
			return err
		}
	}

	// Cache it to be returned in Retrieve().
	m.ownMessages = append(m.ownMessages, message)

	return nil
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

// RetrieveAll retrieves all previously fetched messages
func (m *Messenger) RetrieveAll(ctx context.Context, c RetrieveConfig) ([]*protocol.Message, error) {
	result, err := m.retrieveLatest(ctx)
	if err != nil {
		return nil, err
	}

	postProcess := newPostProcessor(m, postProcessorConfig{
		MatchChat: true,
		Persist:   true,
	})
	result, err = postProcess.Run(result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to post process messages")
	}

	retrievedMessages, err := m.retrieveSaved(ctx, c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get saved messages")
	}
	result = append(result, retrievedMessages...)

	// Include own messages.
	result = append(result, m.ownMessages...)
	m.ownMessages = nil

	return result, nil
}

func (m *Messenger) retrieveLatest(ctx context.Context) ([]*protocol.Message, error) {
	latest, err := m.transport.RetrieveAllMessages()
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve messages")
	}

	logger := m.logger.With(zap.String("site", "retrieveLatest"))
	logger.Debug("retrieved messages", zap.Int("count", len(latest)))

	var result []*protocol.Message
	for _, transpMessage := range latest {
		protoMessages, err := m.processor.Process(transpMessage.Message)
		if err != nil {
			return nil, err
		}
		result = append(result, protoMessages...)
	}
	return result, nil
}

func (m *Messenger) retrieveSaved(ctx context.Context, c RetrieveConfig) (messages []*protocol.Message, err error) {
	if !m.messagesPersistenceEnabled {
		return nil, nil
	}

	if !c.latest {
		return m.persistence.Messages(c.From, c.To)
	}

	if c.last24Hours {
		to := time.Now()
		from := to.Add(-time.Hour * 24)
		return m.persistence.Messages(from, to)
	}

	return nil, nil
}

// DEPRECATED
func (m *Messenger) RetrieveRawAll() (map[transport.Filter][]*protocol.StatusMessage, error) {
	chatWithMessages, err := m.transport.RetrieveRawAll()
	if err != nil {
		return nil, err
	}

	logger := m.logger.With(zap.String("site", "RetrieveRawAll"))
	result := make(map[transport.Filter][]*protocol.StatusMessage)

	for chat, messages := range chatWithMessages {
		for _, shhMessage := range messages {
			// TODO: fix this to use an exported method.
			statusMessages, err := m.processor.handleMessages(shhMessage, false)
			if err != nil {
				logger.Info("failed to decode messages", zap.Error(err))
				continue
			}

			result[chat] = append(result[chat], statusMessages...)
		}
	}

	err = m.saveContacts(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *Messenger) saveContacts(messages map[transport.Filter][]*protocol.StatusMessage) error {
	allContactsMap := make(map[string]bool)
	var allContacts []Contact
	for _, chatMessages := range messages {
		for _, message := range chatMessages {
			publicKey := message.SigPubKey()
			address := strings.ToLower(crypto.PubkeyToAddress(*publicKey).Hex())

			if _, ok := allContactsMap[address]; ok {
				continue
			}
			contact, err := buildContact(publicKey)
			if err != nil {
				continue
			}

			allContactsMap[address] = true
			allContacts = append(allContacts, *contact)
		}
	}
	return m.persistence.SetContactsGeneratedData(allContacts, nil)
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
func (m *Messenger) MarkMessagesSeen(ids ...string) error {
	return m.persistence.MarkMessagesSeen(ids...)
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
}

func newPostProcessor(m *Messenger, config postProcessorConfig) *postProcessor {
	return &postProcessor{
		myPublicKey: &m.identity.PublicKey,
		persistence: m.persistence,
		logger:      m.logger,
		config:      config,
	}
}

func (p *postProcessor) Run(messages []*protocol.Message) ([]*protocol.Message, error) {
	var err error

	p.logger.Debug("running post processor", zap.Int("messages", len(messages)))

	var fns []func([]*protocol.Message) ([]*protocol.Message, error)

	// Order is important. Persisting messages should be always at the end.
	if p.config.MatchChat {
		fns = append(fns, p.matchMessages)
	}
	if p.config.Persist {
		fns = append(fns, p.saveMessages)
	}

	for _, fn := range fns {
		messages, err = fn(messages)
		if err != nil {
			return nil, err
		}
	}

	return messages, nil
}

func (p *postProcessor) saveMessages(messages []*protocol.Message) ([]*protocol.Message, error) {
	_, err := p.persistence.SaveMessages(messages)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (p *postProcessor) matchMessages(messages []*protocol.Message) ([]*protocol.Message, error) {
	chats, err := p.persistence.Chats()
	if err != nil {
		return nil, err
	}

	result := make([]*protocol.Message, 0, len(messages))
	for _, message := range messages {
		chat, err := p.matchMessage(message, chats)
		if err != nil {
			p.logger.Error("failed to match a chat to a message", zap.Error(err))
			continue
		}
		message.ChatID = chat.ID
		result = append(result, message)
	}
	return result, nil
}

func (p *postProcessor) matchMessage(message *protocol.Message, chats []*Chat) (*Chat, error) {
	if message.SigPubKey == nil {
		p.logger.Error("public key can't be empty")
		return nil, errors.New("received a message with empty public key")
	}

	switch {
	case message.MessageT == protocol.MessageTypePublicGroup:
		// For public messages, all outgoing and incoming messages have the same chatID
		// equal to a public chat name.
		chatID := message.Content.ChatID
		chat := findChatByID(chatID, chats)
		if chat == nil {
			return nil, errors.New("received a public message from non-existing chat")
		}
		return chat, nil
	case message.MessageT == protocol.MessageTypePrivate && isPubKeyEqual(message.SigPubKey, p.myPublicKey):
		// It's a private message coming from us so we rely on Message.Content.ChatID.
		// If chat does not exist, it should be created to support multidevice synchronization.
		chatID := message.Content.ChatID
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
	case message.MessageT == protocol.MessageTypePrivate:
		// It's an incoming private message. ChatID is calculated from the signature.
		// If a chat does not exist, a new one is created and saved.
		chatID := statusproto.EncodeHex(crypto.FromECDSAPub(message.SigPubKey))
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
	case message.MessageT == protocol.MessageTypePrivateGroup:
		// In the case of a group message, ChatID is the same for all messages belonging to a group.
		// It needs to be verified if the signature public key belongs to the chat.
		chatID := message.Content.ChatID
		chat := findChatByID(chatID, chats)
		if chat == nil {
			return nil, errors.New("received group chat message for non-existing chat")
		}

		sigPubKeyHex := statusproto.EncodeHex(crypto.FromECDSAPub(message.SigPubKey))
		for _, member := range chat.Members {
			if member.ID == sigPubKeyHex {
				return chat, nil
			}
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

// VerifyENSName verifies that a registered ENS name matches the expected public key
func (m *Messenger) VerifyENSNames(rpcEndpoint, contractAddress string, ensDetails []ens.ENSDetails) (map[string]ens.ENSResponse, error) {
	verifier := ens.NewVerifier(m.logger)

	ensResponse, err := verifier.CheckBatch(ensDetails, rpcEndpoint, contractAddress)
	if err != nil {
		return nil, err
	}

	// Update contacts
	var contacts []Contact
	for _, details := range ensResponse {
		if details.Error == nil {
			contact, err := buildContact(details.PublicKey)
			if err != nil {
				return nil, err
			}
			contact.ENSVerified = details.Verified
			contact.ENSVerifiedAt = details.VerifiedAt
			contact.Name = details.Name

			contacts = append(contacts, *contact)
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
