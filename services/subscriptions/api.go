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
	return &API{
		node:                node,
		activeSubscriptions: NewSubscriptions(100 * time.Millisecond),
	}
}

func (api *API) SubscribeSignal(method string, args []interface{}) (SubscriptionID, error) {
	var (
		filter    filter
		err       error
		namespace = method[:3]
	)

	rpc := api.node.RPCPrivateClient()

	switch namespace {
	case "shh":
		filter, err = installShhFilter(rpc, method, args)
	case "eth":
		filter, err = installEthFilter(rpc, method, args)
	default:
		err = fmt.Errorf("unexpected namespace: %s", namespace)
	}

	if err != nil {
		return "", fmt.Errorf("[SubscribeSignal] could not subscribe, failed to call %s: %v", method, err)
	}

	return api.activeSubscriptions.Create(namespace, filter)
}

func (api *API) UnsubscribeSignal(id string) error {
	return api.activeSubscriptions.Remove(SubscriptionID(id))
}
