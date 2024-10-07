package ethclient

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

func blockJSONToHeader(blockJSON json.RawMessage) (*types.Header, error) {
	if len(blockJSON) == 0 {
		return nil, ethereum.NotFound
	}

	var header *types.Header
	if err := json.Unmarshal(blockJSON, &header); err != nil {
		return nil, err
	}
	return header, nil
}

func blockJSONToBody(blockJSON json.RawMessage) (*rpcBlock, error) {
	if len(blockJSON) == 0 {
		return nil, ethereum.NotFound
	}

	var body *rpcBlock
	if err := json.Unmarshal(blockJSON, &body); err != nil {
		return nil, err
	}

	return body, nil
}

func blockJSONToTxs(blockJSON json.RawMessage) ([]rpcTransaction, error) {
	if len(blockJSON) == 0 {
		return nil, ethereum.NotFound
	}

	var txs *rpcBlockTransactions
	if err := json.Unmarshal(blockJSON, &txs); err != nil {
		return nil, err
	}

	return txs.Txs, nil
}

type parsedBlockJSON struct {
	header *types.Header
	body   *rpcBlock
	txs    []rpcTransaction
}

func parseBlockJSON(blockJSON json.RawMessage, transactionDetailsFlag bool) (*parsedBlockJSON, error) {
	var header *types.Header
	var body *rpcBlock
	var txs []rpcTransaction
	var err error

	if header, err = blockJSONToHeader(blockJSON); err != nil {
		return nil, err
	}
	if body, err = blockJSONToBody(blockJSON); err != nil {
		return nil, err
	}
	if transactionDetailsFlag {
		if txs, err = blockJSONToTxs(blockJSON); err != nil {
			return nil, err
		}
	}
	return &parsedBlockJSON{
		header: header,
		body:   body,
		txs:    txs,
	}, nil
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	finalized := big.NewInt(int64(rpc.FinalizedBlockNumber))
	if number.Cmp(finalized) == 0 {
		return "finalized"
	}
	safe := big.NewInt(int64(rpc.SafeBlockNumber))
	if number.Cmp(safe) == 0 {
		return "safe"
	}
	return hexutil.EncodeBig(number)
}

type rpcBlock struct {
	Hash        common.Hash   `json:"hash"`
	Number      *hexutil.Big  `json:"number,omitempty"`
	UncleHashes []common.Hash `json:"uncles"`
}

type rpcBlockTransactions struct {
	Txs []rpcTransaction `json:"transactions"`
}

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

func (tx *rpcTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

func isConcreteBlockNumber(blockNumber *big.Int) bool {
	return blockNumber != nil && blockNumber.Cmp(big.NewInt(0)) >= 0
}
