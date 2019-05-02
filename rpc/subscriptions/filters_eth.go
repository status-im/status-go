package subscriptions

import "github.com/status-im/status-go/rpc"

type ethFilter struct {
	id        string
	rpcClient *rpc.Client
}

func InstallEthFilter(rpcClient rpc.Client, method string, args []interface{}) (*whisperFilter, error) {

	if err := validateEthMethod(method); err != nil {
		return err
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

func (ef *ethFilter) getChanges() (interface{}, error) {
	var result []interface{}

	err := ef.rpcClient.Call(&result, "eth_getFilterChanges", wf.getId())

	return result, err
}

func (ef *ethFilter) uninstall() error {
	return ef.rpcClient.Call(nil, "eth_uninstallFilter", wf.getId())
}

func validateEthMethod(method string) error {
	for _, allowedMethod := range []string{
		"eth_newFilter",
		"eth_newBlockFilter",
		"eth_newPendingTransactionFilter",
	} {
		if method == allowedMethod {
			return true
		}
	}

	return false
}
