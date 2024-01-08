package protocol

import (
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
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/communitytokens"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/waku"
)

func TestMessengerCommunitiesSignersSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunitiesSignersSuite))
}

type MessengerCommunitiesSignersSuite struct {
	suite.Suite
	john  *Messenger
	bob   *Messenger
	alice *Messenger

	shh    types.Waku
	logger *zap.Logger

	collectiblesServiceMock *CollectiblesServiceMock

	accountsTestData map[string]string

	mockedBalances map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big // chainID, account, token, balance
}

func (s *MessengerCommunitiesSignersSuite) SetupTest() {

	communities.SetValidateInterval(300 * time.Millisecond)

	s.logger = tt.MustCreateTestLogger()

	s.collectiblesServiceMock = &CollectiblesServiceMock{}

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	aliceAccountAddress := "0x0777100000000000000000000000000000000000"
	bobAccountAddress := "0x0330000000000000000000000000000000000000"
	accountPassword := "QWERTY"

	s.john = s.newMessenger("", []string{})
	s.bob = s.newMessenger(accountPassword, []string{aliceAccountAddress})
	s.alice = s.newMessenger(accountPassword, []string{bobAccountAddress})
	_, err := s.john.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.accountsTestData = make(map[string]string)
	s.accountsTestData[common.PubkeyToHex(&s.bob.identity.PublicKey)] = bobAccountAddress
	s.accountsTestData[common.PubkeyToHex(&s.alice.identity.PublicKey)] = aliceAccountAddress
}

func (s *MessengerCommunitiesSignersSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.john)
	TearDownMessenger(&s.Suite, s.bob)
	TearDownMessenger(&s.Suite, s.alice)
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesSignersSuite) newMessenger(password string, walletAddresses []string) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	accountsManagerMock := &AccountManagerMock{}
	accountsManagerMock.AccountsMap = make(map[string]string)

	for _, walletAddress := range walletAddresses {
		accountsManagerMock.AccountsMap[walletAddress] = types.EncodeHex(crypto.Keccak256([]byte(password)))
	}

	tokenManagerMock := &TokenManagerMock{
		Balances: &s.mockedBalances,
	}

	messenger, err := newCommunitiesTestMessenger(s.shh, privateKey, s.logger, accountsManagerMock, tokenManagerMock, s.collectiblesServiceMock)
	s.Require().NoError(err)

	return messenger
}

func (s *MessengerCommunitiesSignersSuite) createCommunity(controlNode *Messenger) *communities.Community {
	community, _ := createCommunity(&s.Suite, controlNode)
	return community
}

func (s *MessengerCommunitiesSignersSuite) advertiseCommunityTo(controlNode *Messenger, communityID types.HexBytes, user *Messenger) {
	advertiseCommunityTo(&s.Suite, communityID, controlNode, user)
}

func (s *MessengerCommunitiesSignersSuite) joinCommunity(controlNode *Messenger, community *communities.Community, user *Messenger) {
	accTestData := s.accountsTestData[common.PubkeyToHex(&s.alice.identity.PublicKey)]
	array64Bytes := common.HashPublicKey(&s.alice.identity.PublicKey)
	signature := append([]byte{0}, array64Bytes...)

	request := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{accTestData},
		AirdropAddress:    accTestData,
		Signatures:        []types.HexBytes{signature},
	}

	joinCommunity(&s.Suite, community, controlNode, user, request, "")
}

func (s *MessengerCommunitiesSignersSuite) joinOnRequestCommunity(controlNode *Messenger, community *communities.Community, user *Messenger) {
	accTestData := s.accountsTestData[common.PubkeyToHex(&s.alice.identity.PublicKey)]
	array64Bytes := common.HashPublicKey(&s.alice.identity.PublicKey)
	signature := append([]byte{0}, array64Bytes...)

	request := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{accTestData},
		AirdropAddress:    accTestData,
		Signatures:        []types.HexBytes{signature},
	}

	joinOnRequestCommunity(&s.Suite, community, controlNode, user, request)
}

// John crates a community
// Ownership is transferred to Alice
// Alice kick all members Bob and John
// Bob automatically rejoin
// John receive AC notification to share the address and join to the community
// Bob and John accepts the changes

