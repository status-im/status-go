package protocol

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/status-im/status-go/protocol/storenodes"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/protocol/common/shard"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/tt"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"

	mailserversDB "github.com/status-im/status-go/services/mailservers"
	waku2 "github.com/status-im/status-go/wakuv2"
	wakuV2common "github.com/status-im/status-go/wakuv2/common"
)

func TestMessengerStoreNodeCommunitySuite(t *testing.T) {
	suite.Run(t, new(MessengerStoreNodeCommunitySuite))
}

type MessengerStoreNodeCommunitySuite struct {
	suite.Suite

	cancel chan struct{}

	owner     *Messenger
	ownerWaku types.Waku

	bob     *Messenger
	bobWaku types.Waku

	storeNode                 *waku2.Waku
	storeNodeAddress          string
	communityStoreNode        *waku2.Waku
	communityStoreNodeAddress string

	collectiblesServiceMock *CollectiblesServiceMock

	logger *zap.Logger
}

func (s *MessengerStoreNodeCommunitySuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	s.cancel = make(chan struct{}, 10)

	s.collectiblesServiceMock = &CollectiblesServiceMock{}

	s.storeNode, s.storeNodeAddress = s.createStore("store-1")
	s.communityStoreNode, s.communityStoreNodeAddress = s.createStore("store-community")

	s.owner, s.ownerWaku = s.newMessenger("owner", s.storeNodeAddress)
	s.bob, s.bobWaku = s.newMessenger("bob", s.storeNodeAddress)
}

func (s *MessengerStoreNodeCommunitySuite) TearDown() {
	close(s.cancel)
	if s.storeNode != nil {
		s.Require().NoError(s.storeNode.Stop())
	}
	if s.communityStoreNode != nil {
		s.Require().NoError(s.communityStoreNode.Stop())
	}
	if s.owner != nil {
		TearDownMessenger(&s.Suite, s.owner)
	}
	if s.bob != nil {
		TearDownMessenger(&s.Suite, s.bob)
	}
}

func (s *MessengerStoreNodeCommunitySuite) createStore(name string) (*waku2.Waku, string) {
	cfg := testWakuV2Config{
		logger:                 s.logger.Named(name),
		enableStore:            true,
		useShardAsDefaultTopic: false,
		clusterID:              shard.UndefinedShardValue,
	}

	storeNode := NewTestWakuV2(&s.Suite, cfg)
	addresses := storeNode.ListenAddresses()
	s.Require().GreaterOrEqual(len(addresses), 1, "no storenode listen address")
	return storeNode, addresses[0]
}

func (s *MessengerStoreNodeCommunitySuite) newMessenger(name, storenodeAddress string) (*Messenger, types.Waku) {
	localMailserverID := "local-mailserver-007"
	localFleet := "local-fleet-007"

	logger := s.logger.Named(name)
	cfg := testWakuV2Config{
		logger:                 logger,
		enableStore:            false,
		useShardAsDefaultTopic: false,
		clusterID:              shard.UndefinedShardValue,
	}
	wakuV2 := NewTestWakuV2(&s.Suite, cfg)
	wakuV2Wrapper := gethbridge.NewGethWakuV2Wrapper(wakuV2)

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
		Address: storenodeAddress,
		Fleet:   localFleet,
	})
	s.Require().NoError(err)

	options := []Option{
		WithAutoRequestHistoricMessages(false),
		WithCuratedCommunitiesUpdateLoop(false),
	}

	if storenodeAddress != "" {
		options = append(options,
			WithTestStoreNode(&s.Suite, localMailserverID, storenodeAddress, localFleet, s.collectiblesServiceMock),
		)
	}

	messenger, err := newMessengerWithKey(wakuV2Wrapper, privateKey, logger, options)

	s.Require().NoError(err)
	return messenger, wakuV2Wrapper
}

func (s *MessengerStoreNodeCommunitySuite) createCommunityWithChat(m *Messenger) (*communities.Community, *Chat) {
	WaitForAvailableStoreNode(&s.Suite, m, 500*time.Millisecond)

	storeNodeSubscription := s.setupStoreNodeEnvelopesWatcher(nil)

	createCommunityRequest := &requests.CreateCommunity{
		Name:        RandomLettersString(10),
		Description: RandomLettersString(20),
		Color:       RandomColor(),
		Tags:        RandomCommunityTags(3),
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
	}

	response, err := m.CreateCommunity(createCommunityRequest, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)
	s.Require().True(response.Communities()[0].Joined())
	s.Require().True(response.Communities()[0].IsControlNode())
	s.Require().True(response.Communities()[0].IsMemberOwner(&m.identity.PublicKey))

	s.waitForEnvelopes(storeNodeSubscription, 1)

	return response.Communities()[0], response.Chats()[0]
}

