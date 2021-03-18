package protocol

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	userimage "github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/protocol/audio"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/encryption/sharedsecret"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/identity/identicon"
	"github.com/status-im/status-go/protocol/images"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/pushnotificationclient"
	"github.com/status-im/status-go/protocol/pushnotificationserver"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/transport"
	wakutransp "github.com/status-im/status-go/protocol/transport/waku"
	shhtransp "github.com/status-im/status-go/protocol/transport/whisper"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/services/mailservers"
)

type chatContext string

const (
	PubKeyStringLength = 132

	transactionSentTxt = "Transaction sent"

	publicChat  chatContext = "public-chat"
	privateChat chatContext = "private-chat"
)

const emojiResendMinDelay = 30
const emojiResendMaxCount = 3

var communityAdvertiseIntervalSecond int64 = 60 * 60

// messageCacheIntervalMs is how long we should keep processed messages in the cache, in ms
var messageCacheIntervalMs uint64 = 1000 * 60 * 60 * 48

// Messenger is a entity managing chats and messages.
// It acts as a bridge between the application and encryption
// layers.
// It needs to expose an interface to manage installations
// because installations are managed by the user.
// Similarly, it needs to expose an interface to manage
// mailservers because they can also be managed by the user.
type Messenger struct {
	node                       types.Node
	config                     *config
	identity                   *ecdsa.PrivateKey
	persistence                *sqlitePersistence
	transport                  transport.Transport
	encryptor                  *encryption.Protocol
	processor                  *common.MessageProcessor
	handler                    *MessageHandler
	ensVerifier                *ens.Verifier
	pushNotificationClient     *pushnotificationclient.Client
	pushNotificationServer     *pushnotificationserver.Server
	communitiesManager         *communities.Manager
	logger                     *zap.Logger
	verifyTransactionClient    EthClient
	featureFlags               common.FeatureFlags
	shutdownTasks              []func() error
	shouldPublishContactCode   bool
	systemMessagesTranslations map[protobuf.MembershipUpdateEvent_EventType]string
	allChats                   map[string]*Chat
	allContacts                map[string]*Contact
	allInstallations           map[string]*multidevice.Installation
	modifiedInstallations      map[string]bool
	installationID             string
	mailserver                 []byte
	database                   *sql.DB
	multiAccounts              *multiaccounts.Database
	account                    *multiaccounts.Account
	mailserversDatabase        *mailservers.Database
	quit                       chan struct{}

	mutex sync.Mutex
}

type dbConfig struct {
	dbPath string
	dbKey  string
}

type EnvelopeEventsInterceptor struct {
	EnvelopeEventsHandler transport.EnvelopeEventsHandler
	Messenger             *Messenger
}

// EnvelopeSent triggered when envelope delivered at least to 1 peer.
func (interceptor EnvelopeEventsInterceptor) EnvelopeSent(identifiers [][]byte) {
	if interceptor.Messenger != nil {
		var ids []string
		for _, identifierBytes := range identifiers {
			ids = append(ids, types.EncodeHex(identifierBytes))
		}

		err := interceptor.Messenger.processSentMessages(ids)
		if err != nil {
			interceptor.Messenger.logger.Info("Messenger failed to process sent messages", zap.Error(err))
		}
	}
	interceptor.EnvelopeEventsHandler.EnvelopeSent(identifiers)
}

// EnvelopeExpired triggered when envelope is expired but wasn't delivered to any peer.
func (interceptor EnvelopeEventsInterceptor) EnvelopeExpired(identifiers [][]byte, err error) {
	//we don't track expired events in Messenger, so just redirect to handler
	interceptor.EnvelopeEventsHandler.EnvelopeExpired(identifiers, err)
}

// MailServerRequestCompleted triggered when the mailserver sends a message to notify that the request has been completed
func (interceptor EnvelopeEventsInterceptor) MailServerRequestCompleted(requestID types.Hash, lastEnvelopeHash types.Hash, cursor []byte, err error) {
	//we don't track mailserver requests in Messenger, so just redirect to handler
	interceptor.EnvelopeEventsHandler.MailServerRequestCompleted(requestID, lastEnvelopeHash, cursor, err)
}

// MailServerRequestExpired triggered when the mailserver request expires
func (interceptor EnvelopeEventsInterceptor) MailServerRequestExpired(hash types.Hash) {
	//we don't track mailserver requests in Messenger, so just redirect to handler
	interceptor.EnvelopeEventsHandler.MailServerRequestExpired(hash)
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
		logger,
	)

	processor, err := common.NewMessageProcessor(
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

	// Initialize push notification server
	var pushNotificationServer *pushnotificationserver.Server
	if c.pushNotificationServerConfig != nil && c.pushNotificationServerConfig.Enabled {
		c.pushNotificationServerConfig.Identity = identity
		pushNotificationServerPersistence := pushnotificationserver.NewSQLitePersistence(database)
		pushNotificationServer = pushnotificationserver.New(c.pushNotificationServerConfig, pushNotificationServerPersistence, processor)
	}

	// Initialize push notification client
	pushNotificationClientPersistence := pushnotificationclient.NewPersistence(database)
	pushNotificationClientConfig := c.pushNotificationClientConfig
	if pushNotificationClientConfig == nil {
		pushNotificationClientConfig = &pushnotificationclient.Config{}
	}

	sqlitePersistence := NewSQLitePersistence(database)
	// Overriding until we handle different identities
	pushNotificationClientConfig.Identity = identity
	pushNotificationClientConfig.Logger = logger
	pushNotificationClientConfig.InstallationID = installationID

	pushNotificationClient := pushnotificationclient.New(pushNotificationClientPersistence, pushNotificationClientConfig, processor, sqlitePersistence)

	ensVerifier := ens.New(node, logger, transp, database, c.verifyENSURL, c.verifyENSContractAddress)

	communitiesManager, err := communities.NewManager(&identity.PublicKey, database, logger, ensVerifier)
	if err != nil {
		return nil, err
	}
	handler := newMessageHandler(identity, logger, sqlitePersistence, communitiesManager, transp, ensVerifier)

	messenger = &Messenger{
		config:                     &c,
		node:                       node,
		identity:                   identity,
		persistence:                sqlitePersistence,
		transport:                  transp,
		encryptor:                  encryptionProtocol,
		processor:                  processor,
		handler:                    handler,
		pushNotificationClient:     pushNotificationClient,
		pushNotificationServer:     pushNotificationServer,
		communitiesManager:         communitiesManager,
		ensVerifier:                ensVerifier,
		featureFlags:               c.featureFlags,
		systemMessagesTranslations: c.systemMessagesTranslations,
		allChats:                   make(map[string]*Chat),
		allContacts:                make(map[string]*Contact),
		allInstallations:           make(map[string]*multidevice.Installation),
		installationID:             installationID,
		modifiedInstallations:      make(map[string]bool),
		verifyTransactionClient:    c.verifyTransactionClient,
		database:                   database,
		multiAccounts:              c.multiAccount,
		mailserversDatabase:        c.mailserversDatabase,
		account:                    c.account,
		quit:                       make(chan struct{}),
		shutdownTasks: []func() error{
			ensVerifier.Stop,
			pushNotificationClient.Stop,
			communitiesManager.Stop,
			encryptionProtocol.Stop,
			transp.ResetFilters,
			transp.Stop,
			func() error { processor.Stop(); return nil },
			// Currently this often fails, seems like it's safe to ignore them
			// https://github.com/uber-go/zap/issues/328
			func() error { _ = logger.Sync; return nil },
			database.Close,
		},
		logger: logger,
	}

	if c.envelopesMonitorConfig != nil {
		interceptor := EnvelopeEventsInterceptor{c.envelopesMonitorConfig.EnvelopeEventsHandler, messenger}
		err := messenger.transport.SetEnvelopeEventsHandler(interceptor)
		if err != nil {
			logger.Info("Unable to set envelopes event handler", zap.Error(err))
		}
	}

	return messenger, nil
}

func (m *Messenger) processSentMessages(ids []string) error {
	for _, id := range ids {
		rawMessage, err := m.persistence.RawMessageByID(id)
		if err != nil {
			return errors.Wrapf(err, "Can't get raw message with id %v", id)
		}

		rawMessage.Sent = true

		err = m.persistence.SaveRawMessage(rawMessage)
		if err != nil {
			return errors.Wrapf(err, "Can't save raw message marked as sent")
		}
	}

	return nil
}

func shouldResendEmojiReaction(message *common.RawMessage, t common.TimeSource) (bool, error) {
	if message.MessageType != protobuf.ApplicationMetadataMessage_EMOJI_REACTION {
		return false, errors.New("Should resend only emoji reactions")
	}

	if message.Sent {
		return false, errors.New("Should resend only non-sent messages")
	}

	if message.SendCount > emojiResendMaxCount {
		return false, nil
	}

	//exponential backoff depends on how many attempts to send message already made
	backoff := uint64(math.Pow(2, float64(message.SendCount-1))) * emojiResendMinDelay * uint64(time.Second)
	backoffElapsed := t.GetCurrentTime() > (message.LastSent + backoff)
	return backoffElapsed, nil
}

func (m *Messenger) resendExpiredEmojiReactions() error {
	ids, err := m.persistence.ExpiredEmojiReactionsIDs(emojiResendMaxCount)
	if err != nil {
		return errors.Wrapf(err, "Can't get expired reactions from db")
	}

	for _, id := range ids {
		rawMessage, err := m.persistence.RawMessageByID(id)
		if err != nil {
			return errors.Wrapf(err, "Can't get raw message with id %v", id)
		}

		if ok, err := shouldResendEmojiReaction(rawMessage, m.getTimesource()); ok {
			err = m.persistence.SaveRawMessage(rawMessage)
			if err != nil {
				return errors.Wrapf(err, "Can't save raw message marked as non-expired")
			}

			err = m.reSendRawMessage(context.Background(), rawMessage.ID)
			if err != nil {
				return errors.Wrapf(err, "Can't resend expired message with id %v", rawMessage.ID)
			}
		} else {
			return err
		}
	}
	return nil
}

