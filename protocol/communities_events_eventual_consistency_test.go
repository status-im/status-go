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
	AdminCommunityEventsSuite

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
	s.AdminCommunityEventsSuite.TearDownTest()
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

// TODO: remove once eventual consistency is implemented
var communityRequestsEventualConsistencyFixed = false

func (s *CommunityEventsEventualConsistencySuite) TestAdminAcceptRejectRequestToJoin() {
	community := setUpOnRequestCommunityAndRoles(s, protobuf.CommunityMember_ROLE_ADMIN, []*Messenger{})

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

	// accept request to join
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = s.admin.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)

	// then reject request to join
	rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = s.admin.DeclineRequestToJoinCommunity(rejectRequestToJoin)
	s.Require().NoError(err)

	// ensure both messages are pushed to waku
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
	s.messagesOrderController.order = messagesOrderAsPosted

	waitForAcceptedRequestToJoin := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return len(sub.AcceptedRequestsToJoin) == 1
	})

	waitOnAdminEventsRejection := waitOnCommunitiesEvent(s.owner, func(s *communities.Subscription) bool {
		return s.CommunityEventsMessageInvalidClock != nil
	})

	_, err = s.owner.RetrieveAll()
	s.Require().NoError(err)

	// first owner handles AcceptRequestToJoinCommunity event
	err = <-waitForAcceptedRequestToJoin
	s.Require().NoError(err)

	// then owner rejects DeclineRequestToJoinCommunity event due to invalid clock
	err = <-waitOnAdminEventsRejection
	s.Require().NoError(err)

	if communityRequestsEventualConsistencyFixed {
		// admin receives rejected DeclineRequestToJoinCommunity event and re-applies it,
		// there is no signal whatsoever, we just wait for admin to process all incoming messages
		_, _ = WaitOnMessengerResponse(s.admin, func(response *MessengerResponse) bool {
			return false
		}, "")

		waitForRejectedRequestToJoin := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
			return len(sub.RejectedRequestsToJoin) == 1
		})

		_, err = s.owner.RetrieveAll()
		s.Require().NoError(err)

		// owner handles DeclineRequestToJoinCommunity event eventually
		err = <-waitForRejectedRequestToJoin
		s.Require().NoError(err)

		// user should be removed from community
		community, err = s.owner.GetCommunityByID(community.ID())
		s.Require().NoError(err)
		s.Require().False(community.HasMember(&user.identity.PublicKey))
	}
}
