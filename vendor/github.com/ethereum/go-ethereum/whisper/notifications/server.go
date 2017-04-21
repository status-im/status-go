package notifications

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/params"
)

const (
	topicSendNotification      = "SEND_NOTIFICATION"
	topicNewChatSession        = "NEW_CHAT_SESSION"
	topicAckNewChatSession     = "ACK_NEW_CHAT_SESSION"
	topicNewDeviceRegistration = "NEW_DEVICE_REGISTRATION"
	topicAckDeviceRegistration = "ACK_DEVICE_REGISTRATION"
)

var (
	ErrServiceInitError = errors.New("notification service has not been properly initialized")
)

// NotificationServer service capable of handling Push Notifications
type NotificationServer struct {
	whisper *whisper.Whisper
	config  *params.WhisperConfig

	nodeID      string            // proposed server will feature this ID
	discovery   *discoveryService // discovery service handles client/server negotiation, when server is selected
	protocolKey *ecdsa.PrivateKey // private key of service, used to encode handshake communication

	clientSessions   map[string]*ClientSession
	clientSessionsMu sync.RWMutex

	chatSessions   map[string]*ChatSession
	chatSessionsMu sync.RWMutex

	deviceSubscriptions   map[string]*DeviceSubscription
	deviceSubscriptionsMu sync.RWMutex

	firebaseProvider NotificationDeliveryProvider

	quit chan struct{}
}

// ClientSession abstracts notification client, which expects notifications whenever
// some envelope can be decoded with session key (key hash is compared for optimization)
type ClientSession struct {
	ClientKey      string      // public key uniquely identifying a client
	SessionKey     []byte      // actual symkey used for client - server communication
	SessionKeyHash common.Hash // The Keccak256Hash of the symmetric key, which is shared between server/client
}

// ChatSession abstracts chat session, which some previously registered client can create.
// ChatSession is used by client for sharing common secret, allowing others to register
// themselves and eventually to trigger notifications.
type ChatSession struct {
	ParentKey      string      // public key uniquely identifying a client session used to create a chat session
	ChatKey        string      // ID that uniquely identifies a chat session
	SessionKey     []byte      // actual symkey used for client - server communication
	SessionKeyHash common.Hash // The Keccak256Hash of the symmetric key, which is shared between server/client
}

// DeviceSubscription stores enough information about a device (or group of devices),
// so that Notification Server can trigger notification on that device(s)
type DeviceSubscription struct {
	DeviceID           string           // ID that will be used as destination
	ChatSessionKeyHash common.Hash      // The Keccak256Hash of the symmetric key, which is shared between server/client
	PubKey             *ecdsa.PublicKey // public key of subscriber (to filter out when notification is triggered)
}

// Init used for service initialization, making sure it is safe to call Start()
func (s *NotificationServer) Init(whisperService *whisper.Whisper, whisperConfig *params.WhisperConfig) {
	s.whisper = whisperService
	s.config = whisperConfig

	s.discovery = NewDiscoveryService(s)
	s.clientSessions = make(map[string]*ClientSession)
	s.chatSessions = make(map[string]*ChatSession)
	s.deviceSubscriptions = make(map[string]*DeviceSubscription)
	s.quit = make(chan struct{})

	// setup providers (FCM only, for now)
	s.firebaseProvider = NewFirebaseProvider(whisperConfig.FirebaseConfig)
}

// Start begins notification loop, in a separate go routine
func (s *NotificationServer) Start(stack *p2p.Server) error {
	if s.whisper == nil {
		return ErrServiceInitError
	}

	// configure nodeID
	if stack != nil {
		if nodeInfo := stack.NodeInfo(); nodeInfo != nil {
			s.nodeID = nodeInfo.ID
		}
	}

	// configure keys
	identity, err := s.config.ReadIdentityFile()
	if err != nil {
		return err
	}
	s.whisper.AddIdentity(identity)
	s.protocolKey = identity
	glog.V(logger.Info).Infoln("protocol pubkey: ", common.ToHex(crypto.FromECDSAPub(&s.protocolKey.PublicKey)))

	// start discovery protocol
	s.discovery.Start()

	glog.V(logger.Info).Infoln("Whisper Notification Server started")
	return nil
}

