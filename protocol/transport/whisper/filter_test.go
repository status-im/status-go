package whisper

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	gethbridge "github.com/status-im/status-go/protocol/bridge/geth"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"

	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/mutecomm/go-sqlcipher"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

func TestFiltersManagerSuite(t *testing.T) {
	suite.Run(t, new(FiltersManagerSuite))
}

type FiltersManagerSuite struct {
	suite.Suite
	chats   *filtersManager
	dbPath  string
	manager []*testKey
	logger  *zap.Logger
}

type testKey struct {
	privateKey       *ecdsa.PrivateKey
	partitionedTopic int
}

func (t *testKey) publicKeyString() string {
	return hex.EncodeToString(crypto.FromECDSAPub(&t.privateKey.PublicKey))
}

func newTestKey(privateKey string, partitionedTopic int) (*testKey, error) {
	key, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}

	return &testKey{
		privateKey:       key,
		partitionedTopic: partitionedTopic,
	}, nil
}

func (s *FiltersManagerSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	keyStrs := []string{
		"c6cbd7d76bc5baca530c875663711b947efa6a86a900a9e8645ce32e5821484e",
		"d51dd64ad19ea84968a308dca246012c00d2b2101d41bce740acd1c650acc509",
	}
	keyTopics := []int{4490, 3991}

	dbFile, err := ioutil.TempFile(os.TempDir(), "filter")
	s.Require().NoError(err)
	s.dbPath = dbFile.Name()

	for i, k := range keyStrs {
		testKey, err := newTestKey(k, keyTopics[i])
		s.Require().NoError(err)
		s.manager = append(s.manager, testKey)
	}

	db, err := sqlite.Open(s.dbPath, "filter-key")
	s.Require().NoError(err)

	whisper := gethbridge.NewGethWhisperWrapper(whisper.New(nil))

	s.chats, err = newFiltersManager(db, whisper, s.manager[0].privateKey, s.logger)
	s.Require().NoError(err)
}

func (s *FiltersManagerSuite) TearDownTest() {
	os.Remove(s.dbPath)
	_ = s.logger.Sync()
}

func (s *FiltersManagerSuite) TestDiscoveryAndPartitionedTopic() {
	_, err := s.chats.Init(nil, nil, true)
	s.Require().NoError(err)

	s.Require().Equal(4, len(s.chats.filters), "It creates four filters")

	discoveryFilter := s.chats.filters[discoveryTopic]
	s.Require().NotNil(discoveryFilter, "It adds the discovery filter")
	s.Require().True(discoveryFilter.Listen)

	s.assertRequiredFilters()
}

func (s *FiltersManagerSuite) TestPartitionedTopicWithDiscoveryDisabled() {
	_, err := s.chats.Init(nil, nil, false)
	s.Require().NoError(err)

	s.Require().Equal(3, len(s.chats.filters), "It creates three filters")

	discoveryFilter := s.chats.filters[discoveryTopic]
	s.Require().Nil(discoveryFilter, "It does not add the discovery filter")

	s.assertRequiredFilters()
}

func (s *FiltersManagerSuite) assertRequiredFilters() {
	partitionedTopic := fmt.Sprintf("contact-discovery-%d", s.manager[0].partitionedTopic)
	personalDiscoveryTopic := fmt.Sprintf("contact-discovery-%s", s.manager[0].publicKeyString())
	contactCodeTopic := contactCodeTopic(&s.manager[0].privateKey.PublicKey)

	personalDiscoveryFilter := s.chats.filters[personalDiscoveryTopic]
	s.Require().NotNil(personalDiscoveryFilter, "It adds the discovery filter")
	s.Require().True(personalDiscoveryFilter.Listen)

	contactCodeFilter := s.chats.filters[contactCodeTopic]
	s.Require().NotNil(contactCodeFilter, "It adds the contact code filter")
	s.Require().True(contactCodeFilter.Listen)

	partitionedFilter := s.chats.filters[partitionedTopic]
	s.Require().NotNil(partitionedFilter, "It adds the partitioned filter")
	s.Require().True(partitionedFilter.Listen)
}
