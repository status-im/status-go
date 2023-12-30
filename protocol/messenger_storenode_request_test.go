package protocol

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/protocol/transport"

	"github.com/status-im/status-go/multiaccounts/accounts"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/common/shard"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/tt"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"

	mailserversDB "github.com/status-im/status-go/services/mailservers"
	waku2 "github.com/status-im/status-go/wakuv2"
	wakuV2common "github.com/status-im/status-go/wakuv2/common"
)

const (
	localFleet              = "local-test-fleet-1"
	localMailserverID       = "local-test-mailserver"
	storeNodeConnectTimeout = 500 * time.Millisecond
	runLocalTests           = false
)

func TestMessengerStoreNodeRequestSuite(t *testing.T) {
	suite.Run(t, new(MessengerStoreNodeRequestSuite))
}

type MessengerStoreNodeRequestSuite struct {
	suite.Suite

	cancel chan struct{}

	owner *Messenger
	bob   *Messenger

	wakuStoreNode    *waku2.Waku
	storeNodeAddress string

	ownerWaku types.Waku
	bobWaku   types.Waku

	logger *zap.Logger
}

type singleResult struct {
	EnvelopesCount   int
	Envelopes        []*wakuV2common.ReceivedMessage
	Error            error
	FetchedCommunity *communities.Community
}

func (r *singleResult) toString() string {
	resultString := ""
	communityString := ""

	if r.FetchedCommunity != nil {
		communityString = fmt.Sprintf("clock: %d (%s), name: %s, members: %d",
			r.FetchedCommunity.Clock(),
			time.Unix(int64(r.FetchedCommunity.Clock()), 0).UTC(),
			r.FetchedCommunity.Name(),
			len(r.FetchedCommunity.Members()),
		)
	}

	if r.Error != nil {
		resultString = fmt.Sprintf("error: %s", r.Error.Error())
	} else {
		resultString = fmt.Sprintf("envelopes fetched: %d, community %s",
			r.EnvelopesCount, communityString)
	}

	for i, envelope := range r.Envelopes {
		resultString += fmt.Sprintf("\n\tenvelope %3.0d: %s, timestamp: %d (%s), size: %d bytes, contentTopic: %s, pubsubTopic: %s",
			i+1,
			envelope.Hash().Hex(),
			envelope.Envelope.Message().GetTimestamp(),
			time.Unix(0, envelope.Envelope.Message().GetTimestamp()).UTC(),
			len(envelope.Envelope.Message().Payload),
			envelope.Envelope.Message().ContentTopic,
			envelope.Envelope.PubsubTopic(),
		)
	}

	return resultString
}

func (s *MessengerStoreNodeRequestSuite) SetupTest() {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	cfg.Development = false
	cfg.DisableStacktrace = true
	s.logger = tt.MustCreateTestLoggerWithConfig(cfg)

	s.cancel = make(chan struct{}, 10)

	storeNodeLogger := s.logger.Named("store-node-waku")
	s.wakuStoreNode = NewWakuV2(&s.Suite, storeNodeLogger, true, true, false)

	storeNodeListenAddresses := s.wakuStoreNode.ListenAddresses()
	s.Require().LessOrEqual(1, len(storeNodeListenAddresses))

	s.storeNodeAddress = storeNodeListenAddresses[0]
	s.logger.Info("store node ready", zap.String("address", s.storeNodeAddress))
}

func (s *MessengerStoreNodeRequestSuite) TearDown() {
	close(s.cancel)
	s.Require().NoError(s.wakuStoreNode.Stop())
	TearDownMessenger(&s.Suite, s.owner)
	TearDownMessenger(&s.Suite, s.bob)
}

func (s *MessengerStoreNodeRequestSuite) createOwner() {
	wakuLogger := s.logger.Named("owner-waku-node")
	wakuV2 := NewWakuV2(&s.Suite, wakuLogger, true, false, false)
	s.ownerWaku = gethbridge.NewGethWakuV2Wrapper(wakuV2)

	messengerLogger := s.logger.Named("owner-messenger")
	s.owner = s.newMessenger(s.ownerWaku, messengerLogger, s.storeNodeAddress)

	// We force the owner to use the store node as relay peer
	err := s.owner.DialPeer(s.storeNodeAddress)
	s.Require().NoError(err)
}