// Stop handles stopping the running notification loop, and all related resources
func (s *NotificationServer) Stop() error {
	close(s.quit)

	if s.whisper == nil {
		return ErrServiceInitError
	}

	if s.discovery != nil {
		s.discovery.Stop()
	}

	glog.V(logger.Info).Infoln("Whisper Notification Server stopped")
	return nil
}

// RegisterClientSession forms a cryptographic link between server and client.
// It does so by sharing a session SymKey and installing filter listening for messages
// encrypted with that key. So, both server and client have a secure way to communicate.
func (s *NotificationServer) RegisterClientSession(session *ClientSession) (sessionKey []byte, err error) {
	s.clientSessionsMu.Lock()
	defer s.clientSessionsMu.Unlock()

	// generate random symmetric session key
	keyName := fmt.Sprintf("%s-%s", "ntfy-client", crypto.Keccak256Hash([]byte(session.ClientKey)).Hex())
	sessionKey, sessionKeyDerived, err := s.makeSessionKey(keyName)
	if err != nil {
		return nil, err
	}

	// populate session key hash (will be used to match decrypted message to a given client id)
	session.SessionKeyHash = crypto.Keccak256Hash(sessionKeyDerived)
	session.SessionKey = sessionKeyDerived

	// append to list of known clients
	// so that it is trivial to go key hash -> client session info
	id := session.SessionKeyHash.Hex()
	s.clientSessions[id] = session

	// setup filter, which will get all incoming messages, that are encrypted with SymKey
	filterID, err := s.installTopicFilter(topicNewChatSession, sessionKeyDerived)
	if err != nil {
		return nil, fmt.Errorf("failed installing filter: %v", err)
	}
	go s.requestProcessorLoop(filterID, topicNewChatSession, s.processNewChatSessionRequest)
	return
}

// RegisterChatSession forms a cryptographic link between server and client.
// This link is meant to be shared with other clients, so that they can use
// the shared SymKey to trigger notifications for devices attached to a given
// chat session.
func (s *NotificationServer) RegisterChatSession(session *ChatSession) (sessionKey []byte, err error) {
	s.chatSessionsMu.Lock()
	defer s.chatSessionsMu.Unlock()

	// generate random symmetric session key
	keyName := fmt.Sprintf("%s-%s", "ntfy-chat", crypto.Keccak256Hash([]byte(session.ParentKey+session.ChatKey)).Hex())
	sessionKey, sessionKeyDerived, err := s.makeSessionKey(keyName)
	if err != nil {
		return nil, err
	}

	// populate session key hash (will be used to match decrypted message to a given client id)
	session.SessionKeyHash = crypto.Keccak256Hash(sessionKeyDerived)
	session.SessionKey = sessionKeyDerived

	// append to list of known clients
	// so that it is trivial to go key hash -> client session info
	id := session.SessionKeyHash.Hex()
	s.chatSessions[id] = session

	// setup filter, to process incoming device registration requests
	filterID1, err := s.installTopicFilter(topicNewDeviceRegistration, sessionKeyDerived)
	if err != nil {
		return nil, fmt.Errorf("failed installing filter: %v", err)
	}
	go s.requestProcessorLoop(filterID1, topicNewDeviceRegistration, s.processNewDeviceRegistrationRequest)

	// setup filter, to process incoming notification trigger requests
	filterID2, err := s.installTopicFilter(topicSendNotification, sessionKeyDerived)
	if err != nil {
		return nil, fmt.Errorf("failed installing filter: %v", err)
	}
	go s.requestProcessorLoop(filterID2, topicSendNotification, s.processSendNotificationRequest)

	return
}

// RegisterDeviceSubscription persists device id, so that it can be used to trigger notifications.
func (s *NotificationServer) RegisterDeviceSubscription(subscription *DeviceSubscription) error {
	s.deviceSubscriptionsMu.Lock()
	defer s.deviceSubscriptionsMu.Unlock()

	// if one passes the same id again, we will just overwrite
	id := fmt.Sprintf("%s-%s", "ntfy-device",
		crypto.Keccak256Hash([]byte(subscription.ChatSessionKeyHash.Hex()+subscription.DeviceID)).Hex())
	s.deviceSubscriptions[id] = subscription

	glog.V(logger.Info).Infof("device registered: %s", subscription.DeviceID)
	return nil
}

