package push_notification_client

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"io"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"go.uber.org/zap"
)

const encryptedPayloadKeyLength = 16
const accessTokenKeyLength = 16

type PushNotificationServer struct {
	publicKey    *ecdsa.PublicKey
	registered   bool
	registeredAt int64
}

type PushNotificationInfo struct {
	AccessToken    string
	InstallationID string
	PublicKey      *ecdsa.PublicKey
}

type Config struct {
	// Identity is our identity key
	Identity *ecdsa.PrivateKey
	// SendEnabled indicates whether we should be sending push notifications
	SendEnabled bool
	// RemoteNotificationsEnabled is whether we should register with a remote server for push notifications
	RemoteNotificationsEnabled bool

	// AllowOnlyFromContacts indicates whether we should be receiving push notifications
	// only from contacts
	AllowOnlyFromContacts bool
	// PushNotificationServers is an array of push notification servers we want to register with
	PushNotificationServers []*PushNotificationServer
	// InstallationID is the installation-id for this device
	InstallationID string

	Logger *zap.Logger

	// TokenType is the type of token
	TokenType protobuf.PushNotificationRegistration_TokenType
}

type Client struct {
	persistence *Persistence
	quit        chan struct{}
	config      *Config

	// lastPushNotificationVersion is the latest known push notification version
	lastPushNotificationVersion uint64

	// AccessToken is the access token that is currently being used
	AccessToken string
	// DeviceToken is the device token for this device
	DeviceToken string

	// randomReader only used for testing so we have deterministic encryption
	reader io.Reader

	//messageProcessor is a message processor used to send and being notified of messages

	messageProcessor *common.MessageProcessor
	//pushNotificationQueryResponses is a channel that listens to pushNotificationResponse

	pushNotificationQueryResponses chan *protobuf.PushNotificationQueryResponse
}

func New(persistence *Persistence, config *Config, processor *common.MessageProcessor) *Client {
	return &Client{
		quit:                           make(chan struct{}),
		config:                         config,
		pushNotificationQueryResponses: make(chan *protobuf.PushNotificationQueryResponse),
		messageProcessor:               processor,
		persistence:                    persistence,
		reader:                         rand.Reader}
}

func (c *Client) Start() error {
	if c.messageProcessor == nil {
		return errors.New("can't start, missing message processor")
	}

	go func() {
		subscription := c.messageProcessor.Subscribe()
		for {
			select {
			case m := <-subscription:
				if err := c.HandleMessageSent(m); err != nil {
					// TODO: log
				}
			case <-c.quit:
				return
			}
		}
	}()
	return nil
}

func (c *Client) Stop() error {
	close(c.quit)
	return nil
}

// This likely will return a channel as it's an asynchrous operation
func fetchNotificationInfoFor(publicKey *ecdsa.PublicKey) error {
	return nil
}

// Sends an actual push notification, where do we get the chatID?
func sendPushNotificationTo(publicKey *ecdsa.PublicKey, chatID string) error {
	return nil
}

// This should schedule:
// 1) Check we have reasonably fresh push notifications info
// 2) Otherwise it should fetch them
// 3) Send a push notification to the devices in question
func (p *Client) HandleMessageSent(sentMessage *common.SentMessage) error {
	return nil
}

func (p *Client) NotifyOnMessageID(messageID []byte) error {
	return nil
}

func (p *Client) mutedChatIDsHashes(chatIDs []string) [][]byte {
	var mutedChatListHashes [][]byte

	for _, chatID := range chatIDs {
		mutedChatListHashes = append(mutedChatListHashes, common.Shake256([]byte(chatID)))
	}

	return mutedChatListHashes
}

func (p *Client) encryptToken(publicKey *ecdsa.PublicKey, token []byte) ([]byte, error) {
	sharedKey, err := ecies.ImportECDSA(p.config.Identity).GenerateShared(
		ecies.ImportECDSAPublic(publicKey),
		accessTokenKeyLength,
		accessTokenKeyLength,
	)
	if err != nil {
		return nil, err
	}
	encryptedToken, err := encryptAccessToken(token, sharedKey, p.reader)
	if err != nil {
		return nil, err
	}
	return encryptedToken, nil
}

func (p *Client) allowedUserList(token []byte, contactIDs []*ecdsa.PublicKey) ([][]byte, error) {
	var encryptedTokens [][]byte
	for _, publicKey := range contactIDs {
		encryptedToken, err := p.encryptToken(publicKey, token)
		if err != nil {
			return nil, err
		}

		encryptedTokens = append(encryptedTokens, encryptedToken)

	}
	return encryptedTokens, nil
}

