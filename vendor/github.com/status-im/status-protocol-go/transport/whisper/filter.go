package whisper

import (
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"math/big"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	whispertypes "github.com/status-im/status-protocol-go/transport/whisper/types"
	"go.uber.org/zap"
)

const (
	discoveryTopic = "contact-discovery"
)

var (
	// The number of partitions.
	nPartitions = big.NewInt(5000)
	minPow      = 0.0
)

type whisperFilter struct {
	FilterID string
	Topic    whispertypes.TopicType
	SymKeyID string
}

// TODO: revise fields encoding/decoding. Some are encoded using hexutil and some using encoding/hex.
type Filter struct {
	// ChatID is the identifier of the chat
	ChatID string `json:"chatId"`
	// FilterID the whisper filter id generated
	FilterID string `json:"filterId"`
	// SymKeyID is the symmetric key id used for symmetric filters
	SymKeyID string `json:"symKeyId"`
	// OneToOne tells us if we need to use asymmetric encryption for this chat
	OneToOne bool `json:"oneToOne"`
	// Identity is the public key of the other recipient for non-public filters.
	// It's encoded using encoding/hex.
	Identity string `json:"identity"`
	// Topic is the whisper topic
	Topic whispertypes.TopicType `json:"topic"`
	// Discovery is whether this is a discovery topic
	Discovery bool `json:"discovery"`
	// Negotiated tells us whether is a negotiated topic
	Negotiated bool `json:"negotiated"`
	// Listen is whether we are actually listening for messages on this chat, or the filter is only created in order to be able to post on the topic
	Listen bool `json:"listen"`
}

func (c *Filter) IsPublic() bool {
	return !c.OneToOne
}

type filtersManager struct {
	whisper     whispertypes.Whisper
	persistence *sqlitePersistence
	privateKey  *ecdsa.PrivateKey
	keys        map[string][]byte // a cache of symmetric manager derived from passwords
	logger      *zap.Logger

	genericDiscoveryTopicEnabled bool

	mutex   sync.Mutex
	filters map[string]*Filter
}

// newFiltersManager returns a new filtersManager.
func newFiltersManager(db *sql.DB, w whispertypes.Whisper, privateKey *ecdsa.PrivateKey, logger *zap.Logger) (*filtersManager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	persistence := newSQLitePersistence(db)

	keys, err := persistence.All()
	if err != nil {
		return nil, err
	}

	return &filtersManager{
		privateKey:  privateKey,
		whisper:     w,
		persistence: persistence,
		keys:        keys,
		filters:     make(map[string]*Filter),
		logger:      logger.With(zap.Namespace("filtersManager")),
	}, nil
}

func (s *filtersManager) Init(
	chatIDs []string,
	publicKeys []*ecdsa.PublicKey,
	genericDiscoveryTopicEnabled bool,
) ([]*Filter, error) {
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

	// Add public, one-to-one and negotiated filters.
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

	var allFilters []*Filter
	for _, f := range s.filters {
		allFilters = append(allFilters, f)
	}
	return allFilters, nil
}

// DEPRECATED
func (s *filtersManager) InitWithFilters(filters []*Filter, genericDiscoveryTopicEnabled bool) ([]*Filter, error) {
	var (
		chatIDs    []string
		publicKeys []*ecdsa.PublicKey
	)

	for _, filter := range filters {
		if filter.Identity != "" && filter.OneToOne {
			publicKey, err := strToPublicKey(filter.Identity)
			if err != nil {
				return nil, err
			}
			publicKeys = append(publicKeys, publicKey)
		} else if filter.ChatID != "" {
			chatIDs = append(chatIDs, filter.ChatID)
		}
	}

	return s.Init(chatIDs, publicKeys, genericDiscoveryTopicEnabled)
}

func (s *filtersManager) Reset() error {
	var filters []*Filter

	s.mutex.Lock()
	for _, f := range s.filters {
		filters = append(filters, f)
	}
	s.mutex.Unlock()

	return s.Remove(filters...)
}

