package protocol

import (
	"bytes"
	"context"
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

type TokenManagerMock struct {
}

func (m *TokenManagerMock) GetAllChainIDs() ([]uint64, error) {
	return []uint64{5}, nil
}

func (m *TokenManagerMock) GetBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, error) {
	return nil, nil
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

	tokenManagerMock := &TokenManagerMock{}

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newCommunitiesTestMessenger(s.shh, privateKey, s.logger, accountsManagerMock, tokenManagerMock)
	s.Require().NoError(err)

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

func (s *MessengerCommunitiesTokenPermissionsSuite) joinCommunityWithAddresses(community *communities.Community, user *Messenger, password string, addresses []string) {
	passwdHash := types.EncodeHex(crypto.Keccak256([]byte(password)))
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID(), Password: passwdHash, AddressesToReveal: addresses}
	joinCommunity(&s.Suite, community, s.owner, user, request)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestCreateTokenPermission() {
	community, _ := createCommunity(&s.Suite, s.owner)

	createTokenPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{uint64(1): "0x123"},
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
	community, _ := createCommunity(&s.Suite, s.owner)

	tokenPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{uint64(1): "0x123"},
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
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.bob.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunitiesSettings(), 1)

	community := response.Communities()[0]

	tokensMetadata := community.CommunityTokensMetadata()
	s.Require().Len(tokensMetadata, 0)

	newToken := &protobuf.CommunityTokenMetadata{
		ContractAddresses: map[uint64]string{3: "0xasd"},
		Description:       "desc1",
		Image:             "IMG1",
		TokenType:         protobuf.CommunityTokenType_ERC721,
		Symbol:            "SMB",
	}

	_, err = community.AddCommunityTokensMetadata(newToken)
	s.Require().NoError(err)
	tokensMetadata = community.CommunityTokensMetadata()
	s.Require().Len(tokensMetadata, 1)

	s.Require().Equal(tokensMetadata[0].ContractAddresses, newToken.ContractAddresses)
	s.Require().Equal(tokensMetadata[0].Description, newToken.Description)
	s.Require().Equal(tokensMetadata[0].Image, newToken.Image)
	s.Require().Equal(tokensMetadata[0].TokenType, newToken.TokenType)
	s.Require().Equal(tokensMetadata[0].Symbol, newToken.Symbol)
	s.Require().Equal(tokensMetadata[0].Name, newToken.Name)
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestRequestAccessWithENSTokenPermission() {
	community, _ := createCommunity(&s.Suite, s.owner)

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

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)

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
	community, _ := createCommunity(&s.Suite, s.owner)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)

	s.joinCommunityWithAddresses(community, s.alice, alicePassword, []string{})
	s.joinCommunityWithAddresses(community, s.bob, bobPassword, []string{})

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
			case common.PubkeyToHex(&s.bob.identity.PublicKey):
				s.Require().Len(member.RevealedAccounts, 1)
				s.Require().Equal(member.RevealedAccounts[0].Address, bobAddress)
			default:
				s.Require().Fail("pubKey does not match expected keys")
			}
		}
	}
}

func (s *MessengerCommunitiesTokenPermissionsSuite) TestJoinedCommunityMembersSelectedSharedAddress() {
	community, _ := createCommunity(&s.Suite, s.owner)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)

	s.joinCommunityWithAddresses(community, s.alice, alicePassword, []string{aliceAddress2})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(2, community.MembersCount())

	for pubKey, member := range community.Members() {
		if pubKey != common.PubkeyToHex(&s.owner.identity.PublicKey) {
			s.Require().Len(member.RevealedAccounts, 1)

			switch pubKey {
			case common.PubkeyToHex(&s.alice.identity.PublicKey):
				s.Require().Equal(member.RevealedAccounts[0].Address, aliceAddress2)
			default:
				s.Require().Fail("pubKey does not match expected keys")
			}
		}
	}
}
