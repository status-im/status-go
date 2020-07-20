package push_notification_client

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

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"go.uber.org/zap"
)

const encryptedPayloadKeyLength = 16
const accessTokenKeyLength = 16
const staleQueryTimeInSeconds = 86400

// maxRetries is the maximum number of attempts we do before giving up registering with a server
const maxRetries int64 = 12

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

type Config struct {
	// Identity is our identity key
	Identity *ecdsa.PrivateKey
	// SendEnabled indicates whether we should be sending push notifications
	SendEnabled bool
	// RemoteNotificationsEnabled is whether we should register with a remote server for push notifications
	RemoteNotificationsEnabled bool

	// allowyFromContactsOnly indicates whether we should be receiving push notifications
	// only from contacts
	allowFromContactsOnly bool

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

	// lastPushNotificationRegistration is the latest known push notification version
	lastPushNotificationRegistration *protobuf.PushNotificationRegistration

	// lastContactIDs is the latest contact ids array
	lastContactIDs []*ecdsa.PublicKey

	// AccessToken is the access token that is currently being used
	AccessToken string
	// deviceToken is the device token for this device
	deviceToken string

	// randomReader only used for testing so we have deterministic encryption
	reader io.Reader

	//messageProcessor is a message processor used to send and being notified of messages
	messageProcessor *common.MessageProcessor

	// registrationLoopQuitChan is a channel to indicate to the registration loop that should be terminating
	registrationLoopQuitChan chan struct{}
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

func (c *Client) subscribeForSentMessages() {
	go func() {
		subscription := c.messageProcessor.Subscribe()
		for {
			select {
			case m, more := <-subscription:
				if !more {
					c.config.Logger.Info("no more")
					return
				}
				c.config.Logger.Info("handling message sent")
				if err := c.HandleMessageSent(m); err != nil {
					c.config.Logger.Error("failed to handle message", zap.Error(err))
				}
			case <-c.quit:
				return
			}
		}
	}()

}

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
	c.deviceToken = lastRegistration.Token
	return nil

}
func (c *Client) stopRegistrationLoop() {
	// stop old registration loop
	if c.registrationLoopQuitChan != nil {
		close(c.registrationLoopQuitChan)
		c.registrationLoopQuitChan = nil
	}
}

func (c *Client) startRegistrationLoop() {
	c.stopRegistrationLoop()
	c.registrationLoopQuitChan = make(chan struct{})
	go c.registrationLoop()
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
	c.startRegistrationLoop()

	return nil
}

func (c *Client) Stop() error {
	close(c.quit)
	if c.registrationLoopQuitChan != nil {
		close(c.registrationLoopQuitChan)
	}
	return nil
}

