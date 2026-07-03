package adapters

import (
	"github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/messaging/utils"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

type ConnectionStatusSubscription struct {
	S *wakutypes.ConnStatusSubscription
	c <-chan types.ConnectionStatus
}

func NewConnectionStatusSubscription(s *wakutypes.ConnStatusSubscription) *ConnectionStatusSubscription {
	return &ConnectionStatusSubscription{
		S: s,
		c: utils.BridgeChannels(s.C, func(status wakutypes.ConnStatus) types.ConnectionStatus {
			return types.ConnectionStatus{
				IsOnline: status.IsOnline,
				State:    status.State,
			}
		}),
	}
}

func (c *ConnectionStatusSubscription) C() <-chan types.ConnectionStatus {
	return c.c
}

func (c *ConnectionStatusSubscription) Unsubscribe() {
	c.S.Unsubscribe()
}
