package protocol

import (
	"fmt"
	"testing"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"

	"github.com/stretchr/testify/suite"
)

func TestMessengerCommunityMetricsSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunityMetricsSuite))
}

type MessengerCommunityMetricsSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerCommunityMetricsSuite) prepareCommunityAndChatIDs() (*communities.Community, []string) {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}
	response, err := s.m.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Communities(), 1)
	community := response.Communities()[0]

	s.Require().Len(community.ChatIDs(), 1)
	chatIDs := community.ChatIDs()

	// Create another chat
	chat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status",
			Emoji:       "👍",
			Description: "status community chat",
		},
	}
	response, err = s.m.CreateCommunityChat(community.ID(), chat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	chatIDs = append(chatIDs, response.Chats()[0].ID)

	return community, chatIDs
}

func (s *MessengerCommunityMetricsSuite) prepareCommunityChatMessages(communityID string, chatIDs []string) {
	s.generateMessages(chatIDs[0], communityID, []uint64{
		// out ouf range messages in the beginning
		1690162000,
		// 1st column, 1 message
		1690372200,
		// 2nd column, 1 message
		1690372800,
		// 3rd column, 1 message
		1690373000,
		// out ouf range messages in the end
		1690373100,
	})

	s.generateMessages(chatIDs[1], communityID, []uint64{
		// out ouf range messages in the beginning
		1690151000,
		// 1st column, 2 messages
		1690372000,
		1690372100,
		// 2nd column, 1 message
		1690372700,
		// 3rd column empty
		// out ouf range messages in the end
		1690373100,
	})
}

func (s *MessengerCommunityMetricsSuite) generateMessages(chatID string, communityID string, timestamps []uint64) {
	var messages []*common.Message
	for i, timestamp := range timestamps {
		message := &common.Message{
			ChatMessage: protobuf.ChatMessage{
				ChatId:      chatID,
				Text:        fmt.Sprintf("Test message %d", i),
				MessageType: protobuf.MessageType_ONE_TO_ONE,
				// NOTE: should we filter content types for messages metrics
				Clock:     timestamp,
				Timestamp: timestamp,
			},
			WhisperTimestamp: timestamp,
			From:             common.PubkeyToHex(&s.m.identity.PublicKey),
			LocalChatID:      chatID,
			CommunityID:      communityID,
			ID:               types.EncodeHex(crypto.Keccak256([]byte(fmt.Sprintf("%s%s%d", chatID, communityID, timestamp)))),
		}

		err := message.PrepareContent(common.PubkeyToHex(&s.m.identity.PublicKey))
		s.Require().NoError(err)

		messages = append(messages, message)
	}
	err := s.m.persistence.SaveMessages(messages)
	s.Require().NoError(err)
}

func (s *MessengerCommunityMetricsSuite) TestCollectCommunityMetricsInvalidRequest() {
	community, _ := s.prepareCommunityAndChatIDs()

	request := &requests.CommunityMetricsRequest{
		CommunityID: community.ID(),
		Type:        requests.CommunityMetricsRequestMessagesTimestamps,
		Intervals: []requests.MetricsIntervalRequest{
			requests.MetricsIntervalRequest{
				StartTimestamp: 1690372400,
				EndTimestamp:   1690371800,
			},
			requests.MetricsIntervalRequest{
				StartTimestamp: 1690371900,
				EndTimestamp:   1690373000,
			},
		},
	}

	// Expect error
	_, err := s.m.CollectCommunityMetrics(request)
	s.Require().Error(err)
	s.Require().Equal(err, requests.ErrInvalidTimestampIntervals)
}

func (s *MessengerCommunityMetricsSuite) TestCollectCommunityMetricsEmptyInterval() {
	community, _ := s.prepareCommunityAndChatIDs()

	request := &requests.CommunityMetricsRequest{
		CommunityID: community.ID(),
		Type:        requests.CommunityMetricsRequestMessagesTimestamps,
	}

	// Expect empty metrics
	resp, err := s.m.CollectCommunityMetrics(request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Entries count should be empty
	s.Require().Len(resp.Intervals, 0)
}

func (s *MessengerCommunityMetricsSuite) TestCollectCommunityMessagesTimestamps() {
	community, chatIDs := s.prepareCommunityAndChatIDs()

	s.prepareCommunityChatMessages(string(community.ID()), chatIDs)

	// Request metrics
	request := &requests.CommunityMetricsRequest{
		CommunityID: community.ID(),
		Type:        requests.CommunityMetricsRequestMessagesTimestamps,
		Intervals: []requests.MetricsIntervalRequest{
			requests.MetricsIntervalRequest{
				StartTimestamp: 1690372000,
				EndTimestamp:   1690372300,
			},
			requests.MetricsIntervalRequest{
				StartTimestamp: 1690372400,
				EndTimestamp:   1690372800,
			},
			requests.MetricsIntervalRequest{
				StartTimestamp: 1690372900,
				EndTimestamp:   1690373000,
			},
		},
	}

	resp, err := s.m.CollectCommunityMetrics(request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().Len(resp.Intervals, 3)

	s.Require().Equal(resp.Intervals[0].Timestamps, []uint64{1690372000, 1690372100, 1690372200})
	s.Require().Equal(resp.Intervals[1].Timestamps, []uint64{1690372700, 1690372800})
	s.Require().Equal(resp.Intervals[2].Timestamps, []uint64{1690373000})
}

func (s *MessengerCommunityMetricsSuite) TestCollectCommunityMessagesCount() {
	community, chatIDs := s.prepareCommunityAndChatIDs()

	s.prepareCommunityChatMessages(string(community.ID()), chatIDs)

	// Request metrics
	request := &requests.CommunityMetricsRequest{
		CommunityID: community.ID(),
		Type:        requests.CommunityMetricsRequestMessagesCount,
		Intervals: []requests.MetricsIntervalRequest{
			requests.MetricsIntervalRequest{
				StartTimestamp: 1690372000,
				EndTimestamp:   1690372300,
			},
			requests.MetricsIntervalRequest{
				StartTimestamp: 1690372400,
				EndTimestamp:   1690372800,
			},
			requests.MetricsIntervalRequest{
				StartTimestamp: 1690372900,
				EndTimestamp:   1690373000,
			},
		},
	}

	resp, err := s.m.CollectCommunityMetrics(request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	s.Require().Len(resp.Intervals, 3)

	s.Require().Equal(resp.Intervals[0].Count, 3)
	s.Require().Equal(resp.Intervals[1].Count, 2)
	s.Require().Equal(resp.Intervals[2].Count, 1)
}