func (s *MessengerStoreNodeRequestSuite) createBob() {
	wakuLogger := s.logger.Named("bob-waku-node")
	wakuV2 := NewWakuV2(&s.Suite, wakuLogger, true, false, false)
	s.bobWaku = gethbridge.NewGethWakuV2Wrapper(wakuV2)

	messengerLogger := s.logger.Named("bob-messenger")
	s.bob = s.newMessenger(s.bobWaku, messengerLogger, s.storeNodeAddress)
}

func (s *MessengerStoreNodeRequestSuite) newMessenger(shh types.Waku, logger *zap.Logger, mailserverAddress string) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	mailserversSQLDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)
	err = sqlite.Migrate(mailserversSQLDb) // migrate default
	s.Require().NoError(err)

	mailserversDatabase := mailserversDB.NewDB(mailserversSQLDb)
	err = mailserversDatabase.Add(mailserversDB.Mailserver{
		ID:      localMailserverID,
		Name:    localMailserverID,
		Address: mailserverAddress,
		Fleet:   localFleet,
	})
	s.Require().NoError(err)

	options := []Option{
		WithMailserversDatabase(mailserversDatabase),
		WithClusterConfig(params.ClusterConfig{
			Fleet: localFleet,
		}),
	}

	messenger, err := newMessengerWithKey(shh, privateKey, logger, options)

	s.Require().NoError(err)
	return messenger
}

