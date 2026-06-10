package transport

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/pbkdf2"

	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	wakuv2 "github.com/status-im/status-go/pkg/messaging/waku"
	wakucommon "github.com/status-im/status-go/pkg/messaging/waku/common"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

type FiltersManager struct {
	persistence KeysPersistence
	privateKey  *ecdsa.PrivateKey
	keys        map[string][]byte // a cache of symmetric manager derived from passwords
	logger      *zap.Logger
	mutex       sync.Mutex
	filters     map[string]*Filter
	// asymKeys and symKeys map a filter's FilterID to the key material used to
	// encode and decode messages on it (exactly one of them has an entry per
	// keyed filter). Runtime-only: FilterIDs are regenerated when filters are
	// recreated on startup, and the password-derived symmetric key material
	// itself is cached by chat id in keys and persisted.
	asymKeys map[string]*ecdsa.PrivateKey
	symKeyMu sync.RWMutex
	symKeys  map[string][]byte
}

// NewFiltersManager returns a new filtersManager.
func NewFiltersManager(persistence KeysPersistence, privateKey *ecdsa.PrivateKey, logger *zap.Logger) (*FiltersManager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	keys, err := persistence.All()
	if err != nil {
		return nil, err
	}

	return &FiltersManager{
		privateKey:  privateKey,
		persistence: persistence,
		keys:        keys,
		filters:     make(map[string]*Filter),
		asymKeys:    make(map[string]*ecdsa.PrivateKey),
		symKeys:     make(map[string][]byte),
		logger:      logger.With(zap.Namespace("filtersManager")),
	}, nil
}

// SymKey returns the symmetric key used to encode and decode messages on the
// filter with the given FilterID, if it is a symmetric filter.
func (f *FiltersManager) SymKey(filterID string) ([]byte, bool) {
	f.symKeyMu.RLock()
	defer f.symKeyMu.RUnlock()
	key, ok := f.symKeys[filterID]
	return key, ok
}

// addSymKey stores the symmetric key of the filter with the given FilterID.
func (f *FiltersManager) addSymKey(filterID string, key []byte) error {
	if len(key) != wakucommon.AESKeyLength {
		return errors.Errorf("wrong key size: %d", len(key))
	}

	f.symKeyMu.Lock()
	defer f.symKeyMu.Unlock()
	f.symKeys[filterID] = key
	return nil
}

func (f *FiltersManager) deleteSymKey(filterID string) {
	f.symKeyMu.Lock()
	defer f.symKeyMu.Unlock()
	delete(f.symKeys, filterID)
}

// WatchersByTopic returns the filters interested in the given (pubsubTopic,
// contentTopic) pair. A filter with no explicit pubsub topic listens on the
// default shard, so it is normalized before comparison — preserving the
// (pubsub, content) routing the waku layer used. A single content topic can be
// shared by distinct chats (content topics are a 4-byte hash of the chat id),
// so the caller must still confirm the match by decoding with each candidate's
// own key.
func (f *FiltersManager) WatchersByTopic(pubsubTopic, contentTopic string) []*Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	var res []*Filter
	for _, filter := range f.filters {
		if filter.ContentTopic.ContentTopic() != contentTopic {
			continue
		}
		filterPubsubTopic := filter.PubsubTopic
		if filterPubsubTopic == "" {
			filterPubsubTopic = wakuv2.DefaultShardPubsubTopic()
		}
		if filterPubsubTopic == pubsubTopic {
			res = append(res, filter)
		}
	}
	return res
}

// AsymKey returns the asymmetric private key used to decode messages received
// on the filter with the given FilterID, if it is an asymmetric filter.
func (f *FiltersManager) AsymKey(filterID string) (*ecdsa.PrivateKey, bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	k, ok := f.asymKeys[filterID]
	return k, ok
}

