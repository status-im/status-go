package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

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
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
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

	s.john = s.newMessenger()
	s.bob = s.newMessenger()
	s.alice = s.newMessenger()
	_, err := s.john.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSignersSuite) TearDownTest() {
	s.Require().NoError(s.john.Shutdown())
	s.Require().NoError(s.bob.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesSignersSuite) newMessengerWithKey(privateKey *ecdsa.PrivateKey) *Messenger {
	messenger, err := newCommunitiesTestMessenger(s.shh, privateKey, s.logger, nil, nil, s.collectiblesServiceMock)
	s.Require().NoError(err)

	return messenger
}

func (s *MessengerCommunitiesSignersSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(privateKey)
}

func (s *MessengerCommunitiesSignersSuite) createCommunity(controlNode *Messenger) *communities.Community {
	community, _ := createCommunity(&s.Suite, controlNode)
	return community
}

func (s *MessengerCommunitiesSignersSuite) advertiseCommunityTo(controlNode *Messenger, community *communities.Community, user *Messenger) {
	advertiseCommunityTo(&s.Suite, community, controlNode, user)
}

func (s *MessengerCommunitiesSignersSuite) joinCommunity(controlNode *Messenger, community *communities.Community, user *Messenger) {
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	joinCommunity(&s.Suite, community, controlNode, user, request)
}

// John crates a community
// Ownership is transferred to Alice
// Both John and Bob accepts the changes

func (s *MessengerCommunitiesSignersSuite) TestControlNodeUpdateSigner() {

	// Create a community
	// Transfer ownership
	// Process message
	community := s.createCommunity(s.john)

	s.advertiseCommunityTo(s.john, community, s.bob)
	s.advertiseCommunityTo(s.john, community, s.alice)

	s.joinCommunity(s.john, community, s.bob)
	s.joinCommunity(s.john, community, s.alice)

	// john as control node publishes community update
	johnDescr := "john's description"
	response, err := s.john.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Name:        community.Name(),
			Description: johnDescr,
			Color:       community.Color(),
			Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
	})
	s.Require().NoError(err)
	s.Require().Equal(johnDescr, response.Communities()[0].Description().Identity.Description)

	// bob accepts community update
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].Description().Identity.Description == johnDescr
		},
		"no communities",
	)
	s.Require().NoError(err)

	// alice accepts community update
	_, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].Description().Identity.Description == johnDescr
		},
		"no communities",
	)
	s.Require().NoError(err)

	// Alice will be transferred the ownership token, and alice will let others know

	var chainID uint64 = 1
	communityAddress := "community-address"
	tokenAddress := "token-address"
	tokenName := "tokenName"
	tokenSymbol := "TSM"

	// Update mock
	// The signer for the community returned by the contracts should be alice
	s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.collectiblesServiceMock.SetMockCollectibleContractData(chainID, tokenAddress,
		&communitytokens.CollectibleContractData{TotalSupply: &bigint.BigInt{}})

	community, err = s.alice.communitiesManager.PromoteSelfToControlNode(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.IsControlNode())

	// Create community token
	_, err = s.alice.SaveCommunityToken(&token.CommunityToken{
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

	err = s.alice.AddCommunityToken(community.IDString(), int(chainID), tokenAddress)
	s.Require().NoError(err)

	// make alice the control node
	transaction := transactions.SendTxArgs{}
	_, err = s.alice.SetCommunitySignerPubKey(context.Background(), community.ID(), chainID, communityAddress, transaction, "password", common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().NoError(err)

	var published bool

	// We check the updated description is published as a signal
	signal.SetMobileSignalHandler(func(data []byte) {
		response := &MessengerResponse{}
		err := json.Unmarshal(data, response)
		if err != nil {
			s.john.logger.Error("failed to unmarshal json", zap.Error(err))
			return
		}

		published = strings.Contains(string(data), community.IDString())
	})

	// john accepts community update
	_, err = WaitOnMessengerResponse(
		s.john,
		func(r *MessengerResponse) bool {
			return published
		},
		"no communities",
	)
	s.Require().NoError(err)

	// We check the control node is correctly set on john to alice
	johnCommunity, err := s.john.communitiesManager.GetByIDString(community.IDString())
	s.Require().NoError(err)
	s.Require().NotNil(johnCommunity)
	s.Require().True(common.IsPubKeyEqual(johnCommunity.ControlNode(), &s.alice.identity.PublicKey))
	s.Require().False(johnCommunity.IsControlNode())

	published = false

	// We check the control node is correctly set on bob
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return published

		},
		"no communities",
	)
	s.Require().NoError(err)

	// We check the control node is correctly set on bob to alice
	bobCommunity, err := s.bob.communitiesManager.GetByIDString(community.IDString())
	s.Require().NoError(err)
	s.Require().NotNil(bobCommunity)
	s.Require().True(common.IsPubKeyEqual(bobCommunity.ControlNode(), &s.alice.identity.PublicKey))
	s.Require().False(bobCommunity.IsControlNode())
}
