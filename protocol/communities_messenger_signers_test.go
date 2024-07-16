package protocol

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"

	utils "github.com/status-im/status-go/common"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/thirdparty"
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

	mockedBalances     communities.BalancesByChain
	mockedCollectibles communities.CollectiblesByChain
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

	accountPassword := "QWERTY"

	s.mockedBalances = make(communities.BalancesByChain)
	s.mockedBalances[testChainID1] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(aliceAddress1)] = make(map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(bobAddress)] = make(map[gethcommon.Address]*hexutil.Big)

	s.mockedCollectibles = make(communities.CollectiblesByChain)
	s.mockedCollectibles[testChainID1] = make(map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress)
	s.mockedCollectibles[testChainID1][gethcommon.HexToAddress(aliceAddress1)] = make(thirdparty.TokenBalancesPerContractAddress)
	s.mockedCollectibles[testChainID1][gethcommon.HexToAddress(bobAddress)] = make(thirdparty.TokenBalancesPerContractAddress)

	s.john = s.newMessenger("", []string{})
	s.bob = s.newMessenger(accountPassword, []string{aliceAddress1})
	s.alice = s.newMessenger(accountPassword, []string{bobAddress})
	_, err := s.john.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.accountsTestData = make(map[string]string)
	s.accountsTestData[common.PubkeyToHex(&s.bob.identity.PublicKey)] = bobAddress
	s.accountsTestData[common.PubkeyToHex(&s.alice.identity.PublicKey)] = aliceAddress1
}

func (s *MessengerCommunitiesSignersSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.john)
	TearDownMessenger(&s.Suite, s.bob)
	TearDownMessenger(&s.Suite, s.alice)
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesSignersSuite) newMessenger(password string, walletAddresses []string) *Messenger {
	communityManagerOptions := []communities.ManagerOption{}

	return newTestCommunitiesMessenger(&s.Suite, s.shh, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			logger:       s.logger,
			extraOptions: []Option{WithCommunityManagerOptions(communityManagerOptions)},
		},
		password:            password,
		walletAddresses:     walletAddresses,
		mockedBalances:      &s.mockedBalances,
		mockedCollectibles:  &s.mockedCollectibles,
		collectiblesService: s.collectiblesServiceMock,
	})
}

func (s *MessengerCommunitiesSignersSuite) createCommunity(controlNode *Messenger) *communities.Community {
	community, _ := createCommunity(&s.Suite, controlNode)
	return community
}

func (s *MessengerCommunitiesSignersSuite) advertiseCommunityTo(controlNode *Messenger, community *communities.Community, user *Messenger) {
	advertiseCommunityTo(&s.Suite, community, controlNode, user)
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
	accTestData := s.accountsTestData[common.PubkeyToHex(&user.identity.PublicKey)]
	array64Bytes := common.HashPublicKey(&user.identity.PublicKey)
	signature := append([]byte{0}, array64Bytes...)

	request := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{accTestData},
		AirdropAddress:    accTestData,
		Signatures:        []types.HexBytes{signature},
	}

	joinOnRequestCommunity(&s.Suite, community, controlNode, user, request)
}

func (s *MessengerCommunitiesSignersSuite) makeAddressSatisfyTheCriteria(chainID uint64, address string, criteria *protobuf.TokenCriteria) {
	makeAddressSatisfyTheCriteria(&s.Suite, s.mockedBalances, s.mockedCollectibles, chainID, address, criteria)
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

	s.advertiseCommunityTo(s.john, community, s.bob)
	s.advertiseCommunityTo(s.john, community, s.alice)

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
		&communities.CollectibleContractData{TotalSupply: &bigint.BigInt{}})

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
			return len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&s.bob.identity.PublicKey) &&
				!r.Communities()[0].Joined() && r.Communities()[0].Spectated() &&
				len(r.ActivityCenterNotifications()) == 0
		},
		"Bob was not kicked from the community",
	)
	s.Require().NoError(err)

	// check that John received kick event, and AC notification msg created
	// John, as ex-owner must manually join the community
	_, err = WaitOnSignaledMessengerResponse(
		s.john,
		func(r *MessengerResponse) bool {
			inSpectateMode := len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&s.john.identity.PublicKey) &&
				!r.Communities()[0].Joined() && r.Communities()[0].Spectated()
			sharedNotificationExist := false
			for _, acNotification := range r.ActivityCenterNotifications() {
				if acNotification.Type == ActivityCenterNotificationTypeShareAccounts {
					sharedNotificationExist = true
					break
				}
			}
			return inSpectateMode && sharedNotificationExist
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
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&s.bob.identity.PublicKey) &&
				r.Communities()[0].Joined() && !r.Communities()[0].Spectated()
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

	s.advertiseCommunityTo(s.john, community, s.bob)
	s.advertiseCommunityTo(s.john, community, s.alice)

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
		&communities.CollectibleContractData{TotalSupply: &bigint.BigInt{}})

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

	s.advertiseCommunityTo(s.john, community, s.alice)

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
		&communities.CollectibleContractData{TotalSupply: &bigint.BigInt{}})

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
	chat := CreateOneToOneChat(common.PubkeyToHex(&s.bob.identity.PublicKey), &s.bob.identity.PublicKey, s.bob.transport)

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err = s.alice.SaveChat(chat)
	s.Require().NoError(err)
	_, err = s.alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	_, err = WaitOnSignaledMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0
		},
		"Community was not advertised to Bob",
	)
	s.Require().NoError(err)

	// Bob joins the community
	s.joinCommunity(s.alice, community, s.bob)

}

