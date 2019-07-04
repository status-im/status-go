package filter

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/messaging/sharedsecret"
	whisper "github.com/status-im/whisper/whisperv6"
	"math/big"
	"sync"
	"time"
)

const (
	discoveryTopic = "contact-discovery"
)

// The number of partitions
var nPartitions = big.NewInt(5000)
var minPow = 0.0

type Filter struct {
	FilterID string
	Topic    whisper.TopicType
	SymKeyID string
}

type Chat struct {
	// ChatID is the identifier of the chat
	ChatID string `json:"chatId"`
	// SymKeyID is the symmetric key id used for symmetric chats
	SymKeyID string `json:"symKeyId"`
	// OneToOne tells us if we need to use asymmetric encryption for this chat
	OneToOne bool `json:"oneToOne"`
	// Listen is whether we are actually listening for messages on this chat, or the filter is only created in order to be able to post on the topic
	Listen bool `json:"listen"`
	// FilterID the whisper filter id generated
	FilterID string `json:"filterId"`
	// Identity is the public key of the other recipient for non-public chats
	Identity string `json:"identity"`
	// Topic is the whisper topic
	Topic whisper.TopicType `json:"topic"`
	// Discovery is whether this is a discovery topic
	Discovery bool `json:"discovery"`
	// Negotiated tells us whether is a negotiated topic
	Negotiated bool `json:"negotiated"`
}

type Messages struct {
	Chat     *Chat              `json:"chat"`
	Messages []*whisper.Message `json:"messages"`
	Error    error              `json:"error"`
}

type Service struct {
	whisper       *whisper.Whisper
	secret        *sharedsecret.Service
	chats         map[string]*Chat
	persistence   Persistence
	mutex         sync.Mutex
	keys          map[string][]byte
	quit          chan struct{}
	onNewMessages func([]*Messages)
}

// New returns a new filter service
func New(w *whisper.Whisper, p Persistence, s *sharedsecret.Service, onNewMessages func([]*Messages)) *Service {
	return &Service{
		whisper:       w,
		secret:        s,
		mutex:         sync.Mutex{},
		persistence:   p,
		chats:         make(map[string]*Chat),
		quit:          make(chan struct{}),
		onNewMessages: onNewMessages,
	}
}

// LoadChat should return a list of newly chats loaded
func (s *Service) Init(chats []*Chat) ([]*Chat, error) {
	log.Debug("Initializing filter service", "chats", chats)

	keys, err := s.persistence.All()
	if err != nil {
		return nil, err
	}
	s.keys = keys

	keyID := s.whisper.SelectedKeyPairID()
	if keyID == "" {
		return nil, errors.New("no key selected")
	}
	myKey, err := s.whisper.GetPrivateKey(keyID)
	if err != nil {
		return nil, err
	}

	// Add our own topic
	log.Debug("Loading one to one chats")
	identityStr := fmt.Sprintf("%x", crypto.FromECDSAPub(&myKey.PublicKey))
	_, err = s.loadOneToOne(myKey, identityStr, true)
	if err != nil {
		log.Error("Error loading one to one chats", "err", err)

		return nil, err
	}

	// Add discovery topic
	log.Debug("Loading discovery topics")
	err = s.loadDiscovery(myKey)
	if err != nil {
		return nil, err
	}

	// Add the various one to one and public chats
	log.Debug("Loading chats")
	for _, chat := range chats {
		_, err = s.load(myKey, chat)
		if err != nil {
			return nil, err
		}
	}

	// Add the negotiated secrets
	log.Debug("Loading negotiated topics")
	secrets, err := s.secret.All()
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		if _, err := s.ProcessNegotiatedSecret(secret); err != nil {
			return nil, err
		}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var allChats []*Chat
	for _, chat := range s.chats {
		allChats = append(allChats, chat)
	}
	log.Debug("Loaded chats")
	return allChats, nil
}

