package filter

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/shhext/chat/topic"
	whisper "github.com/status-im/whisper/whisperv6"
	"math/big"
	"sync"
)

const (
	discoveryTopic = "contact-discovery"
)

// The number of partitions
var nPartitions = big.NewInt(5000)

func toTopic(s string) []byte {
	return crypto.Keccak256([]byte(s))[:whisper.TopicLength]
}

func chatIDToPartitionedTopic(identity string) (string, error) {
	partition := big.NewInt(0)
	publicKeyBytes, err := hex.DecodeString(identity)
	if err != nil {
		return "", err
	}

	publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		return "", err
	}

	partition.Mod(publicKey.X, nPartitions)

	return fmt.Sprintf("contact-discovery-%d", partition), nil
}

type FilterAndTopic struct {
	FilterID string
	Topic    []byte
	SymKeyID string
}

type Chat struct {
	// ChatID is the identifier of the chat
	ChatID string
	// SymKeyID is the symmetric key id used for symmetric chats
	SymKeyID string
	// OneToOne tells us if we need to use asymmetric encryption for this chat
	OneToOne bool
	// Listen is whether we are actually listening for messages on this chat, or the filter is only created in order to be able to post on the topic
	Listen bool
	// FilterID the whisper filter id generated
	FilterID string
	// Identity is the public key of the other recipient for non-public chats
	Identity string
	// Topic is the whisper topic
	Topic []byte
}

type Service struct {
	keyID   string
	whisper *whisper.Whisper
	topic   *topic.Service
	chats   map[string]*Chat
	mutex   sync.Mutex
}

func New(k string, w *whisper.Whisper, t *topic.Service) *Service {
	return &Service{
		keyID:   k,
		whisper: w,
		topic:   t,
		mutex:   sync.Mutex{},
		chats:   make(map[string]*Chat),
	}
}

// LoadDiscovery adds the discovery filter
func (s *Service) LoadDiscovery(myKey *ecdsa.PrivateKey) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	discoveryChat := &Chat{
		ChatID: discoveryTopic,
	}

	discoveryResponse, err := s.AddAsymmetricFilter(myKey, discoveryChat.ChatID, true)
	if err != nil {
		return err
	}

	discoveryChat.Topic = discoveryResponse.Topic
	discoveryChat.FilterID = discoveryResponse.FilterID

	s.chats[discoveryChat.ChatID] = discoveryChat
	return nil
}

func (s *Service) Init(chats []*Chat) error {
	log.Debug("Initializing filter service")
	myKey, err := s.whisper.GetPrivateKey(s.keyID)
	if err != nil {
		return err
	}

	// Add our own topic
	log.Debug("Loading one to one chats")
	identityStr := fmt.Sprintf("%x", crypto.FromECDSAPub(&myKey.PublicKey))
	err = s.LoadOneToOne(myKey, identityStr, true)
	if err != nil {
		log.Error("Error loading one to one chats", "err", err)

		return err
	}

	// Add discovery topic
	log.Debug("Loading discovery topics")
	err = s.LoadDiscovery(myKey)
	if err != nil {
		return err
	}

	// Add the various one to one and public chats
	log.Debug("Loading chats")
	for _, chat := range chats {
		err = s.Load(myKey, chat)
		if err != nil {
			return err
		}
	}

	// Add the negotiated topics
	log.Debug("Loading negotiated topics")
	secrets, err := s.topic.All()
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		s.ProcessNegotiatedSecret(secret)
	}

	return nil
}