func (s *MessengerCommunitiesSignersSuite) testDescriptionSignature(description []byte) {
	var amm protobuf.ApplicationMetadataMessage
	err := proto.Unmarshal(description, &amm)
	s.Require().NoError(err)

	signer, err := utils.RecoverKey(&amm)
	s.Require().NoError(err)
	s.NotNil(signer)
}

func (s *MessengerCommunitiesSignersSuite) forceCommunityChange(community *communities.Community, owner *Messenger, user *Messenger) {
	newDescription := community.DescriptionText() + " new"
	_, err := owner.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
			Name:        community.Name(),
			Color:       community.Color(),
			Description: newDescription,
		},
	})
	s.Require().NoError(err)

	// alice receives new description
	_, err = WaitOnMessengerResponse(user, func(r *MessengerResponse) bool {
		return len(r.Communities()) > 0 && r.Communities()[0].DescriptionText() == newDescription
	}, "new description not received")
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSignersSuite) testSyncCommunity(mintOwnerToken bool) {
	community := s.createCommunity(s.john)
	s.advertiseCommunityTo(s.john, community, s.alice)
	s.joinCommunity(s.john, community, s.alice)

	// FIXME: Remove this workaround when fixed:
	// https://github.com/status-im/status-go/issues/4413
	s.forceCommunityChange(community, s.john, s.alice)

	aliceCommunity, err := s.alice.GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.testDescriptionSignature(aliceCommunity.DescriptionProtocolMessage())

	if mintOwnerToken {
		// john mints owner token
		var chainID uint64 = 1
		tokenAddress := "token-address"
		tokenName := "tokenName"
		tokenSymbol := "TSM"
		communityToken := &token.CommunityToken{
			TokenType:       protobuf.CommunityTokenType_ERC721,
			CommunityID:     community.IDString(),
			Address:         tokenAddress,
			ChainID:         int(chainID),
			Name:            tokenName,
			Supply:          &bigint.BigInt{},
			Symbol:          tokenSymbol,
			PrivilegesLevel: token.OwnerLevel,
		}

		_, err := s.john.SaveCommunityToken(communityToken, nil)
		s.Require().NoError(err)

		// john adds minted owner token to community
		err = s.john.AddCommunityToken(community.IDString(), int(chainID), tokenAddress)
		s.Require().NoError(err)

		// update mock - the signer for the community returned by the contracts should be john
		s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.john.identity.PublicKey))
		s.collectiblesServiceMock.SetMockCommunityTokenData(communityToken)

		// alice accepts community update
		_, err = WaitOnSignaledMessengerResponse(
			s.alice,
			func(r *MessengerResponse) bool {
				return len(r.Communities()) > 0 && len(r.Communities()[0].TokenPermissions()) == 1
			},
			"no communities",
		)
		s.Require().NoError(err)
	}

	// Create alice second instance
	alice2, err := newMessengerWithKey(
		s.shh,
		s.alice.identity,
		s.logger.With(zap.String("name", "alice-2")),
		[]Option{WithCommunityTokensService(s.collectiblesServiceMock)})

	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, alice2)

	// Create communities backup

	clock, _ := s.alice.getLastClockWithRelatedChat()
	communitiesBackup, err := s.alice.backupCommunities(context.Background(), clock)
	s.Require().NoError(err)

	// Find wanted communities in the backup

	var syncCommunityMessages []*protobuf.SyncInstallationCommunity

	for _, b := range communitiesBackup {
		for _, c := range b.Communities {
			if bytes.Equal(c.Id, community.ID()) {
				syncCommunityMessages = append(syncCommunityMessages, c)
			}
		}
	}
	s.Require().Len(syncCommunityMessages, 1)

	s.testDescriptionSignature(syncCommunityMessages[0].Description)

	// Push the backup into second instance

	messageState := alice2.buildMessageState()
	err = alice2.HandleSyncInstallationCommunity(messageState, syncCommunityMessages[0], nil)

	s.Require().NoError(err)
	s.Require().Len(messageState.Response.Communities(), 1)

	expectedControlNode := community.PublicKey()
	if mintOwnerToken {
		expectedControlNode = &s.john.identity.PublicKey
	}

	responseCommunity := messageState.Response.Communities()[0]
	s.Require().Equal(community.IDString(), responseCommunity.IDString())
	s.Require().True(common.IsPubKeyEqual(expectedControlNode, responseCommunity.ControlNode()))
}