func (s *MessengerCommunitiesSignersSuite) TestControlNodeUpdateSigner() {
	// Create a community
	// Transfer ownership
	// Process message
	community := s.createCommunity(s.john)

	s.advertiseCommunityTo(s.john, community.ID(), s.bob)
	s.advertiseCommunityTo(s.john, community.ID(), s.alice)

	s.joinCommunity(s.john, community, s.bob)
	s.joinCommunity(s.john, community, s.alice)

	// john mints owner token
	var chainID uint64 = 1
	tokenAddress := "token-address"
	tokenName := "tokenName"
	tokenSymbol := "TSM"
	_, err := s.john.SaveCommunityToken(&token.CommunityToken{
		TokenType:       protobuf.CommunityTokenType_ERC721,
		CommunityID:     community.IDString(),
		Address:         tokenAddress,
		ChainID:         int(chainID),
		Name:            tokenName,
		Supply:          &bigint.BigInt{},
		Symbol:          tokenSymbol,
		PrivilegesLevel: token.OwnerLevel,
	}, nil)
	s.Require().NoError(err)

	// john adds minted owner token to community
	err = s.john.AddCommunityToken(community.IDString(), int(chainID), tokenAddress)
	s.Require().NoError(err)

	// update mock - the signer for the community returned by the contracts should be john
	s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.john.identity.PublicKey))
	s.collectiblesServiceMock.SetMockCollectibleContractData(chainID, tokenAddress,
		&communitytokens.CollectibleContractData{TotalSupply: &bigint.BigInt{}})

	// bob accepts community update
	_, err = WaitOnSignaledMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && len(r.Communities()[0].TokenPermissions()) == 1
		},
		"no communities",
	)
	s.Require().NoError(err)

	// alice accepts community update
	_, err = WaitOnSignaledMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && len(r.Communities()[0].TokenPermissions()) == 1
		},
		"no communities",
	)
	s.Require().NoError(err)

	// Ownership token will be transferred to Alice and she will kick all members
	// and request kicked members to rejoin
	// the signer for the community returned by the contracts should be alice
	s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))

	response, err := s.alice.PromoteSelfToControlNode(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)

	community, err = s.alice.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.IsControlNode())
	s.Require().True(common.IsPubKeyEqual(community.ControlNode(), &s.alice.identity.PublicKey))
	s.Require().True(community.IsOwner())

	// check that Bob received kick event, also he will receive
	// request to share RevealedAddresses and send request to join to the control node
	_, err = WaitOnSignaledMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&s.bob.identity.PublicKey)
		},
		"Bob was not kicked from the community",
	)
	s.Require().NoError(err)

	// check that John received kick event, and AC notification msg created
	// John, as ex-owner must manually join the community
	_, err = WaitOnSignaledMessengerResponse(
		s.john,
		func(r *MessengerResponse) bool {
			wasKicked := len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&s.john.identity.PublicKey)
			sharedNotificationExist := false
			for _, acNotification := range r.ActivityCenterNotifications() {
				if acNotification.Type == ActivityCenterNotificationTypeShareAccounts {
					sharedNotificationExist = true
					break
				}
			}
			return wasKicked && sharedNotificationExist
		},
		"John was not kicked from the community",
	)
	s.Require().NoError(err)

	// Alice auto-accept requests to join with RevealedAddresses
	_, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && len(r.Communities()[0].Members()) == 2
		},
		"no community update with accepted request",
	)
	s.Require().NoError(err)

	validateResults := func(messenger *Messenger) *communities.Community {
		community, err = messenger.communitiesManager.GetByID(community.ID())
		s.Require().NoError(err)
		s.Require().True(common.IsPubKeyEqual(community.ControlNode(), &s.alice.identity.PublicKey))
		s.Require().Len(community.Members(), 2)
		s.Require().True(community.HasMember(&messenger.identity.PublicKey))

		return community
	}

	community = validateResults(s.alice)
	s.Require().True(community.IsControlNode())
	s.Require().True(community.IsOwner())

	// Bob is a community member again
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&s.bob.identity.PublicKey)
		},
		"Bob was auto-accepted",
	)
	s.Require().NoError(err)

	community = validateResults(s.bob)
	s.Require().False(community.IsControlNode())
	s.Require().False(community.IsOwner())

	// John manually joins the community
	s.joinCommunity(s.alice, community, s.john)

	// Alice change community name

	expectedName := "Alice owns community"

	response, err = s.alice.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
			Name:        expectedName,
			Color:       "#000000",
			Description: "edited community description",
		},
	})

	s.Require().NoError(err)
	s.Require().NotNil(response)

	validateNameInResponse := func(r *MessengerResponse) bool {
		return len(r.Communities()) > 0 && r.Communities()[0].IDString() == community.IDString() &&
			r.Communities()[0].Name() == expectedName
	}

	s.Require().True(validateNameInResponse(response))

	validateNameInDB := func(messenger *Messenger) {
		community, err = messenger.communitiesManager.GetByID(community.ID())
		s.Require().NoError(err)
		s.Require().Equal(expectedName, response.Communities()[0].Name())
	}

	validateNameInDB(s.alice)

	// john accepts community update from alice (new control node)
	_, err = WaitOnMessengerResponse(
		s.john,
		validateNameInResponse,
		"john did not receive community name update",
	)
	s.Require().NoError(err)
	validateNameInDB(s.john)

	// bob accepts community update from alice (new control node)
	_, err = WaitOnMessengerResponse(
		s.bob,
		validateNameInResponse,
		"bob did not receive community name update",
	)
	s.Require().NoError(err)
	validateNameInDB(s.bob)
}

