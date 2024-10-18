package common

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts/ierc20"
)

func PackApprovalInputData(amountIn *big.Int, approvalContractAddress *common.Address) ([]byte, error) {
	if approvalContractAddress == nil || *approvalContractAddress == ZeroAddress() {
		return []byte{}, nil
	}

	erc20ABI, err := abi.JSON(strings.NewReader(ierc20.IERC20ABI))
	if err != nil {
		return []byte{}, err
	}

	return erc20ABI.Pack("approve", approvalContractAddress, amountIn)
}

func GetTokenIdFromSymbol(symbol string) (*big.Int, error) {
	id, success := big.NewInt(0).SetString(symbol, 0)
	if !success {
		return nil, fmt.Errorf("failed to convert %s to big.Int", symbol)
	}
	return id, nil
}