func (c *Client) HandleMessageSent(sentMessage *common.SentMessage) error {
	c.config.Logger.Info("sent message", zap.Any("sent message", sentMessage))
	if !c.config.SendEnabled {
		c.config.Logger.Info("send not enabled, ignoring")
		return nil
	}
	publicKey := sentMessage.PublicKey
	// Check we track this messages fist
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
		return nil
	}

	sendToAllDevices := len(sentMessage.Spec.Installations) == 0

	var installationIDs []string

	anyActionableMessage := sendToAllDevices
	c.config.Logger.Info("send to all devices", zap.Bool("send to all", sendToAllDevices))

	// Collect installationIDs
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
		c.config.Logger.Info("no actionable installation IDs")
		return nil
	}

	c.config.Logger.Info("actionable messages", zap.Any("message-ids", trackedMessageIDs), zap.Any("installation-ids", installationIDs))

	// Check if we queried recently
	queriedAt, err := c.persistence.GetQueriedAt(publicKey)
	if err != nil {
		return err
	}

	// Naively query again if too much time has passed.
	// Here it might not be necessary
	if time.Now().Unix()-queriedAt > staleQueryTimeInSeconds {
		c.config.Logger.Info("querying info")
		err := c.QueryPushNotificationInfo(publicKey)
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
		// have to wait teh maximum amount of time allowed.
		time.Sleep(3 * time.Second)

	}

	c.config.Logger.Info("queried info")
	// Retrieve infos
	info, err := c.GetPushNotificationInfo(publicKey, installationIDs)
	if err != nil {
		c.config.Logger.Error("could not get pn info", zap.Error(err))
		return err
	}

	// Naively dispatch to the first server for now
	// This wait for an acknowledgement and try a different server after a timeout
	// Also we sent a single notification for multiple message ids, need to check with UI what's the desired behavior

	// Sort by server so we tend to hit the same one
	sort.Slice(info, func(i, j int) bool {
		return info[i].ServerPublicKey.X.Cmp(info[j].ServerPublicKey.X) <= 0
	})

	c.config.Logger.Info("retrieved info")

	installationIDsMap := make(map[string]bool)
	// One info per installation id, grouped by server
	actionableInfos := make(map[string][]*PushNotificationInfo)
	for _, i := range info {

		c.config.Logger.Info("queried info", zap.String("id", i.InstallationID))
		if !installationIDsMap[i.InstallationID] {
			serverKey := hex.EncodeToString(crypto.CompressPubkey(i.ServerPublicKey))
			actionableInfos[serverKey] = append(actionableInfos[serverKey], i)
			installationIDsMap[i.InstallationID] = true
		}

	}

	c.config.Logger.Info("actionable info", zap.Int("count", len(actionableInfos)))

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
			MessageId: trackedMessageIDs[0],
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

		// Mark message as sent, this is at-most-once semantic
		// for all messageIDs
		for _, i := range infos {
			for _, messageID := range trackedMessageIDs {

				c.config.Logger.Info("marking as sent ", zap.Binary("mid", messageID), zap.String("id", i.InstallationID))
				if err := c.notifiedOn(publicKey, i.InstallationID, messageID); err != nil {
					return err
				}

			}
		}

	}

	return nil
}

// NotifyOnMessageID keeps track of the message to make sure we notify on it
func (c *Client) NotifyOnMessageID(chatID string, messageID []byte) error {
	return c.persistence.TrackPushNotification(chatID, messageID)
}

