package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestCommunityEventsEventualConsistencySuite(t *testing.T) {
	suite.Run(t, new(CommunityEventsEventualConsistencySuite))
}

type CommunityEventsEventualConsistencySuite struct {
	AdminCommunityEventsSuiteBase

	messagesOrderController *MessagesOrderController
}

func (s *CommunityEventsEventualConsistencySuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()
	s.collectiblesServiceMock = &CollectiblesServiceMock{}

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	wakuWrapper, err := newTestWakuWrapper(&config, s.logger)
	s.Require().NoError(err)
	s.Require().NoError(shh.Start())
	s.shh = wakuWrapper

	s.messagesOrderController = NewMessagesOrderController(messagesOrderRandom)
	s.messagesOrderController.Start(wakuWrapper.SubscribePostEvents())

	s.owner = s.newMessenger("", []string{})
	s.admin = s.newMessenger(accountPassword, []string{eventsSenderAccountAddress})
	s.alice = s.newMessenger(accountPassword, []string{aliceAccountAddress})
	_, err = s.owner.Start()
	s.Require().NoError(err)
	_, err = s.admin.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.mockedBalances = createMockedWalletBalance(&s.Suite)
}

func (s *CommunityEventsEventualConsistencySuite) TearDownTest() {
	s.AdminCommunityEventsSuiteBase.TearDownTest()
	s.messagesOrderController.Stop()
}

func (s *CommunityEventsEventualConsistencySuite) newMessenger(password string, walletAddresses []string) *Messenger {
	return newTestCommunitiesMessenger(&s.Suite, s.shh, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			logger:                  s.logger,
			messagesOrderController: s.messagesOrderController,
		},
		password:            password,
		walletAddresses:     walletAddresses,
		mockedBalances:      &s.mockedBalances,
		collectiblesService: s.collectiblesServiceMock,
	})
}

type requestToJoinActionType int

const (
	requestToJoinAccept requestToJoinActionType = iota
	requestToJoinReject
)

func (s *CommunityEventsEventualConsistencySuite) testRequestsToJoin(actions []requestToJoinActionType, messagesOrder messagesOrderType) {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{})
	s.Require().True(community.IsControlNode())

	// set up additional user that will send request to join
	user := s.newMessenger("", []string{})
	s.SetupAdditionalMessengers([]*Messenger{user})

	advertiseCommunityToUserOldWay(&s.Suite, community, s.owner, user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID(), ENSName: "testName"}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	sentRequest := response.RequestsToJoinCommunity()[0]

	checkRequestToJoin := func(r *MessengerResponse) bool {
		for _, request := range r.RequestsToJoinCommunity() {
			if request.ENSName == requestToJoin.ENSName {
				return true
			}
		}
		return false
	}

	// admin receives request to join
	response, err = WaitOnMessengerResponse(
		s.admin,
		checkRequestToJoin,
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	for _, action := range actions {
		switch action {
		case requestToJoinAccept:
			acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
			_, err = s.admin.AcceptRequestToJoinCommunity(acceptRequestToJoin)
			s.Require().NoError(err)

		case requestToJoinReject:
			rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
			_, err = s.admin.DeclineRequestToJoinCommunity(rejectRequestToJoin)
			s.Require().NoError(err)
		}
	}

	// ensure all messages are pushed to waku
	/*
		FIXME: we should do it smarter, as follows:
		```
		hashes1, err := admin.SendEvent()
		hashes2, err := admin.SendEvent()
		WaitForHashes([][]byte{hashes1, hashes2}, admin.waku)
		s.messagesOrderController.setCustomOrder([][]{hashes1, hashes2})
		```
	*/
	time.Sleep(1 * time.Second)

	// ensure events are received in order
	s.messagesOrderController.order = messagesOrder

	response, err = s.owner.RetrieveAll()
	s.Require().NoError(err)

	lastAction := actions[len(actions)-1]
	responseChecker := func(mr *MessengerResponse) bool {
		if len(mr.RequestsToJoinCommunity()) == 0 || len(mr.Communities()) == 0 {
			return false
		}
		switch lastAction {
		case requestToJoinAccept:
			return mr.RequestsToJoinCommunity()[0].State == communities.RequestToJoinStateAccepted &&
				mr.Communities()[0].HasMember(&user.identity.PublicKey)
		case requestToJoinReject:
			return mr.RequestsToJoinCommunity()[0].State == communities.RequestToJoinStateDeclined &&
				!mr.Communities()[0].HasMember(&user.identity.PublicKey)
		}
		return false
	}

	switch messagesOrder {
	case messagesOrderAsPosted:
		_, err = WaitOnSignaledMessengerResponse(s.owner, responseChecker, "lack of eventual consistency")
		s.Require().NoError(err)
	case messagesOrderReversed:
		s.Require().True(responseChecker(response))
	}
}

func (s *CommunityEventsEventualConsistencySuite) TestAdminAcceptRejectRequestToJoin_InOrder() {
	s.testRequestsToJoin([]requestToJoinActionType{requestToJoinAccept, requestToJoinReject}, messagesOrderAsPosted)
}

func (s *CommunityEventsEventualConsistencySuite) TestAdminAcceptRejectRequestToJoin_OutOfOrder() {
	s.testRequestsToJoin([]requestToJoinActionType{requestToJoinAccept, requestToJoinReject}, messagesOrderReversed)
}

func (s *CommunityEventsEventualConsistencySuite) TestAdminRejectAcceptRequestToJoin_InOrder() {
	s.testRequestsToJoin([]requestToJoinActionType{requestToJoinReject, requestToJoinAccept}, messagesOrderAsPosted)
}

func (s *CommunityEventsEventualConsistencySuite) TestAdminRejectAcceptRequestToJoin_OutOfOrder() {
	s.testRequestsToJoin([]requestToJoinActionType{requestToJoinReject, requestToJoinAccept}, messagesOrderReversed)
}