func (s *MessengerStoreNodeCommunitySuite) requireCommunitiesEqual(c *communities.Community, expected *communities.Community) {
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

func (s *MessengerStoreNodeCommunitySuite) fetchCommunity(m *Messenger, communityShard communities.CommunityShard, expectedCommunity *communities.Community) StoreNodeRequestStats {
	options := []StoreNodeRequestOption{
		WithWaitForResponseOption(true),
	}

	fetchedCommunity, stats, err := m.storeNodeRequestsManager.FetchCommunity(communityShard, options)

	s.Require().NoError(err)
	s.requireCommunitiesEqual(fetchedCommunity, expectedCommunity)

	return stats
}

func (s *MessengerStoreNodeCommunitySuite) setupEnvelopesWatcher(wakuNode *waku2.Waku, topic *wakuV2common.TopicType, cb func(envelope *wakuV2common.ReceivedMessage)) {
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

func (s *MessengerStoreNodeCommunitySuite) setupStoreNodeEnvelopesWatcher(topic *wakuV2common.TopicType) <-chan string {
	storeNodeSubscription := make(chan string, 100)
	s.setupEnvelopesWatcher(s.storeNode, topic, func(envelope *wakuV2common.ReceivedMessage) {
		storeNodeSubscription <- envelope.Hash().String()
	})
	return storeNodeSubscription
}

func (s *MessengerStoreNodeCommunitySuite) waitForEnvelopes(subscription <-chan string, expectedEnvelopesCount int) {
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

func (s *MessengerStoreNodeCommunitySuite) TestSetCommunityStorenodesAndFetch() {
	err := s.owner.DialPeer(s.storeNodeAddress)
	s.Require().NoError(err)
	err = s.bob.DialPeer(s.storeNodeAddress)
	s.Require().NoError(err)

	// Create a community
	community, _ := s.createCommunityWithChat(s.owner)

	// Set the storenode for the community
	_, err = s.owner.SetCommunityStorenodes(&requests.SetCommunityStorenodes{
		CommunityID: community.ID(),
		Storenodes: []storenodes.Storenode{
			{
				StorenodeID: "community-store-node",
				Name:        "community-store-node",
				CommunityID: community.ID(),
				Version:     2,
				Address:     s.communityStoreNodeAddress,
				Fleet:       "aaa",
			},
		},
	})
	s.Require().NoError(err)

	// Bob tetches the community
	s.fetchCommunity(s.bob, community.CommunityShard(), community)
}

func (s *MessengerStoreNodeCommunitySuite) TestSetStorenodeForCommunity_fetchMessagesFromNewStorenode() {
	s.T().Skip("flaky test")
	err := s.owner.DialPeer(s.storeNodeAddress)
	s.Require().NoError(err)
	err = s.bob.DialPeer(s.storeNodeAddress)
	s.Require().NoError(err)

	ownerPeerID := gethbridge.GetGethWakuV2From(s.ownerWaku).PeerID().String()
	bobPeerID := gethbridge.GetGethWakuV2From(s.bobWaku).PeerID().String()

	// 1. Owner creates a community
	community, chat := s.createCommunityWithChat(s.owner)

	// waits for onwer and bob to connect to the store node
	WaitForPeersConnected(&s.Suite, s.storeNode, func() []string {
		return []string{ownerPeerID, bobPeerID}
	})

	// 2. Bob joins the community
	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	joinCommunity(&s.Suite, community, s.owner, s.bob, request, "")

	// waits for onwer and bob to connect to the community store node
	WaitForPeersConnected(&s.Suite, s.communityStoreNode, func() []string {
		err := s.bob.DialPeer(s.communityStoreNodeAddress)
		s.Require().NoError(err)
		err = s.owner.DialPeer(s.communityStoreNodeAddress)
		s.Require().NoError(err)

		return []string{ownerPeerID, bobPeerID}
	})

	// 3. Owner sets the storenode for the community
	_, err = s.owner.SetCommunityStorenodes(&requests.SetCommunityStorenodes{
		CommunityID: community.ID(),
		Storenodes: []storenodes.Storenode{
			{
				StorenodeID: "community-store-node",
				Name:        "community-store-node",
				CommunityID: community.ID(),
				Version:     2,
				Address:     s.communityStoreNodeAddress,
				Fleet:       "aaa",
			},
		},
	})
	s.Require().NoError(err)

	// 5. Bob sends a message to the community chat
	inputMessage := buildTestMessage(*chat)
	_, err = s.bob.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// 6. Owner fetches the message from the new storenode
	err = s.owner.FetchMessages(&requests.FetchMessages{
		ID: chat.ID,
	})
	s.Require().NoError(err)
}
