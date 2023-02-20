package protocol

import (
	"crypto/ecdsa"
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/verification"
)

// The activity center is a place where we store incoming notifications before
// they are shown to the users as new chats, in order to mitigate the impact of spam
// on the messenger

type ActivityCenterType int

const (
	ActivityCenterNotificationNoType ActivityCenterType = iota
	ActivityCenterNotificationTypeNewOneToOne
	ActivityCenterNotificationTypeNewPrivateGroupChat
	ActivityCenterNotificationTypeMention
	ActivityCenterNotificationTypeReply
	ActivityCenterNotificationTypeContactRequest
	ActivityCenterNotificationTypeCommunityInvitation
	ActivityCenterNotificationTypeCommunityRequest
	ActivityCenterNotificationTypeCommunityMembershipRequest
	ActivityCenterNotificationTypeCommunityKicked
	ActivityCenterNotificationTypeContactVerification
)

type ActivityCenterGroup int

const (
	ActivityCenterAll ActivityCenterGroup = iota
	ActivityCenterMentions
	ActivityCenterReplies
	ActivityCenterMembership
	ActivityCenterAdmin
	ActivityCenterContactRequests
	ActivityCenterIdentityVerification
	ActivityCenterTransactions
	ActivityCenterSystem
)

type ActivityCenterMembershipStatus int

const (
	ActivityCenterMembershipStatusIdle ActivityCenterMembershipStatus = iota
	ActivityCenterMembershipStatusPending
	ActivityCenterMembershipStatusAccepted
	ActivityCenterMembershipStatusDeclined
)

var ErrInvalidActivityCenterNotification = errors.New("invalid activity center notification")

type ActivityCenterNotification struct {
	ID                        types.HexBytes                 `json:"id"`
	ChatID                    string                         `json:"chatId"`
	CommunityID               string                         `json:"communityId"`
	MembershipStatus          ActivityCenterMembershipStatus `json:"membershipStatus"`
	Name                      string                         `json:"name"`
	Author                    string                         `json:"author"`
	Type                      ActivityCenterType             `json:"type"`
	LastMessage               *common.Message                `json:"lastMessage"`
	Message                   *common.Message                `json:"message"`
	ReplyMessage              *common.Message                `json:"replyMessage"`
	Timestamp                 uint64                         `json:"timestamp"`
	Read                      bool                           `json:"read"`
	Dismissed                 bool                           `json:"dismissed"`
	Accepted                  bool                           `json:"accepted"`
	ContactVerificationStatus verification.RequestStatus     `json:"contactVerificationStatus"`
}

type ActivityCenterPaginationResponse struct {
	Cursor        string                        `json:"cursor"`
	Notifications []*ActivityCenterNotification `json:"notifications"`
}

type ActivityCenterState struct {
	HasSeen bool `json:"hasSeen"`
}

func (n *ActivityCenterNotification) Valid() error {
	if len(n.ID) == 0 || n.Type == 0 || n.Timestamp == 0 {
		return ErrInvalidActivityCenterNotification
	}
	return nil
}
func (g ActivityCenterGroup) NotificationTypes() []ActivityCenterType {
	switch g {
	case ActivityCenterAll:
		return []ActivityCenterType{
			ActivityCenterNotificationTypeNewPrivateGroupChat,
			ActivityCenterNotificationTypeMention,
			ActivityCenterNotificationTypeReply,
			ActivityCenterNotificationTypeContactRequest,
			ActivityCenterNotificationTypeCommunityInvitation,
			ActivityCenterNotificationTypeCommunityRequest,
			ActivityCenterNotificationTypeCommunityMembershipRequest,
			ActivityCenterNotificationTypeCommunityKicked,
			ActivityCenterNotificationTypeContactVerification,
		}
	case ActivityCenterMentions:
		return []ActivityCenterType{
			ActivityCenterNotificationTypeMention,
		}
	case ActivityCenterReplies:
		return []ActivityCenterType{
			ActivityCenterNotificationTypeReply,
		}
	case ActivityCenterMembership:
		return []ActivityCenterType{
			ActivityCenterNotificationTypeNewPrivateGroupChat,
			ActivityCenterNotificationTypeCommunityInvitation,
			ActivityCenterNotificationTypeCommunityRequest,
			ActivityCenterNotificationTypeCommunityMembershipRequest,
			ActivityCenterNotificationTypeCommunityKicked,
		}
	case ActivityCenterAdmin:
		return []ActivityCenterType{
			ActivityCenterNotificationTypeCommunityMembershipRequest,
		}
	case ActivityCenterContactRequests:
		return []ActivityCenterType{
			ActivityCenterNotificationTypeContactRequest,
		}
	case ActivityCenterIdentityVerification:
		return []ActivityCenterType{
			ActivityCenterNotificationTypeContactVerification,
		}
	case ActivityCenterTransactions:
		return []ActivityCenterType{} // NOTE: for further proposes
	case ActivityCenterSystem:
		return []ActivityCenterType{} // NOTE: for further proposes
	}
	return []ActivityCenterType{}
}

func showMentionOrReplyActivityCenterNotification(publicKey ecdsa.PublicKey, message *common.Message, chat *Chat, responseTo *common.Message) (bool, ActivityCenterType) {
	if chat == nil || !chat.Active || (!chat.CommunityChat() && !chat.PrivateGroupChat()) {
		return false, ActivityCenterNotificationNoType
	}

	if message.Mentioned {
		return true, ActivityCenterNotificationTypeMention
	}

	publicKeyString := common.PubkeyToHex(&publicKey)
	if responseTo != nil && responseTo.From == publicKeyString {
		return true, ActivityCenterNotificationTypeReply
	}

	return false, ActivityCenterNotificationNoType
}
