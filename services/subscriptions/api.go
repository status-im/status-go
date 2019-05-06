package subscriptions

import (
	"fmt"
	"time"

	"github.com/status-im/status-go/node"
)

type API struct {
	node                *node.StatusNode
	activeSubscriptions *Subscriptions
}

func NewPublicAPI(node *node.StatusNode) *API {
	fmt.Println("making newpublicAPI")
	return &API{
		node:                node,
		activeSubscriptions: NewSubscriptions(100 * time.Millisecond),
	}
}

func (api *API) SubscribeSignal(method string, args []interface{}) (SubscriptionID, error) {
	namespace := method[:3]

	var filter filter
	var err error

	fmt.Printf("args = %+v\n", args)

	rpc := api.node.RPCPrivateClient()

	if namespace == "shh" {
		filter, err = installShhFilter(rpc, method, args)
	} else if namespace == "eth" {
		filter, err = installEthFilter(rpc, method, args)
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

func (api *API) shutdown() error {
	return api.activeSubscriptions.removeAll()
}
