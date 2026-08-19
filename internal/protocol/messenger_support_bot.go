package protocol

import (
	"os"
	"strings"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/protocol/requests"
)

const (
	supportBotNewUserMessage = "->"
)

var supportBotChatKey = os.Getenv("STATUS_SUPPORT_BOT_CHAT_KEY")

func (m *Messenger) sendSupportBotContactRequest() error {
	if m.shouldAbortSupportBotContactRequest() {
		return nil
	}

	state, err := m.settings.SupportBotContactRequestState()
	if err != nil {
		return err
	}
	if state == settings.SupportBotContactRequestStateDone {
		return nil
	}
	if state == settings.SupportBotContactRequestStatePendingExisting {
		return m.markSupportBotContactRequestDone()
	}

	chatID, err := requests.ConvertCompressedToLegacyKey(m.config.supportBotChatKey)
	if err != nil {
		return err
	}
	if strings.EqualFold(chatID, m.myHexIdentity()) {
		return m.markSupportBotContactRequestDone()
	}
	if contact, ok := m.allContacts.Load(chatID); ok && contact.Added() {
		return m.markSupportBotContactRequestDone()
	}

	response, err := m.SendContactRequest(m.ctx, &requests.SendContactRequest{
		ID:      chatID,
		Message: supportBotNewUserMessage,
	})
	if err != nil {
		return err
	}
	if err := m.markSupportBotContactRequestDone(); err != nil {
		return err
	}

	m.PublishMessengerResponse(response)
	return nil
}

func (m *Messenger) shouldAbortSupportBotContactRequest() bool {
	select {
	case <-m.quit:
		return true
	case <-m.ctx.Done():
		return true
	default:
		return false
	}
}

func (m *Messenger) markSupportBotContactRequestDone() error {
	return m.settings.SaveSettingField(settings.SupportBotContactRequestState, settings.SupportBotContactRequestStateDone)
}
