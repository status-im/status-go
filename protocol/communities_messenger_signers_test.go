package protocol

import (
	"crypto/ecdsa"
	"testing"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
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
}

func (s *MessengerCommunitiesSignersSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

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
	messenger, err := newCommunitiesTestMessenger(s.shh, privateKey, s.logger, nil, nil)
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

func (s *MessengerCommunitiesSignersSuite) TestControlNodeUpdate() {
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

	// make bob the control node
	bobPubKey := common.PubkeyToHexBytes(&s.bob.identity.PublicKey)
	_, err = s.bob.communitiesManager.UpdateControlNode(community.ID(), bobPubKey)
	s.Require().NoError(err)
	_, err = s.bob.communitiesManager.UpdatePrivateKey(community.ID(), s.bob.identity)
	s.Require().NoError(err)

	// make alice aware of control node change
	_, err = s.alice.communitiesManager.UpdateControlNode(community.ID(), bobPubKey)
	s.Require().NoError(err)

	// john can still publish community update, he doesn't know about control node change yet
	anotherJohnDescr := "another john's description"
	response, err = s.john.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Name:        community.Name(),
			Description: anotherJohnDescr,
			Color:       community.Color(),
			Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
	})
	s.Require().NoError(err)
	s.Require().Equal(anotherJohnDescr, response.Communities()[0].Description().Identity.Description)

	// bob rejects community update as it is not signed by control node
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].Description().Identity.Description == anotherJohnDescr
		},
		"no communities",
	)
	s.Require().ErrorContains(err, "no communities")

	// alice rejects community update as it is not signed by control node
	_, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].Description().Identity.Description == anotherJohnDescr
		},
		"no communities",
	)
	s.Require().ErrorContains(err, "no communities")

	// make john aware of control node change
	_, err = s.john.communitiesManager.UpdateControlNode(community.ID(), bobPubKey)
	s.Require().NoError(err)

	// john can't publish community update anymore
	_, err = s.john.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Name:        community.Name(),
			Description: anotherJohnDescr,
			Color:       community.Color(),
			Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
	})
	s.Require().Error(err, communities.ErrNotEnoughPermissions)

	// FIXME: make community clock timestamp hinted lamport clock
	_, err = s.bob.communitiesManager.IncreaseClock(community.ID())
	s.Require().NoError(err)

	// bob as control node publishes community update
	bobDescr := "bob's description"
	response, err = s.bob.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Name:        community.Name(),
			Description: bobDescr,
			Color:       community.Color(),
			Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
	})
	s.Require().NoError(err)
	s.Require().Equal(bobDescr, response.Communities()[0].Description().Identity.Description)

	// john accepts community update
	_, err = WaitOnMessengerResponse(
		s.john,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].Description().Identity.Description == bobDescr
		},
		"no communities",
	)
	s.Require().NoError(err)

	// alice accepts community update
	_, err = WaitOnMessengerResponse(
		s.alice,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].Description().Identity.Description == bobDescr
		},
		"no communities",
	)
	s.Require().NoError(err)
}
