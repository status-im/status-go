package filter

import (
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"math/big"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	whisper "github.com/status-im/whisper/whisperv6"
	"go.uber.org/zap"
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

// TODO: revise fields encoding/decoding. Some are encoded using hexutil and some using encoding/hex.
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

func (c *Chat) IsPublic() bool {
	return !c.OneToOne
}

type ChatsManager struct {
	whisper     *whisper.Whisper
	persistence *sqlitePersistence
	privateKey  *ecdsa.PrivateKey
	keys        map[string][]byte // a cache of symmetric keys derived from passwords
	logger      *zap.Logger

	genericDiscoveryTopicEnabled bool

	mutex sync.Mutex
	chats map[string]*Chat
}

// New returns a new ChatsManager service
func New(db *sql.DB, w *whisper.Whisper, privateKey *ecdsa.PrivateKey, logger *zap.Logger) (*ChatsManager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	persistence := newSQLitePersistence(db)

	keys, err := persistence.All()
	if err != nil {
		return nil, err
	}

	return &ChatsManager{
		privateKey:  privateKey,
		whisper:     w,
		persistence: persistence,
		keys:        keys,
		chats:       make(map[string]*Chat),
		logger:      logger.With(zap.Namespace("ChatsManager")),
	}, nil
}

func (s *ChatsManager) Init(
	chatIDs []string,
	publicKeys []*ecdsa.PublicKey,
	genericDiscoveryTopicEnabled bool,
) ([]*Chat, error) {
	logger := s.logger.With(zap.String("site", "Init"))

	logger.Info("initializing")

	s.genericDiscoveryTopicEnabled = genericDiscoveryTopicEnabled

	// Load our contact code.
	_, err := s.LoadContactCode(&s.privateKey.PublicKey)
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

	// Add public, one-to-one and negotiated chats.
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

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var allChats []*Chat
	for _, chat := range s.chats {
		allChats = append(allChats, chat)
	}
	return allChats, nil
}

// DEPRECATED
func (s *ChatsManager) InitWithChats(chats []*Chat, genericDiscoveryTopicEnabled bool) ([]*Chat, error) {
	var (
		chatIDs    []string
		publicKeys []*ecdsa.PublicKey
	)

	for _, chat := range chats {
		if chat.Identity != "" && chat.OneToOne {
			publicKey, err := strToPublicKey(chat.Identity)
			if err != nil {
				return nil, err
			}
			publicKeys = append(publicKeys, publicKey)
		} else if chat.ChatID != "" {
			chatIDs = append(chatIDs, chat.ChatID)
		}
	}

	return s.Init(chatIDs, publicKeys, genericDiscoveryTopicEnabled)
}

func (s *ChatsManager) Reset() error {
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

func (s *ChatsManager) ChatByFilterID(filterID string) *Chat {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, chat := range s.chats {
		if chat.FilterID == filterID {
			return chat
		}
	}
	return nil
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
		OneToOne: true,
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
		OneToOne:   true,
	}

	s.chats[chat.ChatID] = chat

	return chat, nil
}

// LoadDiscovery adds 1-2 discovery filters: one for generic discovery topic (if enabled)
// and one for the personal discovery topic.
func (s *ChatsManager) LoadDiscovery() ([]*Chat, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	personalDiscoveryTopic := personalDiscoveryTopic(&s.privateKey.PublicKey)

	// Check if chats are already loaded.
	var result []*Chat

	expectedTopicCount := 1

	if s.genericDiscoveryTopicEnabled {
		expectedTopicCount = 2
		if chat, ok := s.chats[DiscoveryTopic]; ok {
			result = append(result, chat)
		}
	}
	if chat, ok := s.chats[personalDiscoveryTopic]; ok {
		result = append(result, chat)
	}

	if len(result) == expectedTopicCount {
		return result, nil
	}

	var discoveryResponse *Filter
	var err error
	identityStr := publicKeyToStr(&s.privateKey.PublicKey)

	// Load personal discovery
	personalDiscoveryChat := &Chat{
		ChatID:    personalDiscoveryTopic,
		Identity:  identityStr,
		Discovery: true,
		Listen:    true,
		OneToOne:  true,
	}

	discoveryResponse, err = s.addAsymmetric(personalDiscoveryChat.ChatID, true)
	if err != nil {
		return nil, err
	}

	personalDiscoveryChat.Topic = discoveryResponse.Topic
	personalDiscoveryChat.FilterID = discoveryResponse.FilterID

	s.chats[personalDiscoveryChat.ChatID] = personalDiscoveryChat

	if s.genericDiscoveryTopicEnabled {
		// Load generic discovery topic.
		discoveryChat := &Chat{
			ChatID:    DiscoveryTopic,
			Identity:  identityStr,
			Discovery: true,
			Listen:    true,
			OneToOne:  true,
		}

		discoveryResponse, err = s.addAsymmetric(discoveryChat.ChatID, true)
		if err != nil {
			return nil, err
		}

		discoveryChat.Topic = discoveryResponse.Topic
		discoveryChat.FilterID = discoveryResponse.FilterID

		s.chats[discoveryChat.ChatID] = discoveryChat

		return []*Chat{discoveryChat, personalDiscoveryChat}, nil
	}

	return []*Chat{personalDiscoveryChat}, nil
}

// LoadPublic adds a filter for a public chat.
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
		OneToOne: false,
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

// GetNegotiated returns a negotiated chat given an identity
func (s *ChatsManager) GetNegotiated(identity *ecdsa.PublicKey) *Chat {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.chats[negotiatedTopic(identity)]
}

// toTopic converts a string to a whisper topic.
func toTopic(s string) []byte {
	return crypto.Keccak256([]byte(s))[:whisper.TopicLength]
}

func ToTopic(s string) []byte {
	return toTopic(s)
}

func strToPublicKey(str string) (*ecdsa.PublicKey, error) {
	publicKeyBytes, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPubkey(publicKeyBytes)
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
