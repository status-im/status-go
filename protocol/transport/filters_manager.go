package transport

import (
	"bytes"
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
	UnsubscribeMany(ids []string) error
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

func (f *FiltersManager) Init(
	chatIDs []string,
	publicKeys []*ecdsa.PublicKey,
) ([]*Filter, error) {

	// Load our contact code.
	_, err := f.LoadContactCode(&f.privateKey.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load contact code")
	}

	// Load partitioned topic.
	_, err = f.loadMyPartitioned()
	if err != nil {
		return nil, err
	}

	// Add discovery topic.
	_, err = f.LoadDiscovery()
	if err != nil {
		return nil, err
	}

	// Add public, one-to-one and negotiated filters.
	for _, chatID := range chatIDs {
		_, err := f.LoadPublic(chatID)
		if err != nil {
			return nil, err
		}
	}

	for _, publicKey := range publicKeys {
		_, err := f.LoadContactCode(publicKey)
		if err != nil {
			return nil, err
		}
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	var allFilters []*Filter
	for _, f := range f.filters {
		allFilters = append(allFilters, f)
	}
	return allFilters, nil
}

func (f *FiltersManager) InitPublicFilters(chatIDs []string) ([]*Filter, error) {
	var filters []*Filter
	// Add public, one-to-one and negotiated filters.
	for _, chatID := range chatIDs {
		f, err := f.LoadPublic(chatID)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return filters, nil
}

func (f *FiltersManager) InitCommunityFilters(pks []*ecdsa.PrivateKey) ([]*Filter, error) {
	var filters []*Filter
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, pk := range pks {

		identityStr := PublicKeyToStr(&pk.PublicKey)
		rawFilter, err := f.addAsymmetric(identityStr, pk, true)
		if err != nil {
			return nil, err

		}
		filterID := identityStr + "-admin"
		filter := &Filter{
			ChatID:   filterID,
			FilterID: rawFilter.FilterID,
			Topic:    rawFilter.Topic,
			Identity: identityStr,
			Listen:   true,
			OneToOne: true,
		}

		f.filters[filterID] = filter

		filters = append(filters, filter)
	}
	return filters, nil
}

// DEPRECATED
func (f *FiltersManager) InitWithFilters(filters []*Filter) ([]*Filter, error) {
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

	return f.Init(chatIDs, publicKeys)
}

func (f *FiltersManager) Reset() error {
	var filters []*Filter

	f.mutex.Lock()
	for _, f := range f.filters {
		filters = append(filters, f)
	}
	f.mutex.Unlock()

	return f.Remove(filters...)
}

func (f *FiltersManager) Filters() (result []*Filter) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, f := range f.filters {
		result = append(result, f)
	}

	return
}

func (f *FiltersManager) Filter(chatID string) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.filters[chatID]
}

// FilterByFilterID returns a Filter with a given Whisper filter ID.
func (f *FiltersManager) FilterByFilterID(filterID string) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	for _, f := range f.filters {
		if f.FilterID == filterID {
			return f
		}
	}
	return nil
}

func (f *FiltersManager) FilterByTopic(topic []byte) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	for _, f := range f.filters {
		if bytes.Equal(types.TopicTypeToByteArray(f.Topic), topic) {
			return f
		}
	}
	return nil
}

// FiltersByIdentities returns an array of filters for given list of public keys
func (f *FiltersManager) FiltersByIdentities(identities []string) []*Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	identitiesMap := make(map[string]bool)

	for _, identity := range identities {
		identitiesMap[identity] = true
	}

	var filters []*Filter

	for _, filter := range f.filters {
		// Pre-pend 0x before comparing
		if identitiesMap["0x"+filter.Identity] {
			filters = append(filters, filter)
		}
	}
	return filters
}

// FilterByChatID returns a Filter for given chat id
func (f *FiltersManager) FilterByChatID(chatID string) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.filters[chatID]
}

// Remove remove all the filters associated with a chat/identity
func (f *FiltersManager) Remove(filters ...*Filter) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, filter := range filters {
		if err := f.service.Unsubscribe(filter.FilterID); err != nil {
			return err
		}
		if filter.SymKeyID != "" {
			f.service.DeleteSymKey(filter.SymKeyID)
		}
		delete(f.filters, filter.ChatID)
	}

	return nil
}

// Remove remove all the filters associated with a chat/identity
func (f *FiltersManager) RemoveNoListenFilters() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	var filterIDs []string
	var filters []*Filter

	for _, f := range filters {
		if !f.Listen {
			filterIDs = append(filterIDs, f.FilterID)
			filters = append(filters, f)
		}
	}
	if err := f.service.UnsubscribeMany(filterIDs); err != nil {
		return err
	}

	for _, filter := range filters {
		if filter.SymKeyID != "" {
			f.service.DeleteSymKey(filter.SymKeyID)
		}
		delete(f.filters, filter.ChatID)
	}

	return nil
}

// Remove remove all the filters associated with a chat/identity
func (f *FiltersManager) RemoveFilterByChatID(chatID string) (*Filter, error) {
	f.mutex.Lock()
	filter, ok := f.filters[chatID]
	f.mutex.Unlock()

	if !ok {
		return nil, nil
	}

	err := f.Remove(filter)
	if err != nil {
		return nil, err
	}

	return filter, nil
}

// LoadPartitioned creates a filter for a partitioned topic.
func (f *FiltersManager) LoadPartitioned(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen bool) (*Filter, error) {
	return f.loadPartitioned(publicKey, identity, listen, false)
}

// LoadEphemeral creates a filter for a partitioned/personal topic.
func (f *FiltersManager) LoadEphemeral(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen bool) (*Filter, error) {
	return f.loadPartitioned(publicKey, identity, listen, true)
}

