package communities

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

func validateCommunityChat(desc *protobuf.CommunityDescription, chat *protobuf.CommunityChat) error {
	if chat == nil {
		return ErrInvalidCommunityDescription
	}
	if chat.Permissions == nil {
		return ErrInvalidCommunityDescriptionNoChatPermissions
	}
	if chat.Permissions.Access == protobuf.CommunityPermissions_UNKNOWN_ACCESS {
		return ErrInvalidCommunityDescriptionUnknownChatAccess
	}

	for pk := range chat.Members {
		if desc.Members == nil {
			return ErrInvalidCommunityDescriptionMemberInChatButNotInOrg
		}
		// Check member is in the org as well
		if _, ok := desc.Members[pk]; !ok {
			return ErrInvalidCommunityDescriptionMemberInChatButNotInOrg
		}
	}

	return nil
}

func ValidateCommunityDescription(desc *protobuf.CommunityDescription) error {
	if desc == nil {
		return ErrInvalidCommunityDescription
	}
	if desc.Permissions == nil {
		return ErrInvalidCommunityDescriptionNoOrgPermissions
	}
	if desc.Permissions.Access == protobuf.CommunityPermissions_UNKNOWN_ACCESS {
		return ErrInvalidCommunityDescriptionUnknownOrgAccess
	}

	for _, chat := range desc.Chats {
		if err := validateCommunityChat(desc, chat); err != nil {
			return err
		}
	}

	return nil
}