func (f *FiltersManager) Init(
	filtersToInit []FiltersToInitialize,
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
	for _, fi := range filtersToInit {
		_, err := f.LoadPublic(fi.ChatID, fi.PubsubTopic)
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

type FiltersToInitialize struct {
	ChatID      string
	PubsubTopic string
}

func (f *FiltersManager) InitPublicFilters(publicFiltersToInit []FiltersToInitialize) ([]*Filter, error) {
	var filters []*Filter
	// Add public, one-to-one and negotiated filters.
	for _, pf := range publicFiltersToInit {
		f, err := f.LoadPublic(pf.ChatID, pf.PubsubTopic)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return filters, nil
}

type CommunityFilterToInitialize struct {
	PrivKey *ecdsa.PrivateKey
}

func (f *FiltersManager) InitCommunityFilters(communityFiltersToInitialize []CommunityFilterToInitialize) ([]*Filter, error) {
	var filters []*Filter
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, communityFilter := range communityFiltersToInitialize {
		// to satisfy gosec: C601 checks
		cf := communityFilter
		if cf.PrivKey == nil {
			continue
		}

		topics := make([]string, 0)
		topics = append(topics, wakuv2.DefaultShardPubsubTopic())
		topics = append(topics, wakuv2.DefaultNonProtectedPubsubTopic())

		for _, pubsubTopic := range topics {
			pk := &cf.PrivKey.PublicKey
			identityStr := PublicKeyToStr(pk)
			id, topic, err := f.addAsymmetric(identityStr, cf.PrivKey)
			if err != nil {
				f.logger.Debug("could not register community filter", zap.Error(err))
				return nil, err

			}
			filterID := identityStr + "-admin" + pubsubTopic
			filter := &Filter{
				ChatID:       filterID,
				FilterID:     id,
				PubsubTopic:  pubsubTopic,
				ContentTopic: topic,
				Identity:     identityStr,
				Listen:       true,
				OneToOne:     true,
			}

			f.filters[filterID] = filter

			f.logger.Debug("registering filter for", zap.String("chatID", filterID), zap.String("type", "community"), zap.String("pubsubTopic", pubsubTopic), zap.String("contentTopic", topic.String()))

			filters = append(filters, filter)
		}
	}
	return filters, nil
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
		if bytes.Equal(types.TopicTypeToByteArray(f.ContentTopic), topic) {
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

// Remove removes all the filtersToRemove
func (f *FiltersManager) Remove(filtersToRemove ...*Filter) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, filter := range filtersToRemove {
		f.deleteSymKey(filter.FilterID)
		delete(f.asymKeys, filter.FilterID)
		for k, v := range f.filters {
			if filter.FilterID == v.FilterID {
				delete(f.filters, k)
			}
		}
	}

	return nil
}

// Remove remove all the filters associated with a chat/identity
func (f *FiltersManager) RemoveNoListenFilters() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	var filters []*Filter

	for _, v := range f.filters {
		if !v.Listen {
			filters = append(filters, v)
		}
	}

	for _, filter := range filters {
		f.deleteSymKey(filter.FilterID)
		delete(f.asymKeys, filter.FilterID)
		for k, v := range f.filters {
			if filter.FilterID == v.FilterID {
				delete(f.filters, k)
			}
		}
	}

	return nil
}

// RemoveFilterByChatID removes the filters associated with a chat/identity
func (f *FiltersManager) RemoveFilterByChatID(chatID string) (*Filter, error) {
	// TODO: remove subscriptions from waku2 if required. Might need to be implemented in transport

	toRemove := make([]*Filter, 0)
	f.mutex.Lock()
	for _, filter := range f.filters {
		if filter.ChatID == chatID {
			toRemove = append(toRemove, filter)
		}
	}
	f.mutex.Unlock()

	if len(toRemove) == 0 {
		return nil, nil
	}

	err := f.Remove(toRemove...)
	if err != nil {
		return nil, err
	}

	// TODO temporary so not changing the return type, otherwise we should return a slice
	return toRemove[0], nil
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
	id, topic, err := f.addAsymmetric(chatID, identity)
	if err != nil {
		f.logger.Debug("could not register personal topic filter", zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		FilterID:     id,
		ContentTopic: topic,
		Identity:     PublicKeyToStr(publicKey),
		Listen:       listen,
		OneToOne:     true,
	}

	f.filters[chatID] = chat

	f.logger.Debug("registering filter for", zap.String("chatID", chatID), zap.String("type", "personal"), zap.String("topic", topic.String()))

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
	id, topic, err := f.addAsymmetric(chatID, identity)
	if err != nil {
		f.logger.Debug("could not register partitioned topic", zap.String("chatID", chatID), zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		FilterID:     id,
		ContentTopic: topic,
		Identity:     PublicKeyToStr(publicKey),
		Listen:       listen,
		Ephemeral:    ephemeral,
		OneToOne:     true,
	}

	f.filters[chatID] = chat

	f.logger.Debug("registering filter for", zap.String("chatID", chatID), zap.String("type", "partitioned"), zap.String("topic", topic.String()))

	return chat, nil
}

// LoadNegotiated loads a negotiated secret as a filter.
func (f *FiltersManager) LoadNegotiated(secret messagingtypes.NegotiatedSecret) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	chatID := NegotiatedTopic(secret.PublicKey)

	if _, ok := f.filters[chatID]; ok {
		return f.filters[chatID], nil
	}

	keyString := hex.EncodeToString(secret.Key)
	id, topic, err := f.addSymmetric(keyString)
	if err != nil {
		f.logger.Debug("could not register negotiated topic", zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		ContentTopic: topic,
		FilterID:     id,
		Identity:     PublicKeyToStr(secret.PublicKey),
		Negotiated:   true,
		Listen:       true,
		OneToOne:     true,
	}

	f.filters[chat.ChatID] = chat

	f.logger.Debug("registering filter for", zap.String("chatID", chatID), zap.String("type", "negotiated"), zap.String("topic", topic.String()))

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

	id, topic, err := f.addAsymmetric(personalDiscoveryChat.ChatID, f.privateKey)
	if err != nil {
		f.logger.Debug("could not register discovery topic", zap.String("chatID", personalDiscoveryChat.ChatID), zap.Error(err))
		return nil, err
	}

	personalDiscoveryChat.ContentTopic = topic
	personalDiscoveryChat.FilterID = id

	f.filters[personalDiscoveryChat.ChatID] = personalDiscoveryChat

	f.logger.Debug("registering filter for", zap.String("chatID", personalDiscoveryChat.ChatID), zap.String("type", "discovery"), zap.String("topic", personalDiscoveryChat.ContentTopic.String()))

	return []*Filter{personalDiscoveryChat}, nil
}

func (f *FiltersManager) PersonalTopicFilter() *Filter {
	personalDiscoveryTopic := PersonalDiscoveryTopic(&f.privateKey.PublicKey)

	return f.filters[personalDiscoveryTopic]
}

// LoadPublic adds a filter for a public chat with specific pubsubTopic
func (f *FiltersManager) LoadPublic(chatID string, pubsubTopic string) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if chat, ok := f.filters[chatID]; ok {
		if chat.PubsubTopic != pubsubTopic {
			f.logger.Debug("updating pubsub topic for filter",
				zap.String("chatID", chatID),
				zap.String("type", "public"),
				zap.String("oldTopic", chat.PubsubTopic),
				zap.String("newTopic", pubsubTopic),
			)
			chat.PubsubTopic = pubsubTopic
			f.filters[chatID] = chat
		}
		return chat, nil
	}

	id, topic, err := f.addSymmetric(chatID)
	if err != nil {
		f.logger.Debug("could not register public chat topic", zap.String("chatID", chatID), zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		FilterID:     id,
		ContentTopic: topic,
		PubsubTopic:  pubsubTopic,
		Listen:       true,
		OneToOne:     false,
	}

	f.filters[chatID] = chat

	f.logger.Debug("registering filter for",
		zap.String("chatID", chatID),
		zap.String("type", "public"),
		zap.String("ContentTopic", topic.String()),
		zap.String("PubsubTopic", pubsubTopic),
	)

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

	id, topic, err := f.addSymmetric(chatID)
	if err != nil {
		f.logger.Debug("could not register contact code topic", zap.String("chatID", chatID), zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		FilterID:     id,
		ContentTopic: topic,
		Identity:     PublicKeyToStr(pubKey),
		Listen:       true,
	}

	f.filters[chatID] = chat

	f.logger.Debug("registering filter for", zap.String("chatID", chatID), zap.String("type", "contact-code"), zap.String("topic", topic.String()))

	return chat, nil
}

// addSymmetric generates a symmetric filter's FilterID and stores its key,
// deriving it from the chat id if it is not already known. The wire
// subscription is the transport's concern (syncFilterSubscriptions).
func (f *FiltersManager) addSymmetric(chatID string) (string, types.TopicType, error) {
	topic := ToTopic(chatID)

	symKey, ok := f.keys[chatID]
	if !ok {
		// kdf should run no less than 0.1 seconds on an average computer,
		// because it's an once in a session experience
		symKey = pbkdf2.Key([]byte(chatID), nil, 65356, wakucommon.AESKeyLength, sha256.New)
		f.keys[chatID] = symKey

		if err := f.persistence.Add(chatID, symKey); err != nil {
			return "", types.TopicType{}, err
		}
	}

	id, err := wakucommon.GenerateRandomID()
	if err != nil {
		return "", types.TopicType{}, err
	}

	if err := f.addSymKey(id, symKey); err != nil {
		return "", types.TopicType{}, err
	}

	return id, types.BytesToTopic(topic), nil
}

// addAsymmetric generates an asymmetric filter's FilterID and stores its
// private key. The wire subscription is the transport's concern
// (syncFilterSubscriptions). Whether the filter actually receives messages is
// governed by its Listen flag: non-listening filters are skipped during
// routing in handleReceivedMessage.
func (f *FiltersManager) addAsymmetric(chatID string, identity *ecdsa.PrivateKey) (string, types.TopicType, error) {
	topic := ToTopic(chatID)

	id, err := wakucommon.GenerateRandomID()
	if err != nil {
		return "", types.TopicType{}, err
	}

	f.asymKeys[id] = identity

	return id, types.BytesToTopic(topic), nil
}

// GetNegotiated returns a negotiated chat given an identity
func (f *FiltersManager) GetNegotiated(identity *ecdsa.PublicKey) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.filters[NegotiatedTopic(identity)]
}