// LoadPersonal creates a filter for a personal topic.
func (f *FiltersManager) LoadPersonal(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen bool) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	chatID := PersonalDiscoveryTopic(publicKey)
	if _, ok := f.filters[chatID]; ok {
		return f.filters[chatID], nil
	}

	// We set up a filter so we can publish,
	// but we discard envelopes if listen is false.
	filter, err := f.addAsymmetric(chatID, identity, listen)
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

	f.filters[chatID] = chat

	return chat, nil

}

func (f *FiltersManager) loadMyPartitioned() (*Filter, error) {
	return f.loadPartitioned(&f.privateKey.PublicKey, f.privateKey, true, false)
}

func (f *FiltersManager) loadPartitioned(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen, ephemeral bool) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	chatID := PartitionedTopic(publicKey)
	if _, ok := f.filters[chatID]; ok {
		return f.filters[chatID], nil
	}

	// We set up a filter so we can publish,
	// but we discard envelopes if listen is false.
	filter, err := f.addAsymmetric(chatID, identity, listen)
	if err != nil {
		return nil, err
	}

	chat := &Filter{
		ChatID:    chatID,
		FilterID:  filter.FilterID,
		Topic:     filter.Topic,
		Identity:  PublicKeyToStr(publicKey),
		Listen:    listen,
		Ephemeral: ephemeral,
		OneToOne:  true,
	}

	f.filters[chatID] = chat

	return chat, nil
}

// LoadNegotiated loads a negotiated secret as a filter.
func (f *FiltersManager) LoadNegotiated(secret types.NegotiatedSecret) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	chatID := NegotiatedTopic(secret.PublicKey)

	if _, ok := f.filters[chatID]; ok {
		return f.filters[chatID], nil
	}

	keyString := hex.EncodeToString(secret.Key)
	filter, err := f.addSymmetric(keyString)
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

	f.filters[chat.ChatID] = chat

	return chat, nil
}

// LoadDiscovery adds 1 discovery filter
// for the personal discovery topic.
func (f *FiltersManager) LoadDiscovery() ([]*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	personalDiscoveryTopic := PersonalDiscoveryTopic(&f.privateKey.PublicKey)

	// Check if filters are already loaded.
	var result []*Filter

	expectedTopicCount := 1

	if chat, ok := f.filters[personalDiscoveryTopic]; ok {
		result = append(result, chat)
	}

	if len(result) == expectedTopicCount {
		return result, nil
	}

	identityStr := PublicKeyToStr(&f.privateKey.PublicKey)

	// Load personal discovery
	personalDiscoveryChat := &Filter{
		ChatID:    personalDiscoveryTopic,
		Identity:  identityStr,
		Discovery: true,
		Listen:    true,
		OneToOne:  true,
	}

	discoveryResponse, err := f.addAsymmetric(personalDiscoveryChat.ChatID, f.privateKey, true)
	if err != nil {
		return nil, err
	}

	personalDiscoveryChat.Topic = discoveryResponse.Topic
	personalDiscoveryChat.FilterID = discoveryResponse.FilterID

	f.filters[personalDiscoveryChat.ChatID] = personalDiscoveryChat

	return []*Filter{personalDiscoveryChat}, nil
}

func (f *FiltersManager) PersonalTopicFilter() *Filter {
	personalDiscoveryTopic := PersonalDiscoveryTopic(&f.privateKey.PublicKey)

	return f.filters[personalDiscoveryTopic]
}

// LoadPublic adds a filter for a public chat.
func (f *FiltersManager) LoadPublic(chatID string) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if chat, ok := f.filters[chatID]; ok {
		return chat, nil
	}

	filterAndTopic, err := f.addSymmetric(chatID)
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

	f.filters[chatID] = chat

	return chat, nil
}

// LoadContactCode creates a filter for the advertise topic for a given public key.
func (f *FiltersManager) LoadContactCode(pubKey *ecdsa.PublicKey) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	chatID := ContactCodeTopic(pubKey)

	if _, ok := f.filters[chatID]; ok {
		return f.filters[chatID], nil
	}

	contactCodeFilter, err := f.addSymmetric(chatID)
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

	f.filters[chatID] = chat
	return chat, nil
}

// addSymmetric adds a symmetric key filter
func (f *FiltersManager) addSymmetric(chatID string) (*RawFilter, error) {
	var symKeyID string
	var err error

	topic := ToTopic(chatID)
	topics := [][]byte{topic}

	symKey, ok := f.keys[chatID]
	if ok {
		symKeyID, err = f.service.AddSymKeyDirect(symKey)
		if err != nil {
			return nil, err
		}
	} else {
		symKeyID, err = f.service.AddSymKeyFromPassword(chatID)
		if err != nil {
			return nil, err
		}
		if symKey, err = f.service.GetSymKey(symKeyID); err != nil {
			return nil, err
		}
		f.keys[chatID] = symKey

		err = f.persistence.Add(chatID, symKey)
		if err != nil {
			return nil, err
		}
	}

	id, err := f.service.Subscribe(&types.SubscriptionOptions{
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
func (f *FiltersManager) addAsymmetric(chatID string, identity *ecdsa.PrivateKey, listen bool) (*RawFilter, error) {
	var (
		err error
		pow = 1.0 // use PoW high enough to discard all messages for the filter
	)

	if listen {
		pow = minPow
	}

	topic := ToTopic(chatID)
	topics := [][]byte{topic}

	privateKeyID, err := f.service.AddKeyPair(identity)
	if err != nil {
		return nil, err
	}

	id, err := f.service.Subscribe(&types.SubscriptionOptions{
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
func (f *FiltersManager) GetNegotiated(identity *ecdsa.PublicKey) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.filters[NegotiatedTopic(identity)]
}