func (m *Messenger) Start() (*MessengerResponse, error) {
	m.logger.Info("starting messenger", zap.String("identity", types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey))))
	// Start push notification server
	if m.pushNotificationServer != nil {
		if err := m.pushNotificationServer.Start(); err != nil {
			return nil, err
		}
	}

	// Start push notification client
	if m.pushNotificationClient != nil {
		m.handlePushNotificationClientRegistrations(m.pushNotificationClient.SubscribeToRegistrations())

		if err := m.pushNotificationClient.Start(); err != nil {
			return nil, err
		}
	}

	ensSubscription := m.ensVerifier.Subscribe()

	// Subscrbe
	if err := m.ensVerifier.Start(); err != nil {
		return nil, err
	}

	if err := m.communitiesManager.Start(); err != nil {
		return nil, err
	}

	// set shared secret handles
	m.processor.SetHandleSharedSecrets(m.handleSharedSecrets)

	subscriptions, err := m.encryptor.Start(m.identity)
	if err != nil {
		return nil, err
	}

	// handle stored shared secrets
	err = m.handleSharedSecrets(subscriptions.SharedSecrets)
	if err != nil {
		return nil, err
	}

	m.handleEncryptionLayerSubscriptions(subscriptions)
	m.handleCommunitiesSubscription(m.communitiesManager.Subscribe())
	m.handleConnectionChange(m.online())
	m.handleENSVerificationSubscription(ensSubscription)
	m.watchConnectionChange()
	m.watchExpiredEmojis()
	m.watchIdentityImageChanges()
	if err := m.cleanTopics(); err != nil {
		return nil, err
	}
	response := &MessengerResponse{Filters: m.transport.Filters()}

	if m.mailserversDatabase != nil {
		mailserverTopics, err := m.mailserversDatabase.Topics()
		if err != nil {
			return nil, err
		}
		response.MailserverTopics = mailserverTopics

		mailserverRanges, err := m.mailserversDatabase.ChatRequestRanges()
		if err != nil {
			return nil, err
		}
		response.MailserverRanges = mailserverRanges

		mailservers, err := m.mailserversDatabase.Mailservers()
		if err != nil {
			return nil, err
		}
		response.Mailservers = mailservers

	}
	return response, nil
}

// cleanTopics remove any topic that does not have a Listen flag set
func (m *Messenger) cleanTopics() error {
	if m.mailserversDatabase == nil {
		return nil
	}
	var filters []*transport.Filter
	for _, f := range m.transport.Filters() {
		if f.Listen && !f.Ephemeral {
			filters = append(filters, f)
		}
	}

	m.logger.Debug("keeping topics", zap.Any("filters", filters))

	return m.mailserversDatabase.SetTopics(filters)
}

// handle connection change is called each time we go from offline/online or viceversa
func (m *Messenger) handleConnectionChange(online bool) {
	if online {
		if m.pushNotificationClient != nil {
			m.pushNotificationClient.Online()
		}

		if m.shouldPublishContactCode {
			if err := m.publishContactCode(); err != nil {
				m.logger.Error("could not publish on contact code", zap.Error(err))
				return
			}
			m.shouldPublishContactCode = false
		}

	} else {
		if m.pushNotificationClient != nil {
			m.pushNotificationClient.Offline()
		}
	}
	m.ensVerifier.SetOnline(online)
}

func (m *Messenger) online() bool {
	return m.node.PeersCount() > 0
}

func (m *Messenger) buildContactCodeAdvertisement() (*protobuf.ContactCodeAdvertisement, error) {
	if m.pushNotificationClient == nil || !m.pushNotificationClient.Enabled() {
		return nil, nil
	}
	m.logger.Debug("adding push notification info to contact code bundle")
	info, err := m.pushNotificationClient.MyPushNotificationQueryInfo()
	if err != nil {
		return nil, err
	}
	if len(info) == 0 {
		return nil, nil
	}
	return &protobuf.ContactCodeAdvertisement{
		PushNotificationInfo: info,
	}, nil
}

// publishContactCode sends a public message wrapped in the encryption
// layer, which will propagate our bundle
func (m *Messenger) publishContactCode() error {
	var payload []byte
	m.logger.Debug("sending contact code")
	contactCodeAdvertisement, err := m.buildContactCodeAdvertisement()
	if err != nil {
		m.logger.Error("could not build contact code advertisement", zap.Error(err))
	}

	if contactCodeAdvertisement == nil {
		contactCodeAdvertisement = &protobuf.ContactCodeAdvertisement{}
	}
	err = m.attachChatIdentity(contactCodeAdvertisement)
	if err != nil {
		return err
	}

	payload, err = proto.Marshal(contactCodeAdvertisement)
	if err != nil {
		return err
	}

	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
	rawMessage := common.RawMessage{
		LocalChatID: contactCodeTopic,
		MessageType: protobuf.ApplicationMetadataMessage_CONTACT_CODE_ADVERTISEMENT,
		Payload:     payload,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = m.processor.SendPublic(ctx, contactCodeTopic, rawMessage)
	if err != nil {
		m.logger.Warn("failed to send a contact code", zap.Error(err))
	}
	return err
}

// contactCodeAdvertisement attaches a protobuf.ChatIdentity to the given protobuf.ContactCodeAdvertisement,
// if the `shouldPublish` conditions are met
func (m *Messenger) attachChatIdentity(cca *protobuf.ContactCodeAdvertisement) error {
	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
	shouldPublish, err := m.shouldPublishChatIdentity(contactCodeTopic)
	if err != nil {
		return err
	}

	if !shouldPublish {
		return nil
	}

	cca.ChatIdentity, err = m.createChatIdentity(privateChat)
	if err != nil {
		return err
	}

	img, err := m.multiAccounts.GetIdentityImage(m.account.KeyUID, userimage.SmallDimName)
	if err != nil {
		return err
	}
	if img == nil {
		return errors.New("could not find image")
	}

	err = m.persistence.SaveWhenChatIdentityLastPublished(contactCodeTopic, img.Hash())
	if err != nil {
		return err
	}

	return nil
}

// handleStandaloneChatIdentity sends a standalone ChatIdentity message to a public channel if the publish criteria is met
func (m *Messenger) handleStandaloneChatIdentity(chat *Chat) error {
	if chat.ChatType != ChatTypePublic {
		return nil
	}
	shouldPublishChatIdentity, err := m.shouldPublishChatIdentity(chat.ID)
	if err != nil {
		return err
	}
	if !shouldPublishChatIdentity {
		return nil
	}

	ci, err := m.createChatIdentity(publicChat)
	if err != nil {
		return err
	}

	payload, err := proto.Marshal(ci)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID: chat.ID,
		MessageType: protobuf.ApplicationMetadataMessage_CHAT_IDENTITY,
		Payload:     payload,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = m.processor.SendPublic(ctx, chat.ID, rawMessage)
	if err != nil {
		return err
	}

	img, err := m.multiAccounts.GetIdentityImage(m.account.KeyUID, userimage.SmallDimName)
	if err != nil {
		return err
	}
	if img == nil {
		return errors.New("could not find image")
	}

	err = m.persistence.SaveWhenChatIdentityLastPublished(chat.ID, img.Hash())
	if err != nil {
		return err
	}

	return nil
}

// shouldPublishChatIdentity returns true if the last time the ChatIdentity was attached was more than 24 hours ago
func (m *Messenger) shouldPublishChatIdentity(chatID string) (bool, error) {
	if m.account == nil {
		return false, nil
	}

	// Check we have at least one image
	img, err := m.multiAccounts.GetIdentityImage(m.account.KeyUID, userimage.SmallDimName)
	if err != nil {
		return false, err
	}

	if img == nil {
		return false, nil
	}

	lp, hash, err := m.persistence.GetWhenChatIdentityLastPublished(chatID)
	if err != nil {
		return false, err
	}

	if !bytes.Equal(hash, img.Hash()) {
		return true, nil
	}

	return lp == 0 || time.Now().Unix()-lp > 24*60*60, nil
}

// createChatIdentity creates a context based protobuf.ChatIdentity.
// context 'public-chat' will attach only the 'thumbnail' IdentityImage
// context 'private-chat' will attach all IdentityImage
func (m *Messenger) createChatIdentity(context chatContext) (*protobuf.ChatIdentity, error) {
	m.logger.Info(fmt.Sprintf("account keyUID '%s'", m.account.KeyUID))
	m.logger.Info(fmt.Sprintf("context '%s'", context))

	ci := &protobuf.ChatIdentity{
		Clock:   m.transport.GetCurrentTime(),
		EnsName: "", // TODO add ENS name handling to dedicate PR
	}

	ciis := make(map[string]*protobuf.IdentityImage)

	switch context {
	case publicChat:
		m.logger.Info(fmt.Sprintf("handling %s ChatIdentity", context))

		img, err := m.multiAccounts.GetIdentityImage(m.account.KeyUID, userimage.SmallDimName)
		if err != nil {
			return nil, err
		}

		m.logger.Debug(fmt.Sprintf("%s images.IdentityImage '%s'", context, spew.Sdump(img)))

		ciis[userimage.SmallDimName] = m.adaptIdentityImageToProtobuf(img)
		m.logger.Debug(fmt.Sprintf("%s protobuf.IdentityImage '%s'", context, spew.Sdump(ciis)))
		ci.Images = ciis

	case privateChat:
		m.logger.Info(fmt.Sprintf("handling %s ChatIdentity", context))

		imgs, err := m.multiAccounts.GetIdentityImages(m.account.KeyUID)
		if err != nil {
			return nil, err
		}

		m.logger.Debug(fmt.Sprintf("%s images.IdentityImage '%s'", context, spew.Sdump(imgs)))

		for _, img := range imgs {
			ciis[img.Name] = m.adaptIdentityImageToProtobuf(img)
		}
		m.logger.Debug(fmt.Sprintf("%s protobuf.IdentityImage '%s'", context, spew.Sdump(ciis)))
		ci.Images = ciis

	default:
		return ci, fmt.Errorf("unknown ChatIdentity context '%s'", context)
	}

	return ci, nil
}

// adaptIdentityImageToProtobuf Adapts a images.IdentityImage to protobuf.IdentityImage
func (m *Messenger) adaptIdentityImageToProtobuf(img *userimage.IdentityImage) *protobuf.IdentityImage {
	return &protobuf.IdentityImage{
		Payload:    img.Payload,
		SourceType: protobuf.IdentityImage_RAW_PAYLOAD, // TODO add ENS avatar handling to dedicated PR
		ImageType:  images.ImageType(img.Payload),
	}
}

// handleSharedSecrets process the negotiated secrets received from the encryption layer
func (m *Messenger) handleSharedSecrets(secrets []*sharedsecret.Secret) error {
	var result []*transport.Filter
	for _, secret := range secrets {
		fSecret := types.NegotiatedSecret{
			PublicKey: secret.Identity,
			Key:       secret.Key,
		}
		filter, err := m.transport.ProcessNegotiatedSecret(fSecret)
		if err != nil {
			return err
		}
		result = append(result, filter)
	}
	if m.config.onNegotiatedFilters != nil {
		m.config.onNegotiatedFilters(result)
	}

	return nil
}

// handleInstallations adds the installations in the installations map
func (m *Messenger) handleInstallations(installations []*multidevice.Installation) {
	for _, installation := range installations {
		if installation.Identity == contactIDFromPublicKey(&m.identity.PublicKey) {
			if _, ok := m.allInstallations[installation.ID]; !ok {
				m.allInstallations[installation.ID] = installation
				m.modifiedInstallations[installation.ID] = true
			}
		}
	}
}

// handleEncryptionLayerSubscriptions handles events from the encryption layer
func (m *Messenger) handleEncryptionLayerSubscriptions(subscriptions *encryption.Subscriptions) {
	go func() {
		for {
			select {
			case <-subscriptions.SendContactCode:
				if err := m.publishContactCode(); err != nil {
					m.logger.Error("failed to publish contact code", zap.Error(err))
				}
				// we also piggy-back to clean up cached messages
				if err := m.transport.CleanMessagesProcessed(m.getTimesource().GetCurrentTime() - messageCacheIntervalMs); err != nil {
					m.logger.Error("failed to clean processed messages", zap.Error(err))
				}

			case <-subscriptions.Quit:
				m.logger.Debug("quitting encryption subscription loop")
				return
			}
		}
	}()
}

func (m *Messenger) handleENSVerified(records []*ens.VerificationRecord) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var contacts []*Contact
	for _, record := range records {
		m.logger.Info("handling record", zap.Any("record", record))
		contact, ok := m.allContacts[record.PublicKey]
		if !ok {
			m.logger.Info("contact not found")
			continue
		}

		contact.ENSVerified = record.Verified
		contact.Name = record.Name
		contacts = append(contacts, contact)
	}

	m.logger.Info("handled records", zap.Any("contacts", contacts))
	if len(contacts) != 0 {
		if err := m.persistence.SaveContacts(contacts); err != nil {
			m.logger.Error("failed to save contacts", zap.Error(err))
			return
		}
	}

	m.logger.Info("calling on contacts")
	if m.config.onContactENSVerified != nil {
		m.logger.Info("called on contacts")
		response := &MessengerResponse{Contacts: contacts}
		m.config.onContactENSVerified(response)
	}

}

