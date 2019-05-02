package subscriptions

import "github.com/status-im/status-go/rpc"

type whisperFilter struct {
	id        string
	rpcClient *rpc.Client
}

func InstallShhFilter(rpcClient rpc.Cient) (*whisperFilter, error) {

}

func (wf *whisperFilter) getChanges() (interface{}, error) {
	panic("implement me")
}

func (wf *whisperFilter) getId() string {
	return wf.id
}

func (wf *whisperFilter) uninstall() error {
	panic("implement me")
}
