package messaging

import (
	"github.com/status-im/status-go/v10/messaging/adapters"
	"github.com/status-im/status-go/v10/messaging/types"
)

func (a *API) SubscribeToConnStatusChanges() (types.ConnectionStatusSubscription, error) {
	sub, err := a.waku.SubscribeToConnStatusChanges()
	if err != nil {
		return nil, err
	}
	return &adapters.ConnectionStatusSubscription{
		S: sub,
	}, nil
}