func (m *Messenger) handleENSVerificationSubscription(c chan []*ens.VerificationRecord) {
	go func() {
		for {
			select {
			case records, more := <-c:
				if !more {
					m.logger.Info("No more records, quitting")
					return
				}
				if len(records) != 0 {
					m.logger.Info("handling records", zap.Any("records", records))
					m.handleENSVerified(records)
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// watchConnectionChange checks the connection status and call handleConnectionChange when this changes
func (m *Messenger) watchConnectionChange() {
	m.logger.Debug("watching connection changes")
	state := m.online()
	go func() {
		for {
			select {
			case <-time.After(200 * time.Millisecond):
				newState := m.online()
				if state != newState {
					state = newState
					m.logger.Debug("connection changed", zap.Bool("online", state))
					m.handleConnectionChange(state)
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// watchExpiredEmojis regularly checks for expired emojis and invoke their resending
func (m *Messenger) watchExpiredEmojis() {
	m.logger.Debug("watching expired emojis")
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if m.online() {
					err := m.resendExpiredEmojiReactions()
					if err != nil {
						m.logger.Debug("Error when resending expired emoji reactions", zap.Error(err))
					}
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// watchIdentityImageChanges checks for identity images changes and publishes to the contact code when it happens
func (m *Messenger) watchIdentityImageChanges() {
	m.logger.Debug("watching identity image changes")
	if m.multiAccounts == nil {
		return
	}

	channel := m.multiAccounts.SubscribeToIdentityImageChanges()

	go func() {
		for {
			select {
			case <-channel:
				if m.online() {
					if err := m.publishContactCode(); err != nil {
						m.logger.Error("failed to publish contact code", zap.Error(err))
					}

				} else {
					m.shouldPublishContactCode = true
				}
			case <-m.quit:
				return
			}
		}
	}()
}

// handlePushNotificationClientRegistration handles registration events
func (m *Messenger) handlePushNotificationClientRegistrations(c chan struct{}) {
	go func() {
		for {
			_, more := <-c
			if !more {
				return
			}
			if err := m.publishContactCode(); err != nil {
				m.logger.Error("failed to publish contact code", zap.Error(err))
			}

		}
	}()
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

	joinedCommunities, err := m.communitiesManager.Joined()
	if err != nil {
		return err
	}
	for _, org := range joinedCommunities {
		// the org advertise on the public topic derived by the pk
		publicChatIDs = append(publicChatIDs, org.IDString())
	}

	// Init filters for the communities we are an admin of
	var adminCommunitiesPks []*ecdsa.PrivateKey
	adminCommunities, err := m.communitiesManager.Created()
	if err != nil {
		return err
	}

	for _, c := range adminCommunities {
		adminCommunitiesPks = append(adminCommunitiesPks, c.PrivateKey())
	}

	_, err = m.transport.InitCommunityFilters(adminCommunitiesPks)
	if err != nil {
		return err
	}

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
		if !chat.Active || chat.Timeline() {
			continue
		}

		switch chat.ChatType {
		case ChatTypePublic, ChatTypeProfile:
			publicChatIDs = append(publicChatIDs, chat.ID)
		case ChatTypeCommunityChat:
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
	close(m.quit)

	for i, task := range m.shutdownTasks {
		m.logger.Debug("running shutdown task", zap.Int("n", i))
		if tErr := task(); tErr != nil {
			m.logger.Info("shutdown task failed", zap.Error(tErr))
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
	case ChatTypePublic, ChatTypeProfile, ChatTypeTimeline:
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

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	group, err := v1protocol.NewGroupWithCreator(name, clock, m.identity)
	if err != nil {
		return nil, err
	}
	chat.LastClockValue = clock

	chat.updateChatFromGroupMembershipChanges(contactIDFromPublicKey(&m.identity.PublicKey), group)

	clock, _ = chat.NextClockAndTimestamp(m.getTimesource())

	// Add members
	if len(members) > 0 {
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
	}

	recipients, err := stringSliceToPublicKeys(group.Members())

	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	m.allChats[chat.ID] = &chat

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:  recipients,
	})

	if err != nil {
		return nil, err
	}

	chat.updateChatFromGroupMembershipChanges(contactIDFromPublicKey(&m.identity.PublicKey), group)

	response.AddChat(&chat)
	response.Messages = buildSystemMessages(chat.MembershipUpdates, m.systemMessagesTranslations)
	err = m.persistence.SaveMessages(response.Messages)
	if err != nil {
		return nil, err
	}

	return &response, m.saveChat(&chat)
}

func (m *Messenger) CreateGroupChatFromInvitation(name string, chatID string, adminPK string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse
	logger := m.logger.With(zap.String("site", "CreateGroupChatFromInvitation"))
	logger.Info("Creating group chat from invitation", zap.String("name", name))
	chat := CreateGroupChat(m.getTimesource())
	chat.ID = chatID
	chat.Name = name
	chat.InvitationAdmin = adminPK

	response.AddChat(&chat)

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
		return nil, ErrChatNotFound
	}

	group, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return nil, err
	}

	// We save the initial recipients as we want to send updates to also
	// the members kicked out
	oldRecipients, err := stringSliceToPublicKeys(group.Members())
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
	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:  oldRecipients,
	})
	if err != nil {
		return nil, err
	}

	chat.updateChatFromGroupMembershipChanges(contactIDFromPublicKey(&m.identity.PublicKey), group)

	response.AddChat(chat)
	response.Messages = buildSystemMessages(chat.MembershipUpdates, m.systemMessagesTranslations)
	err = m.persistence.SaveMessages(response.Messages)
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
		return nil, ErrChatNotFound
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

	//approve invitations
	for _, member := range members {
		logger.Info("ApproveInvitationByChatIdAndFrom", zap.String("chatID", chatID), zap.Any("member", member))

		groupChatInvitation := &GroupChatInvitation{
			GroupChatInvitation: protobuf.GroupChatInvitation{
				ChatId: chat.ID,
			},
			From: member,
		}

		groupChatInvitation, err = m.persistence.InvitationByID(groupChatInvitation.ID())
		if err != nil && err != common.ErrRecordNotFound {
			return nil, err
		}
		if groupChatInvitation != nil {
			groupChatInvitation.State = protobuf.GroupChatInvitation_APPROVED

			err := m.persistence.SaveInvitation(groupChatInvitation)
			if err != nil {
				return nil, err
			}
			response.Invitations = append(response.Invitations, groupChatInvitation)
		}
	}

	err = group.ProcessEvent(event)
	if err != nil {
		return nil, err
	}

	recipients, err := stringSliceToPublicKeys(group.Members())
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:  recipients,
	})

	if err != nil {
		return nil, err
	}

	chat.updateChatFromGroupMembershipChanges(contactIDFromPublicKey(&m.identity.PublicKey), group)

	response.AddChat(chat)
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	err = m.persistence.SaveMessages(response.Messages)
	if err != nil {
		return nil, err
	}

	return &response, m.saveChat(chat)
}

func (m *Messenger) ChangeGroupChatName(ctx context.Context, chatID string, name string) (*MessengerResponse, error) {
	logger := m.logger.With(zap.String("site", "ChangeGroupChatName"))
	logger.Info("Changing group chat name", zap.String("chatID", chatID), zap.String("name", name))

	m.mutex.Lock()
	defer m.mutex.Unlock()

	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, ErrChatNotFound
	}

	group, err := newProtocolGroupFromChat(chat)
	if err != nil {
		return nil, err
	}

	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	// Add members
	event := v1protocol.NewNameChangedEvent(name, clock)
	event.ChatID = chat.ID
	err = event.Sign(m.identity)
	if err != nil {
		return nil, err
	}

	// Update in-memory group
	err = group.ProcessEvent(event)
	if err != nil {
		return nil, err
	}

	recipients, err := stringSliceToPublicKeys(group.Members())
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:  recipients,
	})

	if err != nil {
		return nil, err
	}

	chat.updateChatFromGroupMembershipChanges(contactIDFromPublicKey(&m.identity.PublicKey), group)

	var response MessengerResponse
	response.AddChat(chat)
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	err = m.persistence.SaveMessages(response.Messages)
	if err != nil {
		return nil, err
	}

	return &response, m.saveChat(chat)
}

func (m *Messenger) SendGroupChatInvitationRequest(ctx context.Context, chatID string, adminPK string,
	message string) (*MessengerResponse, error) {
	logger := m.logger.With(zap.String("site", "SendGroupChatInvitationRequest"))
	logger.Info("Sending group chat invitation request", zap.String("chatID", chatID),
		zap.String("adminPK", adminPK), zap.String("message", message))

	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	// Get chat and clock
	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, ErrChatNotFound
	}
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	invitationR := &GroupChatInvitation{
		GroupChatInvitation: protobuf.GroupChatInvitation{
			Clock:               clock,
			ChatId:              chatID,
			IntroductionMessage: message,
			State:               protobuf.GroupChatInvitation_REQUEST,
		},
		From: types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
	}

	encodedMessage, err := proto.Marshal(invitationR.GetProtobuf())
	if err != nil {
		return nil, err
	}

	spec := common.RawMessage{
		LocalChatID:         adminPK,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_GROUP_CHAT_INVITATION,
		ResendAutomatically: true,
	}

	pkey, err := hex.DecodeString(adminPK[2:])
	if err != nil {
		return nil, err
	}
	// Safety check, make sure is well formed
	adminpk, err := crypto.UnmarshalPubkey(pkey)
	if err != nil {
		return nil, err
	}

	id, err := m.processor.SendPrivate(ctx, adminpk, &spec)
	if err != nil {
		return nil, err
	}

	spec.ID = types.EncodeHex(id)
	spec.SendCount++
	err = m.persistence.SaveRawMessage(&spec)
	if err != nil {
		return nil, err
	}

	response.Invitations = []*GroupChatInvitation{invitationR}

	err = m.persistence.SaveInvitation(invitationR)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (m *Messenger) GetGroupChatInvitations() ([]*GroupChatInvitation, error) {
	return m.persistence.GetGroupChatInvitations()
}

func (m *Messenger) SendGroupChatInvitationRejection(ctx context.Context, invitationRequestID string) (*MessengerResponse, error) {
	logger := m.logger.With(zap.String("site", "SendGroupChatInvitationRejection"))
	logger.Info("Sending group chat invitation reject", zap.String("invitationRequestID", invitationRequestID))

	m.mutex.Lock()
	defer m.mutex.Unlock()

	invitationR, err := m.persistence.InvitationByID(invitationRequestID)
	if err != nil {
		return nil, err
	}

	invitationR.State = protobuf.GroupChatInvitation_REJECTED

	// Get chat and clock
	chat, ok := m.allChats[invitationR.ChatId]
	if !ok {
		return nil, ErrChatNotFound
	}
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	invitationR.Clock = clock

	encodedMessage, err := proto.Marshal(invitationR.GetProtobuf())
	if err != nil {
		return nil, err
	}

	spec := common.RawMessage{
		LocalChatID:         invitationR.From,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_GROUP_CHAT_INVITATION,
		ResendAutomatically: true,
	}

	pkey, err := hex.DecodeString(invitationR.From[2:])
	if err != nil {
		return nil, err
	}
	// Safety check, make sure is well formed
	userpk, err := crypto.UnmarshalPubkey(pkey)
	if err != nil {
		return nil, err
	}

	id, err := m.processor.SendPrivate(ctx, userpk, &spec)
	if err != nil {
		return nil, err
	}

	spec.ID = types.EncodeHex(id)
	spec.SendCount++
	err = m.persistence.SaveRawMessage(&spec)
	if err != nil {
		return nil, err
	}

	var response MessengerResponse

	response.Invitations = []*GroupChatInvitation{invitationR}

	err = m.persistence.SaveInvitation(invitationR)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (m *Messenger) AddAdminsToGroupChat(ctx context.Context, chatID string, members []string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse
	logger := m.logger.With(zap.String("site", "AddAdminsToGroupChat"))
	logger.Info("Add admins to group chat", zap.String("chatID", chatID), zap.Any("members", members))

	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, ErrChatNotFound
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

	recipients, err := stringSliceToPublicKeys(group.Members())
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:  recipients,
	})

	if err != nil {
		return nil, err
	}

	chat.updateChatFromGroupMembershipChanges(contactIDFromPublicKey(&m.identity.PublicKey), group)

	response.AddChat(chat)
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	err = m.persistence.SaveMessages(response.Messages)
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
		return nil, ErrChatNotFound
	}

	_, err := m.Join(chat)
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

	recipients, err := stringSliceToPublicKeys(group.Members())
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:  recipients,
	})
	if err != nil {
		return nil, err
	}

	chat.updateChatFromGroupMembershipChanges(contactIDFromPublicKey(&m.identity.PublicKey), group)

	response.AddChat(chat)
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	err = m.persistence.SaveMessages(response.Messages)
	if err != nil {
		return nil, err
	}

	return &response, m.saveChat(chat)
}

