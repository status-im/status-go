package filter

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	appDB "github.com/status-im/status-go/services/shhext/chat/db"
	"github.com/status-im/status-go/services/shhext/chat/topic"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
)

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

type TestKey struct {
	privateKey       *ecdsa.PrivateKey
	partitionedTopic int
}

func NewTestKey(privateKey string, partitionedTopic int) (*TestKey, error) {
	key, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	return &TestKey{
		privateKey:       key,
		partitionedTopic: partitionedTopic,
	}, nil

}

func (t *TestKey) PublicKeyString() string {
	return fmt.Sprintf("%x", crypto.FromECDSAPub(&t.privateKey.PublicKey))
}

type ServiceTestSuite struct {
	suite.Suite
	service *Service
	path    string
	keys    []*TestKey
}

func (s *ServiceTestSuite) SetupTest() {
	keyStrs := []string{"c6cbd7d76bc5baca530c875663711b947efa6a86a900a9e8645ce32e5821484e", "d51dd64ad19ea84968a308dca246012c00d2b2101d41bce740acd1c650acc509"}
	keyTopics := []int{4490, 3991}

	dbFile, err := ioutil.TempFile(os.TempDir(), "topic")

	s.Require().NoError(err)
	s.path = dbFile.Name()

	for i, k := range keyStrs {
		testKey, err := NewTestKey(k, keyTopics[i])
		s.Require().NoError(err)

		s.keys = append(s.keys, testKey)
	}

	db, err := appDB.Open(s.path, "", 0)
	s.Require().NoError(err)

	// Build services
	topicService := topic.NewService(topic.NewSQLLitePersistence(db))
	whisper := whisper.New(nil)
	keyID, err := whisper.AddKeyPair(s.keys[0].privateKey)
	s.Require().NoError(err)

	s.service = New(keyID, whisper, topicService)
}

func (s *ServiceTestSuite) TearDownTest() {
	os.Remove(s.path)
}

func (s *ServiceTestSuite) TestDiscoveryAndPartitionedTopic() {
	chats := []*Chat{}
	partitionedTopic := fmt.Sprintf("contact-discovery-%d", s.keys[0].partitionedTopic)
	contactCodeTopic := s.keys[0].PublicKeyString() + "-contact-code"

	err := s.service.Init(chats)
	s.Require().NoError(err)

	s.Require().Equal(3, len(s.service.chats), "It creates two filters")

	discoveryFilter := s.service.chats[discoveryTopic]
	s.Require().NotNil(discoveryFilter, "It adds the discovery filter")

	contactCodeFilter := s.service.chats[contactCodeTopic]
	s.Require().NotNil(contactCodeFilter, "It adds the contact code filter")

	partitionedFilter := s.service.chats[partitionedTopic]
	s.Require().NotNil(partitionedFilter, "It adds the partitioned filter")
}

func (s *ServiceTestSuite) TestPublicAndOneToOneChats() {
	chats := []*Chat{
		&Chat{
			ChatID: "status",
		},
		&Chat{
			ChatID:   s.keys[1].PublicKeyString(),
			Identity: s.keys[1].PublicKeyString(),
			OneToOne: true,
		},
	}
	partitionedTopic := fmt.Sprintf("contact-discovery-%d", s.keys[1].partitionedTopic)
	contactCodeTopic := s.keys[1].PublicKeyString() + "-contact-code"

	err := s.service.Init(chats)
	s.Require().NoError(err)

	s.Require().Equal(6, len(s.service.chats), "It creates two additional filters for the one to one and one for the public chat")

	statusFilter := s.service.chats["status"]
	s.Require().NotNil(statusFilter, "It creates a filter for the public chat")
	s.Require().NotNil(statusFilter.SymKeyID, "It returns a sym key id")

	contactCodeFilter := s.service.chats[contactCodeTopic]
	s.Require().NotNil(contactCodeFilter, "It adds the contact code filter")

	partitionedFilter := s.service.chats[partitionedTopic]
	s.Require().NotNil(partitionedFilter, "It adds the partitioned filter")
}

func (s *ServiceTestSuite) TestNegotiatedTopic() {
	chats := []*Chat{}

	negotiatedTopic1 := s.keys[0].PublicKeyString() + "-negotiated"
	negotiatedTopic2 := s.keys[1].PublicKeyString() + "-negotiated"

	// We send a message to ourselves
	_, _, err := s.service.topic.Send(s.keys[0].privateKey, "0-1", &s.keys[0].privateKey.PublicKey, []string{"0-2"})
	s.Require().NoError(err)

	// We send a message to someone else
	_, _, err = s.service.topic.Send(s.keys[0].privateKey, "0-1", &s.keys[1].privateKey.PublicKey, []string{"0-2"})
	s.Require().NoError(err)

	err = s.service.Init(chats)
	s.Require().NoError(err)

	s.Require().Equal(5, len(s.service.chats), "It creates two additional filters for the negotiated topics")

	negotiatedFilter1 := s.service.chats[negotiatedTopic1]
	s.Require().NotNil(negotiatedFilter1, "It adds the negotiated filter")
	negotiatedFilter2 := s.service.chats[negotiatedTopic2]
	s.Require().NotNil(negotiatedFilter2, "It adds the negotiated filter")
}
