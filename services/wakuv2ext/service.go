package wakuv2ext

import (
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ext"

	wakutypes "github.com/status-im/status-go/waku/types"
)

type Service struct {
	*ext.Service
	w wakutypes.Waku
}

func New(config params.NodeConfig, w wakutypes.Waku, rpcClient *rpc.Client) *Service {
	return &Service{
		Service: ext.New(config, w, rpcClient),
		w:       w,
	}
}

func (s *Service) PublicWakuAPI() wakutypes.PublicWakuAPI {
	return s.w.PublicWakuAPI()
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