func (s *MessengerStoreNodeRequestSuite) createCommunity(m *Messenger) *communities.Community {
	s.waitForAvailableStoreNode(m)

	storeNodeSubscription := s.setupStoreNodeEnvelopesWatcher(nil)

	createCommunityRequest := &requests.CreateCommunity{
		Name:        RandomLettersString(10),
		Description: RandomLettersString(20),
		Color:       RandomColor(),
		Tags:        RandomCommunityTags(3),
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	response, err := m.CreateCommunity(createCommunityRequest, false)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	s.waitForEnvelopes(storeNodeSubscription, 1)

	return response.Communities()[0]
}

func (s *MessengerStoreNodeRequestSuite) requireCommunitiesEqual(c *communities.Community, expected *communities.Community) {
	if expected == nil {
		s.Require().Nil(c)
		return
	}
	s.Require().NotNil(c)
	s.Require().Equal(expected.IDString(), c.IDString())
	s.Require().Equal(expected.Clock(), c.Clock())
	s.Require().Equal(expected.Name(), c.Name())
	s.Require().Equal(expected.Identity().Description, c.Identity().Description)
	s.Require().Equal(expected.Color(), c.Color())
	s.Require().Equal(expected.Tags(), c.Tags())
	s.Require().Equal(expected.Shard(), c.Shard())
}

func (s *MessengerStoreNodeRequestSuite) requireContactsEqual(c *Contact, expected *Contact) {
	s.Require().Equal(expected.DisplayName, c.DisplayName)
	s.Require().Equal(expected.Bio, c.Bio)
	s.Require().Equal(expected.SocialLinks, c.SocialLinks)
}

func (s *MessengerStoreNodeRequestSuite) fetchCommunity(m *Messenger, communityShard communities.CommunityShard, expectedCommunity *communities.Community) StoreNodeRequestStats {
	options := []StoreNodeRequestOption{
		WithWaitForResponseOption(true),
	}

	fetchedCommunity, stats, err := m.storeNodeRequestsManager.FetchCommunity(communityShard, options)

	s.Require().NoError(err)
	s.requireCommunitiesEqual(fetchedCommunity, expectedCommunity)

	return stats
}

func (s *MessengerStoreNodeRequestSuite) fetchProfile(m *Messenger, contactID string, expectedContact *Contact) {
	fetchedContact, err := m.FetchContact(contactID, true)
	s.Require().NoError(err)
	s.Require().NotNil(fetchedContact)
	s.Require().Equal(contactID, fetchedContact.ID)

	if expectedContact != nil {
		s.requireContactsEqual(fetchedContact, expectedContact)
	}
}

func (s *MessengerStoreNodeRequestSuite) waitForAvailableStoreNode(messenger *Messenger) {
	WaitForAvailableStoreNode(&s.Suite, messenger, storeNodeConnectTimeout)
}

func (s *MessengerStoreNodeRequestSuite) setupEnvelopesWatcher(wakuNode *waku2.Waku, topic *wakuV2common.TopicType, cb func(envelope *wakuV2common.ReceivedMessage)) {
	envelopesWatcher := make(chan wakuV2common.EnvelopeEvent, 100)
	envelopesSub := wakuNode.SubscribeEnvelopeEvents(envelopesWatcher)

	go func() {
		defer envelopesSub.Unsubscribe()
		for {
			select {
			case <-s.cancel:
				return

			case envelopeEvent := <-envelopesWatcher:
				if envelopeEvent.Event != wakuV2common.EventEnvelopeAvailable {
					continue
				}
				if topic != nil && *topic != envelopeEvent.Topic {
					continue
				}
				envelope := wakuNode.GetEnvelope(envelopeEvent.Hash)
				cb(envelope)
				s.logger.Debug("envelope available event for fetched content topic",
					zap.Any("envelopeEvent", envelopeEvent),
					zap.Any("envelope", envelope),
				)
			}

		}
	}()
}

func (s *MessengerStoreNodeRequestSuite) setupStoreNodeEnvelopesWatcher(topic *wakuV2common.TopicType) <-chan string {
	storeNodeSubscription := make(chan string, 100)
	s.setupEnvelopesWatcher(s.wakuStoreNode, topic, func(envelope *wakuV2common.ReceivedMessage) {
		storeNodeSubscription <- envelope.Hash().String()
	})
	return storeNodeSubscription
}

func (s *MessengerStoreNodeRequestSuite) waitForEnvelopes(subscription <-chan string, expectedEnvelopesCount int) {
	for i := 0; i < expectedEnvelopesCount; i++ {
		select {
		case <-subscription:
		case <-time.After(5 * time.Second):
			s.Require().Fail("timeout waiting for store node to receive envelopes")
		}
	}
}

func (s *MessengerStoreNodeRequestSuite) TestRequestCommunityInfo() {
	s.createOwner()
	s.createBob()

	community := s.createCommunity(s.owner)

	s.waitForAvailableStoreNode(s.bob)
	s.fetchCommunity(s.bob, community.CommunityShard(), community)
}

func (s *MessengerStoreNodeRequestSuite) TestConsecutiveRequests() {
	s.createOwner()
	s.createBob()

	community := s.createCommunity(s.owner)

	// Test consecutive requests to check that requests in manager are finalized
	// At second request we expect to fetch nothing, because the community is already in the database
	s.fetchCommunity(s.bob, community.CommunityShard(), community)
	s.fetchCommunity(s.bob, community.CommunityShard(), nil)
}

func (s *MessengerStoreNodeRequestSuite) TestSimultaneousCommunityInfoRequests() {
	s.createOwner()
	s.createBob()

	community := s.createCommunity(s.owner)

	storeNodeRequestsCount := 0
	s.bob.storeNodeRequestsManager.onPerformingBatch = func(batch MailserverBatch) {
		storeNodeRequestsCount++
	}

	s.waitForAvailableStoreNode(s.bob)

	wg := sync.WaitGroup{}

	// Make 2 simultaneous fetch requests
	// 1 fetch request = 2 requests to store node (fetch shard and fetch community)
	// only 2 request to store node is expected
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.fetchCommunity(s.bob, community.CommunityShard(), community)
		}()
	}

	wg.Wait()
	s.Require().Equal(2, storeNodeRequestsCount)
}