func (p *Client) buildPushNotificationRegistrationMessage(contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) (*protobuf.PushNotificationRegistration, error) {
	token := uuid.New().String()
	allowedUserList, err := p.allowedUserList([]byte(token), contactIDs)
	if err != nil {
		return nil, err
	}

	options := &protobuf.PushNotificationRegistration{
		AccessToken:     token,
		TokenType:       p.config.TokenType,
		Version:         p.lastPushNotificationVersion + 1,
		InstallationId:  p.config.InstallationID,
		Token:           p.DeviceToken,
		Enabled:         p.config.RemoteNotificationsEnabled,
		BlockedChatList: p.mutedChatIDsHashes(mutedChatIDs),
		AllowedUserList: allowedUserList,
	}
	return options, nil
}

func (c *Client) Register(deviceToken string, contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) error {
	c.DeviceToken = deviceToken
	servers, err := c.persistence.GetServers()
	if err != nil {
		return err
	}

	if len(servers) == 0 {
		return errors.New("no servers to register with")
	}

	registration, err := c.buildPushNotificationRegistrationMessage(contactIDs, mutedChatIDs)
	if err != nil {
		return err
	}

	marshaledRegistration, err := proto.Marshal(registration)
	if err != nil {
		return err
	}

	for _, server := range servers {

		encryptedRegistration, err := c.encryptRegistration(server.publicKey, marshaledRegistration)
		if err != nil {
			return err
		}
		rawMessage := &common.RawMessage{
			Payload:     encryptedRegistration,
			MessageType: protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REGISTRATION,
		}

		_, err = c.messageProcessor.SendPrivate(context.Background(), server.publicKey, rawMessage)

		// Send message and wait for reply

	}
	for {
		select {
		case <-c.quit:
			return nil
		case <-time.After(5 * time.Second):
			return errors.New("no query response received")
		case <-c.pushNotificationQueryResponses:
			return nil
		}
	}
}

// HandlePushNotificationRegistrationResponse should check whether the response was successful or not, retry if necessary otherwise store the result in the database
func (p *Client) HandlePushNotificationRegistrationResponse(response *protobuf.PushNotificationRegistrationResponse) error {
	return nil
}

// HandlePushNotificationAdvertisement should store any info related to push notifications
func (p *Client) HandlePushNotificationAdvertisement(info *protobuf.PushNotificationAdvertisementInfo) error {
	return nil
}

// HandlePushNotificationQueryResponse should update the data in the database for a given user
func (c *Client) HandlePushNotificationQueryResponse(response *protobuf.PushNotificationQueryResponse) error {

	c.config.Logger.Debug("received push notification query response", zap.Any("response", response))
	select {
	case c.pushNotificationQueryResponses <- response:
	default:
		return errors.New("could not process push notification query response")
	}
	return nil
}

// HandlePushNotificationResponse should set the request as processed
func (p *Client) HandlePushNotificationResponse(ack *protobuf.PushNotificationResponse) error {
	return nil
}

func (c *Client) AddPushNotificationServer(publicKey *ecdsa.PublicKey) error {
	c.config.Logger.Debug("adding push notification server", zap.Any("public-key", publicKey))
	currentServers, err := c.persistence.GetServers()
	if err != nil {
		return err
	}

	for _, server := range currentServers {
		if common.IsPubKeyEqual(server.publicKey, publicKey) {
			return errors.New("push notification server already added")
		}
	}

	return c.persistence.UpsertServer(&PushNotificationServer{
		publicKey: publicKey,
	})
}

func (c *Client) RetrievePushNotificationInfo(publicKey *ecdsa.PublicKey) ([]*PushNotificationInfo, error) {
	return nil, nil
	/*
		currentServers, err := c.persistence.GetServers()
		if err != nil {
			return err
		}

		for _, server := range currentServers {
			if common.IsPubKeyEqual(server.publicKey, publicKey) {
				return errors.New("push notification server already added")
			}
		}

		return c.persistence.UpsertServer(&PushNotificationServer{
			publicKey: publicKey,
		})*/
}

func encryptAccessToken(plaintext []byte, key []byte, reader io.Reader) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *Client) encryptRegistration(publicKey *ecdsa.PublicKey, payload []byte) ([]byte, error) {
	sharedKey, err := c.generateSharedKey(publicKey)
	if err != nil {
		return nil, err
	}

	return common.Encrypt(payload, sharedKey, c.reader)
}

func (c *Client) generateSharedKey(publicKey *ecdsa.PublicKey) ([]byte, error) {
	return ecies.ImportECDSA(c.config.Identity).GenerateShared(
		ecies.ImportECDSAPublic(publicKey),
		encryptedPayloadKeyLength,
		encryptedPayloadKeyLength,
	)
}