func (s *MessengerCommunitiesSignersSuite) TestSyncTokenGatedCommunity() {
	testCases := []struct {
		name           string
		mintOwnerToken bool
	}{
		{
			name:           "general community sync",
			mintOwnerToken: false,
		},
		{
			name:           "community with token ownership",
			mintOwnerToken: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.testSyncCommunity(tc.mintOwnerToken)
		})
	}
}

func (s *MessengerCommunitiesSignersSuite) TestWithMintedOwnerTokenApplyCommunityEventsUponMakingDeviceControlNode() {
	community := s.createCommunity(s.john)

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

	// Make sure there is no control node
	s.Require().False(common.IsPubKeyEqual(community.ControlNode(), &s.john.identity.PublicKey))

	// Trick. We need to remove the community private key otherwise the events
	// will be signed and Events will be approved instead of being in Pending State.
	_, err = s.john.RemovePrivateKey(community.ID())
	s.Require().NoError(err)

	request := requests.CreateCommunityTokenPermission{
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

	response, err := s.john.CreateCommunityTokenPermission(&request)
	s.Require().NoError(err)
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].TokenPermissionsAdded, 1)

	addedPermission := func() *communities.CommunityTokenPermission {
		for _, permission := range response.CommunityChanges[0].TokenPermissionsAdded {
			return permission
		}
		return nil
	}()
	s.Require().NotNil(addedPermission)
	s.Require().Equal(communities.TokenPermissionAdditionPending, addedPermission.State)

	messengerReponse, err := s.john.PromoteSelfToControlNode(community.ID())

	s.Require().NoError(err)
	s.Require().Len(messengerReponse.Communities(), 1)

	tokenPermissions := messengerReponse.Communities()[0].TokenPermissions()
	s.Require().Len(tokenPermissions, 2)

	tokenPermissionsMap := make(map[protobuf.CommunityTokenPermission_Type]struct{}, len(tokenPermissions))
	for _, t := range tokenPermissions {
		tokenPermissionsMap[t.Type] = struct{}{}
	}

	s.Require().Len(tokenPermissionsMap, 2)
	s.Require().Contains(tokenPermissionsMap, protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER)
	s.Require().Contains(tokenPermissionsMap, protobuf.CommunityTokenPermission_BECOME_ADMIN)

	for _, v := range tokenPermissions {
		s.Require().Equal(communities.TokenPermissionApproved, v.State)
	}
}

func (s *MessengerCommunitiesSignersSuite) TestWithoutMintedOwnerTokenMakingDeviceControlNodeIsBlocked() {
	community := s.createCommunity(s.john)

	// Make sure there is no control node
	s.Require().False(common.IsPubKeyEqual(community.ControlNode(), &s.john.identity.PublicKey))

	response, err := s.john.PromoteSelfToControlNode(community.ID())
	s.Require().Nil(response)
	s.Require().NotNil(err)
	s.Require().Error(err, "Owner token is needed")
}

