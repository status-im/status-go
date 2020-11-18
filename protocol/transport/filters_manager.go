package transport

import (
	"crypto/ecdsa"
	"encoding/hex"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
)

const (
	minPow = 0.0
)

type RawFilter struct {
	FilterID string
	Topic    types.TopicType
	SymKeyID string
}

type KeysPersistence interface {
	All() (map[string][]byte, error)
	Add(chatID string, key []byte) error
}

type FiltersService interface {
	AddKeyPair(key *ecdsa.PrivateKey) (string, error)
	DeleteKeyPair(keyID string) bool

	AddSymKeyDirect(key []byte) (string, error)
	AddSymKeyFromPassword(password string) (string, error)
	GetSymKey(id string) ([]byte, error)
	DeleteSymKey(id string) bool

	Subscribe(opts *types.SubscriptionOptions) (string, error)
	Unsubscribe(id string) error
}

type FiltersManager struct {
	service     FiltersService
	persistence KeysPersistence
	privateKey  *ecdsa.PrivateKey
	keys        map[string][]byte // a cache of symmetric manager derived from passwords
	logger      *zap.Logger
	mutex       sync.Mutex
	filters     map[string]*Filter
}

// NewFiltersManager returns a new filtersManager.
func NewFiltersManager(persistence KeysPersistence, service FiltersService, privateKey *ecdsa.PrivateKey, logger *zap.Logger) (*FiltersManager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	keys, err := persistence.All()
	if err != nil {
		return nil, err
	}

	return &FiltersManager{
		privateKey:  privateKey,
		service:     service,
		persistence: persistence,
		keys:        keys,
		filters:     make(map[string]*Filter),
		logger:      logger.With(zap.Namespace("filtersManager")),
	}, nil
}

