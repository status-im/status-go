package pendingtxtracker

import (
	"fmt"
	"math/big"

	eth "github.com/ethereum/go-ethereum/common"

	ac "github.com/status-im/status-go/services/wallet/activity/common"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/common"
)

func GenerateTestPendingTransactions(start int, count int, chainID uint64) []*PendingTransaction {
	if count > 127 {
		panic("can't generate more than 127 distinct transactions")
	}

	txs := make([]*PendingTransaction, count)
	for i := 0; i < count; i++ {
		seed := start + i
		txs[i] = &PendingTransaction{
			Hash:           eth.HexToHash(fmt.Sprintf("0x1%d", seed)),
			From:           eth.HexToAddress(fmt.Sprintf("0x2%d", seed)),
			To:             eth.HexToAddress(fmt.Sprintf("0x3%d", seed)),
			Type:           RegisterENS,
			AdditionalData: "someuser.stateofus.eth",
			Value:          bigint.BigInt{Int: big.NewInt(int64(seed))},
			GasLimit:       bigint.BigInt{Int: big.NewInt(21000)},
			GasPrice:       bigint.BigInt{Int: big.NewInt(int64(seed))},
			ChainID:        common.ChainID(chainID),
			Status:         new(ac.TxStatus),
			AutoDelete:     new(bool),
			Symbol:         "ETH",
			Timestamp:      uint64(seed),
		}
		*txs[i].Status = ac.Pending // set to pending by default
		*txs[i].AutoDelete = true   // set to true by default
	}
	return txs
}

func GenerateTestPendingTransactionsWithSummary(testTxs []TestTxSummary, chainID uint64) []*PendingTransaction {
	genTxs := GenerateTestPendingTransactions(0, len(testTxs), chainID)
	for i, tx := range testTxs {
		if tx.Timestamp > 0 {
			genTxs[i].Timestamp = uint64(tx.Timestamp)
		}
	}
	return genTxs
}

type TestTxSummary struct {
	DontConfirm bool
	// Timestamp will be used to mock the Timestamp if greater than 0
	Timestamp int
}