func (m *Messenger) LeaveGroupChat(ctx context.Context, chatID string, remove bool) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, ErrChatNotFound
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

	recipients, err := stringSliceToPublicKeys(group.Members())
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.processor.EncodeMembershipUpdate(group, nil)
	if err != nil {
		return nil, err
	}
	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE,
		Recipients:  recipients,
	})
	if err != nil {
		return nil, err
	}

	chat.updateChatFromGroupMembershipChanges(contactIDFromPublicKey(&m.identity.PublicKey), group)

	if remove {
		chat.Active = false
	}

	response.AddChat(chat)
	response.Messages = buildSystemMessages([]v1protocol.MembershipUpdateEvent{event}, m.systemMessagesTranslations)
	err = m.persistence.SaveMessages(response.Messages)
	if err != nil {
		return nil, err
	}

	return &response, m.saveChat(chat)
}

func (m *Messenger) reregisterForPushNotifications() error {
	m.logger.Info("contact state changed, re-registering for push notification")
	if m.pushNotificationClient == nil {
		return nil
	}

	return m.pushNotificationClient.Reregister(m.pushNotificationOptions())
}

// pull a message from the database and send it again
func (m *Messenger) reSendRawMessage(ctx context.Context, messageID string) error {
	message, err := m.persistence.RawMessageByID(messageID)
	if err != nil {
		return err
	}

	chat, ok := m.allChats[message.LocalChatID]
	if !ok {
		return errors.New("chat not found")
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             message.Payload,
		MessageType:         message.MessageType,
		Recipients:          message.Recipients,
		ResendAutomatically: message.ResendAutomatically,
		SendCount:           message.SendCount,
	})
	return err
}

// ReSendChatMessage pulls a message from the database and sends it again
func (m *Messenger) ReSendChatMessage(ctx context.Context, messageID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.reSendRawMessage(ctx, messageID)
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
func (m *Messenger) sendToPairedDevices(ctx context.Context, spec common.RawMessage) error {
	hasPairedDevices := m.hasPairedDevices()
	// We send a message to any paired device
	if hasPairedDevices {
		_, err := m.processor.SendPrivate(ctx, &m.identity.PublicKey, &spec)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Messenger) dispatchPairInstallationMessage(ctx context.Context, spec common.RawMessage) ([]byte, error) {
	var err error
	var id []byte

	id, err = m.processor.SendPairInstallation(ctx, &m.identity.PublicKey, spec)

	if err != nil {
		return nil, err
	}
	spec.ID = types.EncodeHex(id)
	spec.SendCount++
	err = m.persistence.SaveRawMessage(&spec)
	if err != nil {
		return nil, err
	}

	return id, nil
}

func (m *Messenger) dispatchMessage(ctx context.Context, spec common.RawMessage) (common.RawMessage, error) {
	var err error
	var id []byte
	logger := m.logger.With(zap.String("site", "dispatchMessage"), zap.String("chatID", spec.LocalChatID))
	chat, ok := m.allChats[spec.LocalChatID]
	if !ok {
		return spec, errors.New("no chat found")
	}

	switch chat.ChatType {
	case ChatTypeOneToOne:
		publicKey, err := chat.PublicKey()
		if err != nil {
			return spec, err
		}

		//SendPrivate will alter message identity and possibly datasyncid, so we save an unchanged
		//message for sending to paired devices later
		specCopyForPairedDevices := spec
		if !common.IsPubKeyEqual(publicKey, &m.identity.PublicKey) {
			id, err = m.processor.SendPrivate(ctx, publicKey, &spec)

			if err != nil {
				return spec, err
			}
		}

		err = m.sendToPairedDevices(ctx, specCopyForPairedDevices)

		if err != nil {
			return spec, err
		}

	case ChatTypePublic, ChatTypeProfile:
		logger.Debug("sending public message", zap.String("chatName", chat.Name))
		id, err = m.processor.SendPublic(ctx, chat.ID, spec)
		if err != nil {
			return spec, err
		}
	case ChatTypeCommunityChat:
		// TODO: add grant
		canPost, err := m.communitiesManager.CanPost(&m.identity.PublicKey, chat.CommunityID, chat.CommunityChatID(), nil)
		if err != nil {
			return spec, err
		}

		// We allow emoji reactions by anyone
		if spec.MessageType != protobuf.ApplicationMetadataMessage_EMOJI_REACTION && !canPost {
			m.logger.Error("can't post on chat", zap.String("chat-id", chat.ID), zap.String("chat-name", chat.Name))

			return spec, errors.New("can't post on chat")
		}

		logger.Debug("sending community chat message", zap.String("chatName", chat.Name))
		id, err = m.processor.SendPublic(ctx, chat.ID, spec)
		if err != nil {
			return spec, err
		}
	case ChatTypePrivateGroupChat:
		logger.Debug("sending group message", zap.String("chatName", chat.Name))
		if spec.Recipients == nil {
			// Anything that is not a membership update message is only dispatched to joined users
			// NOTE: I think here it might make sense to always invite to joined users apart from the
			// initial message
			if spec.MessageType != protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE {
				spec.Recipients, err = chat.JoinedMembersAsPublicKeys()
				if err != nil {
					return spec, err
				}

			} else {
				spec.Recipients, err = chat.MembersAsPublicKeys()
				if err != nil {
					return spec, err
				}
			}
		}

		hasPairedDevices := m.hasPairedDevices()

		if !hasPairedDevices {

			// Filter out my key from the recipients
			n := 0
			for _, recipient := range spec.Recipients {
				if !common.IsPubKeyEqual(recipient, &m.identity.PublicKey) {
					spec.Recipients[n] = recipient
					n++
				}
			}
			spec.Recipients = spec.Recipients[:n]
		}

		// We won't really send the message out if there's no recipients
		if len(spec.Recipients) == 0 {
			spec.Sent = true
		}

		// We skip wrapping in some cases (emoji reactions for example)
		if !spec.SkipGroupMessageWrap {
			spec.MessageType = protobuf.ApplicationMetadataMessage_MEMBERSHIP_UPDATE_MESSAGE
		}

		id, err = m.processor.SendGroup(ctx, spec.Recipients, spec)
		if err != nil {
			return spec, err
		}

	default:
		return spec, errors.New("chat type not supported")
	}
	spec.ID = types.EncodeHex(id)
	spec.SendCount++
	spec.LastSent = m.getTimesource().GetCurrentTime()
	err = m.persistence.SaveRawMessage(&spec)
	if err != nil {
		return spec, err
	}

	return spec, nil
}

// SendChatMessage takes a minimal message and sends it based on the corresponding chat
func (m *Messenger) SendChatMessage(ctx context.Context, message *common.Message) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.sendChatMessage(ctx, message)
}

// SendChatMessages takes a array of messages and sends it based on the corresponding chats
func (m *Messenger) SendChatMessages(ctx context.Context, messages []*common.Message) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	for _, message := range messages {
		messageResponse, err := m.sendChatMessage(ctx, message)
		if err != nil {
			return nil, err
		}
		err = response.Merge(messageResponse)
		if err != nil {
			return nil, err
		}
	}

	return &response, nil
}

