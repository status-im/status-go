package protocol

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"errors"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
)

const adminOnlyThreadCreationError = "only admins can create threads in this community"

type MessagePageWithThreadSummaries struct {
	Messages             []*common.Message `json:"messages"`
	Cursor               string            `json:"cursor"`
	Threads              []*Thread         `json:"threads"`
	ThreadSummariesError string            `json:"threadSummariesError,omitempty"`
}

// MessagesWithThreadSummaries keeps summary work bounded to the returned page.
// Summary failures must not prevent the message page from loading.
func (m *Messenger) MessagesWithThreadSummaries(chatID, cursor string, limit, participantPreviewLimit int) (*MessagePageWithThreadSummaries, error) {
	messages, nextCursor, err := m.MessageByChatID(chatID, "", cursor, limit)
	if err != nil {
		return nil, err
	}
	page := &MessagePageWithThreadSummaries{
		Messages: messages,
		Cursor:   nextCursor,
		Threads:  []*Thread{},
	}
	if len(messages) == 0 || !m.featureFlags.Threads {
		return page, nil
	}
	parentIDs := make([]string, 0, len(messages))
	for _, message := range messages {
		if message.HasThread == nil || *message.HasThread {
			parentIDs = append(parentIDs, message.ID)
		}
	}
	if len(parentIDs) == 0 {
		return page, nil
	}
	threads, err := m.ThreadSummariesByParentMessageIDs(chatID, parentIDs, participantPreviewLimit)
	if err != nil {
		page.ThreadSummariesError = err.Error()
	} else {
		page.Threads = threads
	}
	return page, nil
}

func (m *Messenger) ThreadsByChatID(chatID string) ([]*Thread, error) {
	if !m.featureFlags.Threads {
		return nil, ErrThreadFeatureDisabled
	}
	return m.persistence.ThreadsByChatID(chatID)
}

// ThreadSummariesByParentMessageIDs returns card data only for the requested
// parent messages, keeping summary work aligned with message pagination.
func (m *Messenger) ThreadSummariesByParentMessageIDs(chatID string, parentMessageIDs []string, participantPreviewLimit int) ([]*Thread, error) {
	if !m.featureFlags.Threads {
		return []*Thread{}, nil
	}
	return m.persistence.ThreadsWithSummariesByParentMessageIDs(chatID, parentMessageIDs, participantPreviewLimit)
}

func (m *Messenger) ThreadsByChatIDs(chatIDs []string) ([]*Thread, error) {
	if !m.featureFlags.Threads {
		return nil, ErrThreadFeatureDisabled
	}
	return m.persistence.ThreadsByChatIDs(chatIDs)
}

func (m *Messenger) senderCanCreateThread(chat *Chat, sender *ecdsa.PublicKey) (bool, error) {
	if !chat.SupportsThreads() {
		return false, ErrThreadsNotSupportedForChatType
	}

	if chat.ChatType != ChatTypeCommunityChat {
		return true, nil
	}

	community, err := m.communitiesManager.GetByIDString(chat.CommunityID)
	if err != nil {
		return false, err
	}

	return community.AllowsAllMembersToCreateThread() || (sender != nil && community.IsPrivilegedMember(sender)), nil
}

// CreateThread creates thread metadata for an existing parent message in a chat.
// The parent message must already be stored locally. Permission rules (admin-only
// vs all-members) are enforced for community chats.
func (m *Messenger) CreateThread(chatID string, parentMessageID string) (*MessengerResponse, error) {
	if !m.featureFlags.Threads {
		return nil, ErrThreadFeatureDisabled
	}

	if chatID == "" || parentMessageID == "" {
		return nil, errors.New("chatID and parentMessageID are required")
	}

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		return nil, ErrChatNotFoundError
	}

	// Check if thread already exists for this parent message
	_, threadErr := m.persistence.ThreadByID(chatID, parentMessageID)
	if threadErr == nil {
		return nil, errors.New("thread already exists for this message")
	}
	if !errors.Is(threadErr, common.ErrRecordNotFound) {
		return nil, threadErr
	}

	allowed, err := m.senderCanCreateThread(chat, &m.identity.PublicKey)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New(adminOnlyThreadCreationError)
	}

	parentMsg, msgErr := m.persistence.MessageByID(parentMessageID)
	if errors.Is(msgErr, sql.ErrNoRows) || errors.Is(msgErr, common.ErrRecordNotFound) {
		return nil, errors.New("parent message not found")
	}
	if msgErr != nil {
		return nil, msgErr
	}
	if parentMsg.LocalChatID != chatID {
		return nil, errors.New("parent message not found")
	}

	name := normalizeThreadName(parentMsg.Text)
	if err := m.persistence.UpsertThread(parentMessageID, chatID, parentMessageID, name); err != nil {
		return nil, err
	}

	thread, err := m.persistence.ThreadWithSummaryByID(chatID, parentMessageID)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddThread(thread)
	return response, nil
}

