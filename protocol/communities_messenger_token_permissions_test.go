package protocol

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/account"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

const testChainID1 = 1

const ownerPassword = "123456"
const alicePassword = "qwerty"
const bobPassword = "bob123"

const ownerAddress = "0x0100000000000000000000000000000000000000"
const aliceAddress1 = "0x0200000000000000000000000000000000000000"
const aliceAddress2 = "0x0210000000000000000000000000000000000000"
const bobAddress = "0x0300000000000000000000000000000000000000"

type AccountManagerMock struct {
	AccountsMap map[string]string
}

func (m *AccountManagerMock) GetVerifiedWalletAccount(db *accounts.Database, address, password string) (*account.SelectedExtKey, error) {
	return &account.SelectedExtKey{
		Address: types.HexToAddress(address),
	}, nil
}

func (m *AccountManagerMock) CanRecover(rpcParams account.RecoverParams, revealedAddress types.Address) (bool, error) {
	return true, nil
}

func (m *AccountManagerMock) Sign(rpcParams account.SignParams, verifiedAccount *account.SelectedExtKey) (result types.HexBytes, err error) {
	return types.HexBytes{}, nil
}

func (m *AccountManagerMock) DeleteAccount(address types.Address) error {
	return nil
}

type TokenManagerMock struct {
	Balances *map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big
}

func (m *TokenManagerMock) GetAllChainIDs() ([]uint64, error) {
	chainIDs := make([]uint64, 0, len(*m.Balances))
	for key := range *m.Balances {
		chainIDs = append(chainIDs, key)
	}
	return chainIDs, nil
}

func (m *TokenManagerMock) GetBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, error) {
	time.Sleep(100 * time.Millisecond) // simulate response time
	return *m.Balances, nil
}

type CommunityAndKeyActions struct {
	community  *communities.Community
	keyActions *communities.EncryptionKeyActions
}

type TestCommunitiesKeyDistributor struct {
	CommunitiesKeyDistributorImpl

	subscriptions map[chan *CommunityAndKeyActions]bool
	mutex         sync.RWMutex
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

	subscription := make(chan *CommunityAndKeyActions)
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
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger

	mockedBalances map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big // chainID, account, token, balance
}

func (s *MessengerCommunitiesTokenPermissionsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.owner = s.newMessenger(ownerPassword, []string{ownerAddress})
	s.bob = s.newMessenger(bobPassword, []string{bobAddress})
	s.alice = s.newMessenger(alicePassword, []string{aliceAddress1, aliceAddress2})
	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.mockedBalances = make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(aliceAddress1)] = make(map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(aliceAddress2)] = make(map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(bobAddress)] = make(map[gethcommon.Address]*hexutil.Big)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TearDownTest() {
	s.Require().NoError(s.owner.Shutdown())
	s.Require().NoError(s.bob.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesTokenPermissionsSuite) newMessenger(password string, walletAddresses []string) *Messenger {
	accountsManagerMock := &AccountManagerMock{}
	accountsManagerMock.AccountsMap = make(map[string]string)
	for _, walletAddress := range walletAddresses {
		accountsManagerMock.AccountsMap[walletAddress] = types.EncodeHex(crypto.Keccak256([]byte(password)))
	}

	tokenManagerMock := &TokenManagerMock{
		Balances: &s.mockedBalances,
	}

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newCommunitiesTestMessenger(s.shh, privateKey, s.logger, accountsManagerMock, tokenManagerMock)
	s.Require().NoError(err)

	currentDistributorObj, ok := messenger.communitiesKeyDistributor.(*CommunitiesKeyDistributorImpl)
	s.Require().True(ok)
	messenger.communitiesKeyDistributor = &TestCommunitiesKeyDistributor{
		CommunitiesKeyDistributorImpl: *currentDistributorObj,
		subscriptions:                 map[chan *CommunityAndKeyActions]bool{},
		mutex:                         sync.RWMutex{},
	}

	// add wallet account with keypair
	for _, walletAddress := range walletAddresses {
		kp := accounts.GetProfileKeypairForTest(false, true, false)
		kp.Accounts[0].Address = types.HexToAddress(walletAddress)
		err := messenger.settings.SaveOrUpdateKeypair(kp)
		s.Require().NoError(err)
	}

	walletAccounts, err := messenger.settings.GetAccounts()
	s.Require().NoError(err)
	s.Require().Len(walletAccounts, len(walletAddresses))
	for i := range walletAddresses {
		s.Require().Equal(walletAccounts[i].Type, accounts.AccountTypeGenerated)
	}
	return messenger
}

func (s *MessengerCommunitiesTokenPermissionsSuite) joinCommunity(community *communities.Community, user *Messenger, password string, addresses []string) {
	s.joinCommunityWithAirdropAddress(community, user, password, addresses, "")
}

func (s *MessengerCommunitiesTokenPermissionsSuite) joinCommunityWithAirdropAddress(community *communities.Community, user *Messenger, password string, addresses []string, airdropAddress string) {
	passwdHash := types.EncodeHex(crypto.Keccak256([]byte(password)))
	if airdropAddress == "" && len(addresses) > 0 {
		airdropAddress = addresses[0]
	}

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID(), Password: passwdHash, AddressesToReveal: addresses, AirdropAddress: airdropAddress}
	joinCommunity(&s.Suite, community, s.owner, user, request)
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
	balance, ok := new(big.Int).SetString(criteria.Amount, 10)
	s.Require().True(ok)
	decimalsFactor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(criteria.Decimals)), nil)
	balance.Mul(balance, decimalsFactor)

	s.mockedBalances[chainID][walletAddress][contractAddress] = (*hexutil.Big)(balance)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) waitOnCommunitiesEvent(user *Messenger, condition func(*communities.Subscription) bool) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		for {
			select {
			case sub, more := <-user.communitiesManager.Subscribe():
				if !more {
					errCh <- errors.New("channel closed when waiting for communities event")
					return
				}

				if condition(sub) {
					return
				}

			case <-time.After(500 * time.Millisecond):
				errCh <- errors.New("timed out when waiting for communities event")
				return
			}
		}
	}()

	return errCh
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
				Amount:            "100",
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
			s.Require().Equal(tc.Amount, "100")
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
				Amount:            "100",
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
	tokenPermission.TokenCriteria[0].Amount = "200"
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
			s.Require().Equal(tc.Amount, "200")
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

	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	// We try to join the org
	response, err = s.alice.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin1 := response.RequestsToJoinCommunity[0]
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin1.State)

	// Retrieve request to join
	err = tt.RetryWithBackOff(func() error {
		response, err = s.owner.RetrieveAll()
		return err
	})
	s.Require().NoError(err)
	// We don't expect a requestToJoin in the response because due
	// to missing revealed wallet addresses, the request should've
	// been declined right away
	s.Require().Len(response.RequestsToJoinCommunity, 0)

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

	for pubKey, member := range community.Members() {
		if pubKey != common.PubkeyToHex(&s.owner.identity.PublicKey) {

			switch pubKey {
			case common.PubkeyToHex(&s.alice.identity.PublicKey):
				s.Require().Len(member.RevealedAccounts, 2)
				s.Require().Equal(member.RevealedAccounts[0].Address, aliceAddress1)
				s.Require().Equal(member.RevealedAccounts[1].Address, aliceAddress2)
				s.Require().Equal(true, member.RevealedAccounts[0].IsAirdropAddress)
			case common.PubkeyToHex(&s.bob.identity.PublicKey):
				s.Require().Len(member.RevealedAccounts, 1)
				s.Require().Equal(member.RevealedAccounts[0].Address, bobAddress)
				s.Require().Equal(true, member.RevealedAccounts[0].IsAirdropAddress)
			default:
				s.Require().Fail("pubKey does not match expected keys")
			}
		}
	}
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestJoinedCommunityMembersSelectedSharedAddress() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.alice)

	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress2})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(2, community.MembersCount())

	for pubKey, member := range community.Members() {
		if pubKey != common.PubkeyToHex(&s.owner.identity.PublicKey) {
			s.Require().Len(member.RevealedAccounts, 1)

			switch pubKey {
			case common.PubkeyToHex(&s.alice.identity.PublicKey):
				s.Require().Equal(member.RevealedAccounts[0].Address, aliceAddress2)
				s.Require().Equal(true, member.RevealedAccounts[0].IsAirdropAddress)
			default:
				s.Require().Fail("pubKey does not match expected keys")
			}
		}
	}
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestJoinedCommunityMembersMultipleSelectedSharedAddresses() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.alice)

	s.joinCommunityWithAirdropAddress(community, s.alice, alicePassword, []string{aliceAddress1, aliceAddress2}, aliceAddress2)

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(2, community.MembersCount())

	for pubKey, member := range community.Members() {
		if pubKey != common.PubkeyToHex(&s.owner.identity.PublicKey) {
			s.Require().Len(member.RevealedAccounts, 2)

			switch pubKey {
			case common.PubkeyToHex(&s.alice.identity.PublicKey):
				s.Require().Equal(member.RevealedAccounts[0].Address, aliceAddress1)
				s.Require().Equal(member.RevealedAccounts[1].Address, aliceAddress2)
				s.Require().Equal(true, member.RevealedAccounts[1].IsAirdropAddress)
			default:
				s.Require().Fail("pubKey does not match expected keys")
			}
		}
	}
}

