package protocol

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/common/shard"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
)

func TestMessengerCommunitiesShardingSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunitiesShardingSuite))
}

type MessengerCommunitiesShardingSuite struct {
	suite.Suite

	owner     *Messenger
	ownerWaku types.Waku

	alice                         *Messenger
	aliceWaku                     types.Waku
	aliceUnhandledMessagesTracker *unhandledMessagesTracker

	logger *zap.Logger
}

func (s *MessengerCommunitiesShardingSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	wakuNodes := CreateWakuV2Network(&s.Suite, s.logger, true, []string{"owner", "alice"})

	nodeConfig := defaultTestCommunitiesMessengerNodeConfig()
	nodeConfig.WakuV2Config.UseShardAsDefaultTopic = true

	s.ownerWaku = wakuNodes[0]
	s.owner = newTestCommunitiesMessenger(&s.Suite, s.ownerWaku, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			name:   "owner",
			logger: s.logger,
		},
		nodeConfig: nodeConfig,
	})

	s.aliceUnhandledMessagesTracker = &unhandledMessagesTracker{
		messages: map[protobuf.ApplicationMetadataMessage_Type][]*unhandedMessage{},
	}
	s.aliceWaku = wakuNodes[1]
	s.alice = newTestCommunitiesMessenger(&s.Suite, s.aliceWaku, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			name:                     "alice",
			logger:                   s.logger,
			unhandledMessagesTracker: s.aliceUnhandledMessagesTracker,
		},
		nodeConfig: nodeConfig,
	})

	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesShardingSuite) TearDownTest() {
	if s.owner != nil {
		TearDownMessenger(&s.Suite, s.owner)
	}
	if s.ownerWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.ownerWaku).Stop())
	}
	if s.alice != nil {
		TearDownMessenger(&s.Suite, s.alice)
	}
	if s.aliceWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.aliceWaku).Stop())
	}
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesShardingSuite) testPostToCommunityChat(shard *shard.Shard, community *communities.Community, chat *Chat) {
	_, err := s.owner.SetCommunityShard(&requests.SetCommunityShard{
		CommunityID: community.ID(),
		Shard:       shard,
	})
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.alice, func(mr *MessengerResponse) bool {
		if len(mr.communities) == 0 {
			return false
		}
		if shard == nil {
			return mr.Communities()[0].Shard() == nil
		}
		return mr.Communities()[0].Shard() != nil && mr.Communities()[0].Shard().Index == shard.Index
	}, "shard info not updated")
	s.Require().NoError(err)

	message := buildTestMessage(*chat)
	_, err = s.owner.SendChatMessage(context.Background(), message)
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.alice, func(mr *MessengerResponse) bool {
		return len(mr.messages) > 0 && mr.Messages()[0].ID == message.ID
	}, "message not received")
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesShardingSuite) TestPostToCommunityChat() {
	community, chat := createCommunity(&s.Suite, s.owner)

	advertiseCommunityToUserOldWay(&s.Suite, community, s.owner, s.alice)
	joinCommunity(&s.Suite, community, s.owner, s.alice, &requests.RequestToJoinCommunity{CommunityID: community.ID()}, "")

	// Members should be able to receive messages in a community with sharding enabled.
	{
		shard := &shard.Shard{
			Cluster: shard.MainStatusShardCluster,
			Index:   128,
		}
		s.testPostToCommunityChat(shard, community, chat)
	}

	// Members should be able to receive messages in a community where the sharding configuration has been edited.
	{
		shard := &shard.Shard{
			Cluster: shard.MainStatusShardCluster,
			Index:   256,
		}
		s.testPostToCommunityChat(shard, community, chat)
	}

	// Members should continue to receive messages in a community if sharding is disabled after it was previously enabled.
	{
		s.testPostToCommunityChat(nil, community, chat)
	}
}

func (s *MessengerCommunitiesShardingSuite) TestIgnoreOutdatedShardKey() {
	s.T().Skip("flaky test")

	community, _ := createCommunity(&s.Suite, s.owner)

	advertiseCommunityToUserOldWay(&s.Suite, community, s.owner, s.alice)
	joinCommunity(&s.Suite, community, s.owner, s.alice, &requests.RequestToJoinCommunity{CommunityID: community.ID()}, "")

	shard := &shard.Shard{
		Cluster: shard.MainStatusShardCluster,
		Index:   128,
	}

	// Members should receive shard update.
	{
		response, err := s.owner.SetCommunityShard(&requests.SetCommunityShard{
			CommunityID: community.ID(),
			Shard:       shard,
		})
		s.Require().NoError(err)
		s.Require().Len(response.Communities(), 1)
		community = response.Communities()[0]

		_, err = WaitOnMessengerResponse(s.alice, func(mr *MessengerResponse) bool {
			return len(mr.communities) > 0 && mr.Communities()[0].Shard() != nil && mr.Communities()[0].Shard().Index == shard.Index
		}, "shard info not updated")
		s.Require().NoError(err)
	}

	// Members should ignore outdated shard update.
	{
		// Simulate outdated CommunityShardKey message.
		shard.Index = 256
		communityShardKey := &protobuf.CommunityShardKey{
			Clock:       community.Clock() - 1, // simulate outdated clock
			CommunityId: community.ID(),
			Shard:       shard.Protobuffer(),
		}

		encodedMessage, err := proto.Marshal(communityShardKey)
		s.Require().NoError(err)

		rawMessage := common.RawMessage{
			Recipients:          []*ecdsa.PublicKey{&s.alice.identity.PublicKey},
			ResendAutomatically: true,
			MessageType:         protobuf.ApplicationMetadataMessage_COMMUNITY_SHARD_KEY,
			Payload:             encodedMessage,
		}

		_, err = s.owner.sender.SendPubsubTopicKey(context.Background(), &rawMessage)
		s.Require().NoError(err)

		_, err = WaitOnMessengerResponse(s.alice, func(mr *MessengerResponse) bool {
			msgType := protobuf.ApplicationMetadataMessage_COMMUNITY_SHARD_KEY
			msgs, exists := s.aliceUnhandledMessagesTracker.messages[msgType]
			if !exists {
				return false
			}

			for _, msg := range msgs {
				p := &protobuf.CommunityShardKey{}
				err := proto.Unmarshal(msg.ApplicationLayer.Payload, p)
				if err != nil {
					panic(err)
				}

				if msg.err == communities.ErrOldShardInfo && p.Shard != nil && p.Shard.Index == int32(shard.Index) {
					return true
				}
			}

			return false
		}, "shard info with outdated clock either not received or not ignored")
		s.Require().NoError(err)
	}
}