func (s *Service) Start(checkPeriod time.Duration) {
	ticker := time.NewTicker(checkPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			messages := s.getMessages()

			if len(messages) != 0 {
				s.onNewMessages(messages)
			}
		case <-s.quit:
			return
		}
	}
}

// Stop removes all the filters
func (s *Service) Stop() error {
	close(s.quit)

	var chats []*Chat

	s.mutex.Lock()
	for _, chat := range s.chats {
		chats = append(chats, chat)
	}
	s.mutex.Unlock()

	return s.Remove(chats)
}

// Remove remove all the filters associated with a chat/identity
func (s *Service) Remove(chats []*Chat) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Debug("Removing chats", "chats", chats)

	for _, chat := range chats {
		log.Debug("Removing chat", "chat", chat)
		if err := s.whisper.Unsubscribe(chat.FilterID); err != nil {
			return err
		}
		if chat.SymKeyID != "" {
			s.whisper.DeleteSymKey(chat.SymKeyID)
		}
		delete(s.chats, chat.ChatID)
	}

	return nil
}

// LoadPartitioned creates a filter for a partitioned topic
func (s *Service) LoadPartitioned(myKey *ecdsa.PrivateKey, theirPublicKey *ecdsa.PublicKey, listen bool) (*Chat, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := PublicKeyToPartitionedTopic(theirPublicKey)

	if _, ok := s.chats[chatID]; ok {
		return s.chats[chatID], nil
	}

	// We set up a filter so we can publish, but we discard envelopes if listen is false
	filter, err := s.addAsymmetricFilter(myKey, chatID, listen)
	if err != nil {
		return nil, err
	}

	identityStr := fmt.Sprintf("%x", crypto.FromECDSAPub(theirPublicKey))

	chat := &Chat{
		ChatID:    chatID,
		FilterID:  filter.FilterID,
		Topic:     filter.Topic,
		Listen:    listen,
		Identity:  identityStr,
		Discovery: true,
	}

	s.chats[chatID] = chat

	return chat, nil
}

// Load creates filters for a given chat, and returns all the created filters
func (s *Service) Load(chat *Chat) ([]*Chat, error) {
	keyID := s.whisper.SelectedKeyPairID()
	if keyID == "" {
		return nil, errors.New("no key selected")
	}
	myKey, err := s.whisper.GetPrivateKey(keyID)

	if err != nil {
		return nil, err
	}
	return s.load(myKey, chat)
}

func ContactCodeTopic(identity string) string {
	return "0x" + identity + "-contact-code"
}

// Get returns a negotiated chat given an identity
func (s *Service) GetNegotiated(identity *ecdsa.PublicKey) *Chat {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.chats[negotiatedID(identity)]
}

// GetByID returns a chat by chatID
func (s *Service) GetByID(chatID string) *Chat {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.chats[chatID]
}

// ProcessNegotiatedSecret adds a filter based on the agreed secret
func (s *Service) ProcessNegotiatedSecret(secret *sharedsecret.Secret) (*Chat, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := negotiatedID(secret.Identity)
	// If we already have a chat do nothing
	if _, ok := s.chats[chatID]; ok {
		return s.chats[chatID], nil
	}

	keyString := fmt.Sprintf("%x", secret.Key)
	filter, err := s.addSymmetric(keyString)
	if err != nil {
		return nil, err
	}

	identityStr := fmt.Sprintf("%x", crypto.FromECDSAPub(secret.Identity))

	chat := &Chat{
		ChatID:     chatID,
		Topic:      filter.Topic,
		SymKeyID:   filter.SymKeyID,
		FilterID:   filter.FilterID,
		Identity:   identityStr,
		Listen:     true,
		Negotiated: true,
	}

	log.Debug("Processing negotiated secret", "chat-id", chatID, "topic", filter.Topic)

	s.chats[chat.ChatID] = chat
	return chat, nil
}

// ToTopic converts a string to a whisper topic
func ToTopic(s string) []byte {
	return crypto.Keccak256([]byte(s))[:whisper.TopicLength]
}

