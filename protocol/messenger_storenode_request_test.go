package protocol

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/status-im/status-go/multiaccounts/accounts"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/protocol/common"
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

	mailserversDB "github.com/status-im/status-go/services/mailservers"
	waku2 "github.com/status-im/status-go/wakuv2"
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

func (s *MessengerStoreNodeRequestSuite) SetupTest() {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	cfg.Development = false
	cfg.DisableStacktrace = true
	s.logger = tt.MustCreateTestLoggerWithConfig(cfg)

	s.cancel = make(chan struct{}, 10)

	storeNodeLogger := s.logger.Named("store-node-waku")
	s.wakuStoreNode = NewWakuV2(&s.Suite, storeNodeLogger, true, true)

	storeNodeListenAddresses := s.wakuStoreNode.ListenAddresses()
	s.Require().LessOrEqual(1, len(storeNodeListenAddresses))

	s.storeNodeAddress = storeNodeListenAddresses[0]
	s.logger.Info("store node ready", zap.String("address", s.storeNodeAddress))
}

func (s *MessengerStoreNodeRequestSuite) TearDown() {
	close(s.cancel)
	s.wakuStoreNode.Stop() // nolint: errcheck
	s.owner.Shutdown()     // nolint: errcheck
	s.bob.Shutdown()       // nolint: errcheck
}

func (s *MessengerStoreNodeRequestSuite) createOwner() {
	wakuLogger := s.logger.Named("owner-waku-node")
	wakuV2 := NewWakuV2(&s.Suite, wakuLogger, true, false)
	s.ownerWaku = gethbridge.NewGethWakuV2Wrapper(wakuV2)

	messengerLogger := s.logger.Named("owner-messenger")
	s.owner = s.newMessenger(s.ownerWaku, messengerLogger, s.storeNodeAddress)

	// We force the owner to use the store node as relay peer
	err := s.owner.DialPeer(s.storeNodeAddress)
	s.Require().NoError(err)
}

func (s *MessengerStoreNodeRequestSuite) createBob() {
	wakuLogger := s.logger.Named("bob-waku-node")
	wakuV2 := NewWakuV2(&s.Suite, wakuLogger, true, false)
	s.bobWaku = gethbridge.NewGethWakuV2Wrapper(wakuV2)

	messengerLogger := s.logger.Named("bob-messenger")
	s.bob = s.newMessenger(s.bobWaku, messengerLogger, s.storeNodeAddress)
}

func (s *MessengerStoreNodeRequestSuite) newMessenger(shh types.Waku, logger *zap.Logger, mailserverAddress string) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	mailserversSQLDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
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

	createCommunityRequest := &requests.CreateCommunity{
		Name:        RandomLettersString(10),
		Description: RandomLettersString(20),
		Color:       RandomColor(),
		Tags:        RandomCommunityTags(3),
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	response, err := s.owner.CreateCommunity(createCommunityRequest, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	return response.Communities()[0]
}

func (s *MessengerStoreNodeRequestSuite) requireCommunitiesEqual(c *communities.Community, expected *communities.Community) {
	s.Require().Equal(expected.Name(), c.Name())
	s.Require().Equal(expected.Identity().Description, c.Identity().Description)
	s.Require().Equal(expected.Color(), c.Color())
	s.Require().Equal(expected.Tags(), c.Tags())
}

func (s *MessengerStoreNodeRequestSuite) requireContactsEqual(c *Contact, expected *Contact) {
	s.Require().Equal(expected.DisplayName, c.DisplayName)
	s.Require().Equal(expected.Bio, c.Bio)
	s.Require().Equal(expected.SocialLinks, c.SocialLinks)
}

func (s *MessengerStoreNodeRequestSuite) fetchCommunity(m *Messenger, communityShard communities.CommunityShard, expectedCommunityInfo *communities.Community) StoreNodeRequestStats {
	fetchedCommunity, stats, err := m.storeNodeRequestsManager.FetchCommunity(communityShard, true)

	s.Require().NoError(err)
	s.Require().NotNil(fetchedCommunity)
	s.Require().Equal(communityShard.CommunityID, fetchedCommunity.IDString())

	if expectedCommunityInfo != nil {
		s.requireCommunitiesEqual(fetchedCommunity, expectedCommunityInfo)
	}

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

func (s *MessengerStoreNodeRequestSuite) TestRequestCommunityInfo() {
	s.createOwner()
	s.createBob()

	s.waitForAvailableStoreNode(s.owner)
	community := s.createCommunity(s.owner)

	s.waitForAvailableStoreNode(s.bob)
	s.fetchCommunity(s.bob, community.CommunityShard(), community)
}

func (s *MessengerStoreNodeRequestSuite) TestConsecutiveRequests() {
	s.createOwner()
	s.createBob()

	s.waitForAvailableStoreNode(s.owner)
	community := s.createCommunity(s.owner)

	// Test consecutive requests to check that requests in manager are finalized
	for i := 0; i < 2; i++ {
		s.waitForAvailableStoreNode(s.bob)
		s.fetchCommunity(s.bob, community.CommunityShard(), community)
	}
}

func (s *MessengerStoreNodeRequestSuite) TestSimultaneousCommunityInfoRequests() {
	s.createOwner()
	s.createBob()

	s.waitForAvailableStoreNode(s.owner)
	community := s.createCommunity(s.owner)

	storeNodeRequestsCount := 0
	s.bob.storeNodeRequestsManager.onPerformingBatch = func(batch MailserverBatch) {
		storeNodeRequestsCount++
	}

	s.waitForAvailableStoreNode(s.bob)

	wg := sync.WaitGroup{}

	// Make 2 simultaneous fetch requests, only 1 request to store node is expected
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.fetchCommunity(s.bob, community.CommunityShard(), community)
		}()
	}

	wg.Wait()
	s.Require().Equal(1, storeNodeRequestsCount)
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

	s.waitForAvailableStoreNode(s.owner)
	community := s.createCommunity(s.owner)

	// WaitForAvailableStoreNode is done internally
	s.fetchCommunity(s.bob, community.CommunityShard(), community)
}

