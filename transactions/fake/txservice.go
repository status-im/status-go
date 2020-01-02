package fake

import (
	"context"

	"github.com/golang/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// NewTestServer returns a mocked test server
func NewTestServer(ctrl *gomock.Controller) (*rpc.Server, *MockPublicTransactionPoolAPI) {
	srv := rpc.NewServer()
	svc := NewMockPublicTransactionPoolAPI(ctrl)
	if err := srv.RegisterName("eth", svc); err != nil {
		panic(err)
	}
	return srv, svc
}

// CallArgs copied from module go-ethereum/internal/ethapi
type CallArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      hexutil.Uint64  `json:"gas"`
	GasPrice hexutil.Big     `json:"gasPrice"`
	Value    hexutil.Big     `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
}

// PublicTransactionPoolAPI used to generate mock by mockgen util.
// This was done because PublicTransactionPoolAPI is located in internal/ethapi module
// and there is no easy way to generate mocks from internal modules.
type PublicTransactionPoolAPI interface {
	GasPrice(ctx context.Context) (*hexutil.Big, error)
	EstimateGas(ctx context.Context, args CallArgs) (hexutil.Uint64, error)
	GetTransactionCount(ctx context.Context, address common.Address, blockNr rpc.BlockNumber) (*hexutil.Uint64, error)
	SendRawTransaction(ctx context.Context, encodedTx hexutil.Bytes) (common.Hash, error)
}
