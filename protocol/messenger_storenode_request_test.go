package protocol

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/protocol/communities/token"
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
	"github.com/status-im/status-go/t/helpers"

	"github.com/status-im/status-go/services/communitytokens"
	mailserversDB "github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/services/wallet/bigint"
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

	collectiblesServiceMock *CollectiblesServiceMock

	logger *zap.Logger
}

type singleResult struct {
	EnvelopesCount   int
	Envelopes        []*wakuV2common.ReceivedMessage
	ShardEnvelopes   []*wakuV2common.ReceivedMessage
	Error            error
	FetchedCommunity *communities.Community
}

func (r *singleResult) ShardEnvelopesHashes() []string {
	out := make([]string, 0, len(r.ShardEnvelopes))
	for _, e := range r.ShardEnvelopes {
		out = append(out, e.Hash().String())
	}
	return out
}

func (r *singleResult) EnvelopesHashes() []string {
	out := make([]string, 0, len(r.Envelopes))
	for _, e := range r.Envelopes {
		out = append(out, e.Hash().String())
	}
	return out
}

func (r *singleResult) toString() string {
	resultString := ""
	communityString := ""

	if r.FetchedCommunity != nil {
		communityString = fmt.Sprintf("clock: %d, name: '%s', members: %d",
			r.FetchedCommunity.Clock(),
			r.FetchedCommunity.Name(),
			len(r.FetchedCommunity.Members()),
		)
	}

	if r.Error != nil {
		resultString = fmt.Sprintf("error: %s", r.Error.Error())
	} else {
		resultString = fmt.Sprintf("envelopes fetched: %d, community - %s",
			r.EnvelopesCount, communityString)
	}

	for i, envelope := range r.ShardEnvelopes {
		resultString += fmt.Sprintf("\n\tshard envelope %3.0d: %s, timestamp: %d (%s), size: %d bytes, contentTopic: %s, pubsubTopic: %s",
			i+1,
			envelope.Hash().Hex(),
			envelope.Envelope.Message().GetTimestamp(),
			time.Unix(0, envelope.Envelope.Message().GetTimestamp()).UTC(),
			len(envelope.Envelope.Message().Payload),
			envelope.Envelope.Message().ContentTopic,
			envelope.Envelope.PubsubTopic(),
		)
	}

	for i, envelope := range r.Envelopes {
		resultString += fmt.Sprintf("\n\tdescription envelope %3.0d: %s, timestamp: %d (%s), size: %d bytes, contentTopic: %s, pubsubTopic: %s",
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
	s.logger = tt.MustCreateTestLogger()

	s.cancel = make(chan struct{}, 10)

	s.collectiblesServiceMock = &CollectiblesServiceMock{}

	s.createStore()
}

func (s *MessengerStoreNodeRequestSuite) TearDown() {
	close(s.cancel)
	s.Require().NoError(s.wakuStoreNode.Stop())
	TearDownMessenger(&s.Suite, s.owner)
	TearDownMessenger(&s.Suite, s.bob)
}

func (s *MessengerStoreNodeRequestSuite) createStore() {
	cfg := testWakuV2Config{
		logger:                 s.logger.Named("store-waku"),
		enableStore:            true,
		useShardAsDefaultTopic: false,
		clusterID:              shard.UndefinedShardValue,
	}

	s.wakuStoreNode = NewTestWakuV2(&s.Suite, cfg)
	s.storeNodeAddress = s.wakuListenAddress(s.wakuStoreNode)
	s.logger.Info("store node ready", zap.String("address", s.storeNodeAddress))
}

func (s *MessengerStoreNodeRequestSuite) createOwner() {

	cfg := testWakuV2Config{
		logger:                 s.logger.Named("owner-waku"),
		enableStore:            false,
		useShardAsDefaultTopic: false,
		clusterID:              shard.UndefinedShardValue,
	}

	wakuV2 := NewTestWakuV2(&s.Suite, cfg)
	s.ownerWaku = gethbridge.NewGethWakuV2Wrapper(wakuV2)

	messengerLogger := s.logger.Named("owner-messenger")
	s.owner = s.newMessenger(s.ownerWaku, messengerLogger, s.storeNodeAddress)

	// We force the owner to use the store node as relay peer
	WaitForPeersConnected(&s.Suite, gethbridge.GetGethWakuV2From(s.ownerWaku), func() []string {
		err := s.owner.DialPeer(s.storeNodeAddress)
		s.Require().NoError(err)
		return []string{s.wakuStoreNode.PeerID().String()}
	})
}

func (s *MessengerStoreNodeRequestSuite) createBob() {
	cfg := testWakuV2Config{
		logger:                 s.logger.Named("bob-waku"),
		enableStore:            false,
		useShardAsDefaultTopic: false,
		clusterID:              shard.UndefinedShardValue,
	}
	wakuV2 := NewTestWakuV2(&s.Suite, cfg)
	s.bobWaku = gethbridge.NewGethWakuV2Wrapper(wakuV2)

	messengerLogger := s.logger.Named("bob-messenger")
	s.bob = s.newMessenger(s.bobWaku, messengerLogger, s.storeNodeAddress)
}

func (s *MessengerStoreNodeRequestSuite) newMessenger(shh types.Waku, logger *zap.Logger, mailserverAddress string) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	options := []Option{
		WithAutoRequestHistoricMessages(false),
	}

	if mailserverAddress != "" {
		options = append(options,
			WithTestStoreNode(&s.Suite, localMailserverID, mailserverAddress, localFleet, s.collectiblesServiceMock),
		)
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
	s.Require().Equal(expected.TokenPermissions(), c.TokenPermissions())
	s.Require().Equal(expected.CommunityTokensMetadata(), c.CommunityTokensMetadata())
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < expectedEnvelopesCount; i++ {
		select {
		case <-subscription:
		case <-ctx.Done():
			err := fmt.Sprintf("timeout waiting for store node to receive envelopes, received: %d, expected: %d", i, expectedEnvelopesCount)
			s.Require().Fail(err)
		}
	}
}

func (s *MessengerStoreNodeRequestSuite) wakuListenAddress(waku *waku2.Waku) string {
	addresses := waku.ListenAddresses()
	s.Require().LessOrEqual(1, len(addresses))
	return addresses[0]
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
	s.Require().Equal(3, stats.FetchedPagesCount)
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
	s.waitForAvailableStoreNode(s.bob)
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

	community := s.createCommunity(s.owner)

	expectedShard := &shard.Shard{
		Cluster: shard.MainStatusShardCluster,
		Index:   23,
	}

	shardRequest := &requests.SetCommunityShard{
		CommunityID: community.ID(),
		Shard:       expectedShard,
	}

	shardTopic := transport.CommunityShardInfoTopic(community.IDString())
	contentContentTopic := wakuV2common.BytesToTopic(transport.ToTopic(shardTopic))
	storeNodeSubscription := s.setupStoreNodeEnvelopesWatcher(&contentContentTopic)

	_, err := s.owner.SetCommunityShard(shardRequest)
	s.Require().NoError(err)

	s.waitForEnvelopes(storeNodeSubscription, 1)

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

	const descriptionsCount = 4
	community := s.createCommunity(s.owner)
	contentTopic := wakuV2common.BytesToTopic(transport.ToTopic(community.IDString()))
	storeNodeSubscription := s.setupStoreNodeEnvelopesWatcher(&contentTopic)

	// Push a few descriptions to the store node
	for i := 0; i < descriptionsCount-1; i++ {
		err := s.owner.publishOrg(community, false)
		s.Require().NoError(err)
	}

	// Wait for store node to receive envelopes
	s.waitForEnvelopes(storeNodeSubscription, descriptionsCount-1)

	// Subscribe to received envelope
	bobWakuV2 := gethbridge.GetGethWakuV2From(s.bobWaku)

	var receivedEnvelopes []*wakuV2common.ReceivedMessage
	s.setupEnvelopesWatcher(bobWakuV2, &contentTopic, func(envelope *wakuV2common.ReceivedMessage) {
		receivedEnvelopes = append(receivedEnvelopes, envelope)
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

	// Ensure all expected envelopes were received
	s.Require().Equal(descriptionsCount, len(receivedEnvelopes))

	// We check that each next envelope fetched is newer than the previous one
	for i := 1; i < len(receivedEnvelopes); i++ {
		s.Require().Less(
			receivedEnvelopes[i].Envelope.Message().GetTimestamp(),
			receivedEnvelopes[i-1].Envelope.Message().GetTimestamp())
	}
}

/*
	TestFetchRealCommunity is not actually a test, but an utility to check the community description in all of the store nodes.
	It's intended to only run locally and shouldn't be executed in CI, because it relies on connection to the real network.

	TODO: It would be nice to move this code to a real utility in /cmd.
		  It should allow us to fairly verify the community owner and do other good things.

	To run this test, first set `runLocalTests` to true.
	Then carefully set all of communityID, communityShard, fleet and other const variables.

	NOTE: I only tested it with the default parameters, but in theory it should work for any configuration.
*/

type testFetchRealCommunityExampleTokenInfo struct {
	ChainID         uint64
	ContractAddress string
}

var testFetchRealCommunityExample = []struct {
	CommunityID            string
	CommunityShard         *shard.Shard // WARNING: I didn't test a sharded community
	Fleet                  string
	UseShardAsDefaultTopic bool
	ClusterID              uint16
	UserPrivateKeyString   string // When empty a new user will be created
	// Setup OwnerPublicKey and CommunityTokens if the community has owner token
	// This is needed to mock the owner verification
	OwnerPublicKey  string
	CommunityTokens []testFetchRealCommunityExampleTokenInfo
	// Fill these if you know what envelopes are expected.
	// The test will fail if fetched array doesn't equal to the expected one.
	CheckExpectedEnvelopes       bool
	ExpectedShardEnvelopes       []string
	ExpectedDescriptionEnvelopes []string
}{
	{
		//Example 1, status.prod fleet
		CommunityID:            "0x03073514d4c14a7d10ae9fc9b0f05abc904d84166a6ac80add58bf6a3542a4e50a",
		CommunityShard:         nil,
		Fleet:                  params.FleetStatusProd,
		UseShardAsDefaultTopic: false,
		ClusterID:              shard.UndefinedShardValue,
	},
	{
		// Example 3, shards.test fleet
		// https://status.app/c/CxiACi8KFGFwIHJlcSAxIHN0dCBiZWMgbWVtEgdkc2Fkc2FkGAMiByM0MzYwREYqAxkrHAM=#zQ3shwDYZHtrLE7NqoTGjTWzWUu6hom5D4qxfskLZfgfyGRyL
		CommunityID:            "0x03f64be95ed5c925022265f9250f538f65ed3dcf6e4ef6c139803dc02a3487ae7b",
		Fleet:                  params.FleetShardsTest,
		UseShardAsDefaultTopic: true,
		ClusterID:              shard.MainStatusShardCluster,

		CheckExpectedEnvelopes: true,
		ExpectedShardEnvelopes: []string{
			"0x8173eecd7ff9ebcaae3dde0e704daf9bdeb6d33b0d8505a67e7dc56d0d8fc07c",
			"0x596bbafbe0e0b625d165378cd4c7641a4d23aa1145c705aad666ddeaf60c88cd",
			"0x8a1ee798f3657da5a463e5f878ab2455d05b8f552359b58330ccd7fa4f5624b0",
			"0x97bcde2103a01984bb45a8590a6cb6972411445a1b2d40e181d5f2b5366fa5f1",
			"0x26e3c0c880d1a2c4e81bf4fffbdb8b7e1ecc91fce7c6a05ee87d200d62ffc11e",
			"0x1a8820bd61ebcc9de75c25f31c9b05eb6e880a5a4902679bb6ce2f43f61bf159",
			"0xce450cfb5f79d761f34dea5b2ccec63751886e43ae63477e12f517c31f800aeb",
			"0x9607bd1cf08355c44bcce055da197ba177201882736fa8874910194ccdaa8760",
			"0x0c4b989ca69f529e571e6ea8b3230a85e057d8b2ae6147d1fedc2a01f2816ed6",
			"0xe40ea64c9007a064b6324b614976510f2a433c9f84d87139df8f66b536e37ee4",
			"0x7a028466a095e40650bb0ef16e903309b0c38c5a7cb7e2e9debd0acf2151448d",
			"0x96419c6be375b2b348778d4694e3a491de84eecde601d5d405a0e72e9cece4a1",
			"0xbcaeb5e86128638fab7203428daddd741df44ceeabe7d9d25936a10cd0a8b808",
			"0x2e0b5872cb5a7c9a3273048eb2dfcb1d6a28faad3fe307a7db6c2dbaca9ce462",
			"0xfa96bbe4125514ce73c52ef3ccb1c4ad9c4ad4afe8803de8ab9309fe9483b1d0",
			"0xdeddfc82f70cce77c26959d91851fbc33afe648428c3e6ea349b2a2456b92111",
			"0x5b12f17d7b712071f57bb48b7dcd0d6568ff5e7c3f8b3811013aea8dde9c6243",
			"0x18928fd044482c75518162104d487e6fe504f086eb8c5e9f21aa4bce2811d0fa",
			"0x543c156ced76138d69229a39425a0a1cabd617770e023333c10501b979f52d61",
			"0xf46ea6bf5ab6a70662bcf227cc5d2c8c7a70ce42a88e5bb7ebe9e598668a8ae2",
			"0xedb9628dc1ce5b0ec899c3813dd4159a2e06fb3dc88ffaae047e927c804ad0b3",
			"0x16248eccc3544af3fc4a73467d0925ca2f3741eb623516ee369f710e4aa8a3ef",
			"0x6a85f784a9004b56bbb47d87f5541173f05bd61ff5b26e41c714adbb5516e9ff",
			"0x91e320be2cb5c6178027390cbce165fe088a1a35e1442382064ddbc9aabea8a2",
			"0x676496dd36ae40e184863725bfb7425e46d916f73f4b0dd5d10324f4e9325da6",
		},
		ExpectedDescriptionEnvelopes: []string{
			"0xe2c38667ee160861b3dc5a00e4422f47de1303c8b61f0a33c4853ce0b71d0ae4",
			"0x7d8392baa9dd134e43287e58d69b8c9f50aa5c144adda6a3c7d32f00c5dea309",
			"0xf74918d445709ccf9c29e776d27e9b7dc31f25a28473e8fcd89ed9de8a2e6df4",
			"0xafd7e9b6245d88ea2b4fe70265aa3c5ce5618827c1092f8e3058315ac27c5b98",
			"0x4d16bd4fcfc2d8736dc29d8c7287671f7e1df62af74943dc40a39ae18f388a07",
			"0xd101f14bcf6bb9a5e72b934b3e74ebe3f77774037cb5b193803b264d69bfb9bf",
			"0xaa857389e8886401678690bb4dbc664486bd7039427ca53e826197b303696cff",
			"0x4443448e575824330a96d5114a9a7d3fe0ee7168ab6c7a646057ca4502fb91d2",
			"0xfbb79b8d0ee1a61109c543cbe580412fe6d23de33d163006f74fef4addbdad37",
			"0x51bc55c732e0e9db40fd8865a8ae20cbd99bfd4f95c62cf8b591e793d9b642d2",
			"0xc28ca742c0e159a60941fb68ee5832016b510afe54e8ee8bb2200495ac29f1e3",
			"0xf44f1714743a55f0170ba3627390d84cb9307326028abf7236222c93104db833",
			"0x69d067da262b124d2eb6a8ba9f08e0a1ad66afb8d7ff3641a256992fff53a6de",
			"0xa86a08457854fbea347cdd92fd390a330a9971967c551bebc53b18ccfea876fe",
			"0x592850f8bc3c6079826f168971746bd2a1b50a5011fe3b233ead6c72c92a3373",
			"0x1b3151d7b9b37350e86c937dc2e7964d472815bbf9275714bcb16a0c4327fe3d",
			"0x1f2beca64e52f996127647cf3f3abd2e4fe501646fc39e98713ae064333388b3",
			"0xeafc8f9d3114426c08748e6171710874074fb1eb732d9364830ce9b58955c83c",
			"0x6e014a5ec75465efaf036353ec5811b8f710f608a04002925eba3a0a37a30423",
			"0xf7e12a5829cf90e4272132e7b62c5bf0dd09100c8d498c66c9740a798db559b3",
			"0x62da7e828862c3c8692cbd077c5a62266d811764184378e55f9f1066510b4652",
			"0x74201394d05b914bc7e6ce4d2fcf0b119491c80644f48ac9fd37b842e4a0275a",
			"0x9d4c0b1be53810c45c2fa744baa8d16ef4ae3a319b09043f4b4e053127461bf0",
			"0x1937979514ea1dba8ab3b621fc3d0a3f6246b4bf1f9b4073888b8dbd9b4a765a",
			"0xf767f3f36fdecb5ef6650542232334df836bfca1e7f72f1215df50d3f9f9c9bc",
			"0x3a06002325502bc39a77962241fe274d4e88f61762194d321f9cc95272ed4a74",
			"0x13caae58261c181d4974d7a68e0b7c8580c3cc569840179d53ae76407548d8b8",
			"0x36f3b4afbcc4177a7aef26ad567839ffce51896d4c40d0a08d222cebd1255e3b",
			"0x159e685bbab26a5d54ab817c93c9c610055bcf2af75290abcc9a84f1b85a2de9",
			"0x1fd5ff73d7ea9a19f282bd0716f04a5e86b7c515839f0c721f66b3fe99161054",
			"0x95b1e9ada4913ca809c9c28fc225a21753f18a90253660750900c78f79ad2a00",
			"0x4334826934a7cbfb7446ec9d581fa6433c5d1f7f51b97f24717f55cffa320c65",
			"0x0c01d07108c448797ffd14a2152fd38d1764c8a9c5e2f3da12f70551588add7a",
			"0xe7071c6587fc277c4f4c0d7e4575de1a0843d3cf6c2a4aac79be79edc1608038",
			"0x5da4e482f3e6eacf080db685e00c199c8cbbad9a8f43b1d94944426444a7a84a",
			"0x638f551acdd7ccffd6a40ad12bbba1da8fd8a58157fdf9625b12d4a95b4eef71",
			"0xa1a52c28e0481f6004d98bbce906676fff67f04246454bd33fba02c640355af0",
			"0xe0300eb9e0f215ace491b1104665b48b9f6bff039af40e0cfc52a3ce766e747f",
			"0xd092c04d51ee963d59953324d84188a0c1636e8600cb0f5f6f3f4f826d70c8f3",
			"0x8d94bbfee687d534361fc3069079cf4e4f7db2a179d24e6419f67e38b5f0bd34",
			"0x1fd7a4d2c04fca3875126b7a951f619b4da0000ca47496df0c2fb1048a145108",
			"0xbbefbd116cbb23de193318b328412addc500af965d31ba481d70fa1d9e99461d",
			"0xaa4e0e8bd820438e22b93371bda24a29922d33c15fb312b343d2e81a22cbdd95",
			"0x76aef29ea4dde107c22c520efb2a4516b69ae83bc237281d9990f68397d801f5",
			"0x804789119513a065d892cba5d240cb4d89d7329aeee93fcd8e85379a4d362fc9",
			"0x9029b4a13903a3369e3466f1bfabae3f26b6721628db138eaba25c1f55f6fc1e",
			"0xee38c209cb95035a289034c737e5775877145efea31b2a01f7c9241ff02f3e92",
			"0x3e76da87895ca821db3b7ed7dc6949557c620d9cbcaf97af39ed4955d37b734e",
			"0xf5e77eb8f9a5c52e09a56dd5e461bfee6cf9a73e1253f1d41bcde81fe3646997",
			"0x083e06375c366283e541b249ea8646c3f31feb970078e95861ea399f0a57d09a",
			"0xcd7db07ba557ec1ba0104909fdb958661c60c82213a75e8d15e7b262ef4f58b7",
			"0x57b49dc83e1d3ac7b56bb7d758a9bf448339593311103bce4f0a53028587d577",
			"0xb08cd92a5ec6f44a6129a60107132ca17d5fef29fb2bc5ebba14028d57a8038e",
			"0x76959f98c8c734307a985294c26f008f3f705912aab02a5b3a0602a8598c02e2",
			"0xd8ad7df58ffeec20b16a140bdd91484a34fca3ad7fc602043530ca63c307809a",
			"0x6872ef39653bb208ef51f5b11c4bda3eb494a13f5da14e33687d30aef99ef383",
			"0x0fabebc2e0c02f4add94886216b01ecdbd34929ad303d1a20a4505bf729038f6",
			"0xc18270cd532bb3d34f62704feb722c40be48aedb5ad38d4d09fd67f5843b686d",
			"0xbc16217ece82998783d4209ed3bc3f2f33d92630e43933b0129eb8b792500a3f",
			"0xbda651e3b9c82f4bcf5b16252407fc888952820c842c49c06b4f01c8127e359a",
			"0xb4b1799950c6aca3b011ffb775d0f973437d7d46e40cf7b379ff736d08f24eb2",
			"0x38f12fb09c71dd720cacbb2102ac78ad6fbf830558adc7af9fb773f39e728bdc",
			"0x489eb6fa2f5ee5b2a071c7083bf36a0a6cb4ec96049707d25843d9a97b4ac7be",
			"0x64ea5655c8caf89a53c94edd5a47ba750d9fbcf099ec0dcd4026656b044486f1",
			"0x501aee1c5da6aaeaae14abffefbc377b59ebe3fcaa9981bc83bfeffb25344749",
			"0x9a3d360ea866102a6268ffd2001617c442b74b221d131fb3c08ae29bfac18203",
		},
	},
	{
		//Example 1, shards.test fleet
		CommunityID:            "0x02471dd922756a3a50b623e59cf3b99355d6587e43d5c517eb55f9aea9d3fe9fe9",
		Fleet:                  params.FleetShardsTest,
		UseShardAsDefaultTopic: true,
		ClusterID:              shard.MainStatusShardCluster,
		CheckExpectedEnvelopes: true,
		ExpectedShardEnvelopes: []string{
			"0xc3e68e838d09e0117b3f3fd27aabe5f5a509d13e9045263c78e6890953d43547",
			"0x5ee13d052bedb855ce2b9ba6f43c78233fbd4e6539a3bdf156497053c6ddf76d",
			"0xfb6638b7e050f9323a0fe7b84986b5c6f8827965e67e3b3bd0fea21cf24e43de",
		},
		ExpectedDescriptionEnvelopes: []string{
			"0x5b4fa95d430c939c1cbbb26175eabfb4ee058d508c6b4c0e26624958ba02c3ce",
			"0xbf44409ee40dea7816186b37a45dfebabcee59f76855ad5af663ccdf598861ab",
			"0x98d98453f6017517d0114989da0938aad59a3ad9a10839c181f453283f64f5c9",
		},
	},
}

func (s *MessengerStoreNodeRequestSuite) TestFetchRealCommunity() {
	if !runLocalTests {
		return
	}

	exampleToRun := testFetchRealCommunityExample[2]

	// Test configuration
	communityID := exampleToRun.CommunityID
	communityShard := exampleToRun.CommunityShard
	fleet := exampleToRun.Fleet
	useShardAsDefaultTopic := exampleToRun.UseShardAsDefaultTopic
	clusterID := exampleToRun.ClusterID
	userPrivateKeyString := exampleToRun.UserPrivateKeyString
	ownerPublicKey := exampleToRun.OwnerPublicKey
	communityTokens := exampleToRun.CommunityTokens

	// Prepare things depending on the configuration
	nodesList := mailserversDB.DefaultMailserversByFleet(fleet)
	descriptionContentTopic := wakuV2common.BytesToTopic(transport.ToTopic(communityID))
	shardContentTopic := wakuV2common.BytesToTopic(transport.ToTopic(transport.CommunityShardInfoTopic(communityID)))

	communityIDBytes, err := types.DecodeHex(communityID)
	s.Require().NoError(err)

	// update mock - the signer for the community returned by the contracts should be owner
	for _, communityToken := range communityTokens {
		s.collectiblesServiceMock.SetSignerPubkeyForCommunity(communityIDBytes, ownerPublicKey)
		s.collectiblesServiceMock.SetMockCollectibleContractData(communityToken.ChainID, communityToken.ContractAddress,
			&communitytokens.CollectibleContractData{TotalSupply: &bigint.BigInt{}})
	}

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

			wakuLogger := s.logger.Named(fmt.Sprintf("user-waku-%d", i))
			messengerLogger := s.logger.Named(fmt.Sprintf("user-messenger-%d", i))

			cfg := testWakuV2Config{
				logger:                 wakuLogger,
				enableStore:            false,
				useShardAsDefaultTopic: useShardAsDefaultTopic,
				clusterID:              clusterID,
			}
			wakuV2 := NewTestWakuV2(&s.Suite, cfg)
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
				WithCommunityTokensService(s.collectiblesServiceMock),
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

			s.setupEnvelopesWatcher(wakuV2, &shardContentTopic, func(envelope *wakuV2common.ReceivedMessage) {
				result.ShardEnvelopes = append(result.ShardEnvelopes, envelope)
			})

			s.setupEnvelopesWatcher(wakuV2, &descriptionContentTopic, func(envelope *wakuV2common.ReceivedMessage) {
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

	// Check that results has no errors and contain correct envelopes
	for storeNodeName, result := range results {
		s.Require().NoError(result.Error)
		if exampleToRun.CheckExpectedEnvelopes {
			s.Require().Equal(exampleToRun.ExpectedShardEnvelopes, result.ShardEnvelopesHashes(),
				fmt.Sprintf("wrong shard envelopes for store node %s", storeNodeName))
			s.Require().Equal(exampleToRun.ExpectedDescriptionEnvelopes, result.EnvelopesHashes(),
				fmt.Sprintf("wrong envelopes for store node %s", storeNodeName))
		}
	}
}

func (s *MessengerStoreNodeRequestSuite) TestFetchingCommunityWithOwnerToken() {
	s.createOwner()
	s.createBob()

	s.waitForAvailableStoreNode(s.owner)
	community := s.createCommunity(s.owner)

	// owner mints owner token
	var chainID uint64 = 1
	tokenAddress := "token-address"
	tokenName := "tokenName"
	tokenSymbol := "TSM"
	_, err := s.owner.SaveCommunityToken(&token.CommunityToken{
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

	// owner adds minted owner token to community
	err = s.owner.AddCommunityToken(community.IDString(), int(chainID), tokenAddress)
	s.Require().NoError(err)

	// update mock - the signer for the community returned by the contracts should be owner
	s.collectiblesServiceMock.SetSignerPubkeyForCommunity(community.ID(), common.PubkeyToHex(&s.owner.identity.PublicKey))
	s.collectiblesServiceMock.SetMockCollectibleContractData(chainID, tokenAddress,
		&communitytokens.CollectibleContractData{TotalSupply: &bigint.BigInt{}})

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.TokenPermissions(), 1)

	s.waitForAvailableStoreNode(s.bob)

	s.fetchCommunity(s.bob, community.CommunityShard(), community)
}

func (s *MessengerStoreNodeRequestSuite) TestFetchingHistoryWhenOnline() {
	storeAddress := s.storeNodeAddress
	storePeerID := s.wakuStoreNode.PeerID().String()

	// Create messengers
	s.createOwner()
	s.createBob()

	s.logger.Debug("store node info", zap.String("peerID", s.wakuStoreNode.PeerID().String()))
	s.logger.Debug("owner node info", zap.String("peerID", gethbridge.GetGethWakuV2From(s.ownerWaku).PeerID().String()))
	s.logger.Debug("bob node info", zap.String("peerID", gethbridge.GetGethWakuV2From(s.bobWaku).PeerID().String()))

	// Connect to store node to force "online" status
	{
		WaitForPeersConnected(&s.Suite, gethbridge.GetGethWakuV2From(s.bobWaku), func() []string {
			err := s.bob.DialPeer(storeAddress)
			s.Require().NoError(err)
			return []string{storePeerID}
		})
		s.Require().True(s.bob.Online())

		// Wait for bob to fetch backup and historic messages
		time.Sleep(2 * time.Second)
	}

	// bob goes offline
	{
		WaitForConnectionStatus(&s.Suite, gethbridge.GetGethWakuV2From(s.bobWaku), func() bool {
			err := s.bob.DropPeer(storePeerID)
			s.Require().NoError(err)
			return false
		})
		s.Require().False(s.bob.Online())
	}

	// Owner sends a contact request while bob is offline
	{
		// Setup store nodes envelopes watcher
		partitionedTopic := transport.PartitionedTopic(s.bob.IdentityPublicKey())
		topic := transport.ToTopic(partitionedTopic)
		contentTopic := wakuV2common.BytesToTopic(topic)
		storeNodeSubscription := s.setupStoreNodeEnvelopesWatcher(&contentTopic)

		// Send contact request
		response, err := s.owner.SendContactRequest(context.Background(), &requests.SendContactRequest{
			ID:      s.bob.IdentityPublicKeyString(),
			Message: "1",
		})
		s.Require().NoError(err)
		s.Require().NotNil(response)
		s.Require().Len(response.Messages(), 2)

		// Ensure contact request is stored
		s.waitForEnvelopes(storeNodeSubscription, 1)
	}

	// owner goes offline to prevent message resend and any other side effects
	// to go offline we disconnect from both relay and store peers
	WaitForConnectionStatus(&s.Suite, gethbridge.GetGethWakuV2From(s.ownerWaku), func() bool {
		err := s.owner.DropPeer(storePeerID)
		s.Require().NoError(err)
		return false
	})
	s.Require().False(s.owner.Online())

	// bob goes back online, this should trigger fetching historic messages
	{
		// Enable auto request historic messages, so that when bob goes online it will fetch historic messages
		// We don't enable it earlier to control when we connect to the store node.
		s.bob.config.featureFlags.AutoRequestHistoricMessages = true

		WaitForPeersConnected(&s.Suite, gethbridge.GetGethWakuV2From(s.bobWaku), func() []string {
			err := s.bob.DialPeer(storeAddress)
			s.Require().NoError(err)
			return []string{storePeerID}
		})
		s.Require().True(s.bob.Online())

		// Don't  dial the peer, message should be fetched from store node
		response, err := WaitOnMessengerResponse(
			s.bob,
			func(r *MessengerResponse) bool {
				return len(r.Contacts) == 1
			},
			"no contact request received",
		)
		s.Require().NoError(err)
		s.Require().NotNil(response)
		s.Require().Len(response.Contacts, 1)
	}
}
