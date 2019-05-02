package subscriptions

import "github.com/status-im/status-go/rpc"

type ethFilter struct {
	id        string
	rpcClient *rpc.Client
}

func InstallEthFilter(rpcClient rpc.Cient) (*whisperFilter, error) {

}

func (wf *ethFilter) getId() string {
	return wf.id
}

func (wf *ethFilter) getChanges() (interface{}, error) {
	panic("implement me")
}

func (wf *ethFilter) uninstall() error {
	panic("implement me")
}
