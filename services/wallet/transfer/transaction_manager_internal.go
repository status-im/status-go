package transfer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/services/wallet/router/bridge"
)

func (tm *TransactionManager) buildTransactions(bridges map[string]bridge.Bridge) ([]string, error) {
	tm.transactionsForKeycardSigning = make(map[common.Hash]*TransactionDescription)
	var hashes []string
	for _, bridgeTx := range tm.transactionsBridgeData {
		builtTx, err := bridges[bridgeTx.BridgeName].BuildTransaction(bridgeTx)
		if err != nil {
			return hashes, err
		}

		signer := ethTypes.NewLondonSigner(big.NewInt(int64(bridgeTx.ChainID)))
		txHash := signer.Hash(builtTx)

		tm.transactionsForKeycardSigning[txHash] = &TransactionDescription{
			from:    common.Address(bridgeTx.From()),
			chainID: bridgeTx.ChainID,
			builtTx: builtTx,
		}

		hashes = append(hashes, txHash.String())
	}

	return hashes, nil
}
