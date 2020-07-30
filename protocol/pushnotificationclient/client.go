package pushnotificationclient

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"sort"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

// How does sending notifications work?
// 1) Every time a message is scheduled for sending, it will be received on a channel.
//    we keep track on whether we should send a push notification for this message.
// 2) Every time a message is dispatched, we check whether we should send a notification.
//    If so, we query the user info if necessary, check which installations we should be targeting
//    and notify the server if we have information about the user (i.e a token).
//    The logic is complicated by the fact that sometimes messages are batched together (datasync)
//    and the fact that sometimes we send messages to all devices (dh messages).
// 3) The server will notify us if the wrong token is used, in which case a loop will be started that
//    will re-query and re-send the notification, up to a maximum.

// How does registering works?
// We register with the server asynchronously, through a loop, that will try to make sure that
// we have registered with all the servers added, until eventually it gives up.

// A lot of the logic is complicated by the fact that waku/whisper is not req/response, so we just fire a message
// hoping to get a reply at some later stages.

const encryptedPayloadKeyLength = 16
const accessTokenKeyLength = 16
const staleQueryTimeInSeconds = 86400

// maxRegistrationRetries is the maximum number of attempts we do before giving up registering with a server
const maxRegistrationRetries int64 = 12

// maxPushNotificationRetries is the maximum number of attempts before we give up sending a push notification
const maxPushNotificationRetries int64 = 4

// pushNotificationBackoffTime is the step of the exponential backoff
const pushNotificationBackoffTime int64 = 2

// RegistrationBackoffTime is the step of the exponential backoff
const RegistrationBackoffTime int64 = 15

type PushNotificationServer struct {
	PublicKey     *ecdsa.PublicKey `json:"-"`
	Registered    bool             `json:"registered,omitempty"`
	RegisteredAt  int64            `json:"registeredAt,omitempty"`
	LastRetriedAt int64            `json:"lastRetriedAt,omitempty"`
	RetryCount    int64            `json:"retryCount,omitempty"`
	AccessToken   string           `json:"accessToken,omitempty"`
}

func (s *PushNotificationServer) MarshalJSON() ([]byte, error) {
	type ServerAlias PushNotificationServer
	item := struct {
		*ServerAlias
		PublicKeyString string `json:"publicKey"`
	}{
		ServerAlias:     (*ServerAlias)(s),
		PublicKeyString: types.EncodeHex(crypto.FromECDSAPub(s.PublicKey)),
	}

	return json.Marshal(item)
}

type PushNotificationInfo struct {
	AccessToken     string
	InstallationID  string
	PublicKey       *ecdsa.PublicKey
	ServerPublicKey *ecdsa.PublicKey
	RetrievedAt     int64
	Version         uint64
}

type SentNotification struct {
	PublicKey      *ecdsa.PublicKey
	InstallationID string
	LastTriedAt    int64
	RetryCount     int64
	MessageID      []byte
	Success        bool
	Error          protobuf.PushNotificationReport_ErrorType
}

func (s *SentNotification) HashedPublicKey() []byte {
	return common.HashPublicKey(s.PublicKey)
}

type Config struct {
	// Identity is our identity key
	Identity *ecdsa.PrivateKey
	// SendEnabled indicates whether we should be sending push notifications
	SendEnabled bool
	// RemoteNotificationsEnabled is whether we should register with a remote server for push notifications
	RemoteNotificationsEnabled bool

	// AllowyFromContactsOnly indicates whether we should be receiving push notifications
	// only from contacts
	AllowFromContactsOnly bool

	// InstallationID is the installation-id for this device
	InstallationID string

	Logger *zap.Logger
}

type Client struct {
	persistence *Persistence

	config *Config

	// lastPushNotificationRegistration is the latest known push notification version
	lastPushNotificationRegistration *protobuf.PushNotificationRegistration

	// lastContactIDs is the latest contact ids array
	lastContactIDs []*ecdsa.PublicKey

	// AccessToken is the access token that is currently being used
	AccessToken string
	// deviceToken is the device token for this device
	deviceToken string
	// TokenType is the type of token
	tokenType protobuf.PushNotificationRegistration_TokenType
	// APNTopic is the topic of the apn topic for push notification
	apnTopic string

	// randomReader only used for testing so we have deterministic encryption
	reader io.Reader

	//messageProcessor is a message processor used to send and being notified of messages
	messageProcessor *common.MessageProcessor

	// registrationLoopQuitChan is a channel to indicate to the registration loop that should be terminating
	registrationLoopQuitChan chan struct{}

	// resendingLoopQuitChan is a channel to indicate to the send loop that should be terminating
	resendingLoopQuitChan chan struct{}

	quit chan struct{}
}

func New(persistence *Persistence, config *Config, processor *common.MessageProcessor) *Client {
	return &Client{
		quit:             make(chan struct{}),
		config:           config,
		messageProcessor: processor,
		persistence:      persistence,
		reader:           rand.Reader,
	}
}