func (s *MessengerStoreNodeRequestSuite) TestRequestNonExistentCommunity() {
	// On test start store node database is empty, so just request any valid community ID.
	request := FetchCommunityRequest{
		CommunityKey:    "0x036dc11a0663f88e15912f0adb68c3c5f68ca0ca7a233f1a88ff923a3d39b2cf07",
		Shard:           nil,
		TryDatabase:     false,
		WaitForResponse: true,
	}

	s.createBob()

	s.waitForAvailableStoreNode(s.bob)
	fetchedCommunity, err := s.bob.FetchCommunity(&request)

	s.Require().NoError(err)
	s.Require().Nil(fetchedCommunity)
}

func (s *MessengerStoreNodeRequestSuite) TestRequestCommunityInfoWithStoreNodeDisconnected() {
	s.createOwner()
	s.createBob()

	community := s.createCommunity(s.owner)

	// WaitForAvailableStoreNode is done internally
	s.fetchCommunity(s.bob, community.CommunityShard(), community)
}

func (s *MessengerStoreNodeRequestSuite) TestRequestCommunityPagingAlgorithm() {
	const spamAmount = defaultStoreNodeRequestPageSize + initialStoreNodeRequestPageSize

	s.createOwner()
	s.createBob()

	// Create a community
	community := s.createCommunity(s.owner)
	contentTopic := wakuV2common.BytesToTopic(transport.ToTopic(community.IDString()))
	storeNodeSubscription := s.setupStoreNodeEnvelopesWatcher(&contentTopic)

	// Push spam to the same ContentTopic & PubsubTopic
	// The first requested page size is 1. All subsequent pages are limited to 20.
	// We want to test the algorithm, so we push 21 spam envelopes.
	for i := 0; i < spamAmount; i++ {
		spamMessage := common.RawMessage{
			Payload:             RandomBytes(16),
			Sender:              community.PrivateKey(),
			SkipEncryptionLayer: true,
			MessageType:         protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION,
			PubsubTopic:         community.PubsubTopic(),
		}
		_, err := s.owner.sender.SendPublic(context.Background(), community.IDString(), spamMessage)
		s.Require().NoError(err)
	}

	// Wait the store node to receive envelopes
	s.waitForEnvelopes(storeNodeSubscription, spamAmount)

	// Fetch the community
	stats := s.fetchCommunity(s.bob, community.CommunityShard(), community)

	// Expect 3 pages and 23 (24 spam + 1 community description + 1 general channel description) envelopes to be fetched.
	// First we fetch a more up-to-date, but an invalid spam message, fail to decrypt it as community description,
	// then we fetch another page of data and successfully decrypt a community description.
	s.Require().Equal(spamAmount+1, stats.FetchedEnvelopesCount)
	s.Require().Equal(2, stats.FetchedPagesCount) // TODO: revert to 3 when fixed: https://github.com/waku-org/nwaku/issues/2317
}

func (s *MessengerStoreNodeRequestSuite) TestRequestCommunityWithSameContentTopic() {
	s.createOwner()
	s.createBob()

	// Create 2 communities
	community1 := s.createCommunity(s.owner)
	community2 := s.createCommunity(s.owner)

	description2, err := community2.MarshaledDescription()
	s.Require().NoError(err)

	// Push community2 description to the same ContentTopic & PubsubTopic as community1.
	// This way we simulate 2 communities with same ContentTopic.
	spamMessage := common.RawMessage{
		Payload:             description2,
		Sender:              community2.PrivateKey(),
		SkipEncryptionLayer: true,
		MessageType:         protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION,
		PubsubTopic:         community1.PubsubTopic(),
	}
	_, err = s.owner.sender.SendPublic(context.Background(), community1.IDString(), spamMessage)
	s.Require().NoError(err)

	// Fetch the community
	s.fetchCommunity(s.bob, community1.CommunityShard(), community1)
}

func (s *MessengerStoreNodeRequestSuite) TestRequestMultipleCommunities() {
	s.createOwner()
	s.createBob()

	// Create 2 communities
	community1 := s.createCommunity(s.owner)
	community2 := s.createCommunity(s.owner)

	fetchedCommunities := map[string]*communities.Community{}

	err := WaitOnSignaledCommunityFound(s.bob,
		func() {
			err := s.bob.fetchCommunities([]communities.CommunityShard{
				community1.CommunityShard(),
				community2.CommunityShard(),
			})
			s.Require().NoError(err)
		},
		func(community *communities.Community) bool {
			fetchedCommunities[community.IDString()] = community
			return len(fetchedCommunities) == 2
		},
		1*time.Second,
		"communities were not signalled in time",
	)

	s.Require().NoError(err)
	s.Require().Contains(fetchedCommunities, community1.IDString())
	s.Require().Contains(fetchedCommunities, community2.IDString())
}

