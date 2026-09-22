package common

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/internal/contracts/ierc20"

	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
)

func IsProcessorBridge(name string) bool {
	// LI.FI and Relay do both swaps and bridges through a single processor.
	return name == pathProcessorCommon.ProcessorBridgeHopName ||
		name == pathProcessorCommon.ProcessorLiFiName ||
		name == pathProcessorCommon.ProcessorRelayName
}

func IsProcessorSwap(name string) bool {
	return name == pathProcessorCommon.ProcessorSwapParaswapName ||
		name == pathProcessorCommon.ProcessorLiFiName ||
		name == pathProcessorCommon.ProcessorRelayName
}

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

// UnpackApprovalInputData decodes ERC20 `approve(spender, amount)` calldata.
func UnpackApprovalInputData(data []byte) (common.Address, *big.Int, error) {
	erc20ABI, err := abi.JSON(strings.NewReader(ierc20.IERC20ABI))
	if err != nil {
		return common.Address{}, nil, err
	}

	method := erc20ABI.Methods["approve"]
	if len(data) < len(method.ID)+64 || !bytes.Equal(data[:len(method.ID)], method.ID) {
		return common.Address{}, nil, fmt.Errorf("not an ERC20 approve call")
	}

	args, err := method.Inputs.Unpack(data[len(method.ID):])
	if err != nil {
		return common.Address{}, nil, err
	}
	if len(args) != 2 {
		return common.Address{}, nil, fmt.Errorf("unexpected approve arguments: %d", len(args))
	}

	spender, ok := args[0].(common.Address)
	if !ok {
		return common.Address{}, nil, fmt.Errorf("unexpected approve spender type")
	}
	amount, ok := args[1].(*big.Int)
	if !ok {
		return common.Address{}, nil, fmt.Errorf("unexpected approve amount type")
	}
	return spender, amount, nil
}

func ExtractCoordinates(pubkey string) ([32]byte, [32]byte) {
	x, _ := hex.DecodeString(pubkey[4:68])
	y, _ := hex.DecodeString(pubkey[68:132])

	var xByte [32]byte
	copy(xByte[:], x)

	var yByte [32]byte
	copy(yByte[:], y)

	return xByte, yByte
}

func NameHash(name string) common.Hash {
	node := common.Hash{}

	if len(name) > 0 {
		labels := strings.Split(name, ".")

		for i := len(labels) - 1; i >= 0; i-- {
			labelSha := crypto.Keccak256Hash([]byte(labels[i]))
			node = crypto.Keccak256Hash(node.Bytes(), labelSha.Bytes())
		}
	}

	return node
}

func ValidateENSUsername(username string) error {
	if !strings.HasSuffix(username, ".eth") {
		return fmt.Errorf("username must end with .eth")
	}

	return nil
}

func UsernameToLabel(username string) [32]byte {
	usernameHashed := crypto.Keccak256([]byte(username))
	var label [32]byte
	copy(label[:], usernameHashed)

	return label
}
