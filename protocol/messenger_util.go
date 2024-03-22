package protocol

import "github.com/status-im/status-go/protocol/common"

// GetResendTypeForChat returns the resend type for a chat.
// This function currently infers the ResendType from the chat type.
// However, it is recommended to explicitly determine the ResendType based on
// specific message characteristics to avoid implicit assumptions. This ensures
// that each message dictates its ResendType based on its own properties and
// context, rather than the chat type it is associated with.
func GetResendTypeForChat(chat *Chat) common.ResendType {
	if chat.ChatType == ChatTypeOneToOne {
		return common.ResendTypeDataSync
	}
	return common.ResendTypeRawMessage
}