// SendChatMessage takes a minimal message and sends it based on the corresponding chat
func (m *Messenger) sendChatMessage(ctx context.Context, message *common.Message) (*MessengerResponse, error) {
	if message.ImagePath != "" {
		file, err := os.Open(message.ImagePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		payload, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err

		}
		image := protobuf.ImageMessage{
			Payload: payload,
			Type:    images.ImageType(payload),
		}
		message.Payload = &protobuf.ChatMessage_Image{Image: &image}

	} else if message.CommunityID != "" {
		community, err := m.communitiesManager.GetByIDString(message.CommunityID)
		if err != nil {
			return nil, err
		}

		if community == nil {
			return nil, errors.New("community not found")
		}

		wrappedCommunity, err := community.ToBytes()
		if err != nil {
			return nil, err
		}
		message.Payload = &protobuf.ChatMessage_Community{Community: wrappedCommunity}

		message.ContentType = protobuf.ChatMessage_COMMUNITY

	} else if message.AudioPath != "" {
		file, err := os.Open(message.AudioPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		payload, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err

		}
		audioMessage := message.GetAudio()
		if audioMessage == nil {
			return nil, errors.New("no audio has been passed")
		}
		audioMessage.Payload = payload
		audioMessage.Type = audio.Type(payload)
		message.Payload = &protobuf.ChatMessage_Audio{Audio: audioMessage}
		err = os.Remove(message.AudioPath)
		if err != nil {
			return nil, err
		}
	}

	var response MessengerResponse

	// A valid added chat is required.
	chat, ok := m.allChats[message.ChatId]
	if !ok {
		return nil, errors.New("Chat not found")
	}

	err := m.handleStandaloneChatIdentity(chat)
	if err != nil {
		return nil, err
	}

	err = extendMessageFromChat(message, chat, &m.identity.PublicKey, m.getTimesource())
	if err != nil {
		return nil, err
	}

	encodedMessage, err := m.encodeChatEntity(chat, message)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		LocalChatID:          chat.ID,
		SendPushNotification: m.featureFlags.PushNotifications,
		Payload:              encodedMessage,
		MessageType:          protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
		ResendAutomatically:  true,
	}
	rawMessage, err = m.dispatchMessage(ctx, rawMessage)
	if err != nil {
		return nil, err
	}

	if rawMessage.Sent {
		message.OutgoingStatus = common.OutgoingStatusSent
	}
	message.ID = rawMessage.ID
	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.getTimesource())
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	response.Messages, err = m.pullMessagesAndResponsesFromDB([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)
	return &response, m.saveChat(chat)
}

