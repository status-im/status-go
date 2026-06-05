package common

import (
	crand "crypto/rand"
	mrand "math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Since the reception pipeline moved into the transport
// (status-im/status-go#7464), the waku-side Filters collection is only a
// wire-subscription registry. These tests cover that registry; decoding,
// matching and buffering are covered by the transport's tests.

type FilterTestCase struct {
	f      *Filter
	id     string
	alive  bool
	msgCnt int
}

func createLogger(t *testing.T) *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	require.NoError(t, err)
	return logger
}

func generateFilter(t *testing.T, symmetric bool) (*Filter, error) {
	var f Filter
	f.PubsubTopic = "test"

	const topicNum = 8
	f.ContentTopics = make(TopicSet, topicNum)
	for i := 0; i < topicNum; i++ {
		topic := make([]byte, 4)
		_, err := crand.Read(topic) // nolint: gosec
		require.NoError(t, err)
		topic[0] = 0x01

		f.ContentTopics[BytesToTopic(topic)] = struct{}{}
	}

	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	f.Src = &key.PublicKey

	if symmetric {
		f.KeySym = make([]byte, AESKeyLength)
		_, err := crand.Read(f.KeySym) // nolint: gosec
		require.NoError(t, err)
		f.SymKeyHash = crypto.Keccak256Hash(f.KeySym)
	} else {
		f.KeyAsym, err = crypto.GenerateKey()
		require.NoError(t, err)
	}

	return &f, nil
}

func generateTestCases(t *testing.T, SizeTestFilters int) []FilterTestCase {
	cases := make([]FilterTestCase, SizeTestFilters)
	for i := 0; i < SizeTestFilters; i++ {
		f, _ := generateFilter(t, true)
		cases[i].f = f
		cases[i].alive = mrand.Int()&1 == 0 // nolint: gosec
	}
	return cases
}

func TestInstallFilters(t *testing.T) {
	const SizeTestFilters = 256
	filters := NewFilters(createLogger(t))
	tst := generateTestCases(t, SizeTestFilters)

	var err error
	var j string
	for i := 0; i < SizeTestFilters; i++ {
		j, err = filters.Install(tst[i].f)
		require.NoError(t, err)

		tst[i].id = j
		require.Len(t, j, KeyIDSize*2)
	}

	for _, testCase := range tst {
		if !testCase.alive {
			filters.Uninstall(testCase.id)
		}
	}

	for _, testCase := range tst {
		fil := filters.Get(testCase.id)
		exist := fil != nil
		require.Equal(t, exist, testCase.alive)
	}
}

func TestInstallSymKeyGeneratesHash(t *testing.T) {
	filters := NewFilters(createLogger(t))
	filter, _ := generateFilter(t, true)

	// save the current SymKeyHash for comparison
	initialSymKeyHash := filter.SymKeyHash

	// ensure the SymKeyHash is invalid, for Install to recreate it
	var invalid common.Hash
	filter.SymKeyHash = invalid

	_, err := filters.Install(filter)
	require.NoError(t, err)

	for i, b := range filter.SymKeyHash {
		require.Equal(t, b, initialSymKeyHash[i])
	}
}

func TestInstallIdenticalFilters(t *testing.T) {
	filters := NewFilters(createLogger(t))
	filter1, _ := generateFilter(t, true)

	// Copy the first filter since some of its fields
	// are randomly generated.
	filter2 := &Filter{
		KeySym:        filter1.KeySym,
		PubsubTopic:   filter1.PubsubTopic,
		ContentTopics: filter1.ContentTopics,
	}

	_, err := filters.Install(filter1)
	require.NoError(t, err)

	_, err = filters.Install(filter2)
	require.NoError(t, err)
}

func TestInstallFilterWithSymAndAsymKeys(t *testing.T) {
	filters := NewFilters(createLogger(t))
	filter1, _ := generateFilter(t, true)

	asymKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Copy the first filter since some of its fields
	// are randomly generated.
	filter := &Filter{
		KeySym:        filter1.KeySym,
		KeyAsym:       asymKey,
		PubsubTopic:   filter1.PubsubTopic,
		ContentTopics: filter1.ContentTopics,
	}

	_, err = filters.Install(filter)
	require.Error(t, err)
}
