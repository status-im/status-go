package transfer

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
)

func (tm *TransactionManager) buildTransactions(pathProcessors map[string]pathprocessor.PathProcessor) ([]string, error) {
	tm.transactionsForKeycardSigning = make(map[common.Hash]*TransactionDescription)
	var hashes []string
	for _, bridgeTx := range tm.multipathTransactionsData {
		builtTx, err := pathProcessors[bridgeTx.Name].BuildTransaction(bridgeTx)
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