// SyncDevices sends all public chats and contacts to paired devices
// TODO remove use of photoPath in contacts
func (m *Messenger) SyncDevices(ctx context.Context, ensName, photoPath string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	myID := contactIDFromPublicKey(&m.identity.PublicKey)

	if _, err := m.sendContactUpdate(ctx, myID, ensName, photoPath); err != nil {
		return err
	}

	for _, chat := range m.allChats {
		if !chat.Timeline() && !chat.ProfileUpdates() && chat.Public() && chat.Active {
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

	_, err = m.dispatchPairInstallationMessage(ctx, common.RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_PAIR_INSTALLATION,
		ResendAutomatically: true,
	})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)

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

	_, err = m.dispatchMessage(ctx, common.RawMessage{
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
		Clock:         clock,
		Id:            contact.ID,
		EnsName:       contact.Name,
		LocalNickname: contact.LocalNickname,
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
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

	// All contacts in memory
	AllContacts map[string]*Contact
	// List of contacts modified
	ModifiedContacts map[string]bool
	// All installations in memory
	AllInstallations map[string]*multidevice.Installation
	// List of communities modified
	ModifiedInstallations map[string]bool
	// List of filters
	AllFilters map[string]*transport.Filter
	// Map of existing messages
	ExistingMessagesMap map[string]bool
	// EmojiReactions is a list of emoji reactions for the current batch
	// indexed by from-message-id-emoji-type
	EmojiReactions map[string]*EmojiReaction
	// GroupChatInvitations is a list of invitation requests or rejections
	GroupChatInvitations map[string]*GroupChatInvitation
	// Response to the client
	Response *MessengerResponse
	// Timesource is a time source for clock values/timestamps.
	Timesource common.TimeSource
}

func (m *Messenger) markDeliveredMessages(acks [][]byte) {
	for _, ack := range acks {
		//get message ID from database by datasync ID, with at-least-one
		// semantic
		messageIDBytes, err := m.persistence.MarkAsConfirmed(ack, true)
		if err != nil {
			m.logger.Info("got datasync acknowledge for message we don't have in db", zap.String("ack", hex.EncodeToString(ack)))
			continue
		}

		messageID := messageIDBytes.String()
		//mark messages as delivered
		err = m.UpdateMessageOutgoingStatus(messageID, common.OutgoingStatusDelivered)
		if err != nil {
			m.logger.Debug("Can't set message status as delivered", zap.Error(err))
		}

		//send signal to client that message status updated
		if m.config.messageDeliveredHandler != nil {
			message, err := m.persistence.MessageByID(messageID)
			if err != nil {
				m.logger.Debug("Can't get message from database", zap.Error(err))
				continue
			}
			m.config.messageDeliveredHandler(message.LocalChatID, messageID)
		}
	}
}

// addNewMessageNotification takes a common.Message and generates a new MessageNotificationBody and appends it to the
// []Response.Notifications if the message is m.New
func (r *ReceivedMessageState) addNewMessageNotification(publicKey ecdsa.PublicKey, m *common.Message, responseTo *common.Message) error {
	if !m.New {
		return nil
	}

	pubKey, err := m.GetSenderPubKey()
	if err != nil {
		return err
	}
	contactID := contactIDFromPublicKey(pubKey)

	chat := r.AllChats[m.LocalChatID]
	notification := MessageNotificationBody{
		Message: m,
		Contact: r.AllContacts[contactID],
		Chat:    chat,
	}

	if showNotification(publicKey, notification, responseTo) {
		r.Response.Notifications = append(
			r.Response.Notifications,
			notification,
		)
	}

	return nil
}

func (m *Messenger) handleRetrievedMessages(chatWithMessages map[transport.Filter][]*types.Message) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	messageState := &ReceivedMessageState{
		AllChats:              m.allChats,
		AllContacts:           m.allContacts,
		ModifiedContacts:      make(map[string]bool),
		AllInstallations:      m.allInstallations,
		ModifiedInstallations: m.modifiedInstallations,
		ExistingMessagesMap:   make(map[string]bool),
		EmojiReactions:        make(map[string]*EmojiReaction),
		AllFilters:            make(map[string]*transport.Filter),
		GroupChatInvitations:  make(map[string]*GroupChatInvitation),
		Response:              &MessengerResponse{},
		Timesource:            m.getTimesource(),
	}

	logger := m.logger.With(zap.String("site", "RetrieveAll"))

	for _, messages := range chatWithMessages {
		var processedMessages []string
		for _, shhMessage := range messages {
			// Indicates tha all messages in the batch have been processed correctly
			allMessagesProcessed := true
			statusMessages, acks, err := m.processor.HandleMessages(shhMessage, true)
			if err != nil {
				logger.Info("failed to decode messages", zap.Error(err))
				continue
			}
			m.markDeliveredMessages(acks)

			logger.Debug("processing messages further", zap.Int("count", len(statusMessages)))

			for _, msg := range statusMessages {
				publicKey := msg.SigPubKey()

				m.handleInstallations(msg.Installations)
				err := m.handleSharedSecrets(msg.SharedSecrets)
				if err != nil {
					// log and continue, non-critical error
					logger.Warn("failed to handle shared secrets")
				}

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
					logger.Debug("messageExists", zap.String("messageID", messageID))
					continue
				}

				var contact *Contact
				if c, ok := messageState.AllContacts[senderID]; ok {
					contact = c
				} else {
					c, err := buildContact(senderID, publicKey)
					if err != nil {
						logger.Info("failed to build contact", zap.Error(err))
						allMessagesProcessed = false
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
					switch msg.ParsedMessage.Interface().(type) {
					case protobuf.MembershipUpdateMessage:
						logger.Debug("Handling MembershipUpdateMessage")

						rawMembershipUpdate := msg.ParsedMessage.Interface().(protobuf.MembershipUpdateMessage)

						err = m.handler.HandleMembershipUpdate(messageState, messageState.AllChats[rawMembershipUpdate.ChatId], rawMembershipUpdate, m.systemMessagesTranslations)
						if err != nil {
							logger.Warn("failed to handle MembershipUpdate", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.ChatMessage:
						logger.Debug("Handling ChatMessage")
						messageState.CurrentMessageState.Message = msg.ParsedMessage.Interface().(protobuf.ChatMessage)
						err = m.handler.HandleChatMessage(messageState)
						if err != nil {
							logger.Warn("failed to handle ChatMessage", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.PairInstallation:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}
						p := msg.ParsedMessage.Interface().(protobuf.PairInstallation)
						logger.Debug("Handling PairInstallation", zap.Any("message", p))
						err = m.handler.HandlePairInstallation(messageState, p)
						if err != nil {
							logger.Warn("failed to handle PairInstallation", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncInstallationContact:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncInstallationContact)
						logger.Debug("Handling SyncInstallationContact", zap.Any("message", p))
						err = m.handler.HandleSyncInstallationContact(messageState, p)
						if err != nil {
							logger.Warn("failed to handle SyncInstallationContact", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SyncInstallationPublicChat:
						if !common.IsPubKeyEqual(messageState.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
							logger.Warn("not coming from us, ignoring")
							continue
						}

						p := msg.ParsedMessage.Interface().(protobuf.SyncInstallationPublicChat)
						logger.Debug("Handling SyncInstallationPublicChat", zap.Any("message", p))
						added := m.handler.HandleSyncInstallationPublicChat(messageState, p)

						// We re-register as we want to receive mentions from the newly joined public chat
						if added {
							logger.Debug("newly synced public chat, re-registering for push notifications")
							err := m.reregisterForPushNotifications()
							if err != nil {

								allMessagesProcessed = false
								logger.Warn("could not re-register for push notifications", zap.Error(err))
								continue
							}
						}

					case protobuf.RequestAddressForTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.RequestAddressForTransaction)
						logger.Debug("Handling RequestAddressForTransaction", zap.Any("message", command))
						err = m.handler.HandleRequestAddressForTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle RequestAddressForTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.SendTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.SendTransaction)
						logger.Debug("Handling SendTransaction", zap.Any("message", command))
						err = m.handler.HandleSendTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle SendTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.AcceptRequestAddressForTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.AcceptRequestAddressForTransaction)
						logger.Debug("Handling AcceptRequestAddressForTransaction")
						err = m.handler.HandleAcceptRequestAddressForTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle AcceptRequestAddressForTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.DeclineRequestAddressForTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.DeclineRequestAddressForTransaction)
						logger.Debug("Handling DeclineRequestAddressForTransaction")
						err = m.handler.HandleDeclineRequestAddressForTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle DeclineRequestAddressForTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.DeclineRequestTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.DeclineRequestTransaction)
						logger.Debug("Handling DeclineRequestTransaction")
						err = m.handler.HandleDeclineRequestTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle DeclineRequestTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.RequestTransaction:
						command := msg.ParsedMessage.Interface().(protobuf.RequestTransaction)
						logger.Debug("Handling RequestTransaction")
						err = m.handler.HandleRequestTransaction(messageState, command)
						if err != nil {
							logger.Warn("failed to handle RequestTransaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.ContactUpdate:
						logger.Debug("Handling ContactUpdate")
						contactUpdate := msg.ParsedMessage.Interface().(protobuf.ContactUpdate)
						err = m.handler.HandleContactUpdate(messageState, contactUpdate)
						if err != nil {
							logger.Warn("failed to handle ContactUpdate", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.PushNotificationQuery:
						logger.Debug("Received PushNotificationQuery")
						if m.pushNotificationServer == nil {
							continue
						}
						logger.Debug("Handling PushNotificationQuery")
						if err := m.pushNotificationServer.HandlePushNotificationQuery(publicKey, msg.ID, msg.ParsedMessage.Interface().(protobuf.PushNotificationQuery)); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationQuery", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue
					case protobuf.PushNotificationRegistrationResponse:
						logger.Debug("Received PushNotificationRegistrationResponse")
						if m.pushNotificationClient == nil {
							continue
						}
						logger.Debug("Handling PushNotificationRegistrationResponse")
						if err := m.pushNotificationClient.HandlePushNotificationRegistrationResponse(publicKey, msg.ParsedMessage.Interface().(protobuf.PushNotificationRegistrationResponse)); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationRegistrationResponse", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue
					case protobuf.ContactCodeAdvertisement:
						logger.Debug("Received ContactCodeAdvertisement")

						cca := msg.ParsedMessage.Interface().(protobuf.ContactCodeAdvertisement)
						if cca.ChatIdentity != nil {

							logger.Debug("Received ContactCodeAdvertisement ChatIdentity")
							err = m.handler.HandleChatIdentity(messageState, *cca.ChatIdentity)
							if err != nil {
								allMessagesProcessed = false
								logger.Warn("failed to handle ContactCodeAdvertisement ChatIdentity", zap.Error(err))
								// No continue as Chat Identity may fail but the rest of the cca may process fine.
							}
						}

						if m.pushNotificationClient == nil {
							continue
						}
						logger.Debug("Handling ContactCodeAdvertisement")
						if err := m.pushNotificationClient.HandleContactCodeAdvertisement(publicKey, cca); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle ContactCodeAdvertisement", zap.Error(err))
						}

						// We continue in any case, no changes to messenger
						continue

					case protobuf.PushNotificationResponse:
						logger.Debug("Received PushNotificationResponse")
						if m.pushNotificationClient == nil {
							continue
						}
						logger.Debug("Handling PushNotificationResponse")
						if err := m.pushNotificationClient.HandlePushNotificationResponse(publicKey, msg.ParsedMessage.Interface().(protobuf.PushNotificationResponse)); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationResponse", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue

					case protobuf.PushNotificationQueryResponse:
						logger.Debug("Received PushNotificationQueryResponse")
						if m.pushNotificationClient == nil {
							continue
						}
						logger.Debug("Handling PushNotificationQueryResponse")
						if err := m.pushNotificationClient.HandlePushNotificationQueryResponse(publicKey, msg.ParsedMessage.Interface().(protobuf.PushNotificationQueryResponse)); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationQueryResponse", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue

					case protobuf.PushNotificationRequest:
						logger.Debug("Received PushNotificationRequest")
						if m.pushNotificationServer == nil {
							continue
						}
						logger.Debug("Handling PushNotificationRequest")
						if err := m.pushNotificationServer.HandlePushNotificationRequest(publicKey, msg.ID, msg.ParsedMessage.Interface().(protobuf.PushNotificationRequest)); err != nil {
							allMessagesProcessed = false
							logger.Warn("failed to handle PushNotificationRequest", zap.Error(err))
						}
						// We continue in any case, no changes to messenger
						continue
					case protobuf.EmojiReaction:
						logger.Debug("Handling EmojiReaction")
						err = m.handler.HandleEmojiReaction(messageState, msg.ParsedMessage.Interface().(protobuf.EmojiReaction))
						if err != nil {
							logger.Warn("failed to handle EmojiReaction", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.GroupChatInvitation:
						logger.Debug("Handling GroupChatInvitation")
						err = m.handler.HandleGroupChatInvitation(messageState, msg.ParsedMessage.Interface().(protobuf.GroupChatInvitation))
						if err != nil {
							logger.Warn("failed to handle GroupChatInvitation", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.ChatIdentity:
						logger.Debug("Received ChatIdentity")
						err = m.handler.HandleChatIdentity(messageState, msg.ParsedMessage.Interface().(protobuf.ChatIdentity))
						if err != nil {
							logger.Warn("failed to handle ChatIdentity", zap.Error(err))
							allMessagesProcessed = false
							continue
						}

					case protobuf.CommunityDescription:
						logger.Debug("Handling CommunityDescription")
						err = m.handler.HandleCommunityDescription(messageState, publicKey, msg.ParsedMessage.Interface().(protobuf.CommunityDescription), msg.DecryptedPayload)
						if err != nil {
							logger.Warn("failed to handle CommunityDescription", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.CommunityInvitation:
						logger.Debug("Handling CommunityInvitation")
						invitation := msg.ParsedMessage.Interface().(protobuf.CommunityInvitation)
						err = m.handler.HandleCommunityInvitation(messageState, publicKey, invitation, invitation.CommunityDescription)
						if err != nil {
							logger.Warn("failed to handle CommunityInvitation", zap.Error(err))
							allMessagesProcessed = false
							continue
						}
					case protobuf.CommunityRequestToJoin:
						logger.Debug("Handling CommunityRequestToJoin")
						request := msg.ParsedMessage.Interface().(protobuf.CommunityRequestToJoin)
						err = m.handler.HandleCommunityRequestToJoin(messageState, publicKey, request)
						if err != nil {
							logger.Warn("failed to handle CommunityRequestToJoin", zap.Error(err))
							continue
						}

					default:
						// Check if is an encrypted PushNotificationRegistration
						if msg.Type == protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REGISTRATION {
							logger.Debug("Received PushNotificationRegistration")
							if m.pushNotificationServer == nil {
								continue
							}
							logger.Debug("Handling PushNotificationRegistration")
							if err := m.pushNotificationServer.HandlePushNotificationRegistration(publicKey, msg.ParsedMessage.Interface().([]byte)); err != nil {
								allMessagesProcessed = false
								logger.Warn("failed to handle PushNotificationRegistration", zap.Error(err))
							}
							// We continue in any case, no changes to messenger
							continue
						}

						logger.Debug("message not handled", zap.Any("messageType", reflect.TypeOf(msg.ParsedMessage.Interface())))

					}
				} else {
					logger.Debug("parsed message is nil")
				}
			}

			// Process any community changes
			for _, changes := range messageState.Response.CommunityChanges {
				if changes.ShouldMemberJoin {
					response, err := m.joinCommunity(changes.Community.ID())
					if err != nil {
						logger.Error("cannot join community", zap.Error(err))
						continue
					}

					if err := messageState.Response.Merge(response); err != nil {
						logger.Error("cannot merge join community response", zap.Error(err))
						continue
					}

				} else if changes.ShouldMemberLeave {
					response, err := m.leaveCommunity(changes.Community.ID())
					if err != nil {
						logger.Error("cannot join community", zap.Error(err))
						continue
					}

					if err := messageState.Response.Merge(response); err != nil {
						logger.Error("cannot merge join community response", zap.Error(err))
						continue

					}

				}
			}

			// Clean up as not used by clients currently
			messageState.Response.CommunityChanges = nil

			if allMessagesProcessed {
				processedMessages = append(processedMessages, types.EncodeHex(shhMessage.Hash))
			}
		}

		if len(processedMessages) != 0 {
			if err := m.transport.ConfirmMessagesProcessed(processedMessages, m.getTimesource().GetCurrentTime()); err != nil {
				logger.Warn("failed to confirm processed messages", zap.Error(err))
			}
		}
	}

	var contactsToSave []*Contact
	for id := range messageState.ModifiedContacts {
		contact := messageState.AllContacts[id]
		if contact != nil {
			// We save all contacts so we can pull back name/image,
			// but we only send to client those
			// that have some custom fields
			contactsToSave = append(contactsToSave, contact)
			if contact.HasCustomFields() {
				messageState.Response.Contacts = append(messageState.Response.Contacts, contact)
			}
		}
	}

	for _, filter := range messageState.AllFilters {
		messageState.Response.Filters = append(messageState.Response.Filters, filter)
	}

	// Hydrate chat alias and identicon
	for id := range messageState.Response.chats {
		chat := messageState.AllChats[id]
		if chat.OneToOne() {
			contact, ok := m.allContacts[chat.ID]
			if ok {
				chat.Alias = contact.Alias
				chat.Identicon = contact.Identicon
			}
		}

		messageState.Response.AddChat(chat)
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
	if len(messageState.Response.chats) > 0 {
		err = m.saveChats(messageState.Response.Chats())
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

	for _, emojiReaction := range messageState.EmojiReactions {
		messageState.Response.EmojiReactions = append(messageState.Response.EmojiReactions, emojiReaction)
	}

	for _, groupChatInvitation := range messageState.GroupChatInvitations {
		messageState.Response.Invitations = append(messageState.Response.Invitations, groupChatInvitation)
	}

	if len(contactsToSave) > 0 {
		err = m.persistence.SaveContacts(contactsToSave)
		if err != nil {
			return nil, err
		}
	}

	newMessagesIds := map[string]struct{}{}
	for _, message := range messageState.Response.Messages {
		newMessagesIds[message.ID] = struct{}{}
	}

	messagesWithResponses, err := m.pullMessagesAndResponsesFromDB(messageState.Response.Messages)
	if err != nil {
		return nil, err
	}
	messagesByID := map[string]*common.Message{}
	for _, message := range messagesWithResponses {
		messagesByID[message.ID] = message
	}
	messageState.Response.Messages = messagesWithResponses

	for _, message := range messageState.Response.Messages {
		if _, ok := newMessagesIds[message.ID]; ok {
			message.New = true

			// Create notification body to be eventually passed to `localnotifications.SendMessageNotifications()`
			if err = messageState.addNewMessageNotification(m.identity.PublicKey, message, messagesByID[message.ResponseTo]); err != nil {
				return nil, err
			}
		}
	}

	// Reset installations
	m.modifiedInstallations = make(map[string]bool)

	return messageState.Response, nil
}

func showNotification(publicKey ecdsa.PublicKey, n MessageNotificationBody, responseTo *common.Message) bool {
	if n.Chat != nil && n.Chat.ChatType == ChatTypeOneToOne {
		return true
	}

	publicKeyString := common.PubkeyToHex(&publicKey)
	mentioned := false
	for _, mention := range n.Message.Mentions {
		if publicKeyString == mention {
			mentioned = true
		}
	}

	if mentioned {
		return true
	}

	if responseTo != nil {
		return responseTo.From == publicKeyString
	}

	return false
}

// SetMailserver sets the currently used mailserver
func (m *Messenger) SetMailserver(peer []byte) {
	m.mailserver = peer
}

func (m *Messenger) RequestHistoricMessages(
	ctx context.Context,
	from, to uint32,
	cursor []byte,
) ([]byte, error) {
	if m.mailserver == nil {
		return nil, errors.New("no mailserver selected")
	}
	return m.transport.SendMessagesRequest(ctx, m.mailserver, from, to, cursor)
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

func (m *Messenger) MessageByID(id string) (*common.Message, error) {
	return m.persistence.MessageByID(id)
}

func (m *Messenger) MessagesExist(ids []string) (map[string]bool, error) {
	return m.persistence.MessagesExist(ids)
}

func (m *Messenger) MessageByChatID(chatID, cursor string, limit int) ([]*common.Message, string, error) {
	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return nil, "", err
	}

	if chat.Timeline() {
		var chatIDs = []string{"@" + contactIDFromPublicKey(&m.identity.PublicKey)}
		contacts, err := m.persistence.Contacts()
		if err != nil {
			return nil, "", err
		}
		for _, contact := range contacts {
			if contact.IsAdded() {
				chatIDs = append(chatIDs, "@"+contact.ID)
			}
		}
		return m.persistence.MessageByChatIDs(chatIDs, cursor, limit)
	}
	return m.persistence.MessageByChatID(chatID, cursor, limit)
}

func (m *Messenger) SaveMessages(messages []*common.Message) error {
	return m.persistence.SaveMessages(messages)
}

func (m *Messenger) DeleteMessage(id string) error {
	return m.persistence.DeleteMessage(id)
}

func (m *Messenger) DeleteMessagesByChatID(id string) error {
	return m.persistence.DeleteMessagesByChatID(id)
}

func (m *Messenger) ClearHistory(id string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.clearHistory(id)

}

func (m *Messenger) clearHistory(id string) (*MessengerResponse, error) {
	chat, ok := m.allChats[id]
	if !ok {
		return nil, ErrChatNotFound
	}

	clock, _ := chat.NextClockAndTimestamp(m.transport)

	err := m.persistence.ClearHistory(chat, clock)
	if err != nil {
		return nil, err
	}

	m.allChats[id] = chat

	response := &MessengerResponse{}
	response.AddChat(chat)
	return response, nil
}

// MarkMessagesSeen marks messages with `ids` as seen in the chat `chatID`.
// It returns the number of affected messages or error. If there is an error,
// the number of affected messages is always zero.
func (m *Messenger) MarkMessagesSeen(chatID string, ids []string) (uint64, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	count, err := m.persistence.MarkMessagesSeen(chatID, ids)
	if err != nil {
		return 0, err
	}
	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return 0, err
	}
	m.allChats[chatID] = chat
	return count, nil
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

// MuteChat signals to the messenger that we don't want to be notified
// on new messages from this chat
func (m *Messenger) MuteChat(chatID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	chat, ok := m.allChats[chatID]
	if !ok {
		return errors.New("chat not found")
	}

	err := m.persistence.MuteChat(chatID)
	if err != nil {
		return err
	}

	chat.Muted = true
	m.allChats[chat.ID] = chat

	return m.reregisterForPushNotifications()
}

// UnmuteChat signals to the messenger that we want to be notified
// on new messages from this chat
func (m *Messenger) UnmuteChat(chatID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	chat, ok := m.allChats[chatID]
	if !ok {
		return errors.New("chat not found")
	}

	err := m.persistence.UnmuteChat(chatID)
	if err != nil {
		return err
	}

	chat.Muted = false
	m.allChats[chat.ID] = chat
	return m.reregisterForPushNotifications()
}

func (m *Messenger) UpdateMessageOutgoingStatus(id, newOutgoingStatus string) error {
	return m.persistence.UpdateMessageOutgoingStatus(id, newOutgoingStatus)
}

// Identicon returns an identicon based on the input string
func Identicon(id string) (string, error) {
	return identicon.GenerateBase64(id)
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

	message := &common.Message{}
	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
	if err != nil {
		return nil, err
	}

	message.MessageType = protobuf.MessageType_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	message.Seen = true
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
	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_REQUEST_TRANSACTION,
		ResendAutomatically: true,
	})

	message.CommandParameters = &common.CommandParameters{
		ID:           rawMessage.ID,
		Value:        value,
		Address:      address,
		Contract:     contract,
		CommandState: common.CommandStateRequestTransaction,
	}

	if err != nil {
		return nil, err
	}
	messageID := rawMessage.ID

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

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)
	response.Messages = []*common.Message{message}
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

	message := &common.Message{}
	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
	if err != nil {
		return nil, err
	}

	message.MessageType = protobuf.MessageType_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	message.Seen = true
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
	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_REQUEST_ADDRESS_FOR_TRANSACTION,
		ResendAutomatically: true,
	})

	message.CommandParameters = &common.CommandParameters{
		ID:           rawMessage.ID,
		From:         from,
		Value:        value,
		Contract:     contract,
		CommandState: common.CommandStateRequestAddressForTransaction,
	}

	if err != nil {
		return nil, err
	}
	messageID := rawMessage.ID

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

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)
	response.Messages = []*common.Message{message}
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
	message.Seen = true
	message.OutgoingStatus = common.OutgoingStatusSending

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

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_ACCEPT_REQUEST_ADDRESS_FOR_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters.Address = address
	message.CommandParameters.CommandState = common.CommandStateRequestAddressForTransactionAccepted

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)
	response.Messages = []*common.Message{message}
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
	message.Seen = true
	message.OutgoingStatus = common.OutgoingStatusSending
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

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters.CommandState = common.CommandStateRequestTransactionDeclined

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)
	response.Messages = []*common.Message{message}
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
	message.Seen = true
	message.OutgoingStatus = common.OutgoingStatusSending
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

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_DECLINE_REQUEST_ADDRESS_FOR_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters.CommandState = common.CommandStateRequestAddressForTransactionDeclined

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)
	response.Messages = []*common.Message{message}
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
	message.Seen = true
	message.Text = transactionSentTxt
	message.OutgoingStatus = common.OutgoingStatusSending

	// Hide previous message
	previousMessage, err := m.persistence.MessageByCommandID(chatID, messageID)
	if err != nil && err != common.ErrRecordNotFound {
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

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SEND_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters.TransactionHash = transactionHash
	message.CommandParameters.Signature = signature
	message.CommandParameters.CommandState = common.CommandStateTransactionSent

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)
	response.Messages = []*common.Message{message}
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

	message := &common.Message{}
	err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
	if err != nil {
		return nil, err
	}

	message.MessageType = protobuf.MessageType_ONE_TO_ONE
	message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
	message.LocalChatID = chatID

	clock, timestamp := chat.NextClockAndTimestamp(m.transport)
	message.Clock = clock
	message.WhisperTimestamp = timestamp
	message.Seen = true
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

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SEND_TRANSACTION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	message.ID = rawMessage.ID
	message.CommandParameters = &common.CommandParameters{
		TransactionHash: transactionHash,
		Value:           value,
		Contract:        contract,
		Signature:       signature,
		CommandState:    common.CommandStateTransactionSent,
	}

	err = message.PrepareContent()
	if err != nil {
		return nil, err
	}

	err = chat.UpdateFromMessage(message, m.transport)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{message})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)
	response.Messages = []*common.Message{message}
	return &response, m.saveChat(chat)
}

func (m *Messenger) ValidateTransactions(ctx context.Context, addresses []types.Address) (*MessengerResponse, error) {
	if m.verifyTransactionClient == nil {
		return nil, nil
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

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
		var message *common.Message
		chatID := contactIDFromPublicKey(validationResult.Transaction.From)
		chat, ok := m.allChats[chatID]
		if !ok {
			chat = OneToOneFromPublicKey(validationResult.Transaction.From, m.transport)
		}
		if validationResult.Message != nil {
			message = validationResult.Message
		} else {
			message = &common.Message{}
			err := extendMessageFromChat(message, chat, &m.identity.PublicKey, m.transport)
			if err != nil {
				return nil, err
			}
		}

		message.MessageType = protobuf.MessageType_ONE_TO_ONE
		message.ContentType = protobuf.ChatMessage_TRANSACTION_COMMAND
		message.LocalChatID = chatID
		message.OutgoingStatus = ""

		clock, timestamp := chat.NextClockAndTimestamp(m.transport)
		message.Clock = clock
		message.Timestamp = timestamp
		message.WhisperTimestamp = timestamp
		message.Text = "Transaction received"
		message.Seen = false

		message.ID = validationResult.Transaction.MessageID
		if message.CommandParameters == nil {
			message.CommandParameters = &common.CommandParameters{}
		} else {
			message.CommandParameters = validationResult.Message.CommandParameters
		}

		message.CommandParameters.Value = validationResult.Value
		message.CommandParameters.Contract = validationResult.Contract
		message.CommandParameters.Address = validationResult.Address
		message.CommandParameters.CommandState = common.CommandStateTransactionSent
		message.CommandParameters.TransactionHash = validationResult.Transaction.TransactionHash

		err = message.PrepareContent()
		if err != nil {
			return nil, err
		}

		err = chat.UpdateFromMessage(message, m.transport)
		if err != nil {
			return nil, err
		}

		if message.CommandParameters.ID != "" {
			// Hide previous message
			previousMessage, err := m.persistence.MessageByCommandID(chatID, message.CommandParameters.ID)
			if err != nil && err != common.ErrRecordNotFound {
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
		response.AddChat(chat)

		contact, err := m.getOrBuildContactFromMessage(message)
		if err != nil {
			return nil, err
		}
		response.Notifications = append(response.Notifications, MessageNotificationBody{
			Message: message,
			Contact: contact,
			Chat:    chat,
		})

	}

	if len(response.Messages) > 0 {
		err = m.SaveMessages(response.Messages)
		if err != nil {
			return nil, err
		}
	}
	return &response, nil
}

// pullMessagesAndResponsesFromDB pulls all the messages and the one that have
// been replied to from the database
func (m *Messenger) pullMessagesAndResponsesFromDB(messages []*common.Message) ([]*common.Message, error) {
	var messageIDs []string
	for _, message := range messages {
		messageIDs = append(messageIDs, message.ID)
		if message.ResponseTo != "" {
			messageIDs = append(messageIDs, message.ResponseTo)
		}

	}
	// We pull from the database all the messages & replies involved,
	// so we let the db build the correct messages
	return m.persistence.MessagesByIDs(messageIDs)
}

func (m *Messenger) SignMessage(message string) ([]byte, error) {
	hash := crypto.TextHash([]byte(message))
	return crypto.Sign(hash, m.identity)
}

func (m *Messenger) getTimesource() common.TimeSource {
	return m.transport
}

// AddPushNotificationsServer adds a push notification server
func (m *Messenger) AddPushNotificationsServer(ctx context.Context, publicKey *ecdsa.PublicKey, serverType pushnotificationclient.ServerType) error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	return m.pushNotificationClient.AddPushNotificationsServer(publicKey, serverType)
}

// RemovePushNotificationServer removes a push notification server
func (m *Messenger) RemovePushNotificationServer(ctx context.Context, publicKey *ecdsa.PublicKey) error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	return m.pushNotificationClient.RemovePushNotificationServer(publicKey)
}

// UnregisterFromPushNotifications unregister from any server
func (m *Messenger) UnregisterFromPushNotifications(ctx context.Context) error {
	return m.pushNotificationClient.Unregister()
}

// DisableSendingPushNotifications signals the client not to send any push notification
func (m *Messenger) DisableSendingPushNotifications() error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	m.pushNotificationClient.DisableSending()
	return nil
}

// EnableSendingPushNotifications signals the client to send push notifications
func (m *Messenger) EnableSendingPushNotifications() error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	m.pushNotificationClient.EnableSending()
	return nil
}