func (s *MessengerStoreNodeRequestSuite) TestRequestWithoutWaitingResponse() {
	s.createOwner()
	s.createBob()

	// Create a community
	community := s.createCommunity(s.owner)

	request := FetchCommunityRequest{
		CommunityKey:    community.IDString(),
		Shard:           nil,
		TryDatabase:     false,
		WaitForResponse: false,
	}

	fetchedCommunities := map[string]*communities.Community{}

	err := WaitOnSignaledCommunityFound(s.bob,
		func() {
			fetchedCommunity, err := s.bob.FetchCommunity(&request)
			s.Require().NoError(err)
			s.Require().Nil(fetchedCommunity)
		},
		func(community *communities.Community) bool {
			fetchedCommunities[community.IDString()] = community
			return len(fetchedCommunities) == 1
		},
		1*time.Second,
		"communities weren't signalled",
	)

	s.Require().NoError(err)
	s.Require().Len(fetchedCommunities, 1)
	s.Require().Contains(fetchedCommunities, community.IDString())

	s.requireCommunitiesEqual(fetchedCommunities[community.IDString()], community)
}

func (s *MessengerStoreNodeRequestSuite) TestRequestProfileInfo() {
	s.createOwner()

	// Set keypair (to be able to set displayName)
	ownerProfileKp := accounts.GetProfileKeypairForTest(true, false, false)
	ownerProfileKp.KeyUID = s.owner.account.KeyUID
	ownerProfileKp.Accounts[0].KeyUID = s.owner.account.KeyUID

	err := s.owner.settings.SaveOrUpdateKeypair(ownerProfileKp)
	s.Require().NoError(err)

	// Set display name, this will also publish contact code
	err = s.owner.SetDisplayName("super-owner")
	s.Require().NoError(err)

	s.createBob()
	s.fetchProfile(s.bob, s.owner.selfContact.ID, s.owner.selfContact)
}

// TestSequentialUpdates checks that making updates to the community
// immediately results in new store node fetched information.
// Before adding storeNodeSubscription we had problems with the test setup that we didn't have a mechanism to wait for store node to
// receive and process new messages.
func (s *MessengerStoreNodeRequestSuite) TestSequentialUpdates() {
	s.createOwner()
	s.createBob()

	community := s.createCommunity(s.owner)
	s.fetchCommunity(s.bob, community.CommunityShard(), community)

	contentTopic := wakuV2common.BytesToTopic(transport.ToTopic(community.IDString()))
	communityName := community.Name()

	storeNodeSubscription := s.setupStoreNodeEnvelopesWatcher(&contentTopic)

	for i := 0; i < 3; i++ {
		// Change community name, this will automatically publish a new community description
		ownerEditRequest := &requests.EditCommunity{
			CommunityID: community.ID(),
			CreateCommunity: requests.CreateCommunity{
				Name:        fmt.Sprintf("%s-%d", communityName, i),
				Description: community.DescriptionText(),
				Color:       community.Color(),
				Membership:  community.Permissions().Access,
			},
		}
		_, err := s.owner.EditCommunity(ownerEditRequest)
		s.Require().NoError(err)

		s.waitForEnvelopes(storeNodeSubscription, 1)

		// Get updated community from the database
		community, err = s.owner.communitiesManager.GetByID(community.ID())
		s.Require().NoError(err)
		s.Require().NotNil(community)

		s.fetchCommunity(s.bob, community.CommunityShard(), community)
	}
}

