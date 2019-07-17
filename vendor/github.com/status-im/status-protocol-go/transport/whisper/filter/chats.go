package filter

import (
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"log"
	"math/big"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	whisper "github.com/status-im/whisper/whisperv6"
)

const (
	DiscoveryTopic = "contact-discovery"
)

var (
	// The number of partitions.
	nPartitions = big.NewInt(5000)
	minPow      = 0.0
)

type Filter struct {
	FilterID string
	Topic    whisper.TopicType
	SymKeyID string
}

type NegotiatedSecret struct {
	PublicKey *ecdsa.PublicKey
	Key       []byte
}

type Chat struct {
	// ChatID is the identifier of the chat
	ChatID string `json:"chatId"`
	// FilterID the whisper filter id generated
	FilterID string `json:"filterId"`
	// SymKeyID is the symmetric key id used for symmetric chats
	SymKeyID string `json:"symKeyId"`
	// OneToOne tells us if we need to use asymmetric encryption for this chat
	OneToOne bool `json:"oneToOne"`
	// Identity is the public key of the other recipient for non-public chats.
	// It's encoded using encoding/hex.
	Identity string `json:"identity"`
	// Topic is the whisper topic
	Topic whisper.TopicType `json:"topic"`
	// Discovery is whether this is a discovery topic
	Discovery bool `json:"discovery"`
	// Negotiated tells us whether is a negotiated topic
	Negotiated bool `json:"negotiated"`
	// Listen is whether we are actually listening for messages on this chat, or the filter is only created in order to be able to post on the topic
	Listen bool `json:"listen"`
}

type ChatsManager struct {
	whisper     *whisper.Whisper
	persistence *sqlitePersistence
	privateKey  *ecdsa.PrivateKey
	keys        map[string][]byte

	mutex sync.Mutex
	chats map[string]*Chat
}

// New returns a new filter service
func New(db *sql.DB, w *whisper.Whisper, privateKey *ecdsa.PrivateKey) (*ChatsManager, error) {
	persistence, err := newSQLitePersistence(db)
	if err != nil {
		return nil, err
	}

	return &ChatsManager{
		privateKey:  privateKey,
		whisper:     w,
		persistence: persistence,
		chats:       make(map[string]*Chat),
	}, nil
}

func (s *ChatsManager) Init(chatIDs []string, publicKeys []*ecdsa.PublicKey, negotiated []NegotiatedSecret) ([]*Chat, error) {
	log.Printf("[FiltersManager::Init] initializing")

	keys, err := s.persistence.All()
	if err != nil {
		return nil, err
	}
	s.keys = keys

	// Load our contact code.
	_, err = s.LoadContactCode(&s.privateKey.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load contact code")
	}

	// Load partitioned topic.
	_, err = s.loadMyPartitioned()
	if err != nil {
		return nil, err
	}

	// Add discovery topic.
	_, err = s.LoadDiscovery()
	if err != nil {
		return nil, err
	}

	// Add public, one-to-one and generic chats.
	for _, chatID := range chatIDs {
		_, err := s.LoadPublic(chatID)
		if err != nil {
			return nil, err
		}
	}

	for _, publicKey := range publicKeys {
		_, err := s.LoadContactCode(publicKey)
		if err != nil {
			return nil, err
		}
	}

	for _, secret := range negotiated {
		if _, err := s.LoadNegotiated(secret); err != nil {
			return nil, err
		}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var allChats []*Chat
	for _, chat := range s.chats {
		allChats = append(allChats, chat)
	}
	return allChats, nil
}

func (s *ChatsManager) Uninitialize() error {
	var chats []*Chat

	s.mutex.Lock()
	for _, chat := range s.chats {
		chats = append(chats, chat)
	}
	s.mutex.Unlock()

	return s.Remove(chats...)
}

func (s *ChatsManager) Chats() (result []*Chat) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, chat := range s.chats {
		result = append(result, chat)
	}

	return
}

// ChatByID returns a chat by id.
func (s *ChatsManager) ChatByID(chatID string) *Chat {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.chats[chatID]
}

func (s *ChatsManager) ChatsByPublicKey(publicKey *ecdsa.PublicKey) (result []*Chat) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	identityStr := publicKeyToStr(publicKey)

	for _, chat := range s.chats {
		if chat.Identity == identityStr {
			result = append(result, chat)
		}
	}

	return
}

