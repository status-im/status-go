package adapters

import (
	"github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/messaging/utils"
	wakutypes "github.com/status-im/status-go/waku/types"
)

type ConnectionStatusSubscription struct {
	S *wakutypes.ConnStatusSubscription
}

func (c *ConnectionStatusSubscription) C() <-chan types.ConnectionStatus {
	return utils.BridgeChannels(c.S.C, func(status wakutypes.ConnStatus) types.ConnectionStatus {
		return types.ConnectionStatus{
			IsOnline: status.IsOnline,
		}
	})
}

func (c *ConnectionStatusSubscription) Unsubscribe() {
	c.S.Unsubscribe()
}
