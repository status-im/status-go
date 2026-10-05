package protocol

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func seedThreadBackup(t *testing.T, m *Messenger) []*protobuf.BackedUpThread {
	t.Helper()
	peer, err := crypto.GenerateKey()
	require.NoError(t, err)
	chat := CreateOneToOneChat("thread-backup", &peer.PublicKey, m.getTimesource())
	require.NoError(t, m.SaveChat(chat))

	threadID := "backed-up-thread"
	require.NoError(t, m.SaveMessages([]*common.Message{
		{
			ID: "backed-up-parent", LocalChatID: chat.ID, From: m.selfContact.ID,
			ChatMessage: &protobuf.ChatMessage{ChatId: chat.ID, Text: "Parent", Clock: 10, ContentType: protobuf.ChatMessage_TEXT_PLAIN, MessageType: protobuf.MessageType_ONE_TO_ONE},
		},
		{
			ID: "backed-up-reply", LocalChatID: chat.ID, From: m.selfContact.ID,
			ChatMessage: &protobuf.ChatMessage{ChatId: chat.ID, Text: "Reply", Clock: 20, ResponseTo: "backed-up-parent", ThreadId: &threadID, ContentType: protobuf.ChatMessage_TEXT_PLAIN, MessageType: protobuf.MessageType_ONE_TO_ONE},
		},
	}))
	threads := []*protobuf.BackedUpThread{
		{ThreadId: threadID, ChatId: chat.ID, ParentMessageId: "backed-up-parent", Name: "Thread name", ReadMessagesAtClockValue: 25},
		{ThreadId: "empty-thread", ChatId: chat.ID, ParentMessageId: "absent-parent", Name: "Empty thread"},
		{ThreadId: "placeholder-thread", ChatId: chat.ID, ParentMessageId: "placeholder-thread", ReadMessagesAtClockValue: 30},
	}
	require.NoError(t, m.persistence.SaveBackedUpThreads(threads))
	return threads
}

func assertThreadBackupRestored(t *testing.T, m *Messenger, expected []*protobuf.BackedUpThread) {
	t.Helper()
	actual, err := m.persistence.AllThreadsForBackup()
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	reply, err := m.persistence.MessageByID("backed-up-reply")
	require.NoError(t, err)
	require.Equal(t, expected[0].ThreadId, reply.GetThreadId())
	require.Equal(t, "backed-up-parent", reply.ResponseTo)
	parent, err := m.persistence.MessageByID("backed-up-parent")
	require.NoError(t, err)
	require.Empty(t, parent.GetThreadId())
}

func (s *MessengerLocalBackupSuite) TestLocalBackupThreads() {
	for _, enabled := range []bool{false, true} {
		s.Run(fmt.Sprintf("feature-enabled-%t", enabled), func() {
			source, receiver := s.anotherMessenger(), s.anotherMessenger()
			source.featureFlags.Threads = enabled
			receiver.featureFlags.Threads = enabled
			threads := seedThreadBackup(s.T(), source)

			s.Require().NoError(source.settings.SaveSetting(settings.MessagesBackupEnabled.GetReactName(), false))
			data, err := source.ExportBackup()
			s.Require().NoError(err)
			var backup protobuf.MessengerLocalBackup
			s.Require().NoError(proto.Unmarshal(data, &backup))
			s.Require().Empty(backup.Messages)
			s.Require().Empty(backup.Threads)

			s.Require().NoError(source.settings.SaveSetting(settings.MessagesBackupEnabled.GetReactName(), true))
			data, err = source.ExportBackup()
			s.Require().NoError(err)
			s.Require().NoError(proto.Unmarshal(data, &backup))
			s.Require().Len(backup.Messages, 2)
			s.Require().Equal(threads, backup.Threads)

			s.Require().NoError(receiver.settings.SaveSetting(settings.MessagesBackupEnabled.GetReactName(), false))
			s.Require().NoError(receiver.ImportBackup(data))
			s.Require().NoError(receiver.ImportBackup(data))
			assertThreadBackupRestored(s.T(), receiver, threads)

			response := &MessengerResponse{}
			s.Require().NoError(receiver.addBackedUpThreadsToResponse(response, threads))
			if enabled {
				s.Require().Len(response.Threads(), 3)
				for _, thread := range response.Threads() {
					if thread.ThreadID == threads[0].ThreadId {
						s.Require().EqualValues(2, thread.MessagesCount)
						s.Require().NotNil(thread.LastMessage)
						s.Require().Equal("Reply", thread.LastMessage.Text)
					}
				}
			} else {
				s.Require().Empty(response.Threads())
			}
		})
	}
}

func (s *MessengerLocalBackupSuite) TestLocalBackupMetadataOnlyThreads() {
	s.m.featureFlags.Threads = false
	threads := []*protobuf.BackedUpThread{{
		ThreadId: "empty-thread", ChatId: "missing-chat", ParentMessageId: "missing-parent", Name: "Empty", ReadMessagesAtClockValue: 12,
	}}
	s.Require().NoError(s.m.persistence.SaveBackedUpThreads(threads))
	s.Require().NoError(s.m.settings.SaveSetting(settings.MessagesBackupEnabled.GetReactName(), true))
	data, err := s.m.ExportBackup()
	s.Require().NoError(err)
	var backup protobuf.MessengerLocalBackup
	s.Require().NoError(proto.Unmarshal(data, &backup))
	s.Require().Empty(backup.Messages)
	s.Require().Equal(threads, backup.Threads)
	receiver := s.anotherMessenger()
	s.Require().NoError(receiver.ImportBackup(data))
	actual, err := receiver.persistence.AllThreadsForBackup()
	s.Require().NoError(err)
	s.Require().Equal(threads, actual)
}

