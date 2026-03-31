package messaging

import (
	"github.com/status-im/status-go/pkg/messaging/adapters"
	"github.com/status-im/status-go/pkg/messaging/types"
)

func (a *API) SubscribeToConnStatusChanges() (types.ConnectionStatusSubscription, error) {
	sub, err := a.core.waku.SubscribeToConnStatusChanges()
	if err != nil {
		return nil, err
	}
	return adapters.NewConnectionStatusSubscription(sub), nil
}
