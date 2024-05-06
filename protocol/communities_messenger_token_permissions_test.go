package protocol

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
)

const testChainID1 = 1

const ownerPassword = "123456"
const alicePassword = "qwerty"
const bobPassword = "bob123"

const ownerAddress = "0x0100000000000000000000000000000000000000"
const aliceAddress1 = "0x0200000000000000000000000000000000000000"
const aliceAddress2 = "0x0210000000000000000000000000000000000000"
const bobAddress = "0x0300000000000000000000000000000000000000"

type CommunityAndKeyActions struct {
	community  *communities.Community
	keyActions *communities.EncryptionKeyActions
}

type TestCommunitiesKeyDistributor struct {
	CommunitiesKeyDistributorImpl

	subscriptions map[chan *CommunityAndKeyActions]bool
	mutex         sync.RWMutex
}

func (tckd *TestCommunitiesKeyDistributor) Generate(community *communities.Community, keyActions *communities.EncryptionKeyActions) error {
	return tckd.CommunitiesKeyDistributorImpl.Generate(community, keyActions)
}

func (tckd *TestCommunitiesKeyDistributor) Distribute(community *communities.Community, keyActions *communities.EncryptionKeyActions) error {
	err := tckd.CommunitiesKeyDistributorImpl.Distribute(community, keyActions)
	if err != nil {
		return err
	}

	// notify distribute finished
	tckd.mutex.RLock()
	for s := range tckd.subscriptions {
		s <- &CommunityAndKeyActions{
			community:  community,
			keyActions: keyActions,
		}
	}
	tckd.mutex.RUnlock()

	return nil
}

func (tckd *TestCommunitiesKeyDistributor) waitOnKeyDistribution(condition func(*CommunityAndKeyActions) bool) <-chan error {
	errCh := make(chan error, 1)

	subscription := make(chan *CommunityAndKeyActions, 40)
	tckd.mutex.Lock()
	tckd.subscriptions[subscription] = true
	tckd.mutex.Unlock()

	go func() {
		defer func() {
			close(errCh)

			tckd.mutex.Lock()
			delete(tckd.subscriptions, subscription)
			tckd.mutex.Unlock()
			close(subscription)
		}()

		for {
			select {
			case s, more := <-subscription:
				if !more {
					errCh <- errors.New("channel closed when waiting for key distribution")
					return
				}

				if condition(s) {
					return
				}

			case <-time.After(500 * time.Millisecond):
				errCh <- errors.New("timed out when waiting for key distribution")
				return
			}
		}
	}()

	return errCh
}

func TestMessengerCommunitiesTokenPermissionsSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunitiesTokenPermissionsSuite))
}

type MessengerCommunitiesTokenPermissionsSuite struct {
	suite.Suite
	owner *Messenger
	bob   *Messenger
	alice *Messenger

	ownerWaku types.Waku
	bobWaku   types.Waku
	aliceWaku types.Waku

	logger *zap.Logger

	mockedBalances          map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big // chainID, account, token, balance
	collectiblesServiceMock *CollectiblesServiceMock
}

func (s *MessengerCommunitiesTokenPermissionsSuite) SetupTest() {
	// Initialize with nil to avoid panics in TearDownTest
	s.owner = nil
	s.bob = nil
	s.alice = nil
	s.ownerWaku = nil
	s.bobWaku = nil
	s.aliceWaku = nil

	s.logger = tt.MustCreateTestLogger()

	wakuNodes := CreateWakuV2Network(&s.Suite, s.logger, false, []string{"owner", "bob", "alice"})

	s.ownerWaku = wakuNodes[0]
	s.owner = s.newMessenger(ownerPassword, []string{ownerAddress}, s.ownerWaku, "owner", []Option{})

	s.bobWaku = wakuNodes[1]
	s.bob = s.newMessenger(bobPassword, []string{bobAddress}, s.bobWaku, "bob", []Option{})

	s.aliceWaku = wakuNodes[2]
	s.alice = s.newMessenger(alicePassword, []string{aliceAddress1, aliceAddress2}, s.aliceWaku, "alice", []Option{})

	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.resetMockedBalances()

}

func (s *MessengerCommunitiesTokenPermissionsSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.owner)
	TearDownMessenger(&s.Suite, s.bob)
	TearDownMessenger(&s.Suite, s.alice)
	if s.ownerWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.ownerWaku).Stop())
	}
	if s.bobWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.bobWaku).Stop())
	}
	if s.aliceWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.aliceWaku).Stop())
	}
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesTokenPermissionsSuite) newMessenger(password string, walletAddresses []string, waku types.Waku, name string, extraOptions []Option) *Messenger {
	return newTestCommunitiesMessenger(&s.Suite, waku, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			logger:       s.logger.Named(name),
			extraOptions: extraOptions,
		},
		password:            password,
		walletAddresses:     walletAddresses,
		mockedBalances:      &s.mockedBalances,
		collectiblesService: s.collectiblesServiceMock,
	})
}

func (s *MessengerCommunitiesTokenPermissionsSuite) joinCommunity(community *communities.Community, user *Messenger, password string, addresses []string) {
	s.joinCommunityWithAirdropAddress(community, user, password, addresses, "")
}

