package subscriptions

import (
	"fmt"
	"sync"
	"time"

	"github.com/status-im/status-go/rpc"
)

type API struct {
	rpcClient           *rpc.Client
	activeSubscriptions Subscriptions
}

func (api *API) SubscribeSignal(method string, args ...interface{}) (SubscriptionID, error) {
	namespace := "shh"

	var filterID string

	err := rpcClient.Call(&filterID, method, args)
	if err != nil {
		return -1, fmt.Errorf("[SubscribeSignal] could not subscribe, failed to call %s: %v", method, err)
	}

	return api.activeSubscriptions.Add(namespace, filterID)
}

func (api *API) UnsubscribeSignal(id rpc.ID) error {
	return api.activeSubscriptions.Remove(SubscriptionID(id))
}

func (api *API) ClearSignalSubscriptions() {
	return api.activeSubscriptions.RemoveAll()
}