func (s *MessengerCommunitiesSignersSuite) TestAutoAcceptOnOwnershipChangeRequestRequired() {
	community, _ := createOnRequestCommunity(&s.Suite, s.john)

	s.advertiseCommunityTo(s.john, community.ID(), s.bob)
	s.advertiseCommunityTo(s.john, community.ID(), s.alice)

	s.joinOnRequestCommunity(s.john, community, s.bob)
	s.joinOnRequestCommunity(s.john, community, s.alice)

	// john mints owner token
	var chainID uint64 = 1
	tokenAddress := "token-address"
	tokenName := "tokenName"
	tokenSymbol := "TSM"
	_, err := s.john.SaveCommunityToken(&token.CommunityToken{
		TokenType:       protobuf.CommunityTokenType_ERC721,
		CommunityID:     community.IDString(),
		Address:         tokenAddress,
		ChainID:         int(chainID),
		Name:            tokenName,
		Supply:          &bigint.BigInt{},
		Symbol:          tokenSymbol,
		PrivilegesLevel: token.OwnerLevel,
	}, nil)
	s.Require().NoError(err)

	err = s.john.AddCommunityToken(community.IDString(), int(chainID), tokenAddress)
	s.Require().NoError(err)

	// set john as contract owner
	s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.john.identity.PublicKey))
	s.collectiblesServiceMock.SetMockCollectibleContractData(chainID, tokenAddress,
		&communitytokens.CollectibleContractData{TotalSupply: &bigint.BigInt{}})

	hasTokenPermission := func(r *MessengerResponse) bool {
		return len(r.Communities()) > 0 && r.Communities()[0].HasTokenPermissions()
	}

	// bob received owner permissions
	_, err = WaitOnSignaledMessengerResponse(
		s.bob,
		hasTokenPermission,
		"no communities with token permission for Bob",
	)
	s.Require().NoError(err)

	// alice received owner permissions
	_, err = WaitOnSignaledMessengerResponse(
		s.alice,
		hasTokenPermission,
		"no communities with token permission for Alice",
	)
	s.Require().NoError(err)

	// simulate Alice received owner token
	s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))

	// after receiving owner token - set up control node, set up owner role, kick all members
	// and request kicked members to rejoin
	response, err := s.alice.PromoteSelfToControlNode(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)
	community, err = s.alice.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.IsControlNode())
	s.Require().True(common.IsPubKeyEqual(community.ControlNode(), &s.alice.identity.PublicKey))
	s.Require().True(community.IsOwner())

	// check that client received kick event
	// Bob will receive request to share RevealedAddresses and send request to join to the control node
	_, err = WaitOnSignaledMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&s.bob.identity.PublicKey)
		},
		"Bob was not kicked from the community",
	)
	s.Require().NoError(err)

	// check that client received kick event
	// John will receive request to share RevealedAddresses and send request to join to the control node
	_, err = WaitOnSignaledMessengerResponse(
		s.john,
		func(r *MessengerResponse) bool {
			wasKicked := len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&s.john.identity.PublicKey)
			sharedNotificationExist := false
			for _, acNotification := range r.ActivityCenterNotifications() {
				if acNotification.Type == ActivityCenterNotificationTypeShareAccounts {
					sharedNotificationExist = true
					break
				}
			}
			return wasKicked && sharedNotificationExist
		},
		"John was not kicked from the community",
	)
	s.Require().NoError(err)

	// Alice auto-accept requests to join with RevealedAddresses
	_, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && len(r.Communities()[0].Members()) == 2
		},
		"no community update with accepted request",
	)
	s.Require().NoError(err)

	validateResults := func(messenger *Messenger) *communities.Community {
		community, err = messenger.communitiesManager.GetByID(community.ID())
		s.Require().NoError(err)
		s.Require().True(common.IsPubKeyEqual(community.ControlNode(), &s.alice.identity.PublicKey))
		s.Require().Len(community.Members(), 2)
		s.Require().True(community.HasMember(&messenger.identity.PublicKey))

		return community
	}

	community = validateResults(s.alice)
	s.Require().True(community.IsControlNode())
	s.Require().True(community.IsOwner())

	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&s.bob.identity.PublicKey)
		},
		"Bob was auto-accepted",
	)
	s.Require().NoError(err)

	community = validateResults(s.bob)
	s.Require().False(community.IsControlNode())
	s.Require().False(community.IsOwner())
}

