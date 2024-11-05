package common

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

func FullDomainName(username string) string {
	return username + "." + StatusDomain
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
