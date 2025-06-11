package wakuv2ext

import (
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ext"
)

type Service struct {
	*ext.Service
}

func New(config params.NodeConfig, rpcClient *rpc.Client) *Service {
	return &Service{
		Service: ext.New(config, rpcClient),
	}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []gethrpc.API {
	apis := []gethrpc.API{
		{
			Namespace: "wakuext",
			Version:   "1.0",
			Service:   NewPublicAPI(s),
			Public:    false,
		},
	}
	return apis
}
