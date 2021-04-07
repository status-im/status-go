package protocol

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
)

type ActivityCenterType int

const (
	ActivityCenterNotificationTypeNewOneToOne = iota + 1
	ActivityCenterNotificationTypeNewPrivateGroupChat
)

var ErrInvalidActivityCenterNotification = errors.New("invalid activity center notification")

type ActivityCenterNotification struct {
	ID          types.HexBytes     `json:"id"`
	ChatID      string             `json:"chatId"`
	Type        ActivityCenterType `json:"type"`
	LastMessage *common.Message    `json:"lastMessage"`
	Timestamp   uint64             `json:"timestamp"`
	Read        bool               `json:"read"`
	Dismissed   bool               `json:"dismissed"`
	Accepted    bool               `json:"accepted"`
}

type ActivityCenterPaginationResponse struct {
	Cursor        string                        `json:"cursor"`
	Notifications []*ActivityCenterNotification `json:"notifications"`
}

func (n *ActivityCenterNotification) Valid() error {
	if len(n.ID) == 0 || n.Type == 0 || n.Timestamp == 0 {
		return ErrInvalidActivityCenterNotification
	}
	return nil

}
