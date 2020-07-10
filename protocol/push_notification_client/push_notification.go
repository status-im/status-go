package push_notification_client

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"sort"

	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"go.uber.org/zap"
)

const encryptedPayloadKeyLength = 16
const accessTokenKeyLength = 16
const staleQueryTimeInSeconds = 86400

type PushNotificationServer struct {
	PublicKey    *ecdsa.PublicKey
	Registered   bool
	RegisteredAt int64
	AccessToken  string
}

type PushNotificationInfo struct {
	AccessToken     string
	InstallationID  string
	PublicKey       *ecdsa.PublicKey
	ServerPublicKey *ecdsa.PublicKey
	RetrievedAt     int64
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
}

func New(persistence *Persistence, config *Config, processor *common.MessageProcessor) *Client {
	return &Client{
		quit:             make(chan struct{}),
		config:           config,
		messageProcessor: processor,
		persistence:      persistence,
		reader:           rand.Reader}
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
					c.config.Logger.Error("failed to handle message", zap.Error(err))
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

type notificationSendingSpec struct {
	serverPublicKey *ecdsa.PublicKey
	installationID  string
	messageID       []byte
}

// The message has been sent
// We should:
// 1) Check whether we should notify on anything
// 2) Refresh info if necessaary
// 3) Sent push notifications
func (c *Client) HandleMessageSent(sentMessage *common.SentMessage) error {
	if !c.config.SendEnabled {
		return nil
	}
	publicKey := sentMessage.PublicKey
	var installationIDs []string

	var notificationSpecs []*notificationSendingSpec

	//Find if there's any actionable message
	for _, messageID := range sentMessage.MessageIDs {
		for _, installation := range sentMessage.Spec.Installations {
			installationID := installation.ID
			shouldNotify, err := c.shouldNotifyOn(publicKey, installationID, messageID)
			if err != nil {
				return err
			}
			if shouldNotify {
				notificationSpecs = append(notificationSpecs, &notificationSendingSpec{
					installationID: installationID,
					messageID:      messageID,
				})
				installationIDs = append(installationIDs, installation.ID)
			}
		}
	}

	// Is there anything we should be notifying on?
	if len(installationIDs) == 0 {
		return nil
	}

	// Check if we queried recently
	queriedAt, err := c.persistence.GetQueriedAt(publicKey)
	if err != nil {
		return err
	}

	// Naively query again if too much time has passed.
	// Here it might not be necessary
	if time.Now().Unix()-queriedAt > staleQueryTimeInSeconds {
		err := c.QueryPushNotificationInfo(publicKey)
		if err != nil {
			return err
		}
		// This is just horrible, but for now will do,
		// the issue is that we don't really know how long it will
		// take to reply, as there might be multiple servers
		// replying to us.
		// The only time we are 100% certain that we can proceed is
		// when we have non-stale info for each device, but
		// most devices are not going to be registered, so we'd still
		// have to wait teh maximum amount of time allowed.
		time.Sleep(3 * time.Second)

	}
	// Retrieve infos
	info, err := c.GetPushNotificationInfo(publicKey, installationIDs)
	if err != nil {
		return err
	}

	// Naively dispatch to the first server for now
	// This wait for an acknowledgement and try a different server after a timeout
	// Also we sent a single notification for multiple message ids, need to check with UI what's the desired behavior

	// Sort by server so we tend to hit the same one
	sort.Slice(info, func(i, j int) bool {
		return info[i].ServerPublicKey.X.Cmp(info[j].ServerPublicKey.X) <= 0
	})

	installationIDsMap := make(map[string]bool)
	// One info per installation id, grouped by server
	actionableInfos := make(map[string][]*PushNotificationInfo)
	for _, i := range info {
		if !installationIDsMap[i.InstallationID] {
			serverKey := hex.EncodeToString(crypto.CompressPubkey(i.ServerPublicKey))
			actionableInfos[serverKey] = append(actionableInfos[serverKey], i)
			installationIDsMap[i.InstallationID] = true
		}

	}

	for _, infos := range actionableInfos {
		var pushNotifications []*protobuf.PushNotification
		for _, i := range infos {
			// TODO: Add ChatID, message, public_key
			pushNotifications = append(pushNotifications, &protobuf.PushNotification{
				AccessToken:    i.AccessToken,
				PublicKey:      common.HashPublicKey(publicKey),
				InstallationId: i.InstallationID,
			})

		}
		request := &protobuf.PushNotificationRequest{
			MessageId: sentMessage.MessageIDs[0],
			Requests:  pushNotifications,
		}
		serverPublicKey := infos[0].ServerPublicKey

		payload, err := proto.Marshal(request)
		if err != nil {
			return err
		}

		rawMessage := &common.RawMessage{
			Payload:     payload,
			MessageType: protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REQUEST,
		}

		// TODO: We should use the messageID for the response
		_, err = c.messageProcessor.SendPrivate(context.Background(), serverPublicKey, rawMessage)

		if err != nil {
			return err
		}
	}

	return nil
}

// NotifyOnMessageID keeps track of the message to make sure we notify on it
func (c *Client) NotifyOnMessageID(chatID string, messageID []byte) error {
	return c.persistence.TrackPushNotification(chatID, messageID)
}

func (c *Client) shouldNotifyOn(publicKey *ecdsa.PublicKey, installationID string, messageID []byte) (bool, error) {
	return c.persistence.ShouldSentNotificationFor(publicKey, installationID, messageID)
}

func (c *Client) notifiedOn(publicKey *ecdsa.PublicKey, installationID string, messageID []byte) error {
	return c.persistence.NotifiedOn(publicKey, installationID, messageID)
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

func (c *Client) Register(deviceToken string, contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) ([]*PushNotificationServer, error) {
	c.DeviceToken = deviceToken
	servers, err := c.persistence.GetServers()
	if err != nil {
		return nil, err
	}

	if len(servers) == 0 {
		return nil, errors.New("no servers to register with")
	}

	registration, err := c.buildPushNotificationRegistrationMessage(contactIDs, mutedChatIDs)
	if err != nil {
		return nil, err
	}

	marshaledRegistration, err := proto.Marshal(registration)
	if err != nil {
		return nil, err
	}

	var serverPublicKeys []*ecdsa.PublicKey
	for _, server := range servers {

		// Reset server registration data
		server.Registered = false
		server.RegisteredAt = 0
		server.AccessToken = registration.AccessToken
		serverPublicKeys = append(serverPublicKeys, server.PublicKey)

		if err := c.persistence.UpsertServer(server); err != nil {
			return nil, err
		}

		// Dispatch message
		encryptedRegistration, err := c.encryptRegistration(server.PublicKey, marshaledRegistration)
		if err != nil {
			return nil, err
		}
		rawMessage := &common.RawMessage{
			Payload:     encryptedRegistration,
			MessageType: protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REGISTRATION,
		}

		_, err = c.messageProcessor.SendPrivate(context.Background(), server.PublicKey, rawMessage)

		if err != nil {
			return nil, err
		}

	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// This code polls the database for server registrations, giving up
	// after 5 seconds
	for {
		select {
		case <-c.quit:
			return servers, nil
		case <-ctx.Done():
			c.config.Logger.Debug("Context done")
			return servers, nil
		case <-time.After(200 * time.Millisecond):
			servers, err = c.persistence.GetServersByPublicKey(serverPublicKeys)
			if err != nil {
				return nil, err
			}

			allRegistered := true
			for _, server := range servers {
				allRegistered = allRegistered && server.Registered
			}

			// If any of the servers we haven't registered yet, continue
			if !allRegistered {
				continue
			}

			// all have registered,cancel context and return
			cancel()
			return servers, nil
		}
	}
}

// HandlePushNotificationRegistrationResponse should check whether the response was successful or not, retry if necessary otherwise store the result in the database
func (c *Client) HandlePushNotificationRegistrationResponse(publicKey *ecdsa.PublicKey, response protobuf.PushNotificationRegistrationResponse) error {
	c.config.Logger.Debug("received push notification registration response", zap.Any("response", response))
	// TODO: handle non successful response and match request id
	// Not successful ignore for now
	if !response.Success {
		return errors.New("response was not successful")
	}

	servers, err := c.persistence.GetServersByPublicKey([]*ecdsa.PublicKey{publicKey})
	if err != nil {
		return err
	}
	// We haven't registered with this server
	if len(servers) != 1 {
		return errors.New("not registered with this server, ignoring")
	}

	server := servers[0]
	server.Registered = true
	server.RegisteredAt = time.Now().Unix()

	return c.persistence.UpsertServer(server)
}

// HandlePushNotificationAdvertisement should store any info related to push notifications
func (p *Client) HandlePushNotificationAdvertisement(info *protobuf.PushNotificationAdvertisementInfo) error {
	return nil
}

// HandlePushNotificationQueryResponse should update the data in the database for a given user
func (c *Client) HandlePushNotificationQueryResponse(serverPublicKey *ecdsa.PublicKey, response protobuf.PushNotificationQueryResponse) error {

	c.config.Logger.Debug("received push notification query response", zap.Any("response", response))
	if len(response.Info) == 0 {
		return errors.New("empty response from the server")
	}

	publicKey, err := c.persistence.GetQueryPublicKey(response.MessageId)
	if err != nil {
		return err
	}
	if publicKey == nil {
		c.config.Logger.Debug("query not found")
		return nil
	}
	var pushNotificationInfo []*PushNotificationInfo
	for _, info := range response.Info {
		if bytes.Compare(info.PublicKey, common.HashPublicKey(publicKey)) != 0 {
			c.config.Logger.Warn("reply for different key, ignoring")
			continue
		}
		pushNotificationInfo = append(pushNotificationInfo, &PushNotificationInfo{
			PublicKey:       publicKey,
			ServerPublicKey: serverPublicKey,
			AccessToken:     info.AccessToken,
			InstallationID:  info.InstallationId,
			RetrievedAt:     time.Now().Unix(),
		})

	}
	err = c.persistence.SavePushNotificationInfo(pushNotificationInfo)
	if err != nil {
		c.config.Logger.Error("failed to save push notifications", zap.Error(err))
		return err
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
		if common.IsPubKeyEqual(server.PublicKey, publicKey) {
			return errors.New("push notification server already added")
		}
	}

	return c.persistence.UpsertServer(&PushNotificationServer{
		PublicKey: publicKey,
	})
}

func (c *Client) QueryPushNotificationInfo(publicKey *ecdsa.PublicKey) error {
	hashedPublicKey := common.HashPublicKey(publicKey)
	query := &protobuf.PushNotificationQuery{
		PublicKeys: [][]byte{hashedPublicKey},
	}
	encodedMessage, err := proto.Marshal(query)
	if err != nil {
		return err
	}

	rawMessage := &common.RawMessage{
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_QUERY,
	}

	encodedPublicKey := hex.EncodeToString(hashedPublicKey)
	c.config.Logger.Debug("sending query")
	messageID, err := c.messageProcessor.SendPublic(context.Background(), encodedPublicKey, rawMessage)

	if err != nil {
		return err
	}

	return c.persistence.SavePushNotificationQuery(publicKey, messageID)
}

func (c *Client) GetPushNotificationInfo(publicKey *ecdsa.PublicKey, installationIDs []string) ([]*PushNotificationInfo, error) {
	return c.persistence.GetPushNotificationInfo(publicKey, installationIDs)
}

func (c *Client) listenToPublicKeyQueryTopic(hashedPublicKey []byte) error {
	encodedPublicKey := hex.EncodeToString(hashedPublicKey)
	return c.messageProcessor.JoinPublic(encodedPublicKey)
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