// RemoveSubscriber uninstalls subscriber
func (s *NotificationServer) RemoveSubscriber(id string) {
	s.clientSessionsMu.Lock()
	defer s.clientSessionsMu.Unlock()

	delete(s.clientSessions, id)
}

// processNewChatSessionRequest processes incoming client requests of type:
// client has a session key, and ready to create a new chat session (which is
// a bag of subscribed devices, basically)
func (s *NotificationServer) processNewChatSessionRequest(msg *whisper.ReceivedMessage) error {
	s.clientSessionsMu.RLock()
	defer s.clientSessionsMu.RUnlock()

	var parsedMessage struct {
		ChatID string `json:"chat"`
	}
	if err := json.Unmarshal(msg.Payload, &parsedMessage); err != nil {
		return err
	}

	if msg.Src == nil {
		return errors.New("message 'from' field is required")
	}

	clientSession, ok := s.clientSessions[msg.SymKeyHash.Hex()]
	if !ok {
		return errors.New("client session not found")
	}

	// register chat session
	parentKey := hex.EncodeToString(crypto.FromECDSAPub(msg.Src))
	sessionKey, err := s.RegisterChatSession(&ChatSession{
		ParentKey: parentKey,
		ChatKey:   parsedMessage.ChatID,
	})
	if err != nil {
		return err
	}

	// confirm that chat has been successfully created
	msgParams := whisper.MessageParams{
		Dst:      msg.Src,
		KeySym:   clientSession.SessionKey,
		Topic:    MakeTopic([]byte(topicAckNewChatSession)),
		Payload:  []byte(`{"server": "0x` + s.nodeID + `", "key": "0x` + hex.EncodeToString(sessionKey) + `"}`),
		TTL:      uint32(s.config.TTL),
		PoW:      s.config.MinimumPoW,
		WorkTime: 5,
	}
	response := whisper.NewSentMessage(&msgParams)
	env, err := response.Wrap(&msgParams)
	if err != nil {
		return fmt.Errorf("failed to wrap server response message: %v", err)
	}

	if err := s.whisper.Send(env); err != nil {
		return fmt.Errorf("failed to send server response message: %v", err)
	}

	glog.V(logger.Debug).Infof("server confirms chat creation (dst: %v, topic: %x)", msgParams.Dst, msgParams.Topic)
	return nil
}

// processNewDeviceRegistrationRequest processes incoming client requests of type:
// client has a session key, creates chat, and obtains chat SymKey (to be shared with
// others). Then using that chat SymKey client registers it's device ID with server.
func (s *NotificationServer) processNewDeviceRegistrationRequest(msg *whisper.ReceivedMessage) error {
	s.chatSessionsMu.RLock()
	defer s.chatSessionsMu.RUnlock()

	var parsedMessage struct {
		DeviceID string `json:"device"`
	}
	if err := json.Unmarshal(msg.Payload, &parsedMessage); err != nil {
		return err
	}

	if msg.Src == nil {
		return errors.New("message 'from' field is required")
	}

	chatSession, ok := s.chatSessions[msg.SymKeyHash.Hex()]
	if !ok {
		return errors.New("chat session not found")
	}

	if len(parsedMessage.DeviceID) <= 0 {
		return errors.New("'device' cannot be empty")
	}

	// register chat session
	err := s.RegisterDeviceSubscription(&DeviceSubscription{
		DeviceID:           parsedMessage.DeviceID,
		ChatSessionKeyHash: chatSession.SessionKeyHash,
		PubKey:             msg.Src,
	})
	if err != nil {
		return err
	}

	// confirm that client has been successfully subscribed
	msgParams := whisper.MessageParams{
		Dst:      msg.Src,
		KeySym:   chatSession.SessionKey,
		Topic:    MakeTopic([]byte(topicAckDeviceRegistration)),
		Payload:  []byte(`{"server": "0x` + s.nodeID + `"}`),
		TTL:      uint32(s.config.TTL),
		PoW:      s.config.MinimumPoW,
		WorkTime: 5,
	}
	response := whisper.NewSentMessage(&msgParams)
	env, err := response.Wrap(&msgParams)
	if err != nil {
		return fmt.Errorf("failed to wrap server response message: %v", err)
	}

	if err := s.whisper.Send(env); err != nil {
		return fmt.Errorf("failed to send server response message: %v", err)
	}

	glog.V(logger.Debug).Infof("server confirms device registration (dst: %v, topic: %x)", msgParams.Dst, msgParams.Topic)
	return nil
}