func (c *Client) Start() error {
	if c.messageProcessor == nil {
		return errors.New("can't start, missing message processor")
	}

	err := c.loadLastPushNotificationRegistration()
	if err != nil {
		return err
	}

	c.subscribeForSentMessages()
	c.subscribeForScheduledMessages()
	c.startRegistrationLoop()
	c.startResendingLoop()

	return nil
}

func (c *Client) Stop() error {
	close(c.quit)
	c.stopRegistrationLoop()
	c.stopResendingLoop()
	return nil
}

// Unregister unregisters from all the servers
func (c *Client) Unregister() error {
	// stop registration loop
	c.stopRegistrationLoop()

	registration := c.buildPushNotificationUnregisterMessage()
	err := c.saveLastPushNotificationRegistration(registration, nil)
	if err != nil {
		return err
	}

	// reset servers
	err = c.resetServers()
	if err != nil {
		return err
	}

	// and asynchronously register
	c.startRegistrationLoop()
	return nil
}

// Registered returns true if we registered with all the servers
func (c *Client) Registered() (bool, error) {
	servers, err := c.persistence.GetServers()
	if err != nil {
		return false, err
	}

	for _, s := range servers {
		if !s.Registered {
			return false, nil
		}
	}

	return true, nil
}

func (c *Client) GetSentNotification(hashedPublicKey []byte, installationID string, messageID []byte) (*SentNotification, error) {
	return c.persistence.GetSentNotification(hashedPublicKey, installationID, messageID)
}

func (c *Client) GetServers() ([]*PushNotificationServer, error) {
	return c.persistence.GetServers()
}

func (c *Client) Reregister(contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) error {
	c.config.Logger.Debug("re-registering")
	if len(c.deviceToken) == 0 {
		c.config.Logger.Info("no device token, not registering")
		return nil
	}

	return c.Register(c.deviceToken, c.apnTopic, c.tokenType, contactIDs, mutedChatIDs)
}

// Register registers with all the servers
func (c *Client) Register(deviceToken, apnTopic string, tokenType protobuf.PushNotificationRegistration_TokenType, contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) error {
	// stop registration loop
	c.stopRegistrationLoop()

	// reset servers
	err := c.resetServers()
	if err != nil {
		return err
	}

	c.deviceToken = deviceToken
	c.apnTopic = apnTopic
	c.tokenType = tokenType

	registration, err := c.buildPushNotificationRegistrationMessage(contactIDs, mutedChatIDs)
	if err != nil {
		return err
	}

	err = c.saveLastPushNotificationRegistration(registration, contactIDs)
	if err != nil {
		return err
	}

	c.startRegistrationLoop()

	return nil
}

// HandlePushNotificationRegistrationResponse should check whether the response was successful or not, retry if necessary otherwise store the result in the database
func (c *Client) HandlePushNotificationRegistrationResponse(publicKey *ecdsa.PublicKey, response protobuf.PushNotificationRegistrationResponse) error {
	c.config.Logger.Debug("received push notification registration response", zap.Any("response", response))

	// Not successful ignore for now
	if !response.Success {
		return errors.New("response was not successful")
	}

	servers, err := c.persistence.GetServersByPublicKey([]*ecdsa.PublicKey{publicKey})
	if err != nil {
		return err
	}

	// we haven't registered with this server
	if len(servers) != 1 {
		return errors.New("not registered with this server, ignoring")
	}

	server := servers[0]
	server.Registered = true
	server.RegisteredAt = time.Now().Unix()

	return c.persistence.UpsertServer(server)
}