// This test is intended to only run locally to test how fast is a big community fetched
// Shouldn't be executed in CI, because it relies on connection to status.prod store nodes.
func (s *MessengerStoreNodeRequestSuite) TestRequestBigCommunity() {
	if !runLocalTests {
		return
	}

	// Status CC community
	const communityID = "0x03073514d4c14a7d10ae9fc9b0f05abc904d84166a6ac80add58bf6a3542a4e50a"

	communityShard := communities.CommunityShard{
		CommunityID: communityID,
		Shard:       nil,
	}

	wakuLogger := s.logger.Named("user-waku-node")
	messengerLogger := s.logger.Named("user-messenger")

	wakuV2 := NewWakuV2(&s.Suite, wakuLogger, true, false)
	userWaku := gethbridge.NewGethWakuV2Wrapper(wakuV2)

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	mailserversSQLDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)

	mailserversDatabase := mailserversDB.NewDB(mailserversSQLDb)
	s.Require().NoError(err)

	options := []Option{
		WithMailserversDatabase(mailserversDatabase),
		WithClusterConfig(params.ClusterConfig{
			Fleet: params.FleetStatusProd,
		}),
	}

	// Create bob without `createBob` without func to force status.prod fleet
	s.bob, err = newMessengerWithKey(userWaku, privateKey, messengerLogger, options)
	s.Require().NoError(err)

	fetchedCommunity, stats, err := s.bob.storeNodeRequestsManager.FetchCommunity(communityShard, true)

	s.Require().NoError(err)
	s.Require().NotNil(fetchedCommunity)
	s.Require().Equal(communityID, fetchedCommunity.IDString())

	s.Require().Equal(initialStoreNodeRequestPageSize, stats.FetchedEnvelopesCount)
	s.Require().Equal(1, stats.FetchedPagesCount)
}

func (s *MessengerStoreNodeRequestSuite) TestRequestCommunityPagingAlgorithm() {
	s.createOwner()
	s.createBob()

	// Create a community
	s.waitForAvailableStoreNode(s.owner)
	community := s.createCommunity(s.owner)

	// Push spam to the same ContentTopic & PubsubTopic
	// The first requested page size is 1. All subsequent pages are limited to 20.
	// We want to test the algorithm, so we push 21 spam envelopes.
	for i := 0; i < defaultStoreNodeRequestPageSize+initialStoreNodeRequestPageSize; i++ {
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

	// Fetch the community
	stats := s.fetchCommunity(s.bob, community.CommunityShard(), community)

	// Expect 3 pages and 23 (24 spam + 1 community description + 1 general channel description) envelopes to be fetched.
	// First we fetch a more up-to-date, but an invalid spam message, fail to decrypt it as community description,
	// then we fetch another page of data and successfully decrypt a community description.
	s.Require().Equal(defaultStoreNodeRequestPageSize+initialStoreNodeRequestPageSize+2, stats.FetchedEnvelopesCount)
	s.Require().Equal(3, stats.FetchedPagesCount)
}

func (s *MessengerStoreNodeRequestSuite) TestRequestCommunityWithSameContentTopic() {
	s.createOwner()
	s.createBob()

	// Create a community
	s.waitForAvailableStoreNode(s.owner)
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
	s.waitForAvailableStoreNode(s.owner)
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
	s.waitForAvailableStoreNode(s.owner)
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