func (s *MessengerCommunitiesTokenPermissionsSuite) joinCommunityWithAirdropAddress(community *communities.Community, user *Messenger, password string, addresses []string, airdropAddress string) {
	passwdHash := types.EncodeHex(crypto.Keccak256([]byte(password)))
	if airdropAddress == "" && len(addresses) > 0 {
		airdropAddress = addresses[0]
	}

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID(), AddressesToReveal: addresses, AirdropAddress: airdropAddress}
	joinCommunity(&s.Suite, community, s.owner, user, request, passwdHash)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) advertiseCommunityTo(community *communities.Community, user *Messenger) {
	advertiseCommunityTo(&s.Suite, community, s.owner, user)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) createCommunity() (*communities.Community, *Chat) {
	return createCommunity(&s.Suite, s.owner)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) sendChatMessage(sender *Messenger, chatID string, text string) *common.Message {
	return sendChatMessage(&s.Suite, sender, chatID, text)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) makeAddressSatisfyTheCriteria(chainID uint64, address string, criteria *protobuf.TokenCriteria) {
	walletAddress := gethcommon.HexToAddress(address)
	contractAddress := gethcommon.HexToAddress(criteria.ContractAddresses[chainID])
	balance, ok := new(big.Int).SetString(criteria.AmountInWei, 10)
	s.Require().True(ok)

	s.mockedBalances[chainID][walletAddress][contractAddress] = (*hexutil.Big)(balance)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) resetMockedBalances() {
	s.mockedBalances = make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(aliceAddress1)] = make(map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(aliceAddress2)] = make(map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(bobAddress)] = make(map[gethcommon.Address]*hexutil.Big)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) waitOnKeyDistribution(condition func(*CommunityAndKeyActions) bool) <-chan error {
	testCommunitiesKeyDistributor, ok := s.owner.communitiesKeyDistributor.(*TestCommunitiesKeyDistributor)
	s.Require().True(ok)
	return testCommunitiesKeyDistributor.waitOnKeyDistribution(condition)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestCreateTokenPermission() {

	community, _ := s.createCommunity()

	createTokenPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{uint64(testChainID1): "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	response, err := s.owner.CreateCommunityTokenPermission(createTokenPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	tokenPermissions := response.Communities()[0].TokenPermissions()
	for _, tokenPermission := range tokenPermissions {
		for _, tc := range tokenPermission.TokenCriteria {
			s.Require().Equal(tc.Type, protobuf.CommunityTokenType_ERC20)
			s.Require().Equal(tc.Symbol, "TEST")
			s.Require().Equal(tc.AmountInWei, "100000000000000000000")
			s.Require().Equal(tc.Amount, "100") // automatically upgraded deprecated amount
			s.Require().Equal(tc.Decimals, uint64(18))
		}
	}
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestEditTokenPermission() {

	community, _ := s.createCommunity()

	tokenPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	response, err := s.owner.CreateCommunityTokenPermission(tokenPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	tokenPermissions := response.Communities()[0].TokenPermissions()

	var tokenPermissionID string
	for id := range tokenPermissions {
		tokenPermissionID = id
	}

	tokenPermission.TokenCriteria[0].Symbol = "TESTUpdated"
	tokenPermission.TokenCriteria[0].AmountInWei = "200000000000000000000"
	tokenPermission.TokenCriteria[0].Decimals = uint64(20)

	editTokenPermission := &requests.EditCommunityTokenPermission{
		PermissionID:                   tokenPermissionID,
		CreateCommunityTokenPermission: *tokenPermission,
	}

	response2, err := s.owner.EditCommunityTokenPermission(editTokenPermission)
	s.Require().NoError(err)
	// wait for `checkMemberPermissions` to finish
	time.Sleep(1 * time.Second)
	s.Require().NotNil(response2)
	s.Require().Len(response2.Communities(), 1)

	tokenPermissions = response2.Communities()[0].TokenPermissions()
	for _, tokenPermission := range tokenPermissions {
		for _, tc := range tokenPermission.TokenCriteria {
			s.Require().Equal(tc.Type, protobuf.CommunityTokenType_ERC20)
			s.Require().Equal(tc.Symbol, "TESTUpdated")
			s.Require().Equal(tc.AmountInWei, "200000000000000000000")
			s.Require().Equal(tc.Amount, "2") // automatically upgraded deprecated amount
			s.Require().Equal(tc.Decimals, uint64(20))
		}
	}
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestCommunityTokensMetadata() {

	community, _ := s.createCommunity()

	tokensMetadata := community.CommunityTokensMetadata()
	s.Require().Len(tokensMetadata, 0)

	newToken := &protobuf.CommunityTokenMetadata{
		ContractAddresses: map[uint64]string{testChainID1: "0xasd"},
		Description:       "desc1",
		Image:             "IMG1",
		TokenType:         protobuf.CommunityTokenType_ERC721,
		Symbol:            "SMB",
		Decimals:          3,
	}

	_, err := community.AddCommunityTokensMetadata(newToken)
	s.Require().NoError(err)
	tokensMetadata = community.CommunityTokensMetadata()
	s.Require().Len(tokensMetadata, 1)

	s.Require().Equal(tokensMetadata[0].ContractAddresses, newToken.ContractAddresses)
	s.Require().Equal(tokensMetadata[0].Description, newToken.Description)
	s.Require().Equal(tokensMetadata[0].Image, newToken.Image)
	s.Require().Equal(tokensMetadata[0].TokenType, newToken.TokenType)
	s.Require().Equal(tokensMetadata[0].Symbol, newToken.Symbol)
	s.Require().Equal(tokensMetadata[0].Name, newToken.Name)
	s.Require().Equal(tokensMetadata[0].Decimals, newToken.Decimals)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestRequestAccessWithENSTokenPermission() {
	community, _ := s.createCommunity()

	createTokenPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:       protobuf.CommunityTokenType_ENS,
				EnsPattern: "test.stateofus.eth",
			},
		},
	}

	response, err := s.owner.CreateCommunityTokenPermission(createTokenPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	s.advertiseCommunityTo(community, s.alice)

	// Make sure declined requests are 0
	declinedRequests, err := s.owner.DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(declinedRequests, 0)

	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	// We try to join the org
	response, err = s.alice.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin1 := response.RequestsToJoinCommunity()[0]
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin1.State)

	// Retrieve request to join
	err = tt.RetryWithBackOff(func() error {
		_, err = s.owner.RetrieveAll()
		if err != nil {
			return err
		}
		declinedRequests, err := s.owner.DeclinedRequestsToJoinForCommunity(community.ID())
		if err != nil {
			return err
		}
		if len(declinedRequests) != 1 {
			return errors.New("there should be one declined request")
		}
		if !bytes.Equal(requestToJoin1.ID, declinedRequests[0].ID) {
			return errors.New("wrong declined request")
		}
		return nil
	})
	s.Require().NoError(err)

	// Ensure alice is not a member of the community
	allCommunities, err := s.owner.Communities()
	if bytes.Equal(allCommunities[0].ID(), community.ID()) {
		s.Require().False(allCommunities[0].HasMember(&s.alice.identity.PublicKey))
	}
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestJoinedCommunityMembersSharedAddress() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.alice)
	s.advertiseCommunityTo(community, s.bob)

	s.joinCommunity(community, s.alice, alicePassword, []string{})
	s.joinCommunity(community, s.bob, bobPassword, []string{})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(3, community.MembersCount())

	// Check owner's DB for revealed accounts
	for pubKey := range community.Members() {
		if pubKey != common.PubkeyToHex(&s.owner.identity.PublicKey) {
			revealedAccounts, err := s.owner.communitiesManager.GetRevealedAddresses(community.ID(), pubKey)
			s.Require().NoError(err)
			switch pubKey {
			case common.PubkeyToHex(&s.alice.identity.PublicKey):
				s.Require().Len(revealedAccounts, 2)
				s.Require().Equal(revealedAccounts[0].Address, aliceAddress1)
				s.Require().Equal(revealedAccounts[1].Address, aliceAddress2)
				s.Require().Equal(true, revealedAccounts[0].IsAirdropAddress)
			case common.PubkeyToHex(&s.bob.identity.PublicKey):
				s.Require().Len(revealedAccounts, 1)
				s.Require().Equal(revealedAccounts[0].Address, bobAddress)
				s.Require().Equal(true, revealedAccounts[0].IsAirdropAddress)
			default:
				s.Require().Fail("pubKey does not match expected keys")
			}
		}
	}

	// Check Bob's DB for revealed accounts
	revealedAccountsInBobsDB, err := s.bob.communitiesManager.GetRevealedAddresses(community.ID(), common.PubkeyToHex(&s.bob.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().Len(revealedAccountsInBobsDB, 1)
	s.Require().Equal(revealedAccountsInBobsDB[0].Address, bobAddress)
	s.Require().Equal(true, revealedAccountsInBobsDB[0].IsAirdropAddress)

	// Check Alices's DB for revealed accounts
	revealedAccountsInAlicesDB, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().Len(revealedAccountsInAlicesDB, 2)
	s.Require().Equal(revealedAccountsInAlicesDB[0].Address, aliceAddress1)
	s.Require().Equal(revealedAccountsInAlicesDB[1].Address, aliceAddress2)
	s.Require().Equal(true, revealedAccountsInAlicesDB[0].IsAirdropAddress)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestJoinedCommunityMembersSelectedSharedAddress() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.alice)

	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress2})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(2, community.MembersCount())

	alicePubkey := common.PubkeyToHex(&s.alice.identity.PublicKey)

	// Check owner's DB for revealed accounts
	revealedAccounts, err := s.owner.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(revealedAccounts, 1)
	s.Require().Equal(revealedAccounts[0].Address, aliceAddress2)
	s.Require().Equal(true, revealedAccounts[0].IsAirdropAddress)

	// Check Alice's DB for revealed accounts
	revealedAccountsInAlicesDB, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(revealedAccountsInAlicesDB, 1)
	s.Require().Equal(revealedAccountsInAlicesDB[0].Address, aliceAddress2)
	s.Require().Equal(true, revealedAccountsInAlicesDB[0].IsAirdropAddress)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestJoinedCommunityMembersMultipleSelectedSharedAddresses() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.alice)

	s.joinCommunityWithAirdropAddress(community, s.alice, alicePassword, []string{aliceAddress1, aliceAddress2}, aliceAddress2)

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(2, community.MembersCount())

	alicePubkey := common.PubkeyToHex(&s.alice.identity.PublicKey)

	// Check owner's DB for revealed accounts
	revealedAccounts, err := s.owner.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(revealedAccounts, 2)
	s.Require().Equal(revealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(revealedAccounts[1].Address, aliceAddress2)
	s.Require().Equal(true, revealedAccounts[1].IsAirdropAddress)

	// Check Alice's DB for revealed accounts
	revealedAccountsInAlicesDB, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(revealedAccountsInAlicesDB, 2)
	s.Require().Equal(revealedAccountsInAlicesDB[0].Address, aliceAddress1)
	s.Require().Equal(revealedAccountsInAlicesDB[1].Address, aliceAddress2)
	s.Require().Equal(true, revealedAccountsInAlicesDB[1].IsAirdropAddress)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestEditSharedAddresses() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.alice)

	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress2})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(2, community.MembersCount())

	alicePubkey := common.PubkeyToHex(&s.alice.identity.PublicKey)

	revealedAccounts, err := s.owner.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)

	s.Require().Len(revealedAccounts, 1)
	s.Require().Equal(revealedAccounts[0].Address, aliceAddress2)
	s.Require().Equal(true, revealedAccounts[0].IsAirdropAddress)

	alicesRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(alicesRevealedAccounts, 1)
	s.Require().Equal(alicesRevealedAccounts[0].Address, aliceAddress2)
	s.Require().Equal(true, alicesRevealedAccounts[0].IsAirdropAddress)

	request := &requests.EditSharedAddresses{CommunityID: community.ID(), AddressesToReveal: []string{aliceAddress1}, AirdropAddress: aliceAddress1}

	signingParams, err := s.alice.GenerateJoiningCommunityRequestsForSigning(common.PubkeyToHex(&s.alice.identity.PublicKey), community.ID(), request.AddressesToReveal)
	s.Require().NoError(err)

	passwdHash := types.EncodeHex(crypto.Keccak256([]byte(alicePassword)))
	for i := range signingParams {
		signingParams[i].Password = passwdHash
	}
	signatures, err := s.alice.SignData(signingParams)
	s.Require().NoError(err)

	updateAddresses := len(request.AddressesToReveal) == 0
	if updateAddresses {
		request.AddressesToReveal = make([]string, len(signingParams))
	}
	for i := range signingParams {
		request.AddressesToReveal[i] = signingParams[i].Address
		request.Signatures = append(request.Signatures, types.FromHex(signatures[i]))
	}
	if updateAddresses {
		request.AirdropAddress = request.AddressesToReveal[0]
	}

	response, err := s.alice.EditSharedAddressesForCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Retrieve address change
	err = tt.RetryWithBackOff(func() error {
		response, err := s.owner.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities in response (address change reception)")
		}
		return nil
	})
	s.Require().NoError(err)
	revealedAccounts, err = s.owner.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)

	s.Require().Len(revealedAccounts, 1)
	s.Require().Equal(revealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(true, revealedAccounts[0].IsAirdropAddress)

	// Retrieve community description change
	err = tt.RetryWithBackOff(func() error {
		response, err := s.alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities in response (address change reception)")
		}
		return nil
	})
	s.Require().NoError(err)

	alicesRevealedAccounts, err = s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(alicesRevealedAccounts, 1)
	s.Require().Equal(alicesRevealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(true, alicesRevealedAccounts[0].IsAirdropAddress)
}