// Remove remove all the filters associated with a chat/identity
func (s *ChatsManager) Remove(chats ...*Chat) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, chat := range chats {
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

// LoadPartitioned creates a filter for a partitioned topic.
func (s *ChatsManager) LoadPartitioned(publicKey *ecdsa.PublicKey) (*Chat, error) {
	return s.loadPartitioned(publicKey, false)
}

func (s *ChatsManager) loadMyPartitioned() (*Chat, error) {
	return s.loadPartitioned(&s.privateKey.PublicKey, true)
}

func (s *ChatsManager) loadPartitioned(publicKey *ecdsa.PublicKey, listen bool) (*Chat, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := partitionedTopic(publicKey)
	if _, ok := s.chats[chatID]; ok {
		return s.chats[chatID], nil
	}

	// We set up a filter so we can publish,
	// but we discard envelopes if listen is false.
	filter, err := s.addAsymmetric(chatID, listen)
	if err != nil {
		return nil, err
	}

	chat := &Chat{
		ChatID:   chatID,
		FilterID: filter.FilterID,
		Topic:    filter.Topic,
		Identity: publicKeyToStr(publicKey),
		Listen:   listen,
	}

	s.chats[chatID] = chat

	return chat, nil
}

// LoadNegotiated loads a negotiated secret as a filter.
func (s *ChatsManager) LoadNegotiated(secret NegotiatedSecret) (*Chat, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := negotiatedTopic(secret.PublicKey)

	if _, ok := s.chats[chatID]; ok {
		return s.chats[chatID], nil
	}

	keyString := hex.EncodeToString(secret.Key)
	filter, err := s.addSymmetric(keyString)
	if err != nil {
		return nil, err
	}

	chat := &Chat{
		ChatID:     chatID,
		Topic:      filter.Topic,
		SymKeyID:   filter.SymKeyID,
		FilterID:   filter.FilterID,
		Identity:   publicKeyToStr(secret.PublicKey),
		Negotiated: true,
		Listen:     true,
	}

	s.chats[chat.ChatID] = chat

	return chat, nil
}

// loadDiscovery adds two discovery filters: for generic discovery topic
// and for the personal discovery topic.
func (s *ChatsManager) LoadDiscovery() ([]*Chat, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	personalDiscoveryTopic := personalDiscoveryTopic(&s.privateKey.PublicKey)

	// Check if chats are already loaded.
	var result []*Chat

	if chat, ok := s.chats[DiscoveryTopic]; ok {
		result = append(result, chat)
	}
	if chat, ok := s.chats[personalDiscoveryTopic]; ok {
		result = append(result, chat)
	}

	if len(result) == 2 {
		return result, nil
	}

	// Load generic discovery topic.
	identityStr := publicKeyToStr(&s.privateKey.PublicKey)

	discoveryChat := &Chat{
		ChatID:    DiscoveryTopic,
		Identity:  identityStr,
		Discovery: true,
		Listen:    true,
	}

	discoveryResponse, err := s.addAsymmetric(discoveryChat.ChatID, true)
	if err != nil {
		return nil, err
	}

	discoveryChat.Topic = discoveryResponse.Topic
	discoveryChat.FilterID = discoveryResponse.FilterID

	s.chats[discoveryChat.ChatID] = discoveryChat

	// Load personal discovery
	personalDiscoveryChat := &Chat{
		ChatID:    personalDiscoveryTopic,
		Identity:  identityStr,
		Discovery: true,
		Listen:    true,
	}

	discoveryResponse, err = s.addAsymmetric(personalDiscoveryChat.ChatID, true)
	if err != nil {
		return nil, err
	}

	personalDiscoveryChat.Topic = discoveryResponse.Topic
	personalDiscoveryChat.FilterID = discoveryResponse.FilterID

	s.chats[personalDiscoveryChat.ChatID] = personalDiscoveryChat

	return []*Chat{discoveryChat, personalDiscoveryChat}, nil
}

// loadPublic adds a filter for a public chat.
func (s *ChatsManager) LoadPublic(chatID string) (*Chat, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if chat, ok := s.chats[chatID]; ok {
		return chat, nil
	}

	filterAndTopic, err := s.addSymmetric(chatID)
	if err != nil {
		return nil, err
	}

	chat := &Chat{
		ChatID:   chatID,
		FilterID: filterAndTopic.FilterID,
		SymKeyID: filterAndTopic.SymKeyID,
		Topic:    filterAndTopic.Topic,
		Listen:   true,
	}

	s.chats[chatID] = chat

	return chat, nil
}

// LoadContactCode creates a filter for the advertise topic for a given public key.
func (s *ChatsManager) LoadContactCode(pubKey *ecdsa.PublicKey) (*Chat, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := contactCodeTopic(pubKey)

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
		Identity: publicKeyToStr(pubKey),
		Listen:   true,
	}

	s.chats[chatID] = chat
	return chat, nil
}