// PublicKeyToPartitionedTopic returns the associated partitioned topic string
// with the given public key
func PublicKeyToPartitionedTopic(publicKey *ecdsa.PublicKey) string {
	partition := big.NewInt(0)
	partition.Mod(publicKey.X, nPartitions)
	return fmt.Sprintf("contact-discovery-%d", partition.Int64())
}

// PublicKeyToPartitionedTopicBytes returns the bytes of the partitioned topic
// associated with the given public key
func PublicKeyToPartitionedTopicBytes(publicKey *ecdsa.PublicKey) []byte {
	return ToTopic(PublicKeyToPartitionedTopic(publicKey))
}

// loadDiscovery adds the discovery filter
func (s *Service) loadDiscovery(myKey *ecdsa.PrivateKey) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.chats[discoveryTopic]; ok {
		return nil
	}

	identityStr := fmt.Sprintf("%x", crypto.FromECDSAPub(&myKey.PublicKey))

	discoveryChat := &Chat{
		ChatID:    discoveryTopic,
		Listen:    true,
		Identity:  identityStr,
		Discovery: true,
	}

	discoveryResponse, err := s.addAsymmetricFilter(myKey, discoveryChat.ChatID, true)
	if err != nil {
		return err
	}

	discoveryChat.Topic = discoveryResponse.Topic
	discoveryChat.FilterID = discoveryResponse.FilterID

	s.chats[discoveryChat.ChatID] = discoveryChat

	// Load personal discovery
	personalDiscoveryTopic := fmt.Sprintf("contact-discovery-%s", identityStr)
	personalDiscoveryChat := &Chat{
		ChatID:    personalDiscoveryTopic,
		Listen:    true,
		Identity:  identityStr,
		Discovery: true,
	}

	discoveryResponse, err = s.addAsymmetricFilter(myKey, personalDiscoveryChat.ChatID, true)
	if err != nil {
		return err
	}

	personalDiscoveryChat.Topic = discoveryResponse.Topic
	personalDiscoveryChat.FilterID = discoveryResponse.FilterID

	s.chats[personalDiscoveryChat.ChatID] = personalDiscoveryChat

	return nil
}

// loadPublic adds a filter for a public chat
func (s *Service) loadPublic(chat *Chat) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.chats[chat.ChatID]; ok {
		return nil
	}

	filterAndTopic, err := s.addSymmetric(chat.ChatID)
	if err != nil {
		return err
	}

	chat.FilterID = filterAndTopic.FilterID
	chat.SymKeyID = filterAndTopic.SymKeyID
	chat.Topic = filterAndTopic.Topic
	chat.Listen = true

	s.chats[chat.ChatID] = chat
	return nil
}

// loadOneToOne creates two filters for a given chat, one listening to the contact codes
// and another on the partitioned topic, if listen is specified.
func (s *Service) loadOneToOne(myKey *ecdsa.PrivateKey, identity string, listen bool) ([]*Chat, error) {
	var chats []*Chat
	contactCodeChat, err := s.loadContactCode(identity)
	if err != nil {
		return nil, err
	}

	chats = append(chats, contactCodeChat)

	if listen {
		publicKeyBytes, err := hex.DecodeString(identity)
		if err != nil {
			return nil, err
		}

		publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
		if err != nil {
			return nil, err
		}

		partitionedChat, err := s.LoadPartitioned(myKey, publicKey, listen)
		if err != nil {
			return nil, err
		}

		chats = append(chats, partitionedChat)
	}
	return chats, nil
}

// loadContactCode creates a filter for the topic are advertised for a given identity
func (s *Service) loadContactCode(identity string) (*Chat, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := ContactCodeTopic(identity)
	if _, ok := s.chats[chatID]; ok {
		return s.chats[chatID], nil
	}

	contactCodeFilter, err := s.addSymmetric(chatID)
	if err != nil {
		return nil, err
	}
	chat := &Chat{
		ChatID:   chatID,
		FilterID: contactCodeFilter.FilterID,
		Topic:    contactCodeFilter.Topic,
		SymKeyID: contactCodeFilter.SymKeyID,
		Identity: identity,
		Listen:   true,
	}

	s.chats[chatID] = chat
	return chat, nil
}

