package subscriptions

import (
	"fmt"

	"github.com/status-im/status-go/rpc"
)

type ethFilter struct {
	id        string
	rpcClient *rpc.Client
}

func InstallEthFilter(rpcClient *rpc.Client, method string, args []interface{}) (*ethFilter, error) {

	if err := validateEthMethod(method); err != nil {
		return nil, err
	}

	var result string

	err := rpcClient.Call(&result, method, args)

	if err != nil {
		return nil, err
	}

	filter := &ethFilter{
		id:        result,
		rpcClient: rpcClient,
	}

	return filter, nil

}

func (ef *ethFilter) getId() string {
	return ef.id
}

func (ef *ethFilter) getChanges() ([]interface{}, error) {
	var result []interface{}

	err := ef.rpcClient.Call(&result, "eth_getFilterChanges", ef.getId())

	return result, err
}

func (ef *ethFilter) uninstall() error {
	return ef.rpcClient.Call(nil, "eth_uninstallFilter", ef.getId())
}

func validateEthMethod(method string) error {
	for _, allowedMethod := range []string{
		"eth_newFilter",
		"eth_newBlockFilter",
		"eth_newPendingTransactionFilter",
	} {
		if method == allowedMethod {
			return nil
		}
	}

	return fmt.Errorf("unexpected filter method: %s", method)
}