// processSendNotificationRequest processes incoming client requests of type:
// when client has session key, and ready to use it to send notifications
func (s *NotificationServer) processSendNotificationRequest(msg *whisper.ReceivedMessage) error {
	s.deviceSubscriptionsMu.RLock()
	defer s.deviceSubscriptionsMu.RUnlock()

	for _, subscriber := range s.deviceSubscriptions {
		if subscriber.ChatSessionKeyHash == msg.SymKeyHash {
			if whisper.IsPubKeyEqual(msg.Src, subscriber.PubKey) {
				continue // no need to notify ourselves
			}

			if s.firebaseProvider != nil {
				err := s.firebaseProvider.Send(subscriber.DeviceID, string(msg.Payload))
				if err != nil {
					glog.V(logger.Info).Infof("cannot send notification: %v", err)
				}
			}
		}
	}

	return nil
}

// installTopicFilter installs Whisper filter using symmetric key
func (s *NotificationServer) installTopicFilter(topicName string, topicKey []byte) (filterID string, err error) {
	topic := MakeTopic([]byte(topicName))
	filter := whisper.Filter{
		KeySym:    topicKey,
		Topics:    []whisper.TopicType{topic},
		AcceptP2P: true,
	}
	filterID, err = s.whisper.Watch(&filter)
	if err != nil {
		return "", fmt.Errorf("failed installing filter: %v", err)
	}

	glog.V(logger.Debug).Infof("installed topic filter %v for topic %x (%s)", filterID, topic, topicName)
	return
}

// installKeyFilter installs Whisper filter using asymmetric key
func (s *NotificationServer) installKeyFilter(topicName string, key *ecdsa.PrivateKey) (filterID string, err error) {
	topic := MakeTopic([]byte(topicName))
	filter := whisper.Filter{
		KeyAsym:   key,
		Topics:    []whisper.TopicType{topic},
		AcceptP2P: true,
	}
	filterID, err = s.whisper.Watch(&filter)
	if err != nil {
		return "", fmt.Errorf("failed installing filter: %v", err)
	}

	glog.V(logger.Info).Infof("installed key filter %v for topic %x (%s)", filterID, topic, topicName)
	return
}

// requestProcessorLoop processes incoming client requests, by listening to a given filter,
// and executing process function on each incoming message
func (s *NotificationServer) requestProcessorLoop(filterID string, topicWatched string, fn messageProcessingFn) {
	glog.V(logger.Detail).Infof("request processor started: %s", topicWatched)

	filter := s.whisper.GetFilter(filterID)
	if filter == nil {
		glog.V(logger.Warn).Infof("filter is not installed: %s (for topic '%s')", filterID, topicWatched)
		return
	}

	ticker := time.NewTicker(time.Millisecond * 50)

	for {
		select {
		case <-ticker.C:
			messages := filter.Retrieve()
			for _, msg := range messages {
				if err := fn(msg); err != nil {
					glog.V(logger.Warn).Infof("failed processing incoming request: %v", err)
				}
			}
		case <-s.quit:
			glog.V(logger.Detail).Infof("request processor stopped: %s", topicWatched)
			return
		}
	}
}

// makeSessionKey generates and saves random SymKey, allowing to establish secure
// channel between server and client
func (s *NotificationServer) makeSessionKey(keyName string) (sessionKey, sessionKeyDerived []byte, err error) {
	// wipe out previous occurrence of symmetric key
	s.whisper.DeleteSymKey(keyName)

	sessionKey, err = makeSessionKey()
	if err != nil {
		return nil, nil, err
	}
	s.whisper.AddSymKey(keyName, sessionKey)
	sessionKeyDerived = s.whisper.GetSymKey(keyName)

	return
}