func (s *MessengerCommunitiesSignersSuite) TestControlNodeDeviceChanged() {
	// Note: we don't have any specific check if control node device changed,
	// so in this test we will just call twice 'PromoteSelfToControlNode'
	community, _ := createOnRequestCommunity(&s.Suite, s.john)

	// john mints owner token
	ownerTokenAddress := "token-address"
	_, err := s.john.SaveCommunityToken(&token.CommunityToken{
		TokenType:       protobuf.CommunityTokenType_ERC721,
		CommunityID:     community.IDString(),
		Address:         ownerTokenAddress,
		ChainID:         int(testChainID1),
		Name:            "ownerToken",
		Supply:          &bigint.BigInt{},
		Symbol:          "OT",
		PrivilegesLevel: token.OwnerLevel,
	}, nil)
	s.Require().NoError(err)

	err = s.john.AddCommunityToken(community.IDString(), int(testChainID1), ownerTokenAddress)
	s.Require().NoError(err)

	// john mints TM token
	tokenMasterTokenAddress := "token-master-address"
	_, err = s.john.SaveCommunityToken(&token.CommunityToken{
		TokenType:       protobuf.CommunityTokenType_ERC721,
		CommunityID:     community.IDString(),
		Address:         tokenMasterTokenAddress,
		ChainID:         int(testChainID1),
		Name:            "tokenMasterToken",
		Supply:          &bigint.BigInt{},
		Symbol:          "TMT",
		PrivilegesLevel: token.MasterLevel,
	}, nil)
	s.Require().NoError(err)

	err = s.john.AddCommunityToken(community.IDString(), int(testChainID1), tokenMasterTokenAddress)
	s.Require().NoError(err)

	// set john as contract owner
	s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.john.identity.PublicKey))
	s.collectiblesServiceMock.SetMockCollectibleContractData(testChainID1, ownerTokenAddress,
		&communities.CollectibleContractData{TotalSupply: &bigint.BigInt{}})
	s.collectiblesServiceMock.SetMockCollectibleContractData(testChainID1, tokenMasterTokenAddress,
		&communities.CollectibleContractData{TotalSupply: &bigint.BigInt{}})

	community, err = s.john.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(common.IsPubKeyEqual(community.ControlNode(), &s.john.identity.PublicKey))

	var tokenMasterTokenCriteria *protobuf.TokenCriteria
	for _, permission := range community.TokenPermissions() {
		if permission.Type == protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER {
			s.Require().Len(permission.TokenCriteria, 1)
			tokenMasterTokenCriteria = permission.TokenCriteria[0]
			break
		}
	}
	s.Require().NotNil(tokenMasterTokenCriteria)

	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenMasterTokenCriteria)

	waitOnAliceCommunityValidation := waitOnCommunitiesEvent(s.alice, func(sub *communities.Subscription) bool {
		return sub.TokenCommunityValidated != nil
	})
	s.advertiseCommunityTo(s.john, community, s.alice)
	err = <-waitOnAliceCommunityValidation
	s.Require().NoError(err)

	waitOnBobCommunityValidation := waitOnCommunitiesEvent(s.bob, func(sub *communities.Subscription) bool {
		return sub.TokenCommunityValidated != nil
	})
	s.advertiseCommunityTo(s.john, community, s.bob)
	err = <-waitOnBobCommunityValidation
	s.Require().NoError(err)

	s.joinOnRequestCommunity(s.john, community, s.alice)
	s.joinOnRequestCommunity(s.john, community, s.bob)

	community, err = s.john.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(checkRoleBasedOnThePermissionType(protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, &s.bob.identity.PublicKey, community))

	// Simulate control node device changed and new control node device has all members revealed addresses
	response, err := s.john.PromoteSelfToControlNode(community.ID())
	s.Require().NoError(err)
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].MembersRemoved, 0)

	// Simulate control node device changed and new control node device does not have bob's revealed addresses
	bobRequestID := communities.CalculateRequestID(s.bob.IdentityPublicKeyString(), community.ID())
	err = s.john.communitiesManager.RemoveRequestToJoinRevealedAddresses(bobRequestID)
	s.Require().NoError(err)

	// due to test execution is fast, we update request to join clock
	clock := uint64(time.Now().Unix() - 2)
	err = s.john.communitiesManager.UpdateClockInRequestToJoin(bobRequestID, clock)
	s.Require().NoError(err)

	_, err = s.john.PromoteSelfToControlNode(community.ID())

	s.Require().NoError(err)
	community, err = s.john.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.Members(), 2)
	for _, chat := range community.Chats() {
		s.Require().Len(chat.Members, 2)
	}

	// Bob will receive request to share RevealedAddresses and send request to join to the control node
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) == 1 && !r.Communities()[0].HasMember(&s.bob.identity.PublicKey) &&
				r.Communities()[0].Spectated() && len(r.ActivityCenterNotifications()) == 0
		},
		"Bob was not soft kicked from the community",
	)
	s.Require().NoError(err)

	// check that alice was not soft kicked
	_, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&s.alice.identity.PublicKey) &&
				!r.Communities()[0].Spectated() && len(r.ActivityCenterNotifications()) == 0 &&
				r.Communities()[0].Joined()
		},
		"Alice was kicked from the community",
	)
	s.Require().NoError(err)

	// John auto-accept requests to join with RevealedAddresses
	_, err = WaitOnMessengerResponse(
		s.john,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && len(r.Communities()[0].Members()) == 3
		},
		"no community update with accepted request",
	)
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&s.bob.identity.PublicKey) &&
				r.Communities()[0].Joined() && !r.Communities()[0].Spectated() && r.Communities()[0].IsTokenMaster()
		},
		"Bob was auto-accepted",
	)
	s.Require().NoError(err)
}