func (c *Client) shouldNotifyOn(publicKey *ecdsa.PublicKey, installationID string, messageID []byte) (bool, error) {
	if len(installationID) == 0 {
		return c.persistence.ShouldSendNotificationToAllInstallationIDs(publicKey, messageID)
	} else {
		return c.persistence.ShouldSendNotificationFor(publicKey, installationID, messageID)
	}
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

func (p *Client) decryptToken(publicKey *ecdsa.PublicKey, token []byte) ([]byte, error) {
	sharedKey, err := ecies.ImportECDSA(p.config.Identity).GenerateShared(
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

func (c *Client) allowedUserList(token []byte, contactIDs []*ecdsa.PublicKey) ([][]byte, error) {
	// If we allow everyone, don't set the list
	if !c.config.allowFromContactsOnly {
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
// and return a new one in that case
func (c *Client) getToken(contactIDs []*ecdsa.PublicKey) string {
	if c.lastPushNotificationRegistration == nil || len(c.lastPushNotificationRegistration.AccessToken) == 0 || c.shouldRefreshToken(c.lastContactIDs, contactIDs) {
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
	allowedUserList, err := c.allowedUserList([]byte(token), contactIDs)
	if err != nil {
		return nil, err
	}

	options := &protobuf.PushNotificationRegistration{
		AccessToken:           token,
		TokenType:             c.config.TokenType,
		Version:               c.getVersion(),
		InstallationId:        c.config.InstallationID,
		Token:                 c.deviceToken,
		AllowFromContactsOnly: c.config.allowFromContactsOnly,
		Enabled:               c.config.RemoteNotificationsEnabled,
		BlockedChatList:       c.mutedChatIDsHashes(mutedChatIDs),
		AllowedUserList:       allowedUserList,
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

// shouldRefreshToken tells us whether we should pull a new token, that's only necessary when a contact is removed
func (c *Client) shouldRefreshToken(oldContactIDs, newContactIDs []*ecdsa.PublicKey) bool {
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

// We calculate if it's too early to retry, by exponentially backing off
func shouldRetryRegisteringWithServer(server *PushNotificationServer) bool {
	return time.Now().Unix() > nextServerRetry(server)
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
		server.RetryCount += 1
		server.LastRetriedAt = time.Now().Unix()
		server.AccessToken = ""

		if err := c.persistence.UpsertServer(server); err != nil {
			return err
		}

	}

	return nil
}

func (c *Client) registerWithServer(registration *protobuf.PushNotificationRegistration, server *PushNotificationServer) error {
	// Reset server registration data
	server.Registered = false
	server.RegisteredAt = 0
	server.RetryCount += 1
	server.LastRetriedAt = time.Now().Unix()
	server.AccessToken = registration.AccessToken

	if err := c.persistence.UpsertServer(server); err != nil {
		return err
	}

	grant, err := c.buildGrantSignature(server.PublicKey, registration.AccessToken)
	if err != nil {
		c.config.Logger.Error("failed to build grant", zap.Error(err))
		return err
	}

	registration.Grant = grant

	marshaledRegistration, err := proto.Marshal(registration)
	if err != nil {
		return err
	}

	// Dispatch message
	encryptedRegistration, err := c.encryptRegistration(server.PublicKey, marshaledRegistration)
	if err != nil {
		return err
	}
	rawMessage := &common.RawMessage{
		Payload:     encryptedRegistration,
		MessageType: protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REGISTRATION,
	}

	_, err = c.messageProcessor.SendPrivate(context.Background(), server.PublicKey, rawMessage)

	if err != nil {
		return err
	}
	return nil
}

func (c *Client) registrationLoop() error {
	for {
		c.config.Logger.Info("running registration loop")
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
			if !server.Registered {
				nonRegisteredServers = append(nonRegisteredServers, server)
			}
		}
		if len(nonRegisteredServers) == 0 {
			c.config.Logger.Debug("registered with all servers, quitting registration loop")
			return nil
		}

		c.config.Logger.Info("Trying to register with", zap.Int("servers", len(nonRegisteredServers)))

		var lowestNextRetry int64

		for _, server := range nonRegisteredServers {
			nR := nextServerRetry(server)
			c.config.Logger.Info("Next retry", zap.Int64("now", time.Now().Unix()), zap.Int64("next", nR))
			if shouldRetryRegisteringWithServer(server) {
				c.config.Logger.Info("registering with server", zap.Any("server", server))
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
		select {

		case <-time.After(waitFor * time.Second):
		case <-c.registrationLoopQuitChan:
			return nil

		}
	}
}

func (c *Client) Unregister() error {
	// stop registration loop
	c.stopRegistrationLoop()

	registration := c.buildPushNotificationUnregisterMessage()
	err := c.SaveLastPushNotificationRegistration(registration, nil)
	if err != nil {
		return err
	}
	// Reset servers
	err = c.resetServers()
	if err != nil {
		return err
	}

	// and asynchronously register
	c.startRegistrationLoop()
	return nil
}

func (c *Client) SaveLastPushNotificationRegistration(registration *protobuf.PushNotificationRegistration, contactIDs []*ecdsa.PublicKey) error {
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

func (c *Client) GetServers() ([]*PushNotificationServer, error) {
	return c.persistence.GetServers()
}

func (c *Client) Register(deviceToken string, contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) error {
	// stop registration loop
	c.stopRegistrationLoop()

	c.deviceToken = deviceToken

	registration, err := c.buildPushNotificationRegistrationMessage(contactIDs, mutedChatIDs)
	if err != nil {
		return err
	}

	err = c.SaveLastPushNotificationRegistration(registration, contactIDs)
	if err != nil {
		return err
	}

	c.startRegistrationLoop()

	return nil
}

// HandlePushNotificationRegistrationResponse should check whether the response was successful or not, retry if necessary otherwise store the result in the database
func (c *Client) HandlePushNotificationRegistrationResponse(publicKey *ecdsa.PublicKey, response protobuf.PushNotificationRegistrationResponse) error {
	c.config.Logger.Info("received push notification registration response", zap.Any("response", response))
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

func (c *Client) handleAllowedUserList(publicKey *ecdsa.PublicKey, allowedUserList [][]byte) string {
	c.config.Logger.Info("handling allowed user list")
	for _, encryptedToken := range allowedUserList {
		token, err := c.decryptToken(publicKey, encryptedToken)
		if err != nil {
			c.config.Logger.Warn("could not decrypt token", zap.Error(err))
			continue
		}
		c.config.Logger.Info("decrypted token")
		return string(token)
	}
	return ""
}

// HandlePushNotificationQueryResponse should update the data in the database for a given user
func (c *Client) HandlePushNotificationQueryResponse(serverPublicKey *ecdsa.PublicKey, response protobuf.PushNotificationQueryResponse) error {

	c.config.Logger.Info("received push notification query response", zap.Any("response", response))
	if len(response.Info) == 0 {
		return errors.New("empty response from the server")
	}

	publicKey, err := c.persistence.GetQueryPublicKey(response.MessageId)
	if err != nil {
		return err
	}
	if publicKey == nil {
		c.config.Logger.Info("query not found")
		return nil
	}
	var pushNotificationInfo []*PushNotificationInfo
	for _, info := range response.Info {
		if bytes.Compare(info.PublicKey, common.HashPublicKey(publicKey)) != 0 {
			c.config.Logger.Warn("reply for different key, ignoring")
			continue
		}

		accessToken := info.AccessToken

		if len(accessToken) == 0 && len(info.AllowedUserList) != 0 {
			accessToken = c.handleAllowedUserList(publicKey, info.AllowedUserList)

		}

		if len(accessToken) == 0 {
			c.config.Logger.Info("not in the allowed users list")
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
func (p *Client) HandlePushNotificationResponse(serverKey *ecdsa.PublicKey, response protobuf.PushNotificationResponse) error {
	return nil
}

func (c *Client) RemovePushNotificationServer(publicKey *ecdsa.PublicKey) error {
	c.config.Logger.Info("removing push notification server", zap.Any("public-key", publicKey))
	//TODO: this needs implementing. It requires unregistering from the server and
	// likely invalidate the device token of the user
	return errors.New("not implemented")
}

func (c *Client) AddPushNotificationServer(publicKey *ecdsa.PublicKey) error {
	c.config.Logger.Info("adding push notification server", zap.Any("public-key", publicKey))
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
	c.config.Logger.Info("sending query")
	messageID, err := c.messageProcessor.SendPublic(context.Background(), encodedPublicKey, rawMessage)

	if err != nil {
		return err
	}

	return c.persistence.SavePushNotificationQuery(publicKey, messageID)
}

func (c *Client) GetPushNotificationInfo(publicKey *ecdsa.PublicKey, installationIDs []string) ([]*PushNotificationInfo, error) {
	if len(installationIDs) == 0 {
		return c.persistence.GetPushNotificationInfoByPublicKey(publicKey)
	} else {
		return c.persistence.GetPushNotificationInfo(publicKey, installationIDs)
	}
}

func (c *Client) EnableSending() {
	c.config.SendEnabled = true
}

func (c *Client) DisableSending() {
	c.config.SendEnabled = false
}

func (c *Client) EnablePushNotificationsFromContactsOnly(contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) error {
	c.config.allowFromContactsOnly = true
	if c.lastPushNotificationRegistration != nil {
		return c.Register(c.deviceToken, contactIDs, mutedChatIDs)
	}
	return nil
}

func (c *Client) DisablePushNotificationsFromContactsOnly(contactIDs []*ecdsa.PublicKey, mutedChatIDs []string) error {
	c.config.allowFromContactsOnly = false
	if c.lastPushNotificationRegistration != nil {
		return c.Register(c.deviceToken, contactIDs, mutedChatIDs)
	}
	return nil
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
