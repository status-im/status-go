package tokenbalances

import (
	wCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
)

var NativeTokenAddress = wCommon.ZeroAddress()

type AccountAddress = multistandardbalance.AccountAddress
type ContractAddress = multistandardbalance.ContractAddress