// NOTE(cammellos): Disabling for now as flaky, for some reason does not pass on CI, but passes locally
/*
func (s *MessengerCommunitiesTokenPermissionsSuite) TestBecomeMemberPermissions() {
	// Create a store node
	// This is needed to fetch the messages after rejoining the community
	var err error

	storeNodeLogger := s.logger.Named("store-node-waku")
	wakuStoreNode := NewTestWakuV2(&s.Suite, storeNodeLogger, true, true, false, shard.UndefinedShardValue)

	storeNodeListenAddresses := wakuStoreNode.ListenAddresses()
	s.Require().LessOrEqual(1, len(storeNodeListenAddresses))

	storeNodeAddress := storeNodeListenAddresses[0]
	s.logger.Info("store node ready", zap.String("address", storeNodeAddress))

	// Create messengers

	wakuNodes := CreateWakuV2Network(&s.Suite, s.logger, false, []string{"owner", "bob"})
	s.ownerWaku = wakuNodes[0]
	s.bobWaku = wakuNodes[1]

	options := []Option{
		WithTestStoreNode(&s.Suite, localMailserverID, storeNodeAddress, localFleet, s.collectiblesServiceMock),
	}

	s.owner = s.newMessenger(ownerPassword, []string{ownerAddress}, s.ownerWaku, "owner", options)
	s.Require().NoError(err)

	_, err = s.owner.Start()
	s.Require().NoError(err)

	s.bob = s.newMessenger(bobPassword, []string{bobAddress}, s.bobWaku, "bob", options)
	s.Require().NoError(err)

	_, err = s.bob.Start()
	s.Require().NoError(err)

	// Force the owner to use the store node as relay peer

	err = s.owner.DialPeer(storeNodeAddress)
	s.Require().NoError(err)

	// Create a community

	community, chat := s.createCommunity()

	// bob joins the community
	s.advertiseCommunityTo(community, s.bob)
	s.joinCommunityWithAirdropAddress(community, s.bob, bobPassword, []string{bobAddress}, "")

	messages := []string{
		"1-message", // RandomLettersString(10), // successful message on open community
		"2-message", // RandomLettersString(11), // failing message on encrypted community
		"3-message", // RandomLettersString(12), // successful message on encrypted community
	}

	// send message to the channel
	msg := s.sendChatMessage(s.owner, chat.ID, messages[0])
	s.logger.Debug("owner sent a message",
		zap.String("messageText", msg.Text),
		zap.String("messageID", msg.ID),
	)

	// bob can read the message
	response, err := WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			for _, message := range r.messages {
				if message.Text == msg.Text {
					return true
				}
			}
			return false
		},
		"first message not received",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(msg.Text, response.Messages()[0].Text)

	bobMessages, _, err := s.bob.MessageByChatID(msg.ChatId, "", 10)
	s.Require().NoError(err)
	s.Require().Len(bobMessages, 1)
	s.Require().Equal(messages[0], bobMessages[0].Text)

	// setup become member permission
	permissionRequest := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				Amount:            "100",
				Decimals:          uint64(18),
			},
		},
	}

	waitOnBobToBeKicked := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return len(sub.Community.Members()) == 1
	})

	response, err = s.owner.CreateCommunityTokenPermission(&permissionRequest)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	err = <-waitOnBobToBeKicked
	s.Require().NoError(err)

	// bob should be kicked from the community,
	// because he doesn't meet the criteria
	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.Members(), 1)

	// bob receives community changes
	// chats and members should be empty,
	// this info is available only to members
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) == 1 && len(community.TokenPermissions()) > 0 && r.Communities()[0].IDString() == community.IDString() && !r.Communities()[0].Joined()
		},
		"no community that satisfies criteria",
	)
	s.Require().NoError(err)

	// We are not member of the community anymore, so we need to refetch
	// the data, since we would not be pulling it anymore
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			_, err := s.bob.FetchCommunity(&FetchCommunityRequest{WaitForResponse: true, TryDatabase: false, CommunityKey: community.IDString()})
			if err != nil {
				return false
			}
			c, err := s.bob.communitiesManager.GetByID(community.ID())
			return err == nil && c != nil && len(c.TokenPermissions()) > 0 && !c.Joined()
		},
		"no token permissions",
	)

	s.Require().NoError(err)

	// bob tries to join, but he doesn't satisfy so the request isn't sent
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID(), AddressesToReveal: []string{bobAddress}, AirdropAddress: bobAddress}
	_, err = s.bob.RequestToJoinCommunity(request)
	s.Require().ErrorIs(err, communities.ErrPermissionToJoinNotSatisfied)

	// make sure bob does not have a pending request to join
	pendingRequests, err := s.bob.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(pendingRequests, 0)

	// Send chat message while bob is not in the community
	msg = s.sendChatMessage(s.owner, chat.ID, messages[1])

	// make bob satisfy the criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, permissionRequest.TokenCriteria[0])

	waitOnCommunityKeyToBeDistributedToBob := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		return len(sub.community.Description().Members) == 2 &&
			len(sub.keyActions.CommunityKeyAction.Members) == 1 &&
			sub.keyActions.CommunityKeyAction.ActionType == communities.EncryptionKeySendToMembers
	})

	// bob re-joins the community
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	err = <-waitOnCommunityKeyToBeDistributedToBob
	s.Require().NoError(err)

	// send message to channel
	msg = s.sendChatMessage(s.owner, chat.ID, messages[2])
	s.logger.Debug("owner sent a message",
		zap.String("messageText", msg.Text),
		zap.String("messageID", msg.ID),
	)

	// bob can read the message
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			// Bob should have all 3 messages
			bobMessages, _, err = s.bob.MessageByChatID(msg.ChatId, "", 10)
			return err == nil && len(bobMessages) == 3
		},
		"not all 3 messages received",
	)
	bobMessages, _, err = s.bob.MessageByChatID(msg.ChatId, "", 10)
	for _, m := range bobMessages {
		fmt.Printf("ID: %s\n", m.ID)
	}
	s.Require().NoError(err)
}*/