func (s *MessengerCommunitiesTokenPermissionsSuite) validateAliceAddress(community *communities.Community, wantedAddress string) error {
	for pubKey, member := range community.Members() {
		if pubKey != common.PubkeyToHex(&s.owner.identity.PublicKey) {
			s.Require().Len(member.RevealedAccounts, 1)

			switch pubKey {
			case common.PubkeyToHex(&s.alice.identity.PublicKey):
				if member.RevealedAccounts[0].Address != wantedAddress {
					return errors.New("Alice's address does not match the wanted address. Wanted " + wantedAddress + ", Found: " + member.RevealedAccounts[0].Address)
				}
			default:
				return errors.New("pubKey does not match expected keys")
			}
		}
	}
	return nil
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestEditSharedAddresses() {
	community, _ := s.createCommunity()
	s.advertiseCommunityTo(community, s.alice)

	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress2})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(2, community.MembersCount())

	err = s.validateAliceAddress(community, aliceAddress2)
	s.Require().NoError(err)

	passwdHash := types.EncodeHex(crypto.Keccak256([]byte(alicePassword)))
	request := &requests.EditSharedAddresses{CommunityID: community.ID(), Password: passwdHash, AddressesToReveal: []string{aliceAddress1}, AirdropAddress: aliceAddress1}
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
		community := response.Communities()[0]
		return s.validateAliceAddress(community, aliceAddress1)
	})
	s.Require().NoError(err)

	// Also check that the owner has the new address in their DB
	community, err = s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	err = s.validateAliceAddress(community, aliceAddress1)
	s.Require().NoError(err)

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

	// Check that Alice's community is updated with the new addresses
	community, err = s.alice.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	err = s.validateAliceAddress(community, aliceAddress1)
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestBecomeMemberPermissions() {
	community, chat := s.createCommunity()

	// bob joins the community
	s.advertiseCommunityTo(community, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{})

	// send message to the channel
	msg := s.sendChatMessage(s.owner, chat.ID, "hello on open community")

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
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(msg.Text, response.Messages()[0].Text)

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

	waitOnBobToBeKicked := s.waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return len(sub.Community.Members()) == 1
	})
	waitOnCommunityToBeRekeyedOnceBobIsKicked := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		return len(sub.community.Description().Members) == 1 &&
			sub.keyActions.CommunityKeyAction.ActionType == communities.EncryptionKeyRekey
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
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0
		},
		"no community",
	)
	s.Require().NoError(err)

	err = <-waitOnCommunityToBeRekeyedOnceBobIsKicked
	s.Require().NoError(err)

	// send message to channel
	msg = s.sendChatMessage(s.owner, chat.ID, "hello on encrypted community")

	// bob can't read the message
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			for _, message := range r.messages {
				if message.Text == msg.Text {
					return true
				}
			}
			return false
		},
		"no messages",
	)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "no messages")

	// bob tries to join, but he doesn't satisfy so the request isn't sent
	passwdHash := types.EncodeHex(crypto.Keccak256([]byte(bobPassword)))
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID(), Password: passwdHash, AddressesToReveal: []string{bobAddress}, AirdropAddress: bobAddress}
	_, err = s.bob.RequestToJoinCommunity(request)
	s.Require().Error(err)
	s.Require().ErrorContains(err, "permission to join not satisfied")

	// make sure bob does not have a pending request to join
	requests, err := s.bob.MyPendingRequestsToJoin()
	s.Require().NoError(err)
	s.Require().Len(requests, 0)

	// make bob satisfy the criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, permissionRequest.TokenCriteria[0])

	waitOnCommunityKeyToBeDistributedToBob := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		return len(sub.community.Description().Members) == 2 &&
			len(sub.keyActions.CommunityKeyAction.Members) == 1 &&
			sub.keyActions.CommunityKeyAction.ActionType == communities.EncryptionKeySendToMembers
	})

	// bob re-joins the community
	s.joinCommunity(community, s.bob, bobPassword, []string{})

	err = <-waitOnCommunityKeyToBeDistributedToBob
	s.Require().NoError(err)

	// send message to channel
	msg = s.sendChatMessage(s.owner, chat.ID, "hello on encrypted community")

	// bob can read the message
	response, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			for _, message := range r.messages {
				if message.Text == msg.Text {
					return true
				}
			}
			return false
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(msg.Text, response.Messages()[0].Text)
}