func (s *filtersManager) Filters() (result []*Filter) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, f := range s.filters {
		result = append(result, f)
	}

	return
}

func (s *filtersManager) Filter(chatID string) *Filter {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.filters[chatID]
}

// FilterByFilterID returns a Filter with a given Whisper filter ID.
func (s *filtersManager) FilterByFilterID(filterID string) *Filter {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, f := range s.filters {
		if f.FilterID == filterID {
			return f
		}
	}
	return nil
}

func (s *filtersManager) FiltersByPublicKey(publicKey *ecdsa.PublicKey) (result []*Filter) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	identityStr := publicKeyToStr(publicKey)

	for _, f := range s.filters {
		if f.Identity == identityStr {
			result = append(result, f)
		}
	}

	return
}

// Remove remove all the filters associated with a chat/identity
func (s *filtersManager) Remove(filters ...*Filter) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, f := range filters {
		if err := s.whisper.Unsubscribe(f.FilterID); err != nil {
			return err
		}
		if f.SymKeyID != "" {
			s.whisper.DeleteSymKey(f.SymKeyID)
		}
		delete(s.filters, f.ChatID)
	}

	return nil
}

// LoadPartitioned creates a filter for a partitioned topic.
func (s *filtersManager) LoadPartitioned(publicKey *ecdsa.PublicKey) (*Filter, error) {
	return s.loadPartitioned(publicKey, false)
}

func (s *filtersManager) loadMyPartitioned() (*Filter, error) {
	return s.loadPartitioned(&s.privateKey.PublicKey, true)
}

func (s *filtersManager) loadPartitioned(publicKey *ecdsa.PublicKey, listen bool) (*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := partitionedTopic(publicKey)
	if _, ok := s.filters[chatID]; ok {
		return s.filters[chatID], nil
	}

	// We set up a filter so we can publish,
	// but we discard envelopes if listen is false.
	filter, err := s.addAsymmetric(chatID, listen)
	if err != nil {
		return nil, err
	}

	chat := &Filter{
		ChatID:   chatID,
		FilterID: filter.FilterID,
		Topic:    filter.Topic,
		Identity: publicKeyToStr(publicKey),
		Listen:   listen,
		OneToOne: true,
	}

	s.filters[chatID] = chat

	return chat, nil
}

// LoadNegotiated loads a negotiated secret as a filter.
func (s *filtersManager) LoadNegotiated(secret whispertypes.NegotiatedSecret) (*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := negotiatedTopic(secret.PublicKey)

	if _, ok := s.filters[chatID]; ok {
		return s.filters[chatID], nil
	}

	keyString := hex.EncodeToString(secret.Key)
	filter, err := s.addSymmetric(keyString)
	if err != nil {
		return nil, err
	}

	chat := &Filter{
		ChatID:     chatID,
		Topic:      filter.Topic,
		SymKeyID:   filter.SymKeyID,
		FilterID:   filter.FilterID,
		Identity:   publicKeyToStr(secret.PublicKey),
		Negotiated: true,
		Listen:     true,
		OneToOne:   true,
	}

	s.filters[chat.ChatID] = chat

	return chat, nil
}