func (s *MessengerCommunitiesSignersSuite) TestNewOwnerAcceptRequestToJoin() {
	// Create a community
	// Transfer ownership
	// New owner accepts new request to join
	community := s.createCommunity(s.john)

	s.advertiseCommunityTo(s.john, community.ID(), s.alice)

	s.joinCommunity(s.john, community, s.alice)

	// john mints owner token
	var chainID uint64 = 1
	tokenAddress := "token-address"
	tokenName := "tokenName"
	tokenSymbol := "TSM"
	_, err := s.john.SaveCommunityToken(&token.CommunityToken{
		TokenType:       protobuf.CommunityTokenType_ERC721,
		CommunityID:     community.IDString(),
		Address:         tokenAddress,
		ChainID:         int(chainID),
		Name:            tokenName,
		Supply:          &bigint.BigInt{},
		Symbol:          tokenSymbol,
		PrivilegesLevel: token.OwnerLevel,
	}, nil)
	s.Require().NoError(err)

	// john adds minted owner token to community
	err = s.john.AddCommunityToken(community.IDString(), int(chainID), tokenAddress)
	s.Require().NoError(err)

	// update mock - the signer for the community returned by the contracts should be john
	s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.john.identity.PublicKey))
	s.collectiblesServiceMock.SetMockCollectibleContractData(chainID, tokenAddress,
		&communitytokens.CollectibleContractData{TotalSupply: &bigint.BigInt{}})

	// alice accepts community update
	_, err = WaitOnSignaledMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && len(r.Communities()[0].TokenPermissions()) == 1
		},
		"no communities",
	)
	s.Require().NoError(err)

	// Ownership token will be transferred to Alice and she will kick all members
	// and request kicked members to rejoin
	// the signer for the community returned by the contracts should be alice
	s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))

	response, err := s.alice.PromoteSelfToControlNode(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)

	community, err = s.alice.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.IsControlNode())
	s.Require().True(common.IsPubKeyEqual(community.ControlNode(), &s.alice.identity.PublicKey))
	s.Require().True(community.IsOwner())

	// check that John received kick event, also he will receive
	// request to share RevealedAddresses and send request to join to the control node
	_, err = WaitOnSignaledMessengerResponse(
		s.john,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&s.john.identity.PublicKey)
		},
		"John was not kicked from the community",
	)
	s.Require().NoError(err)

	// Alice advertises community to Bob
	s.advertiseCommunityTo(s.alice, community.ID(), s.bob)

	// Bob joins the community
	s.joinCommunity(s.alice, community, s.bob)

}