func (s *MessengerLocalBackupSuite) TestLocalBackupLegacyMessages() {
	data, err := proto.Marshal(&protobuf.MessengerLocalBackup{Messages: []*protobuf.BackedUpMessage{{
		Id: "legacy", ChatId: "legacy-chat", From: s.m.selfContact.ID, Text: "Legacy message", ResponseTo: "ordinary-parent",
	}}})
	s.Require().NoError(err)
	s.Require().NoError(s.m.ImportBackup(data))
	message, err := s.m.persistence.MessageByID("legacy")
	s.Require().NoError(err)
	s.Require().Empty(message.GetThreadId())
	s.Require().Equal("ordinary-parent", message.ResponseTo)
	threads, err := s.m.persistence.AllThreadsForBackup()
	s.Require().NoError(err)
	s.Require().Empty(threads)
}

func (s *MessengerPairingSuite) TestLocalPairingThreads() {
	for _, enabled := range []bool{false, true} {
		s.Run(fmt.Sprintf("feature-enabled-%t", enabled), func() {
			source, receiver := s.anotherMessenger(), s.anotherMessenger()
			source.featureFlags.Threads = enabled
			receiver.featureFlags.Threads = enabled
			threads := seedThreadBackup(s.T(), source)
			s.Require().NoError(source.settings.SaveSetting(settings.MessagesBackupEnabled.GetReactName(), false))

			var rawMessages []*protobuf.RawMessage
			collector := func(_ context.Context, raw common.RawMessage) (common.RawMessage, error) {
				rawMessages = append(rawMessages, &protobuf.RawMessage{Payload: raw.Payload, MessageType: raw.MessageType})
				return raw, nil
			}
			s.Require().NoError(source.SyncDevices(context.Background(), "", "", false, collector))
			for _, raw := range rawMessages {
				s.Require().NotEqual(protobuf.ApplicationMetadataMessage_BACKED_UP_MESSAGE_BATCH, raw.MessageType)
			}

			rawMessages = nil
			s.Require().NoError(source.SyncDevices(context.Background(), "", "", true, collector))
			s.Require().NoError(receiver.HandleSyncRawMessages(rawMessages))
			s.Require().NoError(receiver.HandleSyncRawMessages(rawMessages))
			assertThreadBackupRestored(s.T(), receiver, threads)
		})
	}
}

func (s *MessengerPairingSuite) TestLocalPairingThreadBatchBoundaries() {
	for _, count := range []int{0, 100, 101} {
		s.Run(fmt.Sprintf("threads-%d", count), func() {
			source, receiver := s.anotherMessenger(), s.anotherMessenger()
			source.featureFlags.Threads = false
			receiver.featureFlags.Threads = true
			threads := make([]*protobuf.BackedUpThread, count)
			for i := range threads {
				id := fmt.Sprintf("thread-%03d", i)
				threads[i] = &protobuf.BackedUpThread{ThreadId: id, ChatId: "chat", ParentMessageId: id, Name: id}
			}
			s.Require().NoError(source.persistence.SaveBackedUpThreads(threads))
			var rawMessages []*protobuf.RawMessage
			s.Require().NoError(source.syncMessages(context.Background(), func(_ context.Context, raw common.RawMessage) (common.RawMessage, error) {
				var batch protobuf.BackedUpMessageBatch
				s.Require().NoError(proto.Unmarshal(raw.Payload, &batch))
				s.Require().Empty(batch.Messages)
				s.Require().NotEmpty(batch.Threads)
				s.Require().LessOrEqual(len(batch.Threads), 100)
				rawMessages = append(rawMessages, &protobuf.RawMessage{Payload: raw.Payload, MessageType: raw.MessageType})
				return raw, nil
			}))
			s.Require().Len(rawMessages, (count+99)/100)
			s.Require().NoError(receiver.HandleSyncRawMessages(rawMessages))
			actual, err := receiver.persistence.AllThreadsForBackup()
			s.Require().NoError(err)
			s.Require().Equal(threads, actual)
		})
	}
}

func (s *MessengerPairingSuite) TestNetworkThreadBackupBatchIgnored() {
	s.m.featureFlags.Threads = true
	state := s.m.buildMessageState()
	s.Require().NoError(s.m.HandleBackedUpMessageBatch(context.Background(), state, &protobuf.BackedUpMessageBatch{
		Threads:  []*protobuf.BackedUpThread{{ThreadId: "thread", ChatId: "chat", ParentMessageId: "parent", Name: "Thread"}},
		Messages: []*protobuf.BackedUpMessage{{Id: "reply", ChatId: "chat", ThreadId: "thread"}},
	}, nil))
	threads, err := s.m.persistence.AllThreadsForBackup()
	s.Require().NoError(err)
	s.Require().Empty(threads)
	_, err = s.m.persistence.MessageByID("reply")
	s.Require().ErrorIs(err, common.ErrRecordNotFound)
}
