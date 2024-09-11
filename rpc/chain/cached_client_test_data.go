package chain

import (
	"math/big"
	"math/rand"

	crypto_rand "crypto/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func getRandomTransaction() *types.Transaction {
	nonce := rand.Uint64()
	gasLimit := rand.Uint64()
	gasPrice := rand.Uint64()
	to := common.Address{}
	crypto_rand.Read(to[:])
	value := rand.Uint64()
	data := make([]byte, 32*rand.Intn(10))
	crypto_rand.Read(data)

	tx := types.NewTransaction(nonce, to, big.NewInt(int64(value)), gasLimit, big.NewInt(int64(gasPrice)), data)

	return tx
}

func getRandomBlockHeader() *types.Header {
	header := &types.Header{
		Number:     big.NewInt(rand.Int63()),
		Time:       rand.Uint64(),
		Difficulty: big.NewInt(rand.Int63()),
		ParentHash: common.Hash{},
		Nonce:      types.BlockNonce{},
		MixDigest:  common.Hash{},
	}
	crypto_rand.Read(header.ParentHash[:])
	crypto_rand.Read(header.Nonce[:])
	crypto_rand.Read(header.MixDigest[:])

	return header
}

func getRandomLog() *types.Log {
	log := &types.Log{
		Address:     common.Address{},
		Topics:      []common.Hash{},
		Data:        []byte{},
		BlockNumber: rand.Uint64(),
		TxHash:      common.Hash{},
		TxIndex:     uint(rand.Uint64()),
	}
	crypto_rand.Read(log.Address[:])
	crypto_rand.Read(log.TxHash[:])
	for i := 0; i < rand.Intn(10); i++ {
		hash := common.Hash{}
		crypto_rand.Read(hash[:])
		log.Topics = append(log.Topics, hash)
	}
	crypto_rand.Read(log.Data)

	return log
}

func getRandomReceipt() *types.Receipt {
	receipt := &types.Receipt{
		Status:            rand.Uint64(),
		CumulativeGasUsed: rand.Uint64(),
		Bloom:             types.Bloom{},
		Logs:              []*types.Log{},
	}
	crypto_rand.Read(receipt.Bloom[:])
	for i := 0; i < rand.Intn(10); i++ {
		receipt.Logs = append(receipt.Logs, getRandomLog())
	}

	return receipt
}

func getRandomBlock() *types.Block {
	header := getRandomBlockHeader()

	txs := []*types.Transaction{}
	for i := 0; i < rand.Intn(10); i++ {
		txs = append(txs, getRandomTransaction())
	}

	receipts := []*types.Receipt{}
	for i := 0; i < rand.Intn(10); i++ {
		receipts = append(receipts, getRandomReceipt())
	}

	return types.NewBlock(header, txs, nil, receipts, nil)
}