// addSymmetric adds a symmetric key filter
func (s *ChatsManager) addSymmetric(chatID string) (*Filter, error) {
	var symKeyID string
	var err error

	topic := toTopic(chatID)
	topics := [][]byte{topic}

	symKey, ok := s.keys[chatID]
	if ok {
		symKeyID, err = s.whisper.AddSymKeyDirect(symKey)
		if err != nil {
			return nil, err
		}
	} else {
		symKeyID, err = s.whisper.AddSymKeyFromPassword(chatID)
		if err != nil {
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

// addAsymmetricFilter adds a filter with our private key
// and set minPow according to the listen parameter.
func (s *ChatsManager) addAsymmetric(chatID string, listen bool) (*Filter, error) {
	var (
		err error
		pow = 1.0 // use PoW high enough to discard all messages for the filter
	)

	if listen {
		pow = minPow
	}

	topic := toTopic(chatID)
	topics := [][]byte{topic}

	f := &whisper.Filter{
		KeyAsym:  s.privateKey,
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

// Get returns a negotiated chat given an identity
func (s *ChatsManager) GetNegotiated(identity *ecdsa.PublicKey) *Chat {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.chats[negotiatedTopic(identity)]
}

// DEPRECATED
func (s *ChatsManager) InitDeprecated(chats []*Chat, secrets []NegotiatedSecret) ([]*Chat, error) {
	var (
		chatIDs    []string
		publicKeys []*ecdsa.PublicKey
	)

	for _, chat := range chats {
		if chat.ChatID != "" {
			chatIDs = append(chatIDs, chat.ChatID)
		} else if chat.Identity != "" {
			publicKeyBytes, err := hex.DecodeString(chat.Identity)
			if err != nil {
				return nil, err
			}

			publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
			if err != nil {
				return nil, err
			}

			publicKeys = append(publicKeys, publicKey)
		}
	}

	return s.Init(chatIDs, publicKeys, secrets)
}

// DEPRECATED
func (s *ChatsManager) Load(chat *Chat) ([]*Chat, error) {
	if chat.ChatID != "" {
		chat, err := s.LoadPublic(chat.ChatID)
		return []*Chat{chat}, err
	} else if chat.Identity != "" {
		publicKeyBytes, err := hex.DecodeString(chat.Identity)
		if err != nil {
			return nil, err
		}

		publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
		if err != nil {
			return nil, err
		}

		chat, err := s.LoadContactCode(publicKey)
		return []*Chat{chat}, err
	}

	return nil, errors.New("invalid Chat to load")
}

// toTopic converts a string to a whisper topic.
func toTopic(s string) []byte {
	return crypto.Keccak256([]byte(s))[:whisper.TopicLength]
}

func ToTopic(s string) []byte {
	return toTopic(s)
}

func publicKeyToStr(publicKey *ecdsa.PublicKey) string {
	return hex.EncodeToString(crypto.FromECDSAPub(publicKey))
}

func personalDiscoveryTopic(publicKey *ecdsa.PublicKey) string {
	return "contact-discovery-" + publicKeyToStr(publicKey)
}

// partitionedTopic returns the associated partitioned topic string
// with the given public key.
func partitionedTopic(publicKey *ecdsa.PublicKey) string {
	partition := big.NewInt(0)
	partition.Mod(publicKey.X, nPartitions)
	return "contact-discovery-" + strconv.FormatInt(partition.Int64(), 10)
}

// PublicKeyToPartitionedTopicBytes returns the bytes of the partitioned topic
// associated with the given public key
func PublicKeyToPartitionedTopicBytes(publicKey *ecdsa.PublicKey) []byte {
	return toTopic(partitionedTopic(publicKey))
}

func ContactCodeTopic(publicKey *ecdsa.PublicKey) string {
	return contactCodeTopic(publicKey)
}

func contactCodeTopic(publicKey *ecdsa.PublicKey) string {
	return "0x" + publicKeyToStr(publicKey) + "-contact-code"
}

func negotiatedTopic(publicKey *ecdsa.PublicKey) string {
	return "0x" + publicKeyToStr(publicKey) + "-negotiated"
}
