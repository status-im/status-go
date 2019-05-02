package subscriptions

import (
	"fmt"

	"github.com/status-im/status-go/rpc"
)

type API struct {
	rpcClient           *rpc.Client
	activeSubscriptions Subscriptions
}

func (api *API) SubscribeSignal(method string, args ...interface{}) (SubscriptionID, error) {
	namespace := method[:3]

	var filter filter = nil
	var err error = nil

	if namespace == "shh" {
		filter, err := InstallShhFilter(rpcClient, method, args)
	} else if namespace == "eth" {
		filter, err := InstallEthFilter(rpcClient, method, args)
	} else {
		err = fmt.Errorf("unexpected namespace: %s", namespace)
	}

	if err != nil {
		return -1, fmt.Errorf("[SubscribeSignal] could not subscribe, failed to call %s: %v", method, err)
	}

	return api.activeSubscriptions.Create(namespace, filter)
}

func (api *API) UnsubscribeSignal(id rpc.ID) error {
	return api.activeSubscriptions.Remove(SubscriptionID(id))
}

func (api *API) ClearSignalSubscriptions() {
	return api.activeSubscriptions.RemoveAll()
}
