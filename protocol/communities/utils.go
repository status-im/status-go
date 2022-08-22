package communities

import (
	"fmt"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
)

func CalculateRequestID(publicKey string, communityID types.HexBytes) types.HexBytes {
	idString := fmt.Sprintf("%s-%s", publicKey, communityID)
	return crypto.Keccak256([]byte(idString))
}
