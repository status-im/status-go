package wakuv2ext

import (
	"go.uber.org/zap"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ext"
)

type Service struct {
	*ext.Service
}

func New(config params.NodeConfig, rpcClient *rpc.Client, logger *zap.Logger) *Service {
	return &Service{
		Service: ext.New(config, rpcClient, logger),
	}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []gethrpc.API {
	apis := []gethrpc.API{
		{
			Namespace: "owakuext",
			Version:   "1.0",
			Service:   NewPublicAPI(s),
			Public:    false,
		},
	}
	return apis
}