func (s *MessengerStoreNodeRequestSuite) TestRequestShardAndCommunityInfo() {
	s.createOwner()
	s.createBob()

	s.waitForAvailableStoreNode(s.owner)
	community := s.createCommunity(s.owner)

	expectedShard := &shard.Shard{
		Cluster: shard.MainStatusShardCluster,
		Index:   23,
	}

	shardRequest := &requests.SetCommunityShard{
		CommunityID: community.ID(),
		Shard:       expectedShard,
	}

	_, err := s.owner.SetCommunityShard(shardRequest)
	s.Require().NoError(err)

	s.waitForAvailableStoreNode(s.bob)

	communityShard := community.CommunityShard()

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(community)
	s.Require().NotNil(community.Shard())

	s.fetchCommunity(s.bob, communityShard, community)
}

func (s *MessengerStoreNodeRequestSuite) TestFiltersNotRemoved() {
	s.createOwner()
	s.createBob()

	community := s.createCommunity(s.owner)

	// The owner is a member of the community, so he has a filter for community description content topic.
	// We want to check that filter is not removed by `FetchCommunity` call.
	filterBefore := s.owner.transport.FilterByChatID(community.IDString())
	s.Require().NotNil(filterBefore)

	s.fetchCommunity(s.owner, community.CommunityShard(), nil)

	filterAfter := s.owner.transport.FilterByChatID(community.IDString())
	s.Require().NotNil(filterAfter)

	s.Require().Equal(filterBefore.FilterID, filterAfter.FilterID)
}

func (s *MessengerStoreNodeRequestSuite) TestFiltersRemoved() {
	s.createOwner()
	s.createBob()

	community := s.createCommunity(s.owner)

	// The bob is a member of the community, so he has no filters for community description content topic.
	// We want to check that filter created by `FetchCommunity` is removed on request finish.
	filterBefore := s.bob.transport.FilterByChatID(community.IDString())
	s.Require().Nil(filterBefore)

	s.fetchCommunity(s.bob, community.CommunityShard(), community)

	filterAfter := s.bob.transport.FilterByChatID(community.IDString())
	s.Require().Nil(filterAfter)
}

func (s *MessengerStoreNodeRequestSuite) TestRequestCommunityEnvelopesOrder() {
	s.createOwner()
	s.createBob()

	community := s.createCommunity(s.owner)

	// Push 5 descriptions to the store node
	for i := 0; i < 4; i++ {
		err := s.owner.publishOrg(community)
		s.Require().NoError(err)
	}

	// Subscribe to received envelope

	bobWakuV2 := gethbridge.GetGethWakuV2From(s.bobWaku)
	contentTopic := wakuV2common.BytesToTopic(transport.ToTopic(community.IDString()))

	var prevEnvelope *wakuV2common.ReceivedMessage

	s.setupEnvelopesWatcher(bobWakuV2, &contentTopic, func(envelope *wakuV2common.ReceivedMessage) {
		// We check that each next envelope fetched is newer than the previous one
		if prevEnvelope != nil {
			s.Require().Greater(
				envelope.Envelope.Message().GetTimestamp(),
				prevEnvelope.Envelope.Message().GetTimestamp())
		}
		prevEnvelope = envelope
	})

	// Force a single-envelope page size to be able to check the order.
	// Also force all envelopes to be fetched.

	options := []StoreNodeRequestOption{
		WithWaitForResponseOption(true),
		WithStopWhenDataFound(false),
		WithInitialPageSize(1),
		WithFurtherPageSize(1),
	}

	// Fetch the community

	fetchedCommunity, _, err := s.bob.storeNodeRequestsManager.FetchCommunity(community.CommunityShard(), options)
	s.Require().NoError(err)
	s.requireCommunitiesEqual(fetchedCommunity, community)
}

