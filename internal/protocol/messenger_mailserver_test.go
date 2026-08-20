package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/pkg/messaging"
	"github.com/status-im/status-go/pkg/messaging/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

func TestMessengerFetchLatestCommunityDescriptionsSuite(t *testing.T) {
	suite.Run(t, new(MessengerFetchLatestCommunityDescriptionsSuite))
}

type MessengerFetchLatestCommunityDescriptionsSuite struct {
	suite.Suite

	messagingEnv *messaging.TestMessagingEnvironment
	m            *Messenger
}

func (s *MessengerFetchLatestCommunityDescriptionsSuite) createLocalCommunity() *communities.Community {
	response, err := s.m.CreateCommunity(&requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}, true)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	return response.Communities()[0]
}

func (s *MessengerFetchLatestCommunityDescriptionsSuite) SetupTest() {
	var err error
	s.messagingEnv, err = messaging.NewTestMessagingEnvironment()
	s.Require().NoError(err)
	s.Require().NoError(s.messagingEnv.Setup(s.T()))

	// Use a non-running messenger so no background loop issues store queries and
	// interferes with the intercepted batches.
	s.m, err = newTestMessenger(s.T(), s.messagingEnv, testMessengerConfig{})
	s.Require().NoError(err)
}

type recordedMailserverBatch struct {
	batch             wakutypes.MailserverBatch
	pageLimit         uint64
	shouldProcessNext func(int) (bool, uint64)
	processEnvelopes  bool
}

func (s *MessengerFetchLatestCommunityDescriptionsSuite) recordMailserverBatches() *[]recordedMailserverBatch {
	recorded := &[]recordedMailserverBatch{}
	s.messagingEnv.SetProcessMailserverBatchHook(func(ctx context.Context, batch wakutypes.MailserverBatch, pageLimit uint64, shouldProcessNextPage func(int) (bool, uint64), processEnvelopes bool) error {
		*recorded = append(*recorded, recordedMailserverBatch{
			batch:             batch,
			pageLimit:         pageLimit,
			shouldProcessNext: shouldProcessNextPage,
			processEnvelopes:  processEnvelopes,
		})
		return nil
	})
	s.T().Cleanup(func() {
		s.messagingEnv.SetProcessMailserverBatchHook(nil)
	})
	return recorded
}

func descriptionFilter(chatID, pubsubTopic, contentTopic string) *types.ChatFilter {
	return types.NewChatFilter(&types.ChatFilterConfig{
		ChatID:       chatID,
		PubsubTopic:  pubsubTopic,
		ContentTopic: types.StringToContentTopic(contentTopic),
		Listen:       true,
	})
}

// TestFetchesOnlyNewestPagePerFilter verifies that fetchLatestCommunityDescriptions
// issues exactly one store request per filter that fetches only the newest page
// (pageLimit == 1) and stops immediately, instead of sweeping the whole window.
func (s *MessengerFetchLatestCommunityDescriptionsSuite) TestFetchesOnlyNewestPagePerFilter() {
	recorded := s.recordMailserverBatches()

	filters := []*types.ChatFilter{
		descriptionFilter("0xcommunity1", "/waku/2/pubsub-a", "0xcontenttopic1"),
		descriptionFilter("0xcommunity2", "/waku/2/pubsub-b", "0xcontenttopic2"),
	}

	s.m.fetchLatestCommunityDescriptions(filters)

	s.Require().Len(*recorded, len(filters), "expected exactly one store request per filter")

	for i, rec := range *recorded {
		// Only the newest page must be fetched.
		s.Require().Equal(uint64(1), rec.pageLimit, "page limit must be 1")
		s.Require().False(rec.processEnvelopes)

		// The paging callback must stop immediately, regardless of how many
		// envelopes the first page returned (i.e. never fetch a second page).
		s.Require().NotNil(rec.shouldProcessNext)
		cont, next := rec.shouldProcessNext(1000)
		s.Require().False(cont, "must not fetch a second page")
		s.Require().Equal(uint64(0), next)

		// Each batch must target exactly the filter's description topic.
		s.Require().Len(rec.batch.Topics, 1)
		s.Require().Equal(filters[i].PubsubTopic(), rec.batch.PubsubTopic)
		s.Require().Equal([]string{filters[i].ChatID()}, rec.batch.ChatIDs)
	}
}

// TestNoFiltersIsNoop verifies that no store request is made when there are no
// community description filters to fetch.
func (s *MessengerFetchLatestCommunityDescriptionsSuite) TestNoFiltersIsNoop() {
	recorded := s.recordMailserverBatches()

	s.m.fetchLatestCommunityDescriptions(nil)

	s.Require().Empty(*recorded, "no store request should be made when there are no filters")
}

func (s *MessengerFetchLatestCommunityDescriptionsSuite) TestLatestDescriptionUsesRollingMonth() {
	recorded := s.recordMailserverBatches()
	filter := descriptionFilter("0xcommunity1", "/waku/2/pubsub-a", "0xcontenttopic1")
	from, to := s.m.calculateMailserverTimeBounds(oneMonthDuration)

	s.m.fetchLatestCommunityDescriptions([]*types.ChatFilter{filter})

	s.Require().Len(*recorded, 1)
	s.Require().Equal(from, (*recorded)[0].batch.From)
	s.Require().Equal(to, (*recorded)[0].batch.To)
}

// TestReusedFilterForgottenForNonMember verifies that a description filter
// reused by a store-node request is forgotten for a community that is neither
// joined nor spectated (so it does not leave a live subscription behind), and
// kept for a joined/spectated community and for contact requests.
func (s *MessengerFetchLatestCommunityDescriptionsSuite) TestReusedFilterForgottenForNonMember() {
	// A validly-shaped community id (compressed pubkey) that is not in the DB.
	unknownID := "0x02b5bdaf5a25fcfe2ee14c501fab1836b8de57f61621080c3d52073d16de0d98d6"
	s.Require().True(s.m.storeNodeRequestsManager.reusedFilterShouldForget(storeNodeCommunityRequest, unknownID),
		"a community that is neither joined nor spectated must not keep a reused filter")

	joined := s.createLocalCommunity()
	s.Require().False(s.m.storeNodeRequestsManager.reusedFilterShouldForget(storeNodeCommunityRequest, joined.IDString()),
		"a joined/spectated community keeps its live description subscription")

	s.Require().False(s.m.storeNodeRequestsManager.reusedFilterShouldForget(storeNodeContactRequest, unknownID),
		"contact requests are unaffected")
}