func (s *MessengerCommunitiesTokenPermissionsSuite) TestJoinCommunityWithAdminPermission() {

	community, _ := s.createCommunity()

	// setup become admin permission
	permissionRequest := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_ADMIN,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	response, err := s.owner.CreateCommunityTokenPermission(&permissionRequest)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	s.advertiseCommunityTo(community, s.bob)

	// Bob should still be able to join even if there is a permission to be an admin
	s.joinCommunity(community, s.bob, bobPassword, []string{})

	// Verify that we have Bob's revealed account
	revealedAccounts, err := s.owner.GetRevealedAccounts(community.ID(), common.PubkeyToHex(&s.bob.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().Len(revealedAccounts, 1)
	s.Require().Equal(bobAddress, revealedAccounts[0].Address)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestSharedAddressesReturnsRevealedAccount() {
	community, _ := s.createCommunity()

	permissionRequest := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	response, err := s.owner.CreateCommunityTokenPermission(&permissionRequest)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	s.advertiseCommunityTo(community, s.alice)

	s.joinCommunity(community, s.alice, bobPassword, []string{})

	revealedAccounts, err := s.alice.GetRevealedAccounts(community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().Len(revealedAccounts, 2)
	s.Require().Equal(aliceAddress1, revealedAccounts[0].Address)
	s.Require().Equal(aliceAddress2, revealedAccounts[1].Address)

	sharedAddresses, err := s.alice.getSharedAddresses(community.ID(), []string{})
	s.Require().NoError(err)
	s.Require().Len(sharedAddresses, 2)
	s.Require().Equal(sharedAddresses[0].String(), revealedAccounts[0].Address)
	s.Require().Equal(sharedAddresses[1].String(), revealedAccounts[1].Address)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestJoinCommunityAsMemberWithMemberAndAdminPermission() {
	community, _ := s.createCommunity()

	waitOnCommunityPermissionCreated := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return sub.Community.HasTokenPermissions()
	})

	// setup become member permission
	permissionRequestMember := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}
	response, err := s.owner.CreateCommunityTokenPermission(&permissionRequestMember)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	err = <-waitOnCommunityPermissionCreated
	s.Require().NoError(err)

	// setup become admin permission
	permissionRequestAdmin := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_ADMIN,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x124"},
				Symbol:            "TESTADMIN",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}
	response, err = s.owner.CreateCommunityTokenPermission(&permissionRequestAdmin)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	waitOnCommunityPermissionCreated = waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return len(sub.Community.TokenPermissions()) == 2
	})

	err = <-waitOnCommunityPermissionCreated
	s.Require().NoError(err)

	// make bob satisfy the member criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, permissionRequestMember.TokenCriteria[0])

	s.advertiseCommunityTo(response.Communities()[0], s.bob)

	// Bob should still be able to join even though he doesn't satisfy the admin requirement
	// because he satisfies the member one
	s.joinCommunity(community, s.bob, bobPassword, []string{})

	// Verify that we have Bob's revealed account
	revealedAccounts, err := s.owner.GetRevealedAccounts(community.ID(), common.PubkeyToHex(&s.bob.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().Len(revealedAccounts, 1)
	s.Require().Equal(bobAddress, revealedAccounts[0].Address)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestJoinCommunityAsAdminWithMemberAndAdminPermission() {

	community, _ := s.createCommunity()

	// setup become member permission
	permissionRequestMember := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}
	response, err := s.owner.CreateCommunityTokenPermission(&permissionRequestMember)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	waitOnCommunityPermissionCreated := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return sub.Community.HasTokenPermissions()
	})

	err = <-waitOnCommunityPermissionCreated
	s.Require().NoError(err)

	// setup become admin permission
	permissionRequestAdmin := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_ADMIN,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x124"},
				Symbol:            "TESTADMIN",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}
	response, err = s.owner.CreateCommunityTokenPermission(&permissionRequestAdmin)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Communities()[0].TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_ADMIN), 1)
	s.Require().Len(response.Communities()[0].TokenPermissions(), 2)

	waitOnCommunityPermissionCreated = waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return len(sub.Community.TokenPermissions()) == 2
	})

	err = <-waitOnCommunityPermissionCreated
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.TokenPermissions(), 2)

	s.advertiseCommunityTo(community, s.bob)

	// make bob satisfy the admin criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, permissionRequestAdmin.TokenCriteria[0])

	// Bob should still be able to join even though he doesn't satisfy the member requirement
	// because he satisfies the admin one
	s.joinCommunity(community, s.bob, bobPassword, []string{})

	// Verify that we have Bob's revealed account
	revealedAccounts, err := s.owner.GetRevealedAccounts(community.ID(), common.PubkeyToHex(&s.bob.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().Len(revealedAccounts, 1)
	s.Require().Equal(bobAddress, revealedAccounts[0].Address)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) testViewChannelPermissions(viewersCanAddReactions bool) {
	community, chat := s.createCommunity()

	// setup channel reactions permissions
	editedChat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: chat.Name,
			Description: chat.Description,
			Emoji:       chat.Emoji,
			Color:       chat.Color,
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		ViewersCanPostReactions: viewersCanAddReactions,
	}

	_, err := s.owner.EditCommunityChat(community.ID(), chat.ID, editedChat)
	s.Require().NoError(err)

	// bob joins the community
	s.advertiseCommunityTo(community, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{})

	// send message to the channel
	msg := s.sendChatMessage(s.owner, chat.ID, "hello on open community")

	// bob can read the message
	response, err := WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			_, ok := r.messages[msg.ID]
			return ok
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(msg.Text, response.Messages()[0].Text)

	waitOnBobToBeKickedFromChannel := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		channel, ok := sub.Community.Chats()[chat.CommunityChatID()]
		return ok && len(channel.Members) == 1
	})
	waitOnChannelToBeRekeyedOnceBobIsKicked := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		action, ok := sub.keyActions.ChannelKeysActions[chat.CommunityChatID()]
		return ok && (action.ActionType == communities.EncryptionKeyRekey || action.ActionType == communities.EncryptionKeyAdd)
	})

	// setup view channel permission
	channelPermissionRequest := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
		ChatIds: []string{chat.ID},
	}

	response, err = s.owner.CreateCommunityTokenPermission(&channelPermissionRequest)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(s.owner.communitiesManager.IsChannelEncrypted(community.IDString(), chat.ID))

	err = <-waitOnBobToBeKickedFromChannel
	s.Require().NoError(err)

	err = <-waitOnChannelToBeRekeyedOnceBobIsKicked
	s.Require().NoError(err)

	// bob receives community changes
	// channel members should be empty,
	// this info is available only to channel members
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			c, err := s.bob.GetCommunityByID(community.ID())
			if err != nil {
				return false
			}
			if c == nil {
				return false
			}
			channel := c.Chats()[chat.CommunityChatID()]
			return channel != nil && len(channel.Members) == 0
		},
		"no community that satisfies criteria",
	)
	s.Require().NoError(err)

	// make bob satisfy channel criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, channelPermissionRequest.TokenCriteria[0])
	defer s.resetMockedBalances() // reset mocked balances, this test in run with different test cases

	waitOnChannelKeyToBeDistributedToBob := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		action, ok := sub.keyActions.ChannelKeysActions[chat.CommunityChatID()]
		if !ok || action.ActionType != communities.EncryptionKeySendToMembers {
			return false
		}
		_, ok = action.Members[common.PubkeyToHex(&s.bob.identity.PublicKey)]
		return ok
	})

	// force owner to reevaluate channel members
	// in production it will happen automatically, by periodic check
	community, _, err = s.owner.communitiesManager.ReevaluateMembers(community.ID())
	s.Require().NoError(err)

	err = <-waitOnChannelKeyToBeDistributedToBob
	s.Require().NoError(err)

	// send message to the channel
	msg = s.sendChatMessage(s.owner, chat.ID, "hello on closed channel")

	// bob can read the message
	response, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			_, ok := r.messages[msg.ID]
			return ok
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(msg.Text, response.Messages()[0].Text)

	// bob can/can't post reactions
	response, err = s.bob.SendEmojiReaction(context.Background(), chat.ID, msg.ID, protobuf.EmojiReaction_THUMBS_UP)
	if !viewersCanAddReactions {
		s.Require().Error(err)
	} else {
		s.Require().NoError(err)
		s.Require().Len(response.emojiReactions, 1)
		reactionMessage := response.EmojiReactions()[0]

		response, err = WaitOnMessengerResponse(
			s.owner,
			func(r *MessengerResponse) bool {
				_, ok := r.emojiReactions[reactionMessage.ID()]
				return ok
			},
			"no reactions received",
		)

		if viewersCanAddReactions {
			s.Require().NoError(err)
			s.Require().Len(response.EmojiReactions(), 1)
			s.Require().Equal(response.EmojiReactions()[0].Type, protobuf.EmojiReaction_THUMBS_UP)
		} else {
			s.Require().Error(err)
			s.Require().Len(response.EmojiReactions(), 0)
		}
	}
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestViewChannelPermissions() {
	testCases := []struct {
		name                    string
		viewersCanPostReactions bool
	}{
		{
			name:                    "viewers are allowed to post reactions",
			viewersCanPostReactions: true,
		},
		{
			name:                    "viewers are forbidden to post reactions",
			viewersCanPostReactions: false,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(*testing.T) {
			s.testViewChannelPermissions(tc.viewersCanPostReactions)
		})
	}
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestMemberRoleGetUpdatedWhenChangingPermissions() {
	community, chat := s.createCommunity()

	// bob joins the community
	s.advertiseCommunityTo(community, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{})

	community, err := s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	// send message to the channel
	msg := s.sendChatMessage(s.owner, chat.ID, "hello on open community")

	// bob can read the message
	response, err := WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			_, ok := r.messages[msg.ID]
			return ok
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(msg.Text, response.Messages()[0].Text)

	waitOnBobToBeKickedFromChannel := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		channel, ok := sub.Community.Chats()[chat.CommunityChatID()]
		return ok && len(channel.Members) == 1
	})
	waitOnChannelToBeRekeyedOnceBobIsKicked := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		action, ok := sub.keyActions.ChannelKeysActions[chat.CommunityChatID()]
		return ok && (action.ActionType == communities.EncryptionKeyRekey || action.ActionType == communities.EncryptionKeyAdd)
	})

	// setup view channel permission
	channelPermissionRequest := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
		ChatIds: []string{chat.ID},
	}

	response, err = s.owner.CreateCommunityTokenPermission(&channelPermissionRequest)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunityChanges[0].TokenPermissionsAdded, 1)
	s.Require().True(s.owner.communitiesManager.IsChannelEncrypted(community.IDString(), chat.ID))

	err = <-waitOnBobToBeKickedFromChannel
	s.Require().NoError(err)

	err = <-waitOnChannelToBeRekeyedOnceBobIsKicked
	s.Require().NoError(err)

	// bob receives community changes
	// channel members should be empty,
	// this info is available only to channel members
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			c, err := s.bob.GetCommunityByID(community.ID())
			if err != nil {
				return false
			}
			if c == nil {
				return false
			}
			channel := c.Chats()[chat.CommunityChatID()]
			return channel != nil && len(channel.Members) == 0
		},
		"no community that satisfies criteria",
	)
	s.Require().NoError(err)

	// make bob satisfy channel criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, channelPermissionRequest.TokenCriteria[0])
	defer s.resetMockedBalances() // reset mocked balances, this test in run with different test cases

	waitOnChannelKeyToBeDistributedToBob := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		action, ok := sub.keyActions.ChannelKeysActions[chat.CommunityChatID()]
		if !ok || action.ActionType != communities.EncryptionKeySendToMembers {
			return false
		}
		_, ok = action.Members[common.PubkeyToHex(&s.bob.identity.PublicKey)]
		return ok
	})

	// force owner to reevaluate channel members
	// in production it will happen automatically, by periodic check
	community, _, err = s.owner.communitiesManager.ReevaluateMembers(community.ID())
	s.Require().NoError(err)

	err = <-waitOnChannelKeyToBeDistributedToBob
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	chatID := strings.TrimPrefix(chat.ID, community.IDString())
	members := community.Chats()[chatID].Members
	s.Require().Len(members, 2)
	// confirm that member is a viewer and not a poster
	s.Require().Equal(protobuf.CommunityMember_CHANNEL_ROLE_VIEWER, members[s.bob.IdentityPublicKeyString()].ChannelRole)

	// send message to the channel
	msg = s.sendChatMessage(s.owner, chat.ID, "hello on closed channel")

	// bob can read the message
	response, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			_, ok := r.messages[msg.ID]
			return ok
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(msg.Text, response.Messages()[0].Text)

	tokenPermissions := community.TokenPermissions()

	var tokenPermissionID string
	for id := range tokenPermissions {
		tokenPermissionID = id
	}

	// Edit permission so that Bob can now be a poster to show that member role can be edited
	channelPermissionRequest.Type = protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL
	editChannelPermissionRequest := requests.EditCommunityTokenPermission{
		PermissionID:                   tokenPermissionID,
		CreateCommunityTokenPermission: channelPermissionRequest,
	}
	response, err = s.owner.EditCommunityTokenPermission(&editChannelPermissionRequest)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(s.owner.communitiesManager.IsChannelEncrypted(community.IDString(), chat.ID))
	s.Require().Len(response.CommunityChanges[0].TokenPermissionsModified, 1)

	// force owner to reevaluate channel members
	// in production it will happen automatically, by periodic check
	community, roles, err := s.owner.communitiesManager.ReevaluateMembers(community.ID())
	s.Require().NoError(err)
	s.Require().Len(roles, 2)
	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	members = community.Chats()[chatID].Members
	s.Require().Len(members, 2)
	// confirm that member is now a poster
	s.Require().Equal(protobuf.CommunityMember_CHANNEL_ROLE_POSTER, members[s.bob.IdentityPublicKeyString()].ChannelRole)

	err = <-waitOnChannelKeyToBeDistributedToBob
	s.Require().NoError(err)

	// wait until bob permissions are upgraded
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			community, err = s.bob.communitiesManager.GetByID(community.ID())
			s.Require().NoError(err)
			chats := community.Chats()
			if len(chats) == 0 {
				return false
			}
			if chats[chat.ID] == nil {
				return false
			}
			members = chats[chat.ID].Members
			return len(members) == 2 && members[s.bob.myHexIdentity()] != nil && members[s.bob.myHexIdentity()].ChannelRole == protobuf.CommunityMember_CHANNEL_ROLE_POSTER
		},
		"bob never got post permissions",
	)

	msg = s.sendChatMessage(s.bob, chat.ID, "hello on closed channel from Bob")

	// owner can read the message
	response, err = WaitOnMessengerResponse(
		s.owner,
		func(r *MessengerResponse) bool {
			_, ok := r.messages[msg.ID]
			return ok
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(msg.Text, response.Messages()[0].Text)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) testReevaluateMemberPrivilegedRoleInOpenCommunity(permissionType protobuf.CommunityTokenPermission_Type) {
	community, _ := s.createCommunity()

	createTokenPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        permissionType,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	response, err := s.owner.CreateCommunityTokenPermission(createTokenPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(response.Communities()[0].HasTokenPermissions())

	waitOnCommunityPermissionCreated := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return sub.Community.HasTokenPermissions()
	})

	err = <-waitOnCommunityPermissionCreated
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.HasTokenPermissions())

	s.advertiseCommunityTo(community, s.alice)

	var tokenPermission *communities.CommunityTokenPermission
	for _, tokenPermission = range community.TokenPermissions() {
		break
	}

	s.makeAddressSatisfyTheCriteria(testChainID1, aliceAddress1, tokenPermission.TokenCriteria[0])

	// join community as a privileged user
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(checkRoleBasedOnThePermissionType(permissionType, &s.alice.identity.PublicKey, community))

	// the control node re-evaluates the roles of the participants, checking that the privileged user has not lost his role
	community, _, err = s.owner.communitiesManager.ReevaluateMembers(community.ID())
	s.Require().NoError(err)
	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(checkRoleBasedOnThePermissionType(permissionType, &s.alice.identity.PublicKey, community))

	// remove privileged token permission and reevaluate member permissions
	deleteTokenPermission := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: tokenPermission.Id,
	}

	response, err = s.owner.DeleteCommunityTokenPermission(deleteTokenPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().False(response.Communities()[0].HasTokenPermissions())

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().False(community.HasTokenPermissions())

	community, _, err = s.owner.communitiesManager.ReevaluateMembers(community.ID())
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))
	s.Require().False(checkRoleBasedOnThePermissionType(permissionType, &s.alice.identity.PublicKey, community))
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestReevaluateMemberAdminRoleInOpenCommunity() {
	s.testReevaluateMemberPrivilegedRoleInOpenCommunity(protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestReevaluateMemberTokenMasterRoleInOpenCommunity() {
	s.testReevaluateMemberPrivilegedRoleInOpenCommunity(protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) testReevaluateMemberPrivilegedRoleInClosedCommunity(permissionType protobuf.CommunityTokenPermission_Type) {
	community, _ := s.createCommunity()

	createTokenPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        permissionType,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	response, err := s.owner.CreateCommunityTokenPermission(createTokenPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().True(response.Communities()[0].HasTokenPermissions())

	createTokenMemberPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x124"},
				Symbol:            "TEST2",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	waitOnCommunityPermissionCreated := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return len(sub.Community.TokenPermissions()) == 2
	})

	response, err = s.owner.CreateCommunityTokenPermission(createTokenMemberPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.HasTokenPermissions())
	s.Require().Len(community.TokenPermissions(), 2)

	err = <-waitOnCommunityPermissionCreated
	s.Require().NoError(err)

	s.advertiseCommunityTo(community, s.alice)

	var tokenPermission *communities.CommunityTokenPermission
	var tokenMemberPermission *communities.CommunityTokenPermission
	for _, permission := range community.TokenPermissions() {
		if permission.Type == protobuf.CommunityTokenPermission_BECOME_MEMBER {
			tokenMemberPermission = permission
		} else {
			tokenPermission = permission
		}
	}

	s.makeAddressSatisfyTheCriteria(testChainID1, aliceAddress1, tokenPermission.TokenCriteria[0])
	s.makeAddressSatisfyTheCriteria(testChainID1, aliceAddress1, tokenMemberPermission.TokenCriteria[0])

	// join community as a privileged user
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(checkRoleBasedOnThePermissionType(permissionType, &s.alice.identity.PublicKey, community))

	// the control node reevaluates the roles of the participants, checking that the privileged user has not lost his role
	community, _, err = s.owner.communitiesManager.ReevaluateMembers(community.ID())
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(checkRoleBasedOnThePermissionType(permissionType, &s.alice.identity.PublicKey, community))

	deleteTokenPermission := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: tokenPermission.Id,
	}

	// remove privileged token permission and reevaluate member permissions
	response, err = s.owner.DeleteCommunityTokenPermission(deleteTokenPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Communities()[0].TokenPermissions(), 1)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(response.Communities()[0].TokenPermissions(), 1)

	community, _, err = s.owner.communitiesManager.ReevaluateMembers(community.ID())
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))
	s.Require().False(checkRoleBasedOnThePermissionType(permissionType, &s.alice.identity.PublicKey, community))

	// delete member permissions and reevaluate user permissions
	deleteMemberTokenPermission := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: tokenMemberPermission.Id,
	}

	response, err = s.owner.DeleteCommunityTokenPermission(deleteMemberTokenPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Communities()[0].TokenPermissions(), 0)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(response.Communities()[0].TokenPermissions(), 0)

	community, _, err = s.owner.communitiesManager.ReevaluateMembers(community.ID())
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.HasMember(&s.alice.identity.PublicKey))

	s.Require().False(checkRoleBasedOnThePermissionType(permissionType, &s.alice.identity.PublicKey, community))
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestReevaluateMemberAdminRoleInClosedCommunity() {
	s.testReevaluateMemberPrivilegedRoleInClosedCommunity(protobuf.CommunityTokenPermission_BECOME_ADMIN)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestReevaluateMemberTokenMasterRoleInClosedCommunity() {
	s.testReevaluateMemberPrivilegedRoleInClosedCommunity(protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func checkRoleBasedOnThePermissionType(permissionType protobuf.CommunityTokenPermission_Type, member *ecdsa.PublicKey, community *communities.Community) bool {
	switch permissionType {
	case protobuf.CommunityTokenPermission_BECOME_ADMIN:
		return community.IsMemberAdmin(member)
	case protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER:
		return community.IsMemberTokenMaster(member)
	default:
		panic("Unknown permission, please, update the test")
	}
}