package protocol

import (
	"context"
	"strings"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/pkg/messaging"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	mailserversDB "github.com/status-im/status-go/services/mailservers"
)

func TestMessengerFetchLatestCommunityDescriptionsSuite(t *testing.T) {
	suite.Run(t, new(MessengerFetchLatestCommunityDescriptionsSuite))
}

type MessengerFetchLatestCommunityDescriptionsSuite struct {
	suite.Suite

	messagingEnv *messaging.TestMessagingEnvironment
	m            *Messenger
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
	s.messagingEnv.SetProcessMailserverBatchHook(func(ctx context.Context, batch wakutypes.MailserverBatch, storenode peer.AddrInfo, pageLimit uint64, shouldProcessNextPage func(int) (bool, uint64), processEnvelopes bool) error {
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

func descriptionFilter(chatID, pubsubTopic, contentTopic string) *types2.ChatFilter {
	return types2.NewChatFilter(&types2.ChatFilterConfig{
		ChatID:       chatID,
		PubsubTopic:  pubsubTopic,
		ContentTopic: types2.StringToContentTopic(contentTopic),
		Listen:       true,
	})
}

// TestFetchesOnlyNewestPagePerFilter verifies that fetchLatestCommunityDescriptions
// issues exactly one store request per filter that fetches only the newest page
// (pageLimit == 1) and stops immediately, instead of sweeping the whole window.
func (s *MessengerFetchLatestCommunityDescriptionsSuite) TestFetchesOnlyNewestPagePerFilter() {
	recorded := s.recordMailserverBatches()

	filters := []*types2.ChatFilter{
		descriptionFilter("0xcommunity1", "/waku/2/pubsub-a", "0xcontenttopic1"),
		descriptionFilter("0xcommunity2", "/waku/2/pubsub-b", "0xcontenttopic2"),
	}

	s.m.fetchLatestCommunityDescriptions(peer.AddrInfo{}, filters)

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

	s.m.fetchLatestCommunityDescriptions(peer.AddrInfo{}, nil)

	s.Require().Empty(*recorded, "no store request should be made when there are no filters")
}

// TestSweepExcludesDescriptionShapedTopics verifies that the historic sweep
// diverts any description-shaped topic (a community ID, even for a community
// that is not joined or spectated — e.g. a temporary filter installed by the
// store node request manager) to the dedicated newest-page fetch instead of
// sweeping it over the whole sync window.
func (s *MessengerFetchLatestCommunityDescriptionsSuite) TestSweepExcludesDescriptionShapedTopics() {
	s.m.mailserversDatabase = mailserversDB.NewDB(s.m.database)
	recorded := s.recordMailserverBatches()

	communityDescriptionChatID := "0x02" + strings.Repeat("a", 64)
	publicChatID := "a-public-chat"

	filters := []*types2.ChatFilter{
		descriptionFilter(communityDescriptionChatID, "/waku/2/default", "0xdesc"),
		descriptionFilter(publicChatID, "/waku/2/default", "0xpublic"),
	}

	_, err := s.m.syncFiltersFrom(peer.AddrInfo{}, filters, 0)
	s.Require().NoError(err)

	var sweptChatIDs, newestFetchChatIDs []string
	for _, rec := range *recorded {
		if rec.pageLimit == 1 {
			newestFetchChatIDs = append(newestFetchChatIDs, rec.batch.ChatIDs...)
		} else {
			sweptChatIDs = append(sweptChatIDs, rec.batch.ChatIDs...)
		}
	}

	s.Require().NotContains(sweptChatIDs, communityDescriptionChatID,
		"description-shaped topic of a non-joined community must not be swept over the whole window")
	s.Require().Contains(newestFetchChatIDs, communityDescriptionChatID,
		"description-shaped topic must still get the dedicated newest-page fetch")
	s.Require().Contains(sweptChatIDs, publicChatID,
		"non-description topics must still be swept")
}

func TestIsCommunityDescriptionChatID(t *testing.T) {
	compressed02 := "0x02" + strings.Repeat("a", 64)
	compressed03 := "0x03" + strings.Repeat("f", 64)
	uncompressed := "0x04" + strings.Repeat("a", 128)

	cases := []struct {
		name   string
		chatID string
		want   bool
	}{
		{"compressed 0x02 community id", compressed02, true},
		{"compressed 0x03 community id", compressed03, true},
		{"member update channel", compressed02 + "-memberUpdate", false},
		{"ping channel", compressed02 + "-ping", false},
		{"uncompressed 1-1 chat id", uncompressed, false},
		{"contact code topic", uncompressed + "-contact-code", false},
		{"uppercase hex", "0x02" + strings.Repeat("A", 64), false},
		{"non-hex 68 chars", "0x02" + strings.Repeat("z", 64), false},
		{"public chat name", "a-public-chat", false},
		{"empty", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isCommunityDescriptionChatID(tc.chatID))
		})
	}
}

func TestMessengerBoundedResolveSuite(t *testing.T) {
	suite.Run(t, new(MessengerBoundedResolveSuite))
}

type MessengerBoundedResolveSuite struct {
	suite.Suite

	messagingEnv *messaging.TestMessagingEnvironment
	m            *Messenger
}

func (s *MessengerBoundedResolveSuite) SetupTest() {
	var err error
	s.messagingEnv, err = messaging.NewTestMessagingEnvironment()
	s.Require().NoError(err)
	s.Require().NoError(s.messagingEnv.Setup(s.T()))

	s.m, err = newTestMessenger(s.T(), s.messagingEnv, testMessengerConfig{})
	s.Require().NoError(err)
}

// a syntactically valid community id that is not present in the local database.
const unknownCommunityID = "0x02" + "0000000000000000000000000000000000000000000000000000000000000000"

func (s *MessengerBoundedResolveSuite) newCommunityRequest(cfg StoreNodeRequestConfig, communityID string) *storeNodeRequest {
	r := s.m.storeNodeRequestsManager.newStoreNodeRequest(context.Background())
	r.requestID = storeNodeRequestID{RequestType: storeNodeCommunityRequest, DataID: communityID}
	r.config = cfg
	return r
}

func (s *MessengerBoundedResolveSuite) createLocalCommunity() *communities.Community {
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

// TestReusedFilterForgottenForNonMember verifies that a description filter
// reused by a store-node request is forgotten for a community that is neither
// joined nor spectated (so it does not leave a live subscription behind), and
// kept for a joined/spectated community and for contact requests.
func (s *MessengerBoundedResolveSuite) TestReusedFilterForgottenForNonMember() {
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

// TestStopsAtPageCapUnresolved verifies that a request whose description never
// resolves (e.g. a description whose segments span past the page cap) stops at
// the configured MaxPages and ends unresolved instead of sweeping the window.
func (s *MessengerBoundedResolveSuite) TestStopsAtPageCapUnresolved() {
	cfg := defaultStoreNodeRequestConfig(storeNodeCommunityRequest)
	s.Require().Equal(uint64(2), cfg.MaxPages)

	r := s.newCommunityRequest(cfg, unknownCommunityID)

	cont, next := r.shouldFetchNextPage(0)
	s.Require().True(cont, "first page must allow a second page")
	s.Require().Equal(cfg.FurtherPageSize, next)

	cont, _ = r.shouldFetchNextPage(0)
	s.Require().False(cont, "must stop once the page cap is reached")
	s.Require().Equal(storeNodeRequestUnresolved, r.result.outcome)
	s.Require().Nil(r.result.community)
}

// TestResolvedStopsImmediately verifies that once a newer description is
// persisted the request stops and reports the resolved outcome.
func (s *MessengerBoundedResolveSuite) TestResolvedStopsImmediately() {
	community := s.createLocalCommunity()

	r := s.newCommunityRequest(defaultStoreNodeRequestConfig(storeNodeCommunityRequest), community.IDString())

	cont, _ := r.shouldFetchNextPage(0)
	s.Require().False(cont, "must stop once the community is resolved")
	s.Require().Equal(storeNodeRequestResolved, r.result.outcome)
	s.Require().NotNil(r.result.community)
}

// TestAlreadyUpToDateIsSuccess verifies that when the newest fetched clock is
// not newer than the locally stored one the request reports already-up-to-date
// (a success) and never sets a resolved community.
func (s *MessengerBoundedResolveSuite) TestAlreadyUpToDateIsSuccess() {
	community := s.createLocalCommunity()

	r := s.newCommunityRequest(defaultStoreNodeRequestConfig(storeNodeCommunityRequest), community.IDString())
	r.minimumDataClock = community.Clock()

	cont, _ := r.shouldFetchNextPage(0)
	s.Require().True(cont, "up-to-date request keeps paging until the cap")
	s.Require().Equal(storeNodeRequestAlreadyUpToDate, r.result.outcome)

	cont, _ = r.shouldFetchNextPage(0)
	s.Require().False(cont, "must stop at the page cap")
	s.Require().Equal(storeNodeRequestAlreadyUpToDate, r.result.outcome)
	s.Require().Nil(r.result.community)
}

// TestFetchWindowPerRequestType verifies communities use a 7-day window and
// contacts a 31-day window, and that these windows are what the request feeds
// into the batch From/To bounds.
func (s *MessengerBoundedResolveSuite) TestFetchWindowPerRequestType() {
	comCfg := defaultStoreNodeRequestConfig(storeNodeCommunityRequest)
	s.Require().Equal(7*oneDayDuration, comCfg.FetchWindow)

	contactCfg := defaultStoreNodeRequestConfig(storeNodeContactRequest)
	s.Require().Equal(oneMonthDuration, contactCfg.FetchWindow)

	from, to := s.m.calculateMailserverTimeBounds(comCfg.FetchWindow)
	s.Require().Equal(7*oneDayDuration, to.Sub(from))

	from, to = s.m.calculateMailserverTimeBounds(contactCfg.FetchWindow)
	s.Require().Equal(oneMonthDuration, to.Sub(from))
}
