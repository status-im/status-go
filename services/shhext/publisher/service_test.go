package publisher

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/status-im/status-go/services/shhext/dedup"
	"github.com/status-im/status-go/services/shhext/filter"
	"github.com/status-im/status-go/services/shhext/whisperutils"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
)

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

type TestKey struct {
	privateKey     *ecdsa.PrivateKey
	keyID          string
	publicKeyBytes hexutil.Bytes
}

type ServiceTestSuite struct {
	suite.Suite
	alice    *Service
	bob      *Service
	aliceKey *TestKey
	bobKey   *TestKey
}

func (s *ServiceTestSuite) SetupTest() {

	dir1, err := ioutil.TempDir("", "publisher-test")
	s.Require().NoError(err)

	config1 := &Config{
		PfsEnabled:     true,
		DataDir:        dir1,
		InstallationID: "1",
	}

	whisper1 := whisper.New(nil)
	err = whisper1.SetMinimumPoW(0)
	s.Require().NoError(err)

	service1 := New(config1, whisper1)

	pk1, err := crypto.GenerateKey()
	s.Require().NoError(err)

	keyID1, err := whisper1.AddKeyPair(pk1)
	s.Require().NoError(err)

	key1 := &TestKey{
		privateKey:     pk1,
		keyID:          keyID1,
		publicKeyBytes: crypto.FromECDSAPub(&pk1.PublicKey),
	}

	s.Require().NoError(err)

	err = service1.Start(func() bool { return true }, false)
	s.Require().NoError(err)

	err = service1.InitProtocolWithPassword("1", "")
	s.Require().NoError(err)
	_, err = service1.LoadFilters([]*filter.Chat{})
	s.Require().NoError(err)

	dir2, err := ioutil.TempDir("", "publisher-test")
	s.Require().NoError(err)

	config2 := &Config{
		PfsEnabled:     true,
		DataDir:        dir2,
		InstallationID: "2",
	}

	whisper2 := whisper.New(nil)
	err = whisper2.SetMinimumPoW(0)
	s.Require().NoError(err)

	service2 := New(config2, whisper2)

	pk2, err := crypto.GenerateKey()
	s.Require().NoError(err)

	keyID2, err := whisper2.AddKeyPair(pk2)
	s.Require().NoError(err)

	key2 := &TestKey{
		privateKey:     pk2,
		keyID:          keyID2,
		publicKeyBytes: crypto.FromECDSAPub(&pk2.PublicKey),
	}

	err = service2.Start(func() bool { return true }, false)
	s.Require().NoError(err)

	err = service2.InitProtocolWithPassword("1", "")
	s.Require().NoError(err)

	_, err = service2.LoadFilters([]*filter.Chat{})
	s.Require().NoError(err)

	s.alice = service1
	s.aliceKey = key1
	s.bob = service2
	s.bobKey = key2
}

func (s *ServiceTestSuite) TestSendDirectMessage() {
	newMessage, err := s.alice.SendDirectMessage(s.aliceKey.keyID, s.bobKey.publicKeyBytes, false, []byte("hello"))
	s.Require().NoError(err)

	message := &whisper.Message{
		Sig:     s.aliceKey.publicKeyBytes,
		Topic:   newMessage.Topic,
		Payload: newMessage.Payload,
		Dst:     newMessage.PublicKey,
	}
	dedupMessage := dedup.DeduplicateMessage{
		DedupID: []byte("1"),
		Message: message,
	}

	err = s.bob.ProcessMessage(dedupMessage)
	s.Require().NoError(err)

	s.Require().Equal([]byte("hello"), message.Payload)
}

func (s *ServiceTestSuite) TestTopic() {
	// We build an initial message
	newMessage1, err := s.alice.SendDirectMessage(s.aliceKey.keyID, s.bobKey.publicKeyBytes, false, []byte("hello"))
	s.Require().NoError(err)

	message1 := &whisper.Message{
		Sig:     s.aliceKey.publicKeyBytes,
		Topic:   newMessage1.Topic,
		Payload: newMessage1.Payload,
		Dst:     newMessage1.PublicKey,
	}

	// We have no information, it should use the discovery topic
	s.Require().Equal(whisperutils.DiscoveryTopicBytes, message1.Topic)

	// We build a contact code from user 2
	newMessage2, err := s.bob.sendContactCode()
	s.Require().NoError(err)
	s.Require().NotNil(newMessage2)

	message2 := &whisper.Message{
		Sig:     s.bobKey.publicKeyBytes,
		Topic:   newMessage2.Topic,
		Payload: newMessage2.Payload,
		Dst:     newMessage2.PublicKey,
	}

	// We receive the contact code
	dedupMessage2 := dedup.DeduplicateMessage{
		DedupID: []byte("1"),
		Message: message2,
	}

	err = s.alice.ProcessMessage(dedupMessage2)
	s.Require().NoError(err)

	// We build another message, this time it should use the partitioned topic
	newMessage3, err := s.alice.SendDirectMessage(s.aliceKey.keyID, s.bobKey.publicKeyBytes, false, []byte("hello"))
	s.Require().NoError(err)

	message3 := &whisper.Message{
		Sig:     s.aliceKey.publicKeyBytes,
		Topic:   newMessage3.Topic,
		Payload: newMessage3.Payload,
		Dst:     newMessage3.PublicKey,
	}
	expectedTopic3 := whisper.BytesToTopic(filter.PublicKeyToPartitionedTopicBytes(&s.bobKey.privateKey.PublicKey))

	s.Require().Equal(expectedTopic3, message3.Topic)

	// We receive the message
	dedupMessage3 := dedup.DeduplicateMessage{
		DedupID: []byte("1"),
		Message: message3,
	}

	err = s.bob.ProcessMessage(dedupMessage3)
	s.Require().NoError(err)

	// We build another message, this time it should use the negotiated topic
	newMessage4, err := s.bob.SendDirectMessage(s.bobKey.keyID, s.aliceKey.publicKeyBytes, false, []byte("hello"))
	s.Require().NoError(err)

	message4 := &whisper.Message{
		Sig:     s.bobKey.publicKeyBytes,
		Topic:   newMessage4.Topic,
		Payload: newMessage4.Payload,
		Dst:     newMessage4.PublicKey,
	}
	sharedSecret, err := ecies.ImportECDSA(s.bobKey.privateKey).GenerateShared(
		ecies.ImportECDSAPublic(&s.aliceKey.privateKey.PublicKey),
		16,
		16)
	s.Require().NoError(err)
	keyString := fmt.Sprintf("%x", sharedSecret)

	negotiatedTopic := whisper.BytesToTopic(filter.ToTopic(keyString))

	s.Require().Equal(negotiatedTopic, message4.Topic)

	// We receive the message
	dedupMessage4 := dedup.DeduplicateMessage{
		DedupID: []byte("1"),
		Message: message4,
	}

	err = s.alice.ProcessMessage(dedupMessage4)
	s.Require().NoError(err)

	// Alice sends another message to Bob, this time it should use the negotiated topic
	newMessage5, err := s.alice.SendDirectMessage(s.aliceKey.keyID, s.bobKey.publicKeyBytes, false, []byte("hello"))
	s.Require().NoError(err)

	message5 := &whisper.Message{
		Sig:     s.aliceKey.publicKeyBytes,
		Topic:   newMessage5.Topic,
		Payload: newMessage5.Payload,
		Dst:     newMessage5.PublicKey,
	}
	s.Require().NoError(err)
	s.Require().Equal(negotiatedTopic, message5.Topic)

}
