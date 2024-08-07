package protocol

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

// TODO: in future adapt this struct to use waku v2 and switch all tests to waku v2
type CommunitiesMessengerTestSuiteBase struct {
	suite.Suite
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh                     types.Waku
	logger                  *zap.Logger
	mockedBalances          communities.BalancesByChain
	mockedCollectibles      communities.CollectiblesByChain
	collectiblesServiceMock *CollectiblesServiceMock
	collectiblesManagerMock *CollectiblesManagerMock
	accountsTestData        map[string][]string
	accountsPasswords       map[string]string
}

func (s *CommunitiesMessengerTestSuiteBase) SetupTest() {
	s.logger = tt.MustCreateTestLogger()
	s.collectiblesServiceMock = &CollectiblesServiceMock{}
	s.accountsTestData = make(map[string][]string)
	s.accountsPasswords = make(map[string]string)
	s.mockedCollectibles = make(communities.CollectiblesByChain)
	s.collectiblesManagerMock = &CollectiblesManagerMock{
		Collectibles: &s.mockedCollectibles,
	}

	s.mockedBalances = make(communities.BalancesByChain)

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())
}

func (s *CommunitiesMessengerTestSuiteBase) TearDownTest() {
	_ = s.logger.Sync()
}

func (s *CommunitiesMessengerTestSuiteBase) newMessenger(password string, walletAddresses []string) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithConfig(testMessengerConfig{
		logger:     s.logger,
		privateKey: privateKey,
	}, password, walletAddresses)
}

func (s *CommunitiesMessengerTestSuiteBase) newMessengerWithKey(privateKey *ecdsa.PrivateKey, password string, walletAddresses []string) *Messenger {
	return s.newMessengerWithConfig(testMessengerConfig{
		privateKey: privateKey,
		logger:     s.logger,
	}, password, walletAddresses)
}

func (s *CommunitiesMessengerTestSuiteBase) newMessengerWithConfig(config testMessengerConfig, password string, walletAddresses []string) *Messenger {
	messenger := newTestCommunitiesMessenger(&s.Suite, s.shh, testCommunitiesMessengerConfig{
		testMessengerConfig: config,
		password:            password,
		walletAddresses:     walletAddresses,
		mockedBalances:      &s.mockedBalances,
		collectiblesService: s.collectiblesServiceMock,
		collectiblesManager: s.collectiblesManagerMock,
	})

	publicKey := messenger.IdentityPublicKeyString()
	s.accountsTestData[publicKey] = walletAddresses
	s.accountsPasswords[publicKey] = password
	return messenger
}

func (s *CommunitiesMessengerTestSuiteBase) joinCommunity(community *communities.Community, controlNode *Messenger, user *Messenger) {
	userPk := user.IdentityPublicKeyString()
	addresses, exists := s.accountsTestData[userPk]
	s.Require().True(exists)
	password, exists := s.accountsPasswords[userPk]
	s.Require().True(exists)
	joinCommunity(&s.Suite, community.ID(), controlNode, user, password, addresses)
}

func (s *CommunitiesMessengerTestSuiteBase) joinOnRequestCommunity(community *communities.Community, controlNode *Messenger, user *Messenger) {
	userPk := user.IdentityPublicKeyString()
	addresses, exists := s.accountsTestData[userPk]
	s.Require().True(exists)
	password, exists := s.accountsPasswords[userPk]
	s.Require().True(exists)
	joinOnRequestCommunity(&s.Suite, community.ID(), controlNode, user, password, addresses)
}

func (s *CommunitiesMessengerTestSuiteBase) createRequestToJoinCommunity(communityID types.HexBytes, user *Messenger) *requests.RequestToJoinCommunity {
	userPk := user.IdentityPublicKeyString()
	addresses, exists := s.accountsTestData[userPk]
	s.Require().True(exists)
	password, exists := s.accountsPasswords[userPk]
	s.Require().True(exists)
	return createRequestToJoinCommunity(&s.Suite, communityID, user, password, addresses)
}

func (s *CommunitiesMessengerTestSuiteBase) makeAddressSatisfyTheCriteria(chainID uint64, address string, criteria *protobuf.TokenCriteria) {
	walletAddress := gethcommon.HexToAddress(address)
	contractAddress := gethcommon.HexToAddress(criteria.ContractAddresses[chainID])
	balance, ok := new(big.Int).SetString(criteria.AmountInWei, 10)
	s.Require().True(ok)

	if _, exists := s.mockedBalances[chainID]; !exists {
		s.mockedBalances[chainID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	}

	if _, exists := s.mockedBalances[chainID][walletAddress]; !exists {
		s.mockedBalances[chainID][walletAddress] = make(map[gethcommon.Address]*hexutil.Big)
	}

	if _, exists := s.mockedBalances[chainID][walletAddress][contractAddress]; !exists {
		s.mockedBalances[chainID][walletAddress][contractAddress] = (*hexutil.Big)(balance)
	}

	makeAddressSatisfyTheCriteria(&s.Suite, s.mockedBalances, s.mockedCollectibles, chainID, address, criteria)
}