// LoadDiscovery adds 1-2 discovery filters: one for generic discovery topic (if enabled)
// and one for the personal discovery topic.
func (s *filtersManager) LoadDiscovery() ([]*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	personalDiscoveryTopic := personalDiscoveryTopic(&s.privateKey.PublicKey)

	// Check if filters are already loaded.
	var result []*Filter

	expectedTopicCount := 1

	if s.genericDiscoveryTopicEnabled {
		expectedTopicCount = 2
		if chat, ok := s.filters[discoveryTopic]; ok {
			result = append(result, chat)
		}
	}
	if chat, ok := s.filters[personalDiscoveryTopic]; ok {
		result = append(result, chat)
	}

	if len(result) == expectedTopicCount {
		return result, nil
	}

	var discoveryResponse *whisperFilter
	var err error
	identityStr := publicKeyToStr(&s.privateKey.PublicKey)

	// Load personal discovery
	personalDiscoveryChat := &Filter{
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

	s.filters[personalDiscoveryChat.ChatID] = personalDiscoveryChat

	if s.genericDiscoveryTopicEnabled {
		// Load generic discovery topic.
		discoveryChat := &Filter{
			ChatID:    discoveryTopic,
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

		s.filters[discoveryChat.ChatID] = discoveryChat

		return []*Filter{discoveryChat, personalDiscoveryChat}, nil
	}

	return []*Filter{personalDiscoveryChat}, nil
}

// LoadPublic adds a filter for a public chat.
func (s *filtersManager) LoadPublic(chatID string) (*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if chat, ok := s.filters[chatID]; ok {
		return chat, nil
	}

	filterAndTopic, err := s.addSymmetric(chatID)
	if err != nil {
		return nil, err
	}

	chat := &Filter{
		ChatID:   chatID,
		FilterID: filterAndTopic.FilterID,
		SymKeyID: filterAndTopic.SymKeyID,
		Topic:    filterAndTopic.Topic,
		Listen:   true,
		OneToOne: false,
	}

	s.filters[chatID] = chat

	return chat, nil
}

// LoadContactCode creates a filter for the advertise topic for a given public key.
func (s *filtersManager) LoadContactCode(pubKey *ecdsa.PublicKey) (*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := contactCodeTopic(pubKey)

	if _, ok := s.filters[chatID]; ok {
		return s.filters[chatID], nil
	}

	contactCodeFilter, err := s.addSymmetric(chatID)
	if err != nil {
		return nil, err
	}

	chat := &Filter{
		ChatID:   chatID,
		FilterID: contactCodeFilter.FilterID,
		Topic:    contactCodeFilter.Topic,
		SymKeyID: contactCodeFilter.SymKeyID,
		Identity: publicKeyToStr(pubKey),
		Listen:   true,
	}

	s.filters[chatID] = chat
	return chat, nil
}

// addSymmetric adds a symmetric key filter
func (s *filtersManager) addSymmetric(chatID string) (*whisperFilter, error) {
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

	f := s.whisper.CreateFilterWrapper(
		nil, symKey,
		minPow,
		topics,
		s.whisper.NewMessageStore())

	id, err := s.whisper.Subscribe(f)
	if err != nil {
		return nil, err
	}

	return &whisperFilter{
		FilterID: id,
		SymKeyID: symKeyID,
		Topic:    whispertypes.BytesToTopic(topic),
	}, nil
}

// addAsymmetricFilter adds a filter with our private key
// and set minPow according to the listen parameter.
func (s *filtersManager) addAsymmetric(chatID string, listen bool) (*whisperFilter, error) {
	var (
		err error
		pow = 1.0 // use PoW high enough to discard all messages for the filter
	)

	if listen {
		pow = minPow
	}

	topic := toTopic(chatID)
	topics := [][]byte{topic}

	f := s.whisper.CreateFilterWrapper(
		s.privateKey, nil,
		pow,
		topics,
		s.whisper.NewMessageStore())

	id, err := s.whisper.Subscribe(f)
	if err != nil {
		return nil, err
	}
	return &whisperFilter{FilterID: id, Topic: whispertypes.BytesToTopic(topic)}, nil
}

// GetNegotiated returns a negotiated chat given an identity
func (s *filtersManager) GetNegotiated(identity *ecdsa.PublicKey) *Filter {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.filters[negotiatedTopic(identity)]
}

func toTopic(s string) []byte {
	return crypto.Keccak256([]byte(s))[:whispertypes.TopicLength]
}

// ToTopic converts a string to a whisper topic.
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

func ContactCodeTopic(publicKey *ecdsa.PublicKey) string {
	return contactCodeTopic(publicKey)
}

func contactCodeTopic(publicKey *ecdsa.PublicKey) string {
	return "0x" + publicKeyToStr(publicKey) + "-contact-code"
}

func negotiatedTopic(publicKey *ecdsa.PublicKey) string {
	return "0x" + publicKeyToStr(publicKey) + "-negotiated"
}