func (s *Service) Stop() error {
	for _, chat := range s.chats {
		if err := s.Remove(chat); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Remove(chat *Chat) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.whisper.Unsubscribe(chat.ChatID); err != nil {
		return err
	}
	if chat.SymKeyID != "" {
		s.whisper.DeleteSymKey(chat.SymKeyID)
	}
	delete(s.chats, chat.ChatID)

	return nil

}

// LoadOneToOne creates two filters for a given chat, one listening to the contact codes
// and another on the partitioned topic. We pass a listen parameter to indicated whether
// we are listening to messages on the partitioned topic
func (s *Service) LoadOneToOne(myKey *ecdsa.PrivateKey, identity string, listen bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	contactCodeChatID := identity + "-contact-code"
	contactCodeFilter, err := s.AddSymmetric(contactCodeChatID)
	if err != nil {
		return err
	}

	s.chats[contactCodeChatID] = &Chat{
		ChatID:   contactCodeChatID,
		FilterID: contactCodeFilter.FilterID,
		Topic:    contactCodeFilter.Topic,
		SymKeyID: contactCodeFilter.SymKeyID,
		Identity: identity,
	}

	partitionedTopicChatID, err := chatIDToPartitionedTopic(identity)
	if err != nil {
		return err
	}
	// We set up a filter so we can publish, but we discard envelopes if listen is false
	partitionedTopicFilter, err := s.AddAsymmetricFilter(myKey, partitionedTopicChatID, listen)
	if err != nil {
		return err
	}
	s.chats[partitionedTopicChatID] = &Chat{
		ChatID:   partitionedTopicChatID,
		FilterID: partitionedTopicFilter.FilterID,
		Topic:    partitionedTopicFilter.Topic,
		Identity: identity,
		Listen:   listen,
	}

	return nil
}

func (s *Service) AddSymmetric(chatID string) (*FilterAndTopic, error) {
	var symKey []byte

	topic := toTopic(chatID)
	topics := [][]byte{topic}

	symKeyID, err := s.whisper.AddSymKeyFromPassword(chatID)
	if err != nil {
		log.Error("SYM KEYN FAILED", "err", err)
		return nil, err
	}

	if symKey, err = s.whisper.GetSymKey(symKeyID); err != nil {
		return nil, err
	}

	f := &whisper.Filter{
		KeySym:   symKey,
		PoW:      0.002,
		AllowP2P: true,
		Topics:   topics,
		Messages: s.whisper.NewMessageStore(),
	}

	id, err := s.whisper.Subscribe(f)
	if err != nil {
		return nil, err
	}

	return &FilterAndTopic{
		FilterID: id,
		SymKeyID: symKeyID,
		Topic:    topic,
	}, nil
}

func (s *Service) AddAsymmetricFilter(keyAsym *ecdsa.PrivateKey, chatID string, listen bool) (*FilterAndTopic, error) {
	var err error
	var pow float64

	if listen {
		pow = 0.002
	} else {
		// Set high pow so we discard messages
		pow = 1
	}

	topic := toTopic(chatID)
	topics := [][]byte{topic}

	f := &whisper.Filter{
		KeyAsym:  keyAsym,
		PoW:      pow,
		AllowP2P: listen,
		Topics:   topics,
		Messages: s.whisper.NewMessageStore(),
	}

	id, err := s.whisper.Subscribe(f)
	if err != nil {
		return nil, err
	}

	return &FilterAndTopic{FilterID: id, Topic: topic}, nil
}

func (s *Service) LoadPublic(chat *Chat) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	filterAndTopic, err := s.AddSymmetric(chat.ChatID)
	if err != nil {
		return err
	}

	// Add mutex
	chat.FilterID = filterAndTopic.FilterID
	chat.SymKeyID = filterAndTopic.SymKeyID
	chat.Topic = filterAndTopic.Topic
	s.chats[chat.ChatID] = chat
	return nil
}

func (s *Service) Load(myKey *ecdsa.PrivateKey, chat *Chat) error {
	var err error
	log.Debug("Loading chat", "chatID", chat.ChatID)

	// Check we haven't already loaded the chat
	if _, ok := s.chats[chat.ChatID]; !ok {
		if chat.OneToOne {
			err = s.LoadOneToOne(myKey, chat.Identity, false)

		} else {
			err = s.LoadPublic(chat)
		}
		if err != nil {
			return err
		}

	}
	return nil
}

func negotiatedID(identity *ecdsa.PublicKey) string {
	return fmt.Sprintf("%x-negotiated", crypto.FromECDSAPub(identity))
}

func (s *Service) Get(identity *ecdsa.PublicKey) *Chat {
	return s.chats[negotiatedID(identity)]
}

func (s *Service) ProcessNegotiatedSecret(secret *topic.Secret) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	keyString := fmt.Sprintf("%x", secret.Key)
	filter, err := s.AddSymmetric(keyString)
	if err != nil {
		return err
	}

	identityStr := fmt.Sprintf("0x%x", crypto.FromECDSAPub(secret.Identity))

	chat := &Chat{
		ChatID:   negotiatedID(secret.Identity),
		Topic:    filter.Topic,
		SymKeyID: filter.SymKeyID,
		Identity: identityStr,
	}

	s.chats[chat.ChatID] = chat
	return nil
}