// TestFetchRealCommunity is intended to only run locally to check the community description in all of the store nodes.
// Shouldn't be executed in CI, because it relies on connection to the real network.
//
// To run this test, first set `runLocalTests` to true.
// Then carefully set all of communityID, communityShard, fleet and other const variables.
// NOTE: I only tested it with the default parameters, but in theory it should work for any configuration.
func (s *MessengerStoreNodeRequestSuite) TestFetchRealCommunity() {
	if !runLocalTests {
		return
	}

	const communityID = "0x03073514d4c14a7d10ae9fc9b0f05abc904d84166a6ac80add58bf6a3542a4e50a"
	var communityShard *shard.Shard

	const fleet = params.FleetStatusProd
	const useShardAsDefaultTopic = false
	const clusterID = 0
	const userPrivateKeyString = "" // When empty a new user will be created
	contentTopic := wakuV2common.BytesToTopic(transport.ToTopic(communityID))
	nodesList := mailserversDB.DefaultMailserversByFleet(fleet)

	results := map[string]singleResult{}
	wg := sync.WaitGroup{}

	// We run a separate request for each node in the fleet.

	for i, mailserver := range nodesList {
		wg.Add(1)

		go func(i int, mailserver mailserversDB.Mailserver) {
			defer wg.Done()

			fmt.Printf("--- starting for %s\n", mailserver.ID)

			result := singleResult{}

			//
			// Create WakuV2 node
			// NOTE: Another option was to create a bare waku node and fetch envelopes directly with it
			// 		 and after that push all of the envelopes to a new messenger and check the result.
			// 		 But this turned out to be harder to implement.
			//

			wakuLogger := s.logger.Named(fmt.Sprintf("user-waku-node-%d", i))
			messengerLogger := s.logger.Named(fmt.Sprintf("user-messenger-%d", i))

			wakuV2 := NewWakuV2(&s.Suite, wakuLogger, true, false, useShardAsDefaultTopic)
			userWaku := gethbridge.NewGethWakuV2Wrapper(wakuV2)

			//
			// Create a messenger to process envelopes
			//

			var privateKeyString = userPrivateKeyString

			if privateKeyString == "" {
				privateKey, err := crypto.GenerateKey()
				s.Require().NoError(err)
				privateKeyString = hexutil.Encode(crypto.FromECDSA(privateKey))
			}

			privateKeyBytes, err := hexutil.Decode(privateKeyString)
			s.Require().NoError(err)
			privateKey, err := crypto.ToECDSA(privateKeyBytes)
			s.Require().NoError(err)

			// Mock a local fleet with single store node
			// This is done by settings custom store nodes in the database

			mailserversSQLDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
			s.Require().NoError(err)
			mailserversDatabase := mailserversDB.NewDB(mailserversSQLDb)

			mailserver.Fleet = localFleet
			err = mailserversDatabase.Add(mailserver)
			s.Require().NoError(err)

			options := []Option{
				WithMailserversDatabase(mailserversDatabase),
				WithClusterConfig(params.ClusterConfig{
					Fleet:     localFleet,
					ClusterID: clusterID,
				}),
			}

			// Create user without `createBob` func to force desired fleet
			user, err := newMessengerWithKey(userWaku, privateKey, messengerLogger, options)
			s.Require().NoError(err)
			defer TearDownMessenger(&s.Suite, user)

			communityAddress := communities.CommunityShard{
				CommunityID: communityID,
				Shard:       communityShard,
			}

			// Setup envelopes watcher to gather fetched envelopes

			s.setupEnvelopesWatcher(wakuV2, &contentTopic, func(envelope *wakuV2common.ReceivedMessage) {
				result.Envelopes = append(result.Envelopes, envelope)
			})

			// Start fetching

			storeNodeRequestOptions := []StoreNodeRequestOption{
				WithWaitForResponseOption(true),
				WithStopWhenDataFound(false),                         // In this test we want all envelopes to be fetched
				WithInitialPageSize(defaultStoreNodeRequestPageSize), // Because we're fetching all envelopes anyway
			}

			fetchedCommunity, stats, err := user.storeNodeRequestsManager.FetchCommunity(communityAddress, storeNodeRequestOptions)

			result.EnvelopesCount = stats.FetchedEnvelopesCount
			result.FetchedCommunity = fetchedCommunity
			result.Error = err

			results[mailserver.ID] = result
		}(i, mailserver)
	}

	// Wait for all requests to finish

	wg.Wait()

	// Print the results

	for storeNodeName, result := range results {
		fmt.Printf("%s --- %s\n", storeNodeName, result.toString())
	}
}
