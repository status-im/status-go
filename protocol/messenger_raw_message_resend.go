package protocol

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"
)

// watchExpiredMessages regularly checks for expired emojis and invoke their resending
func (m *Messenger) watchExpiredMessages() {
	m.logger.Debug("watching expired messages")
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				if m.Online() {
					err := m.resendExpiredMessages()
					if err != nil {
						m.logger.Debug("failed to resend expired message", zap.Error(err))
					}
				}
			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) resendExpiredMessages() error {
	if m.connectionState.Offline {
		return errors.New("offline")
	}

	ids, err := m.persistence.ExpiredMessagesIDs(m.config.messageResendMaxCount)
	if err != nil {
		return errors.Wrapf(err, "Can't get expired reactions from db")
	}

	for _, id := range ids {
		if err := m.processMessageID(id); err != nil {
			m.logger.Error("Error processing message ID when trying resend raw message", zap.String("id", id), zap.Error(err))
		}
	}
	return nil
}

func (m *Messenger) processMessageID(id string) error {
	rawMessage, err := m.persistence.RawMessageByID(id)
	if err != nil {
		return errors.Wrap(err, "Can't get raw message by ID")
	}

	switch rawMessage.ResendMethod {
	case common.ResendMethodSendCommunityMessage:
		return m.handleSendCommunityMessage(rawMessage)
	case common.ResendMethodSendPrivate:
		return m.handleSendPrivateMessage(rawMessage)
	default:
		return m.handleOtherResendMethods(rawMessage)
	}
}

func (m *Messenger) handleSendCommunityMessage(rawMessage *common.RawMessage) error {
	_, err := m.sender.SendCommunityMessage(context.TODO(), rawMessage)
	if err != nil {
		err = errors.Wrap(err, "Can't resend message with SendCommunityMessage")
	}
	m.upsertRawMessageToWatch(rawMessage)
	return err
}

func (m *Messenger) handleSendPrivateMessage(rawMessage *common.RawMessage) error {
	if len(rawMessage.Recipients) == 0 {
		m.logger.Error("No recipients to resend message", zap.String("id", rawMessage.ID))
		m.upsertRawMessageToWatch(rawMessage)
		return errors.New("No recipients to resend message with SendPrivate")
	}

	_, err := m.sender.SendPrivate(context.TODO(), rawMessage.Recipients[0], rawMessage)
	if err != nil {
		err = errors.Wrap(err, "Can't resend message with SendPrivate")
	}
	m.upsertRawMessageToWatch(rawMessage)
	return err
}

func (m *Messenger) handleOtherResendMethods(rawMessage *common.RawMessage) error {
	chat, ok := m.allChats.Load(rawMessage.LocalChatID)
	if !ok {
		m.logger.Error("Can't find chat with id", zap.String("id", rawMessage.LocalChatID))
		return nil // Continue with next message if chat not found
	}

	if !(chat.Public() || chat.CommunityChat()) {
		return nil // Only resend for public or community chats
	}

	ok, err := m.shouldResendMessage(rawMessage, m.getTimesource())
	if err != nil {
		m.logger.Error("Can't check if message should be resent", zap.Error(err))
		return err
	}

	if ok {
		err := m.persistence.SaveRawMessage(rawMessage)
		if err != nil {
			m.logger.Error("Can't save raw message marked as expired", zap.Error(err))
			return err
		}

		err = m.reSendRawMessage(context.Background(), rawMessage.ID)
		if err != nil {
			m.logger.Error("Can't resend expired message", zap.Error(err), zap.String("id", rawMessage.ID))
		}
	}
	return nil
}

func (m *Messenger) shouldResendMessage(message *common.RawMessage, t common.TimeSource) (bool, error) {
	if m.featureFlags.ResendRawMessagesDisabled {
		return false, nil
	}
	//exponential backoff depends on how many attempts to send message already made
	power := math.Pow(2, float64(message.SendCount-1))
	backoff := uint64(power) * uint64(m.config.messageResendMinDelay.Milliseconds())
	backoffElapsed := t.GetCurrentTime() > (message.LastSent + backoff)
	return backoffElapsed, nil
}

// pull a message from the database and send it again
func (m *Messenger) reSendRawMessage(ctx context.Context, messageID string) error {
	message, err := m.persistence.RawMessageByID(messageID)
	if err != nil {
		return err
	}

	chat, ok := m.allChats.Load(message.LocalChatID)
	if !ok {
		return errors.New("chat not found")
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     message.Payload,
		PubsubTopic: message.PubsubTopic,
		MessageType: message.MessageType,
		Recipients:  message.Recipients,
		ResendType:  message.ResendType,
		SendCount:   message.SendCount,
	})
	return err
}

// UpsertRawMessageToWatch insert/update the rawMessage to the database, resend it if necessary.
// relate watch method: Messenger#watchExpiredMessages
func (m *Messenger) UpsertRawMessageToWatch(rawMessage *common.RawMessage) (*common.RawMessage, error) {
	rawMessage.SendCount++
	rawMessage.LastSent = m.getTimesource().GetCurrentTime()
	err := m.persistence.SaveRawMessage(rawMessage)
	if err != nil {
		return nil, err
	}
	return rawMessage, nil
}

func (m *Messenger) upsertRawMessageToWatch(rawMessage *common.RawMessage) {
	_, err := m.UpsertRawMessageToWatch(rawMessage)
	if err != nil {
		// this is unlikely to happen, but we should log it
		m.logger.Error("Can't upsert raw message after SendCommunityMessage", zap.Error(err), zap.String("id", rawMessage.ID))
	}
}