func (m *Messenger) pushNotificationOptions() *pushnotificationclient.RegistrationOptions {
	var contactIDs []*ecdsa.PublicKey
	var mutedChatIDs []string
	var publicChatIDs []string

	for _, contact := range m.allContacts {
		if contact.IsAdded() && !contact.IsBlocked() {
			pk, err := contact.PublicKey()
			if err != nil {
				m.logger.Warn("could not parse contact public key")
				continue
			}
			contactIDs = append(contactIDs, pk)
		} else if contact.IsBlocked() {
			mutedChatIDs = append(mutedChatIDs, contact.ID)
		}
	}
	for _, chat := range m.allChats {
		if chat.Muted {
			mutedChatIDs = append(mutedChatIDs, chat.ID)
		}
		if chat.Active && chat.Public() {
			publicChatIDs = append(publicChatIDs, chat.ID)
		}

	}
	return &pushnotificationclient.RegistrationOptions{
		ContactIDs:    contactIDs,
		MutedChatIDs:  mutedChatIDs,
		PublicChatIDs: publicChatIDs,
	}
}

// RegisterForPushNotification register deviceToken with any push notification server enabled
func (m *Messenger) RegisterForPushNotifications(ctx context.Context, deviceToken, apnTopic string, tokenType protobuf.PushNotificationRegistration_TokenType) error {
	if m.pushNotificationClient == nil {
		return errors.New("push notification client not enabled")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := m.pushNotificationClient.Register(deviceToken, apnTopic, tokenType, m.pushNotificationOptions())
	if err != nil {
		m.logger.Error("failed to register for push notifications", zap.Error(err))
		return err
	}
	return nil
}

// RegisteredForPushNotifications returns whether we successfully registered with all the servers
func (m *Messenger) RegisteredForPushNotifications() (bool, error) {
	if m.pushNotificationClient == nil {
		return false, errors.New("no push notification client")
	}
	return m.pushNotificationClient.Registered()
}

// EnablePushNotificationsFromContactsOnly is used to indicate that we want to received push notifications only from contacts
func (m *Messenger) EnablePushNotificationsFromContactsOnly() error {
	if m.pushNotificationClient == nil {
		return errors.New("no push notification client")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.pushNotificationClient.EnablePushNotificationsFromContactsOnly(m.pushNotificationOptions())
}

// DisablePushNotificationsFromContactsOnly is used to indicate that we want to received push notifications from anyone
func (m *Messenger) DisablePushNotificationsFromContactsOnly() error {
	if m.pushNotificationClient == nil {
		return errors.New("no push notification client")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.pushNotificationClient.DisablePushNotificationsFromContactsOnly(m.pushNotificationOptions())
}

// EnablePushNotificationsBlockMentions is used to indicate that we dont want to received push notifications for mentions
func (m *Messenger) EnablePushNotificationsBlockMentions() error {
	if m.pushNotificationClient == nil {
		return errors.New("no push notification client")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.pushNotificationClient.EnablePushNotificationsBlockMentions(m.pushNotificationOptions())
}

// DisablePushNotificationsBlockMentions is used to indicate that we want to received push notifications for mentions
func (m *Messenger) DisablePushNotificationsBlockMentions() error {
	if m.pushNotificationClient == nil {
		return errors.New("no push notification client")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.pushNotificationClient.DisablePushNotificationsBlockMentions(m.pushNotificationOptions())
}

// GetPushNotificationsServers returns the servers used for push notifications
func (m *Messenger) GetPushNotificationsServers() ([]*pushnotificationclient.PushNotificationServer, error) {
	if m.pushNotificationClient == nil {
		return nil, errors.New("no push notification client")
	}
	return m.pushNotificationClient.GetServers()
}

// StartPushNotificationsServer initialize and start a push notification server, using the current messenger identity key
func (m *Messenger) StartPushNotificationsServer() error {
	if m.pushNotificationServer == nil {
		pushNotificationServerPersistence := pushnotificationserver.NewSQLitePersistence(m.database)
		config := &pushnotificationserver.Config{
			Enabled:  true,
			Logger:   m.logger,
			Identity: m.identity,
		}
		m.pushNotificationServer = pushnotificationserver.New(config, pushNotificationServerPersistence, m.processor)
	}

	return m.pushNotificationServer.Start()
}

// StopPushNotificationServer stops the push notification server if running
func (m *Messenger) StopPushNotificationsServer() error {
	m.pushNotificationServer = nil
	return nil
}

func generateAliasAndIdenticon(pk string) (string, string, error) {
	identicon, err := identicon.GenerateBase64(pk)
	if err != nil {
		return "", "", err
	}

	name, err := alias.GenerateFromPublicKeyString(pk)
	if err != nil {
		return "", "", err
	}
	return name, identicon, nil

}

func (m *Messenger) SendEmojiReaction(ctx context.Context, chatID, messageID string, emojiID protobuf.EmojiReaction_Type) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var response MessengerResponse

	chat, ok := m.allChats[chatID]
	if !ok {
		return nil, ErrChatNotFound
	}
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	emojiR := &EmojiReaction{
		EmojiReaction: protobuf.EmojiReaction{
			Clock:     clock,
			MessageId: messageID,
			ChatId:    chatID,
			Type:      emojiID,
		},
		LocalChatID: chatID,
		From:        types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
	}
	encodedMessage, err := m.encodeChatEntity(chat, emojiR)
	if err != nil {
		return nil, err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:          chatID,
		Payload:              encodedMessage,
		SkipGroupMessageWrap: true,
		MessageType:          protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		// Don't resend using datasync, that would create quite a lot
		// of traffic if clicking too eagelry
		ResendAutomatically: false,
	})
	if err != nil {
		return nil, err
	}

	response.EmojiReactions = []*EmojiReaction{emojiR}
	response.AddChat(chat)

	err = m.persistence.SaveEmojiReaction(emojiR)
	if err != nil {
		return nil, errors.Wrap(err, "Can't save emoji reaction in db")
	}

	return &response, nil
}

func (m *Messenger) EmojiReactionsByChatID(chatID string, cursor string, limit int) ([]*EmojiReaction, error) {
	chat, err := m.persistence.Chat(chatID)
	if err != nil {
		return nil, err
	}

	if chat.Timeline() {
		var chatIDs = []string{"@" + contactIDFromPublicKey(&m.identity.PublicKey)}
		for _, contact := range m.allContacts {
			if contact.IsAdded() {
				chatIDs = append(chatIDs, "@"+contact.ID)
			}
		}
		return m.persistence.EmojiReactionsByChatIDs(chatIDs, cursor, limit)
	}
	return m.persistence.EmojiReactionsByChatID(chatID, cursor, limit)
}

func (m *Messenger) SendEmojiReactionRetraction(ctx context.Context, emojiReactionID string) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	emojiR, err := m.persistence.EmojiReactionByID(emojiReactionID)
	if err != nil {
		return nil, err
	}

	// Check that the sender is the key owner
	pk := types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey))
	if emojiR.From != pk {
		return nil, errors.Errorf("identity mismatch, "+
			"emoji reactions can only be retracted by the reaction sender, "+
			"emoji reaction sent by '%s', current identity '%s'",
			emojiR.From, pk,
		)
	}

	// Get chat and clock
	chat, ok := m.allChats[emojiR.GetChatId()]
	if !ok {
		return nil, ErrChatNotFound
	}
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	// Update the relevant fields
	emojiR.Clock = clock
	emojiR.Retracted = true

	encodedMessage, err := m.encodeChatEntity(chat, emojiR)
	if err != nil {
		return nil, err
	}

	// Send the marshalled EmojiReactionRetraction protobuf
	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:          emojiR.GetChatId(),
		Payload:              encodedMessage,
		SkipGroupMessageWrap: true,
		MessageType:          protobuf.ApplicationMetadataMessage_EMOJI_REACTION,
		// Don't resend using datasync, that would create quite a lot
		// of traffic if clicking too eagelry
		ResendAutomatically: false,
	})
	if err != nil {
		return nil, err
	}

	// Update MessengerResponse
	response := MessengerResponse{}
	emojiR.Retracted = true
	response.EmojiReactions = []*EmojiReaction{emojiR}
	response.AddChat(chat)

	// Persist retraction state for emoji reaction
	err = m.persistence.SaveEmojiReaction(emojiR)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (m *Messenger) encodeChatEntity(chat *Chat, message common.ChatEntity) ([]byte, error) {
	var encodedMessage []byte
	var err error
	l := m.logger.With(zap.String("site", "Send"), zap.String("chatID", chat.ID))

	switch chat.ChatType {
	case ChatTypeOneToOne:
		l.Debug("sending private message")
		message.SetMessageType(protobuf.MessageType_ONE_TO_ONE)
		encodedMessage, err = proto.Marshal(message.GetProtobuf())
		if err != nil {
			return nil, err
		}
	case ChatTypePublic, ChatTypeProfile:
		l.Debug("sending public message", zap.String("chatName", chat.Name))
		message.SetMessageType(protobuf.MessageType_PUBLIC_GROUP)
		encodedMessage, err = proto.Marshal(message.GetProtobuf())
		if err != nil {
			return nil, err
		}
	case ChatTypeCommunityChat:
		l.Debug("sending community chat message", zap.String("chatName", chat.Name))
		// TODO: add grant
		message.SetMessageType(protobuf.MessageType_COMMUNITY_CHAT)
		encodedMessage, err = proto.Marshal(message.GetProtobuf())
		if err != nil {
			return nil, err
		}
	case ChatTypePrivateGroupChat:
		message.SetMessageType(protobuf.MessageType_PRIVATE_GROUP)
		l.Debug("sending group message", zap.String("chatName", chat.Name))
		if !message.WrapGroupMessage() {
			encodedMessage, err = proto.Marshal(message.GetProtobuf())
			if err != nil {
				return nil, err
			}
		} else {

			group, err := newProtocolGroupFromChat(chat)
			if err != nil {
				return nil, err
			}

			encodedMessage, err = m.processor.EncodeAbridgedMembershipUpdate(group, message)
			if err != nil {
				return nil, err
			}
		}

	default:
		return nil, errors.New("chat type not supported")
	}

	return encodedMessage, nil
}

func (m *Messenger) getOrBuildContactFromMessage(msg *common.Message) (*Contact, error) {
	if c, ok := m.allContacts[msg.From]; ok {
		return c, nil
	}

	senderPubKey, err := msg.GetSenderPubKey()
	if err != nil {
		return nil, err
	}
	senderID := contactIDFromPublicKey(senderPubKey)
	c, err := buildContact(senderID, senderPubKey)
	if err != nil {
		return nil, err
	}

	m.allContacts[msg.From] = c
	return c, nil
}
