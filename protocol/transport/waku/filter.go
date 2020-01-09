package waku

import (
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"math/big"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
)

type Filter = transport.Filter

const (
	discoveryTopic = "contact-discovery"
)

var (
	// The number of partitions.
	nPartitions = big.NewInt(5000)
	minPow      = 0.0
)

type rawFilter struct {
	FilterID string
	Topic    types.TopicType
	SymKeyID string
}

type filtersManager struct {
	waku        types.Waku
	persistence *sqlitePersistence
	privateKey  *ecdsa.PrivateKey
	keys        map[string][]byte // a cache of symmetric manager derived from passwords
	logger      *zap.Logger

	genericDiscoveryTopicEnabled bool

	mutex   sync.Mutex
	filters map[string]*Filter
}

// newFiltersManager returns a new filtersManager.
func newFiltersManager(db *sql.DB, w types.Waku, privateKey *ecdsa.PrivateKey, logger *zap.Logger) (*filtersManager, error) {
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
		waku:        w,
		persistence: persistence,
		keys:        keys,
		filters:     make(map[string]*Filter),
		logger:      logger.With(zap.Namespace("filtersManager")),
	}, nil
}

func (s *filtersManager) Init(
	chatIDs []string,
	publicKeys []*ecdsa.PublicKey,
) ([]*Filter, error) {
	logger := s.logger.With(zap.String("site", "Init"))

	logger.Info("initializing")

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
func (s *filtersManager) InitWithFilters(filters []*Filter) ([]*Filter, error) {
	var (
		chatIDs    []string
		publicKeys []*ecdsa.PublicKey
	)

	for _, filter := range filters {
		if filter.Identity != "" && filter.OneToOne {
			publicKey, err := transport.StrToPublicKey(filter.Identity)
			if err != nil {
				return nil, err
			}
			publicKeys = append(publicKeys, publicKey)
		} else if filter.ChatID != "" {
			chatIDs = append(chatIDs, filter.ChatID)
		}
	}

	return s.Init(chatIDs, publicKeys)
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

	identityStr := transport.PublicKeyToStr(publicKey)

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
		if err := s.waku.Unsubscribe(f.FilterID); err != nil {
			return err
		}
		if f.SymKeyID != "" {
			s.waku.DeleteSymKey(f.SymKeyID)
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

	chatID := transport.PartitionedTopic(publicKey)
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
		Identity: transport.PublicKeyToStr(publicKey),
		Listen:   listen,
		OneToOne: true,
	}

	s.filters[chatID] = chat

	return chat, nil
}

// LoadNegotiated loads a negotiated secret as a filter.
func (s *filtersManager) LoadNegotiated(secret types.NegotiatedSecret) (*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := transport.NegotiatedTopic(secret.PublicKey)

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
		Identity:   transport.PublicKeyToStr(secret.PublicKey),
		Negotiated: true,
		Listen:     true,
		OneToOne:   true,
	}

	s.filters[chat.ChatID] = chat

	return chat, nil
}

// LoadDiscovery adds 1 discovery filter
// for the personal discovery topic.
func (s *filtersManager) LoadDiscovery() ([]*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	personalDiscoveryTopic := transport.PersonalDiscoveryTopic(&s.privateKey.PublicKey)

	// Check if filters are already loaded.
	var result []*Filter

	expectedTopicCount := 1

	if chat, ok := s.filters[personalDiscoveryTopic]; ok {
		result = append(result, chat)
	}

	if len(result) == expectedTopicCount {
		return result, nil
	}

	identityStr := transport.PublicKeyToStr(&s.privateKey.PublicKey)

	// Load personal discovery
	personalDiscoveryChat := &Filter{
		ChatID:    personalDiscoveryTopic,
		Identity:  identityStr,
		Discovery: true,
		Listen:    true,
		OneToOne:  true,
	}

	discoveryResponse, err := s.addAsymmetric(personalDiscoveryChat.ChatID, true)
	if err != nil {
		return nil, err
	}

	personalDiscoveryChat.Topic = discoveryResponse.Topic
	personalDiscoveryChat.FilterID = discoveryResponse.FilterID

	s.filters[personalDiscoveryChat.ChatID] = personalDiscoveryChat

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

	chatID := transport.ContactCodeTopic(pubKey)

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
		Identity: transport.PublicKeyToStr(pubKey),
		Listen:   true,
	}

	s.filters[chatID] = chat
	return chat, nil
}

// addSymmetric adds a symmetric key filter
func (s *filtersManager) addSymmetric(chatID string) (*rawFilter, error) {
	var symKeyID string
	var err error

	topic := transport.ToTopic(chatID)
	topics := [][]byte{topic}

	symKey, ok := s.keys[chatID]
	if ok {
		symKeyID, err = s.waku.AddSymKeyDirect(symKey)
		if err != nil {
			return nil, err
		}
	} else {
		symKeyID, err = s.waku.AddSymKeyFromPassword(chatID)
		if err != nil {
			return nil, err
		}
		if symKey, err = s.waku.GetSymKey(symKeyID); err != nil {
			return nil, err
		}
		s.keys[chatID] = symKey

		err = s.persistence.Add(chatID, symKey)
		if err != nil {
			return nil, err
		}
	}

	id, err := s.waku.Subscribe(&types.SubscriptionOptions{
		SymKeyID: symKeyID,
		PoW:      minPow,
		Topics:   topics,
	})
	if err != nil {
		return nil, err
	}

	return &rawFilter{
		FilterID: id,
		SymKeyID: symKeyID,
		Topic:    types.BytesToTopic(topic),
	}, nil
}

// addAsymmetricFilter adds a filter with our private key
// and set minPow according to the listen parameter.
func (s *filtersManager) addAsymmetric(chatID string, listen bool) (*rawFilter, error) {
	var (
		err error
		pow = 1.0 // use PoW high enough to discard all messages for the filter
	)

	if listen {
		pow = minPow
	}

	topic := transport.ToTopic(chatID)
	topics := [][]byte{topic}

	privateKeyID, err := s.waku.AddKeyPair(s.privateKey)
	if err != nil {
		return nil, err
	}

	id, err := s.waku.Subscribe(&types.SubscriptionOptions{
		PrivateKeyID: privateKeyID,
		PoW:          pow,
		Topics:       topics,
	})
	if err != nil {
		return nil, err
	}
	return &rawFilter{FilterID: id, Topic: types.BytesToTopic(topic)}, nil
}

// GetNegotiated returns a negotiated chat given an identity
func (s *filtersManager) GetNegotiated(identity *ecdsa.PublicKey) *Filter {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.filters[transport.NegotiatedTopic(identity)]
}