// StartThreadFromNewMessage sends a thread root and the supplied message as its
// first reply. The root is sent before the reply because its message ID is the
// thread ID.
func (m *Messenger) StartThreadFromNewMessage(ctx context.Context, request *requests.StartThreadFromNewMessage) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	if !m.featureFlags.Threads {
		return nil, ErrThreadFeatureDisabled
	}

	message := request.Message
	chat, ok := m.allChats.Load(message.ChatId)
	if !ok {
		return nil, ErrChatNotFoundError
	}

	allowed, err := m.senderCanCreateThread(chat, &m.identity.PublicKey)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New(adminOnlyThreadCreationError)
	}

	rootText := normalizeThreadName(request.ThreadName)
	if rootText == "" {
		rootText = normalizeThreadName(message.Text)
	}
	if rootText == "" {
		return nil, errors.New("thread name or message text is required")
	}

	root := common.NewMessage()
	root.ChatId = message.ChatId
	root.Text = rootText
	root.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	root.MessageType = message.MessageType

	rootResponse, err := m.SendChatMessage(ctx, root)
	if err != nil {
		return nil, err
	}

	rootMessages := rootResponse.Messages()
	if len(rootMessages) != 1 || rootMessages[0].ID == "" {
		return nil, errors.New("thread root message was not created")
	}

	threadID := rootMessages[0].ID
	threadResponse, err := m.CreateThread(chat.ID, threadID)
	if err != nil {
		return nil, err
	}

	message.ThreadId = &threadID
	message.ResponseTo = threadID
	replyResponse, err := m.SendChatMessage(ctx, message)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	if err := response.Merge(rootResponse); err != nil {
		return nil, err
	}
	if err := response.Merge(threadResponse); err != nil {
		return nil, err
	}
	if err := response.Merge(replyResponse); err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) addThreadsToResponse(response *MessengerResponse, messages []*common.Message) error {
	if !m.featureFlags.Threads || response == nil || len(messages) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	for _, message := range messages {
		if message == nil {
			continue
		}

		threadID := message.GetThreadId()
		if threadID == "" {
			continue
		}

		chatID := message.LocalChatID
		if chatID == "" {
			continue
		}

		key := threadKey(chatID, threadID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		thread, err := m.persistence.ThreadWithSummaryByID(chatID, threadID)
		if errors.Is(err, common.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}

		response.AddThread(thread)
	}

	return nil
}

// addAffectedThreadToResponse includes the recalculated thread summary in the
// same response as a message change, so clients do not need to refetch it.
func (m *Messenger) addAffectedThreadToResponse(response *MessengerResponse, message *common.Message) error {
	if !m.featureFlags.Threads || response == nil || message == nil {
		return nil
	}

	if threadID := message.GetThreadId(); threadID != "" {
		thread, err := m.persistence.ThreadWithSummaryByID(message.LocalChatID, threadID)
		if errors.Is(err, common.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		response.AddThread(thread)
		return nil
	}

	threads, err := m.persistence.ThreadsWithSummariesByParentMessageIDs(
		message.LocalChatID, []string{message.ID}, defaultThreadParticipantPreviewLimit)
	if err != nil {
		return err
	}
	response.AddThreads(threads)
	return nil
}
