package protocol

import "github.com/status-im/status-go/protocol/protobuf"

type UserStatus struct {
	PublicKey  string `json:"public-key,omitempty"`
	StatusType int    `json:"status-type"`
	Clock      uint64 `json:"clock"`
	CustomText string `json:"text"`
}

func ToUserStatus(msg protobuf.StatusUpdate) UserStatus {
	return UserStatus{
		StatusType: int(msg.StatusType),
		Clock:      msg.Clock,
		CustomText: msg.CustomText,
	}
}