// HandlePushNotificationQueryResponse should update the data in the database for a given user
func (c *Client) HandlePushNotificationQueryResponse(serverPublicKey *ecdsa.PublicKey, response protobuf.PushNotificationQueryResponse) error {
	c.config.Logger.Debug("received push notification query response", zap.Any("response", response))
	if len(response.Info) == 0 {
		return errors.New("empty response from the server")
	}

	// get the public key associated with this query
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
		// make sure the public key matches
		if !bytes.Equal(info.PublicKey, common.HashPublicKey(publicKey)) {
			c.config.Logger.Warn("reply for different key, ignoring")
			continue
		}

		accessToken := info.AccessToken

		// the user wants notification from contacts only, try to decrypt the access token to see if we are in their contacts
		if len(accessToken) == 0 && len(info.AllowedKeyList) != 0 {
			accessToken = c.handleAllowedKeyList(publicKey, info.AllowedKeyList)

		}

		// no luck
		if len(accessToken) == 0 {
			c.config.Logger.Debug("not in the allowed key list")
			continue
		}

		// We check the user has allowed this server to store this particular
		// access token, otherwise anyone could reply with a fake token
		// and receive notifications for a user
		if err := c.handleGrant(publicKey, serverPublicKey, info.Grant, accessToken); err != nil {
			c.config.Logger.Warn("grant verification failed, ignoring", zap.Error(err))
			continue
		}

		pushNotificationInfo = append(pushNotificationInfo, &PushNotificationInfo{
			PublicKey:       publicKey,
			ServerPublicKey: serverPublicKey,
			AccessToken:     accessToken,
			InstallationID:  info.InstallationId,
			Version:         info.Version,
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
func (c *Client) HandlePushNotificationResponse(serverKey *ecdsa.PublicKey, response protobuf.PushNotificationResponse) error {
	messageID := response.MessageId
	c.config.Logger.Debug("received response for", zap.Binary("message-id", messageID))
	for _, report := range response.Reports {
		c.config.Logger.Debug("received response", zap.Any("report", report))
		err := c.persistence.UpdateNotificationResponse(messageID, report)
		if err != nil {
			return err
		}
	}

	// Restart resending loop, in case we need to resend some notifications
	c.stopResendingLoop()
	c.startResendingLoop()
	return nil
}

func (c *Client) RemovePushNotificationServer(publicKey *ecdsa.PublicKey) error {
	c.config.Logger.Debug("removing push notification server", zap.Any("public-key", publicKey))
	//TODO: this needs implementing. It requires unregistering from the server and
	// likely invalidate the device token of the user
	return errors.New("not implemented")
}

func (c *Client) AddPushNotificationsServer(publicKey *ecdsa.PublicKey) error {
	c.config.Logger.Debug("adding push notifications server", zap.Any("public-key", publicKey))
	currentServers, err := c.persistence.GetServers()
	if err != nil {
		return err
	}

	for _, server := range currentServers {
		if common.IsPubKeyEqual(server.PublicKey, publicKey) {
			return errors.New("push notification server already added")
		}
	}

	err = c.persistence.UpsertServer(&PushNotificationServer{
		PublicKey: publicKey,
	})
	if err != nil {
		return err
	}

	if c.config.RemoteNotificationsEnabled {
		c.startRegistrationLoop()
	}
	return nil
}

func (c *Client) GetPushNotificationInfo(publicKey *ecdsa.PublicKey, installationIDs []string) ([]*PushNotificationInfo, error) {
	if len(installationIDs) == 0 {
		return c.persistence.GetPushNotificationInfoByPublicKey(publicKey)
	}
	return c.persistence.GetPushNotificationInfo(publicKey, installationIDs)
}

func (c *Client) EnableSending() {
	c.config.SendEnabled = true
}

func (c *Client) DisableSending() {
	c.config.SendEnabled = false
}

func (c *Client) EnablePushNotificationsFromContactsOnly(contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) error {
	c.config.Logger.Debug("enabling push notification from contacts only")
	c.config.AllowFromContactsOnly = true
	if c.lastPushNotificationRegistration != nil {
		c.config.Logger.Debug("re-registering after enabling push notifications from contacts only")
		return c.Register(c.deviceToken, c.apnTopic, c.tokenType, contactIDs, mutedChatIDs)
	}
	return nil
}

func (c *Client) DisablePushNotificationsFromContactsOnly(contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) error {
	c.config.Logger.Debug("disabling push notification from contacts only")
	c.config.AllowFromContactsOnly = false
	if c.lastPushNotificationRegistration != nil {
		c.config.Logger.Debug("re-registering after disabling push notifications from contacts only")
		return c.Register(c.deviceToken, c.apnTopic, c.tokenType, contactIDs, mutedChatIDs)
	}
	return nil
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

// subscribeForSentMessages subscribes for newly sent messages so we can check if we need to send a push notification
func (c *Client) subscribeForSentMessages() {
	go func() {
		c.config.Logger.Debug("subscribing for sent messages")
		subscription := c.messageProcessor.SubscribeToSentMessages()
		for {
			select {
			case m, more := <-subscription:
				if !more {
					c.config.Logger.Debug("no more sent messages, quitting")
					return
				}
				c.config.Logger.Debug("handling message sent")
				if err := c.handleMessageSent(m); err != nil {
					c.config.Logger.Error("failed to handle message", zap.Error(err))
				}
			case <-c.quit:
				return
			}
		}
	}()
}

// subscribeForScheduledMessages subscribes for messages scheduler for dispatch
func (c *Client) subscribeForScheduledMessages() {
	go func() {
		c.config.Logger.Debug("subscribing for scheduled messages")
		subscription := c.messageProcessor.SubscribeToScheduledMessages()
		for {
			select {
			case m, more := <-subscription:
				if !more {
					c.config.Logger.Debug("no more scheduled messages, quitting")
					return
				}
				c.config.Logger.Debug("handling message scheduled")
				if err := c.handleMessageScheduled(m); err != nil {
					c.config.Logger.Error("failed to handle message", zap.Error(err))
				}
			case <-c.quit:
				return
			}
		}
	}()
}

// loadLastPushNotificationRegistration loads from the database the last registration
func (c *Client) loadLastPushNotificationRegistration() error {
	lastRegistration, lastContactIDs, err := c.persistence.GetLastPushNotificationRegistration()
	if err != nil {
		return err
	}
	if lastRegistration == nil {
		lastRegistration = &protobuf.PushNotificationRegistration{}
	}
	c.lastContactIDs = lastContactIDs
	c.lastPushNotificationRegistration = lastRegistration
	c.deviceToken = lastRegistration.DeviceToken
	c.apnTopic = lastRegistration.ApnTopic
	c.tokenType = lastRegistration.TokenType
	return nil
}

func (c *Client) stopRegistrationLoop() {
	// stop old registration loop
	if c.registrationLoopQuitChan != nil {
		close(c.registrationLoopQuitChan)
		c.registrationLoopQuitChan = nil
	}
}

func (c *Client) stopResendingLoop() {
	// stop old registration loop
	if c.resendingLoopQuitChan != nil {
		close(c.resendingLoopQuitChan)
		c.resendingLoopQuitChan = nil
	}
}

func (c *Client) startRegistrationLoop() {
	c.stopRegistrationLoop()
	c.registrationLoopQuitChan = make(chan struct{})
	go func() {
		err := c.registrationLoop()
		if err != nil {
			c.config.Logger.Error("registration loop exited with an error", zap.Error(err))
		}
	}()
}

func (c *Client) startResendingLoop() {
	c.stopResendingLoop()
	c.resendingLoopQuitChan = make(chan struct{})
	go func() {
		err := c.resendingLoop()
		if err != nil {
			c.config.Logger.Error("resending loop exited with an error", zap.Error(err))
		}
	}()
}

// queryNotificationInfo will block and query for the client token, if force is set it
// will ignore the cool off period
func (c *Client) queryNotificationInfo(publicKey *ecdsa.PublicKey, force bool) error {
	c.config.Logger.Debug("retrieving queried at")

	// Check if we queried recently
	queriedAt, err := c.persistence.GetQueriedAt(publicKey)
	if err != nil {
		c.config.Logger.Error("failed to retrieve queried at", zap.Error(err))
		return err
	}
	c.config.Logger.Debug("checking if querying necessary")

	// Naively query again if too much time has passed.
	// Here it might not be necessary
	if force || time.Now().Unix()-queriedAt > staleQueryTimeInSeconds {
		c.config.Logger.Debug("querying info")
		err := c.queryPushNotificationInfo(publicKey)
		if err != nil {
			c.config.Logger.Error("could not query pn info", zap.Error(err))
			return err
		}
		// This is just horrible, but for now will do,
		// the issue is that we don't really know how long it will
		// take to reply, as there might be multiple servers
		// replying to us.
		// The only time we are 100% certain that we can proceed is
		// when we have non-stale info for each device, but
		// most devices are not going to be registered, so we'd still
		// have to wait the maximum amount of time allowed.
		// A better way to handle this is to set a maximum timer of say
		// 3 seconds, but act at a tick every 200ms.
		// That way we still are able to batch multiple push notifications
		// but we don't have to wait every time 3 seconds, which is wasteful
		// This probably will have to be addressed before released
		time.Sleep(3 * time.Second)
	}
	return nil
}

// handleMessageSent is called every time a message is sent. It will check if
// we need to notify on the message, and if so it will try to dispatch a push notification
// messages might be batched, if coming from datasync for example.
func (c *Client) handleMessageSent(sentMessage *common.SentMessage) error {
	c.config.Logger.Debug("sent messages", zap.Any("messageIDs", sentMessage.MessageIDs))

	// Ignore if we are not sending notifications
	if !c.config.SendEnabled {
		c.config.Logger.Debug("send not enabled, ignoring")
		return nil
	}

	publicKey := sentMessage.PublicKey

	// Collect the messageIDs we want to notify on
	var trackedMessageIDs [][]byte

	for _, messageID := range sentMessage.MessageIDs {
		tracked, err := c.persistence.TrackedMessage(messageID)
		if err != nil {
			return err
		}
		if tracked {
			trackedMessageIDs = append(trackedMessageIDs, messageID)
		}
	}

	// Nothing to do
	if len(trackedMessageIDs) == 0 {
		c.config.Logger.Debug("nothing to do for", zap.Any("messageIDs", sentMessage.MessageIDs))
		return nil
	}

	// sendToAllDevices indicates whether the message has been sent using public key encryption only
	// i.e not through the double ratchet. In that case, any device will have received it.
	sendToAllDevices := len(sentMessage.Spec.Installations) == 0

	var installationIDs []string

	anyActionableMessage := sendToAllDevices

	// Check if we should be notifiying those installations
	for _, messageID := range trackedMessageIDs {
		for _, installation := range sentMessage.Spec.Installations {
			installationID := installation.ID
			shouldNotify, err := c.shouldNotifyOn(publicKey, installationID, messageID)
			if err != nil {
				return err
			}
			if shouldNotify {
				anyActionableMessage = true
				installationIDs = append(installationIDs, installation.ID)
			}
		}
	}

	// Is there anything we should be notifying on?
	if !anyActionableMessage {
		c.config.Logger.Debug("no actionable installation IDs")
		return nil
	}

	c.config.Logger.Debug("actionable messages", zap.Any("message-ids", trackedMessageIDs), zap.Any("installation-ids", installationIDs))

	// we send the notifications and return the info of the devices notified
	infos, err := c.sendNotification(publicKey, installationIDs, trackedMessageIDs[0])
	if err != nil {
		return err
	}

	// mark message as sent so we don't notify again
	for _, i := range infos {
		for _, messageID := range trackedMessageIDs {

			c.config.Logger.Debug("marking as sent ", zap.Binary("mid", messageID), zap.String("id", i.InstallationID))
			if err := c.notifiedOn(publicKey, i.InstallationID, messageID); err != nil {
				return err
			}

		}
	}

	return nil
}

// handleMessageScheduled keeps track of the message to make sure we notify on it
func (c *Client) handleMessageScheduled(message *common.RawMessage) error {
	if !message.SendPushNotification {
		return nil
	}
	messageID, err := types.DecodeHex(message.ID)
	if err != nil {
		return err
	}
	return c.persistence.TrackPushNotification(message.LocalChatID, messageID)
}

// shouldNotifyOn check whether we should notify a particular public-key/installation-id/message-id combination
func (c *Client) shouldNotifyOn(publicKey *ecdsa.PublicKey, installationID string, messageID []byte) (bool, error) {
	if len(installationID) == 0 {
		return c.persistence.ShouldSendNotificationToAllInstallationIDs(publicKey, messageID)
	}
	return c.persistence.ShouldSendNotificationFor(publicKey, installationID, messageID)
}

// notifiedOn marks a combination of publickey/installationid/messageID as notified
func (c *Client) notifiedOn(publicKey *ecdsa.PublicKey, installationID string, messageID []byte) error {
	return c.persistence.UpsertSentNotification(&SentNotification{
		PublicKey:      publicKey,
		LastTriedAt:    time.Now().Unix(),
		InstallationID: installationID,
		MessageID:      messageID,
	})
}

func (c *Client) mutedChatIDsHashes(chatIDs []string) [][]byte {
	var mutedChatListHashes [][]byte

	for _, chatID := range chatIDs {
		mutedChatListHashes = append(mutedChatListHashes, common.Shake256([]byte(chatID)))
	}

	return mutedChatListHashes
}

func (c *Client) encryptToken(publicKey *ecdsa.PublicKey, token []byte) ([]byte, error) {
	sharedKey, err := ecies.ImportECDSA(c.config.Identity).GenerateShared(
		ecies.ImportECDSAPublic(publicKey),
		accessTokenKeyLength,
		accessTokenKeyLength,
	)
	if err != nil {
		return nil, err
	}
	encryptedToken, err := encryptAccessToken(token, sharedKey, c.reader)
	if err != nil {
		return nil, err
	}
	return encryptedToken, nil
}

func (c *Client) decryptToken(publicKey *ecdsa.PublicKey, token []byte) ([]byte, error) {
	sharedKey, err := ecies.ImportECDSA(c.config.Identity).GenerateShared(
		ecies.ImportECDSAPublic(publicKey),
		accessTokenKeyLength,
		accessTokenKeyLength,
	)
	if err != nil {
		return nil, err
	}
	decryptedToken, err := common.Decrypt(token, sharedKey)
	if err != nil {
		return nil, err
	}
	return decryptedToken, nil
}

// allowedKeyList builds up a list of encrypted tokens, used for registering with the server
func (c *Client) allowedKeyList(token []byte, contactIDs []*ecdsa.PublicKey) ([][]byte, error) {
	// If we allow everyone, don't set the list
	if !c.config.AllowFromContactsOnly {
		return nil, nil
	}
	var encryptedTokens [][]byte
	for _, publicKey := range contactIDs {
		encryptedToken, err := c.encryptToken(publicKey, token)
		if err != nil {
			return nil, err
		}

		encryptedTokens = append(encryptedTokens, encryptedToken)

	}
	return encryptedTokens, nil
}

// getToken checks if we need to refresh the token
// and return a new one in that case. A token is refreshed only if it's not set
// or if a contact has been removed
func (c *Client) getToken(contactIDs []*ecdsa.PublicKey) string {
	if c.lastPushNotificationRegistration == nil || len(c.lastPushNotificationRegistration.AccessToken) == 0 || c.shouldRefreshToken(c.lastContactIDs, contactIDs, c.lastPushNotificationRegistration.AllowFromContactsOnly, c.config.AllowFromContactsOnly) {
		c.config.Logger.Info("refreshing access token")
		return uuid.New().String()
	}
	return c.lastPushNotificationRegistration.AccessToken
}

func (c *Client) getVersion() uint64 {
	if c.lastPushNotificationRegistration == nil {
		return 1
	}
	return c.lastPushNotificationRegistration.Version + 1
}

func (c *Client) buildPushNotificationRegistrationMessage(contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) (*protobuf.PushNotificationRegistration, error) {
	token := c.getToken(contactIDs)
	allowedKeyList, err := c.allowedKeyList([]byte(token), contactIDs)
	if err != nil {
		return nil, err
	}

	options := &protobuf.PushNotificationRegistration{
		AccessToken:           token,
		TokenType:             c.tokenType,
		ApnTopic:              c.apnTopic,
		Version:               c.getVersion(),
		InstallationId:        c.config.InstallationID,
		DeviceToken:           c.deviceToken,
		AllowFromContactsOnly: c.config.AllowFromContactsOnly,
		Enabled:               c.config.RemoteNotificationsEnabled,
		BlockedChatList:       c.mutedChatIDsHashes(mutedChatIDs),
		AllowedKeyList:        allowedKeyList,
	}
	return options, nil
}

func (c *Client) buildPushNotificationUnregisterMessage() *protobuf.PushNotificationRegistration {
	options := &protobuf.PushNotificationRegistration{
		Version:        c.getVersion(),
		InstallationId: c.config.InstallationID,
		Unregister:     true,
	}
	return options
}

// shouldRefreshToken tells us whether we should create a new token,
// that's only necessary when a contact is removed
// or allowFromContactsOnly is enabled.
// In both cases we want to invalidate any existing token
func (c *Client) shouldRefreshToken(oldContactIDs, newContactIDs []*ecdsa.PublicKey, oldAllowFromContactsOnly, newAllowFromContactsOnly bool) bool {

	// Check if allowFromContactsOnly has just been enabled
	if !oldAllowFromContactsOnly && newAllowFromContactsOnly {
		return true
	}

	newContactIDsMap := make(map[string]bool)
	for _, pk := range newContactIDs {
		newContactIDsMap[types.EncodeHex(crypto.FromECDSAPub(pk))] = true
	}

	for _, pk := range oldContactIDs {
		if ok := newContactIDsMap[types.EncodeHex(crypto.FromECDSAPub(pk))]; !ok {
			return true
		}

	}
	return false
}

func nextServerRetry(server *PushNotificationServer) int64 {
	return server.LastRetriedAt + RegistrationBackoffTime*server.RetryCount*int64(math.Exp2(float64(server.RetryCount)))
}

func nextPushNotificationRetry(pn *SentNotification) int64 {
	return pn.LastTriedAt + pushNotificationBackoffTime*pn.RetryCount*int64(math.Exp2(float64(pn.RetryCount)))
}

// We calculate if it's too early to retry, by exponentially backing off
func shouldRetryRegisteringWithServer(server *PushNotificationServer) bool {
	return time.Now().Unix() >= nextServerRetry(server)
}

// We calculate if it's too early to retry, by exponentially backing off
func shouldRetryPushNotification(pn *SentNotification) bool {
	if pn.RetryCount > maxPushNotificationRetries {
		return false
	}
	return time.Now().Unix() >= nextPushNotificationRetry(pn)
}

func (c *Client) resetServers() error {
	servers, err := c.persistence.GetServers()
	if err != nil {
		return err
	}
	for _, server := range servers {

		// Reset server registration data
		server.Registered = false
		server.RegisteredAt = 0
		server.RetryCount = 0
		server.LastRetriedAt = time.Now().Unix()
		server.AccessToken = ""

		if err := c.persistence.UpsertServer(server); err != nil {
			return err
		}
	}

	return nil
}

// registerWithServer will register with a push notification server. This will use
// the user identity key for dispatching, as the content is in any case signed, so identity needs to be revealed.
func (c *Client) registerWithServer(registration *protobuf.PushNotificationRegistration, server *PushNotificationServer) error {
	// reset server registration data
	server.Registered = false
	server.RegisteredAt = 0
	server.RetryCount++
	server.LastRetriedAt = time.Now().Unix()
	server.AccessToken = registration.AccessToken

	// save
	if err := c.persistence.UpsertServer(server); err != nil {
		return err
	}

	// build grant for this specific server
	grant, err := c.buildGrantSignature(server.PublicKey, registration.AccessToken)
	if err != nil {
		c.config.Logger.Error("failed to build grant", zap.Error(err))
		return err
	}

	registration.Grant = grant

	// marshal message
	marshaledRegistration, err := proto.Marshal(registration)
	if err != nil {
		return err
	}

	// encrypt and dispatch message
	encryptedRegistration, err := c.encryptRegistration(server.PublicKey, marshaledRegistration)
	if err != nil {
		return err
	}
	rawMessage := common.RawMessage{
		Payload:     encryptedRegistration,
		MessageType: protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REGISTRATION,
	}

	_, err = c.messageProcessor.SendPrivate(context.Background(), server.PublicKey, rawMessage)

	if err != nil {
		return err
	}
	return nil
}

// sendNotification sends an actual notification to the push notification server.
// the notification is sent using an ephemeral key to shield the real identity of the sender
func (c *Client) sendNotification(publicKey *ecdsa.PublicKey, installationIDs []string, messageID []byte) ([]*PushNotificationInfo, error) {
	// get latest push notification infos
	err := c.queryNotificationInfo(publicKey, false)
	if err != nil {
		return nil, err
	}
	c.config.Logger.Debug("queried info")

	// retrieve info from the database
	info, err := c.GetPushNotificationInfo(publicKey, installationIDs)
	if err != nil {
		c.config.Logger.Error("could not get pn info", zap.Error(err))
		return nil, err
	}

	// naively dispatch to the first server for now
	// push notifications are only retried for now if a WRONG_TOKEN response is returned.
	// we should also retry if no response at all is received after a timeout.
	// also we send a single notification for multiple message ids, need to check with UI what's the desired behavior

	// sort by server so we tend to hit the same one
	sort.Slice(info, func(i, j int) bool {
		return info[i].ServerPublicKey.X.Cmp(info[j].ServerPublicKey.X) <= 0
	})

	installationIDsMap := make(map[string]bool)

	// one info per installation id, grouped by server
	actionableInfos := make(map[string][]*PushNotificationInfo)
	for _, i := range info {

		if !installationIDsMap[i.InstallationID] {
			serverKey := hex.EncodeToString(crypto.CompressPubkey(i.ServerPublicKey))
			actionableInfos[serverKey] = append(actionableInfos[serverKey], i)
			installationIDsMap[i.InstallationID] = true
		}

	}

	c.config.Logger.Debug("actionable info", zap.Int("count", len(actionableInfos)))

	// add ephemeral key and listen to it
	ephemeralKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	_, err = c.messageProcessor.AddEphemeralKey(ephemeralKey)
	if err != nil {
		return nil, err
	}

	var actionedInfo []*PushNotificationInfo
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
			MessageId: messageID,
			Requests:  pushNotifications,
		}
		serverPublicKey := infos[0].ServerPublicKey

		payload, err := proto.Marshal(request)
		if err != nil {
			return nil, err
		}

		rawMessage := common.RawMessage{
			Payload: payload,
			Sender:  ephemeralKey,
			// we skip encryption as we don't want to save any key material
			// for an ephemeral key, no need to use pfs as these are throw away keys
			SkipEncryption: true,
			MessageType:    protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REQUEST,
		}

		_, err = c.messageProcessor.SendPrivate(context.Background(), serverPublicKey, rawMessage)

		if err != nil {
			return nil, err
		}
		actionedInfo = append(actionedInfo, infos...)
	}
	return actionedInfo, nil
}

func (c *Client) resendNotification(pn *SentNotification) error {
	c.config.Logger.Debug("resending notification")
	pn.RetryCount++
	pn.LastTriedAt = time.Now().Unix()
	err := c.persistence.UpsertSentNotification(pn)
	if err != nil {
		c.config.Logger.Error("failed to upsert notification", zap.Error(err))
		return err
	}

	// re-fetch push notification info
	err = c.queryNotificationInfo(pn.PublicKey, true)
	if err != nil {
		c.config.Logger.Error("failed to query notification info", zap.Error(err))
		return err
	}

	if err != nil {
		c.config.Logger.Error("could not get pn info", zap.Error(err))
		return err
	}

	_, err = c.sendNotification(pn.PublicKey, []string{pn.InstallationID}, pn.MessageID)
	return err
}

// resendingLoop is a loop that is running when push notifications need to be resent, it only runs when needed, it will quit if no work is necessary.
func (c *Client) resendingLoop() error {
	for {
		c.config.Logger.Debug("running resending loop")
		var lowestNextRetry int64

		// fetch retriable notifications
		retriableNotifications, err := c.persistence.GetRetriablePushNotifications()
		if err != nil {
			c.config.Logger.Error("failed retrieving notifications, quitting resending loop", zap.Error(err))
			return err
		}

		if len(retriableNotifications) == 0 {
			c.config.Logger.Debug("no retriable notifications, quitting")
			return nil
		}

		for _, pn := range retriableNotifications {

			// check if we should retry the notification
			if shouldRetryPushNotification(pn) {
				c.config.Logger.Debug("retrying pn")
				err := c.resendNotification(pn)
				if err != nil {
					return err
				}
			}
			// set the lowest next retry if necessary
			nextRetry := nextPushNotificationRetry(pn)
			if lowestNextRetry == 0 || nextRetry < lowestNextRetry {
				lowestNextRetry = nextRetry
			}
		}

		nextRetry := lowestNextRetry - time.Now().Unix()
		// how long should we sleep for?
		waitFor := time.Duration(nextRetry)
		select {

		case <-time.After(waitFor * time.Second):
		case <-c.resendingLoopQuitChan:
			return nil
		}
	}
}

// registrationLoop is a loop that is running when we need to register with a push notification server, it only runs when needed, it will quit if no work is necessary.
func (c *Client) registrationLoop() error {
	for {
		c.config.Logger.Debug("running registration loop")
		servers, err := c.persistence.GetServers()
		if err != nil {
			c.config.Logger.Error("failed retrieving servers, quitting registration loop", zap.Error(err))
			return err
		}
		if len(servers) == 0 {
			c.config.Logger.Debug("nothing to do, quitting registration loop")
			return nil
		}

		var nonRegisteredServers []*PushNotificationServer
		for _, server := range servers {
			if !server.Registered && server.RetryCount < maxRegistrationRetries {
				nonRegisteredServers = append(nonRegisteredServers, server)
			}
		}

		if len(nonRegisteredServers) == 0 {
			c.config.Logger.Debug("registered with all servers, quitting registration loop")
			return nil
		}

		c.config.Logger.Debug("Trying to register with", zap.Int("servers", len(nonRegisteredServers)))

		var lowestNextRetry int64

		for _, server := range nonRegisteredServers {
			if shouldRetryRegisteringWithServer(server) {
				c.config.Logger.Debug("registering with server", zap.Any("server", server))
				err := c.registerWithServer(c.lastPushNotificationRegistration, server)
				if err != nil {
					return err
				}
			}
			nextRetry := nextServerRetry(server)
			if lowestNextRetry == 0 || nextRetry < lowestNextRetry {
				lowestNextRetry = nextRetry
			}
		}

		nextRetry := lowestNextRetry - time.Now().Unix()
		waitFor := time.Duration(nextRetry)
		c.config.Logger.Debug("Waiting for", zap.Any("wait for", waitFor))
		select {

		case <-time.After(waitFor * time.Second):
		case <-c.registrationLoopQuitChan:
			return nil
		}
	}
}

func (c *Client) saveLastPushNotificationRegistration(registration *protobuf.PushNotificationRegistration, contactIDs []*ecdsa.PublicKey) error {
	// stop registration loop
	c.stopRegistrationLoop()

	err := c.persistence.SaveLastPushNotificationRegistration(registration, contactIDs)
	if err != nil {
		return err
	}
	c.lastPushNotificationRegistration = registration
	c.lastContactIDs = contactIDs

	c.startRegistrationLoop()
	return nil
}

// buildGrantSignatureMaterial builds a grant for a specific server.
// We use 3 components:
// 1) The client public key. Not sure this applies to our signature scheme, but best to be conservative. https://crypto.stackexchange.com/questions/15538/given-a-message-and-signature-find-a-public-key-that-makes-the-signature-valid
// 2) The server public key
// 3) The access token
// By verifying this signature, a client can trust the server was instructed to store this access token.

func (c *Client) buildGrantSignatureMaterial(clientPublicKey *ecdsa.PublicKey, serverPublicKey *ecdsa.PublicKey, accessToken string) []byte {
	var signatureMaterial []byte
	signatureMaterial = append(signatureMaterial, crypto.CompressPubkey(clientPublicKey)...)
	signatureMaterial = append(signatureMaterial, crypto.CompressPubkey(serverPublicKey)...)
	signatureMaterial = append(signatureMaterial, []byte(accessToken)...)
	return crypto.Keccak256(signatureMaterial)
}

func (c *Client) buildGrantSignature(serverPublicKey *ecdsa.PublicKey, accessToken string) ([]byte, error) {
	signatureMaterial := c.buildGrantSignatureMaterial(&c.config.Identity.PublicKey, serverPublicKey, accessToken)
	return crypto.Sign(signatureMaterial, c.config.Identity)
}

func (c *Client) handleGrant(clientPublicKey *ecdsa.PublicKey, serverPublicKey *ecdsa.PublicKey, grant []byte, accessToken string) error {
	signatureMaterial := c.buildGrantSignatureMaterial(clientPublicKey, serverPublicKey, accessToken)
	extractedPublicKey, err := crypto.SigToPub(signatureMaterial, grant)
	if err != nil {
		return err
	}

	if !common.IsPubKeyEqual(clientPublicKey, extractedPublicKey) {
		return errors.New("invalid grant")
	}
	return nil
}

// handleAllowedKeyList will try to decrypt a token from the list, to see if we are allowed to send push notification to a given user
func (c *Client) handleAllowedKeyList(publicKey *ecdsa.PublicKey, allowedKeyList [][]byte) string {
	c.config.Logger.Debug("handling allowed key list")
	for _, encryptedToken := range allowedKeyList {
		token, err := c.decryptToken(publicKey, encryptedToken)
		if err != nil {
			c.config.Logger.Warn("could not decrypt token", zap.Error(err))
			continue
		}
		c.config.Logger.Debug("decrypted token")
		return string(token)
	}
	return ""
}

// queryPushNotificationInfo sends a message to any server who has the given user registered.
// it uses an ephemeral key so the identity of the client querying is not disclosed
func (c *Client) queryPushNotificationInfo(publicKey *ecdsa.PublicKey) error {
	hashedPublicKey := common.HashPublicKey(publicKey)
	query := &protobuf.PushNotificationQuery{
		PublicKeys: [][]byte{hashedPublicKey},
	}
	encodedMessage, err := proto.Marshal(query)
	if err != nil {
		return err
	}

	ephemeralKey, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload: encodedMessage,
		Sender:  ephemeralKey,
		// we don't want to wrap in an encryption layer message
		SkipEncryption: true,
		MessageType:    protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_QUERY,
	}

	_, err = c.messageProcessor.AddEphemeralKey(ephemeralKey)
	if err != nil {
		return err
	}

	// this is the topic of message
	encodedPublicKey := hex.EncodeToString(hashedPublicKey)
	messageID, err := c.messageProcessor.SendPublic(context.Background(), encodedPublicKey, rawMessage)

	if err != nil {
		return err
	}

	return c.persistence.SavePushNotificationQuery(publicKey, messageID)
}
