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

func (s *MessengerCommunityMetricsSuite) prepareCommunity() *communities.Community {
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

	return response.Communities()[0]
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

func (s *MessengerCommunityMetricsSuite) TestCollectCommunityMessagesMetricsEmpty() {
	community := s.prepareCommunity()

	request := &requests.CommunityMetricsRequest{
		CommunityID:    community.ID(),
		Type:           requests.CommunityMetricsRequestMessages,
		StartTimestamp: 1690279200,
		EndTimestamp:   1690282800, // one hour
		StepTimestamp:  100,
	}

	// Expect empty metrics
	resp, err := s.m.CollectCommunityMetrics(request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// Entries count should be empty
	s.Require().Len(resp.Entries, 0)
}

func (s *MessengerCommunityMetricsSuite) TestCollectCommunityMessagesMetricsOneChat() {
	community := s.prepareCommunity()

	s.Require().Len(community.ChatIDs(), 1)
	chatId := community.ChatIDs()[0]

	s.generateMessages(chatId, string(community.ID()), []uint64{
		// out ouf range messages in the begining
		1690162000,
		1690371999,
		// 1st column, 3 message
		1690372000,
		1690372100,
		1690372200,
		// 2nd column, 2 messages
		1690372700,
		1690372800,
		// 3rd column, 1 message
		1690373000,
		// out ouf range messages in the end
		1690373100,
		1690374000,
		1690383000,
	})

	// Request metrics
	request := &requests.CommunityMetricsRequest{
		CommunityID:    community.ID(),
		Type:           requests.CommunityMetricsRequestMessages,
		StartTimestamp: 1690372000,
		EndTimestamp:   1690373000, // one hour
		StepTimestamp:  300,
	}

	resp, err := s.m.CollectCommunityMetrics(request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// floor(1000 / 300) == 3
	s.Require().Len(resp.Entries, 3)

	s.Require().Equal(resp.Entries[1690372300], uint(3))
	// No entries for 1690372600
	s.Require().Equal(resp.Entries[1690372900], uint(2))
	s.Require().Equal(resp.Entries[1690373000], uint(1))
}

func (s *MessengerCommunityMetricsSuite) TestCollectCommunityMessagesMetricsMultipleChats() {
	community := s.prepareCommunity()

	s.Require().Len(community.ChatIDs(), 1)
	chatIds := community.ChatIDs()

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
	response, err := s.m.CreateCommunityChat(community.ID(), chat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	chatIds = append(chatIds, response.Chats()[0].ID)

	s.generateMessages(chatIds[0], string(community.ID()), []uint64{
		// out ouf range messages in the begining
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

	s.generateMessages(chatIds[1], string(community.ID()), []uint64{
		// out ouf range messages in the begining
		1690152000,
		// 1st column, 2 messages
		1690372000,
		1690372100,
		// 2nd column, 1 message
		1690372700,
		// 3rd column empty
		// out ouf range messages in the end
		1690373100,
	})

	// Request metrics
	request := &requests.CommunityMetricsRequest{
		CommunityID:    community.ID(),
		Type:           requests.CommunityMetricsRequestMessages,
		StartTimestamp: 1690372000,
		EndTimestamp:   1690373000, // one hour
		StepTimestamp:  300,
	}

	resp, err := s.m.CollectCommunityMetrics(request)
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	// floor(1000 / 300) == 3
	s.Require().Len(resp.Entries, 3)

	s.Require().Equal(resp.Entries[1690372300], uint(3))
	// No entries for 1690372600
	s.Require().Equal(resp.Entries[1690372900], uint(2))
	s.Require().Equal(resp.Entries[1690373000], uint(1))
}