func (s *FiltersManager) Init(
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

func (s *FiltersManager) InitPublicFilters(chatIDs []string) ([]*Filter, error) {
	var filters []*Filter
	// Add public, one-to-one and negotiated filters.
	for _, chatID := range chatIDs {
		f, err := s.LoadPublic(chatID)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return filters, nil
}

// DEPRECATED
func (s *FiltersManager) InitWithFilters(filters []*Filter) ([]*Filter, error) {
	var (
		chatIDs    []string
		publicKeys []*ecdsa.PublicKey
	)

	for _, filter := range filters {
		if filter.Identity != "" && filter.OneToOne {
			publicKey, err := StrToPublicKey(filter.Identity)
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

func (s *FiltersManager) Reset() error {
	var filters []*Filter

	s.mutex.Lock()
	for _, f := range s.filters {
		filters = append(filters, f)
	}
	s.mutex.Unlock()

	return s.Remove(filters...)
}

func (s *FiltersManager) Filters() (result []*Filter) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, f := range s.filters {
		result = append(result, f)
	}

	return
}

func (s *FiltersManager) Filter(chatID string) *Filter {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.filters[chatID]
}

// FilterByFilterID returns a Filter with a given Whisper filter ID.
func (s *FiltersManager) FilterByFilterID(filterID string) *Filter {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, f := range s.filters {
		if f.FilterID == filterID {
			return f
		}
	}
	return nil
}

func (s *FiltersManager) FiltersByPublicKey(publicKey *ecdsa.PublicKey) (result []*Filter) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	identityStr := PublicKeyToStr(publicKey)

	for _, f := range s.filters {
		if f.Identity == identityStr {
			result = append(result, f)
		}
	}

	return
}

// Remove remove all the filters associated with a chat/identity
func (s *FiltersManager) Remove(filters ...*Filter) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, f := range filters {
		if err := s.service.Unsubscribe(f.FilterID); err != nil {
			return err
		}
		if f.SymKeyID != "" {
			s.service.DeleteSymKey(f.SymKeyID)
		}
		delete(s.filters, f.ChatID)
	}

	return nil
}

// Remove remove all the filters associated with a chat/identity
func (s *FiltersManager) RemoveFilterByChatID(chatID string) error {
	s.mutex.Lock()
	filter, ok := s.filters[chatID]
	s.mutex.Unlock()

	if !ok {
		return nil
	}

	return s.Remove(filter)
}

// LoadPartitioned creates a filter for a partitioned topic.
func (s *FiltersManager) LoadPartitioned(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen bool) (*Filter, error) {
	return s.loadPartitioned(publicKey, identity, listen)
}

func (s *FiltersManager) loadMyPartitioned() (*Filter, error) {
	return s.loadPartitioned(&s.privateKey.PublicKey, s.privateKey, true)
}

func (s *FiltersManager) loadPartitioned(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen bool) (*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := PartitionedTopic(publicKey)
	if _, ok := s.filters[chatID]; ok {
		return s.filters[chatID], nil
	}

	// We set up a filter so we can publish,
	// but we discard envelopes if listen is false.
	filter, err := s.addAsymmetric(chatID, identity, listen)
	if err != nil {
		return nil, err
	}

	chat := &Filter{
		ChatID:   chatID,
		FilterID: filter.FilterID,
		Topic:    filter.Topic,
		Identity: PublicKeyToStr(publicKey),
		Listen:   listen,
		OneToOne: true,
	}

	s.filters[chatID] = chat

	return chat, nil
}

// LoadNegotiated loads a negotiated secret as a filter.
func (s *FiltersManager) LoadNegotiated(secret types.NegotiatedSecret) (*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := NegotiatedTopic(secret.PublicKey)

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
		Identity:   PublicKeyToStr(secret.PublicKey),
		Negotiated: true,
		Listen:     true,
		OneToOne:   true,
	}

	s.filters[chat.ChatID] = chat

	return chat, nil
}

// LoadDiscovery adds 1 discovery filter
// for the personal discovery topic.
func (s *FiltersManager) LoadDiscovery() ([]*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	personalDiscoveryTopic := PersonalDiscoveryTopic(&s.privateKey.PublicKey)

	// Check if filters are already loaded.
	var result []*Filter

	expectedTopicCount := 1

	if chat, ok := s.filters[personalDiscoveryTopic]; ok {
		result = append(result, chat)
	}

	if len(result) == expectedTopicCount {
		return result, nil
	}

	identityStr := PublicKeyToStr(&s.privateKey.PublicKey)

	// Load personal discovery
	personalDiscoveryChat := &Filter{
		ChatID:    personalDiscoveryTopic,
		Identity:  identityStr,
		Discovery: true,
		Listen:    true,
		OneToOne:  true,
	}

	discoveryResponse, err := s.addAsymmetric(personalDiscoveryChat.ChatID, s.privateKey, true)
	if err != nil {
		return nil, err
	}

	personalDiscoveryChat.Topic = discoveryResponse.Topic
	personalDiscoveryChat.FilterID = discoveryResponse.FilterID

	s.filters[personalDiscoveryChat.ChatID] = personalDiscoveryChat

	return []*Filter{personalDiscoveryChat}, nil
}

// LoadPublic adds a filter for a public chat.
func (s *FiltersManager) LoadPublic(chatID string) (*Filter, error) {
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
func (s *FiltersManager) LoadContactCode(pubKey *ecdsa.PublicKey) (*Filter, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	chatID := ContactCodeTopic(pubKey)

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
		Identity: PublicKeyToStr(pubKey),
		Listen:   true,
	}

	s.filters[chatID] = chat
	return chat, nil
}

// addSymmetric adds a symmetric key filter
func (s *FiltersManager) addSymmetric(chatID string) (*RawFilter, error) {
	var symKeyID string
	var err error

	topic := ToTopic(chatID)
	topics := [][]byte{topic}

	symKey, ok := s.keys[chatID]
	if ok {
		symKeyID, err = s.service.AddSymKeyDirect(symKey)
		if err != nil {
			return nil, err
		}
	} else {
		symKeyID, err = s.service.AddSymKeyFromPassword(chatID)
		if err != nil {
			return nil, err
		}
		if symKey, err = s.service.GetSymKey(symKeyID); err != nil {
			return nil, err
		}
		s.keys[chatID] = symKey

		err = s.persistence.Add(chatID, symKey)
		if err != nil {
			return nil, err
		}
	}

	id, err := s.service.Subscribe(&types.SubscriptionOptions{
		SymKeyID: symKeyID,
		PoW:      minPow,
		Topics:   topics,
	})
	if err != nil {
		return nil, err
	}

	return &RawFilter{
		FilterID: id,
		SymKeyID: symKeyID,
		Topic:    types.BytesToTopic(topic),
	}, nil
}

// addAsymmetricFilter adds a filter with our private key
// and set minPow according to the listen parameter.
func (s *FiltersManager) addAsymmetric(chatID string, identity *ecdsa.PrivateKey, listen bool) (*RawFilter, error) {
	var (
		err error
		pow = 1.0 // use PoW high enough to discard all messages for the filter
	)

	if listen {
		pow = minPow
	}

	topic := ToTopic(chatID)
	topics := [][]byte{topic}

	privateKeyID, err := s.service.AddKeyPair(identity)
	if err != nil {
		return nil, err
	}

	id, err := s.service.Subscribe(&types.SubscriptionOptions{
		PrivateKeyID: privateKeyID,
		PoW:          pow,
		Topics:       topics,
	})
	if err != nil {
		return nil, err
	}
	return &RawFilter{FilterID: id, Topic: types.BytesToTopic(topic)}, nil
}

// GetNegotiated returns a negotiated chat given an identity
func (s *FiltersManager) GetNegotiated(identity *ecdsa.PublicKey) *Filter {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.filters[NegotiatedTopic(identity)]
}
