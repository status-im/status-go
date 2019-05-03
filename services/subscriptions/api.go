package subscriptions

import (
	"fmt"
	"time"

	"github.com/status-im/status-go/rpc"
)

type API struct {
	rpcClient           *rpc.Client
	activeSubscriptions *Subscriptions
}

func NewPublicAPI(rpcClient *rpc.Client) *API {
	return &API{
		rpcClient:           rpcClient,
		activeSubscriptions: NewSubscriptions(100 * time.Millisecond),
	}
}

func (api *API) SubscribeSignal(method string, args ...interface{}) (SubscriptionID, error) {
	namespace := method[:3]

	var filter filter = nil
	var err error = nil

	if namespace == "shh" {
		filter, err = InstallShhFilter(api.rpcClient, method, args)
	} else if namespace == "eth" {
		filter, err = InstallEthFilter(api.rpcClient, method, args)
	} else {
		err = fmt.Errorf("unexpected namespace: %s", namespace)
	}

	if err != nil {
		return SubscriptionID(""), fmt.Errorf("[SubscribeSignal] could not subscribe, failed to call %s: %v", method, err)
	}

	return api.activeSubscriptions.Create(namespace, filter)
}

func (api *API) UnsubscribeSignal(id string) error {
	return api.activeSubscriptions.Remove(SubscriptionID(id))
}

func (api *API) ClearSignalSubscriptions() error {
	return api.activeSubscriptions.RemoveAll()
}