// addSymmetric adds a symmetric key filter
func (s *Service) addSymmetric(chatID string) (*Filter, error) {
	var symKeyID string
	var err error

	topic := ToTopic(chatID)
	topics := [][]byte{topic}

	symKey, ok := s.keys[chatID]
	if ok {
		log.Debug("Loading from cache", "chat-id", chatID)
		symKeyID, err = s.whisper.AddSymKeyDirect(symKey)
		if err != nil {
			log.Error("Adding symkey failed", "err", err)
			return nil, err
		}
	} else {
		log.Debug("Generating symkey", "chat-id", chatID)
		symKeyID, err = s.whisper.AddSymKeyFromPassword(chatID)
		if err != nil {
			log.Error("Adding symkey from password failed", "err", err)
			return nil, err
		}
		if symKey, err = s.whisper.GetSymKey(symKeyID); err != nil {
			return nil, err
		}
		s.keys[chatID] = symKey

		err = s.persistence.Add(chatID, symKey)
		if err != nil {
			return nil, err
		}
	}

	f := &whisper.Filter{
		KeySym:   symKey,
		PoW:      minPow,
		AllowP2P: true,
		Topics:   topics,
		Messages: s.whisper.NewMessageStore(),
	}

	id, err := s.whisper.Subscribe(f)
	if err != nil {
		return nil, err
	}

	return &Filter{
		FilterID: id,
		SymKeyID: symKeyID,
		Topic:    whisper.BytesToTopic(topic),
	}, nil
}

// addAsymmetricFilter adds a filter with our privatekey, and set minPow according to the listen parameter
func (s *Service) addAsymmetricFilter(keyAsym *ecdsa.PrivateKey, chatID string, listen bool) (*Filter, error) {
	var err error
	var pow float64

	if listen {
		pow = minPow
	} else {
		// Set high pow so we discard messages
		pow = 1
	}

	topic := ToTopic(chatID)
	topics := [][]byte{topic}

	f := &whisper.Filter{
		KeyAsym:  keyAsym,
		PoW:      pow,
		AllowP2P: true,
		Topics:   topics,
		Messages: s.whisper.NewMessageStore(),
	}

	id, err := s.whisper.Subscribe(f)
	if err != nil {
		return nil, err
	}

	return &Filter{FilterID: id, Topic: whisper.BytesToTopic(topic)}, nil
}

func (s *Service) getMessages() []*Messages {
	var response []*Messages
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for chatID := range s.chats {
		messages := s.getMessagesForChat(chatID)
		if messages.Error != nil || len(messages.Messages) != 0 {
			response = append(response, messages)
		}
	}

	return response
}

func (s *Service) getMessagesForChat(chatID string) *Messages {
	response := &Messages{}

	response.Chat = s.chats[chatID]
	if response.Chat == nil {
		response.Error = errors.New("Chat not found")

		return response
	}

	filter := s.whisper.GetFilter(response.Chat.FilterID)
	if filter == nil {
		response.Error = errors.New("Filter not found")
		return response
	}

	receivedMessages := filter.Retrieve()
	response.Messages = make([]*whisper.Message, 0, len(receivedMessages))
	for _, msg := range receivedMessages {
		response.Messages = append(response.Messages, whisper.ToWhisperMessage(msg))
	}

	return response
}

func negotiatedID(identity *ecdsa.PublicKey) string {
	return fmt.Sprintf("0x%x-negotiated", crypto.FromECDSAPub(identity))
}

func (s *Service) load(myKey *ecdsa.PrivateKey, chat *Chat) ([]*Chat, error) {
	log.Debug("Loading chat", "chatID", chat.ChatID)

	if chat.OneToOne {
		return s.loadOneToOne(myKey, chat.Identity, false)

	}
	return []*Chat{chat}, s.loadPublic(chat)
}
