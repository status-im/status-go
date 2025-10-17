package chainutils

//go:generate go tool mockgen -source=interfaces.go -destination=mock/interfaces.go -package=mock_chainutils

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/rpc/chain/ethclient"
	"github.com/status-im/status-go/services/wallet/router/fees"
)

type FeeManager interface {
	SuggestedFees(ctx context.Context, chainID uint64, address common.Address) (suggestedFees *fees.SuggestedFees, noBaseFee bool, noPriorityFee bool, err error)
}

type EthClientGetter interface {
	EthClient(chainID uint64) (ethclient.EthClientInterface, error)
}
