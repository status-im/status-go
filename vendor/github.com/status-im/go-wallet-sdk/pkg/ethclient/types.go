package ethclient

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// BlockWithoutTxs represents a complete block in the Ethereum blockchain without a list of transactions
type BlockWithoutTxs struct {
	Number           *big.Int
	Hash             *common.Hash
	ParentHash       common.Hash
	Nonce            *types.BlockNonce
	Sha3Uncles       common.Hash
	LogsBloom        *types.Bloom
	TransactionsRoot common.Hash
	StateRoot        common.Hash
	ReceiptsRoot     common.Hash
	Miner            common.Address
	Difficulty       *big.Int
	TotalDifficulty  *big.Int
	ExtraData        []byte
	Size             uint64
	GasLimit         uint64
	GasUsed          uint64
	Timestamp        uint64
	BaseFeePerGas    *big.Int
	WithdrawalsRoot  *common.Hash
	BlobGasUsed      *uint64
	ExcessBlobGas    *uint64
	ParentBeaconRoot *common.Hash
	Uncles           []common.Hash
	Withdrawals      []*Withdrawal
}

// UnmarshalJSON implements json.Unmarshaler
func (b *BlockWithoutTxs) UnmarshalJSON(data []byte) error {
	var block blockWithoutTxsJSON
	if err := json.Unmarshal(data, &block); err != nil {
		return err
	}

	b.Number = (*big.Int)(block.Number)
	b.Hash = block.Hash
	b.ParentHash = block.ParentHash
	b.Nonce = block.Nonce
	b.Sha3Uncles = block.Sha3Uncles
	b.LogsBloom = block.LogsBloom
	b.TransactionsRoot = block.TransactionsRoot
	b.StateRoot = block.StateRoot
	b.ReceiptsRoot = block.ReceiptsRoot
	b.Miner = block.Miner
	b.Difficulty = (*big.Int)(block.Difficulty)
	b.TotalDifficulty = (*big.Int)(block.TotalDifficulty)
	b.ExtraData = []byte(block.ExtraData)
	b.Size = uint64(block.Size)
	b.GasLimit = uint64(block.GasLimit)
	b.GasUsed = uint64(block.GasUsed)
	b.Timestamp = uint64(block.Timestamp)
	b.BaseFeePerGas = (*big.Int)(block.BaseFeePerGas)
	b.WithdrawalsRoot = block.WithdrawalsRoot
	b.BlobGasUsed = (*uint64)(block.BlobGasUsed)
	b.ExcessBlobGas = (*uint64)(block.ExcessBlobGas)
	b.ParentBeaconRoot = block.ParentBeaconRoot
	b.Uncles = block.Uncles
	b.Withdrawals = block.Withdrawals
	return nil
}

// MarshalJSON implements json.Marshaler
func (b *BlockWithoutTxs) MarshalJSON() ([]byte, error) {
	block := blockWithoutTxsJSON{
		Number:           (*hexutil.Big)(b.Number),
		Hash:             b.Hash,
		ParentHash:       b.ParentHash,
		Nonce:            b.Nonce,
		Sha3Uncles:       b.Sha3Uncles,
		LogsBloom:        b.LogsBloom,
		TransactionsRoot: b.TransactionsRoot,
		StateRoot:        b.StateRoot,
		ReceiptsRoot:     b.ReceiptsRoot,
		Miner:            b.Miner,
		Difficulty:       (*hexutil.Big)(b.Difficulty),
		TotalDifficulty:  (*hexutil.Big)(b.TotalDifficulty),
		ExtraData:        hexutil.Bytes(b.ExtraData),
		Size:             hexutil.Uint64(b.Size),
		GasLimit:         hexutil.Uint64(b.GasLimit),
		GasUsed:          hexutil.Uint64(b.GasUsed),
		Timestamp:        hexutil.Uint64(b.Timestamp),
		BaseFeePerGas:    (*hexutil.Big)(b.BaseFeePerGas),
		WithdrawalsRoot:  b.WithdrawalsRoot,
		BlobGasUsed:      (*hexutil.Uint64)(b.BlobGasUsed),
		ExcessBlobGas:    (*hexutil.Uint64)(b.ExcessBlobGas),
		ParentBeaconRoot: b.ParentBeaconRoot,
		Uncles:           b.Uncles,
		Withdrawals:      b.Withdrawals,
	}
	return json.Marshal(block)
}

// blockWithoutTxsJSON is the internal type used for JSON marshaling/unmarshaling
type blockWithoutTxsJSON struct {
	Number           *hexutil.Big      `json:"number"`
	Hash             *common.Hash      `json:"hash"`
	ParentHash       common.Hash       `json:"parentHash"`
	Nonce            *types.BlockNonce `json:"nonce"`
	Sha3Uncles       common.Hash       `json:"sha3Uncles"`
	LogsBloom        *types.Bloom      `json:"logsBloom"`
	TransactionsRoot common.Hash       `json:"transactionsRoot"`
	StateRoot        common.Hash       `json:"stateRoot"`
	ReceiptsRoot     common.Hash       `json:"receiptsRoot"`
	Miner            common.Address    `json:"miner"`
	Difficulty       *hexutil.Big      `json:"difficulty"`
	TotalDifficulty  *hexutil.Big      `json:"totalDifficulty"`
	ExtraData        hexutil.Bytes     `json:"extraData"`
	Size             hexutil.Uint64    `json:"size"`
	GasLimit         hexutil.Uint64    `json:"gasLimit"`
	GasUsed          hexutil.Uint64    `json:"gasUsed"`
	Timestamp        hexutil.Uint64    `json:"timestamp"`
	BaseFeePerGas    *hexutil.Big      `json:"baseFeePerGas,omitempty"`
	WithdrawalsRoot  *common.Hash      `json:"withdrawalsRoot,omitempty"`
	BlobGasUsed      *hexutil.Uint64   `json:"blobGasUsed,omitempty"`
	ExcessBlobGas    *hexutil.Uint64   `json:"excessBlobGas,omitempty"`
	ParentBeaconRoot *common.Hash      `json:"parentBeaconRoot,omitempty"`
	Uncles           []common.Hash     `json:"uncles"`
	Withdrawals      []*Withdrawal     `json:"withdrawals,omitempty"`
}

// BlockWithFullTxs represents a complete block in the Ethereum blockchain with a list of full transactions
type BlockWithTxHashes struct {
	Number            *big.Int
	Hash              *common.Hash
	ParentHash        common.Hash
	Nonce             *types.BlockNonce
	Sha3Uncles        common.Hash
	LogsBloom         *types.Bloom
	TransactionsRoot  common.Hash
	StateRoot         common.Hash
	ReceiptsRoot      common.Hash
	Miner             common.Address
	Difficulty        *big.Int
	TotalDifficulty   *big.Int
	ExtraData         []byte
	Size              uint64
	GasLimit          uint64
	GasUsed           uint64
	Timestamp         uint64
	BaseFeePerGas     *big.Int
	WithdrawalsRoot   *common.Hash
	BlobGasUsed       *uint64
	ExcessBlobGas     *uint64
	ParentBeaconRoot  *common.Hash
	TransactionHashes []common.Hash
	Uncles            []common.Hash
	Withdrawals       []*Withdrawal
}

// UnmarshalJSON implements json.Unmarshaler
func (b *BlockWithTxHashes) UnmarshalJSON(data []byte) error {
	var block blockWithTxHashesJSON
	if err := json.Unmarshal(data, &block); err != nil {
		return err
	}

	b.Number = (*big.Int)(block.Number)
	b.Hash = block.Hash
	b.ParentHash = block.ParentHash
	b.Nonce = block.Nonce
	b.Sha3Uncles = block.Sha3Uncles
	b.LogsBloom = block.LogsBloom
	b.TransactionsRoot = block.TransactionsRoot
	b.StateRoot = block.StateRoot
	b.ReceiptsRoot = block.ReceiptsRoot
	b.Miner = block.Miner
	b.Difficulty = (*big.Int)(block.Difficulty)
	b.TotalDifficulty = (*big.Int)(block.TotalDifficulty)
	b.ExtraData = []byte(block.ExtraData)
	b.Size = uint64(block.Size)
	b.GasLimit = uint64(block.GasLimit)
	b.GasUsed = uint64(block.GasUsed)
	b.Timestamp = uint64(block.Timestamp)
	b.BaseFeePerGas = (*big.Int)(block.BaseFeePerGas)
	b.WithdrawalsRoot = block.WithdrawalsRoot
	b.BlobGasUsed = (*uint64)(block.BlobGasUsed)
	b.ExcessBlobGas = (*uint64)(block.ExcessBlobGas)
	b.ParentBeaconRoot = block.ParentBeaconRoot
	b.TransactionHashes = block.TransactionHashes
	b.Uncles = block.Uncles
	b.Withdrawals = block.Withdrawals
	return nil
}

// MarshalJSON implements json.Marshaler
func (b *BlockWithTxHashes) MarshalJSON() ([]byte, error) {
	block := blockWithTxHashesJSON{
		Number:            (*hexutil.Big)(b.Number),
		Hash:              b.Hash,
		ParentHash:        b.ParentHash,
		Nonce:             b.Nonce,
		Sha3Uncles:        b.Sha3Uncles,
		LogsBloom:         b.LogsBloom,
		TransactionsRoot:  b.TransactionsRoot,
		StateRoot:         b.StateRoot,
		ReceiptsRoot:      b.ReceiptsRoot,
		Miner:             b.Miner,
		Difficulty:        (*hexutil.Big)(b.Difficulty),
		TotalDifficulty:   (*hexutil.Big)(b.TotalDifficulty),
		ExtraData:         hexutil.Bytes(b.ExtraData),
		Size:              hexutil.Uint64(b.Size),
		GasLimit:          hexutil.Uint64(b.GasLimit),
		GasUsed:           hexutil.Uint64(b.GasUsed),
		Timestamp:         hexutil.Uint64(b.Timestamp),
		BaseFeePerGas:     (*hexutil.Big)(b.BaseFeePerGas),
		WithdrawalsRoot:   b.WithdrawalsRoot,
		BlobGasUsed:       (*hexutil.Uint64)(b.BlobGasUsed),
		ExcessBlobGas:     (*hexutil.Uint64)(b.ExcessBlobGas),
		ParentBeaconRoot:  b.ParentBeaconRoot,
		TransactionHashes: b.TransactionHashes,
		Uncles:            b.Uncles,
		Withdrawals:       b.Withdrawals,
	}
	return json.Marshal(block)
}

// blockWithTxHashesJSON is the internal type used for JSON marshaling/unmarshaling
type blockWithTxHashesJSON struct {
	Number            *hexutil.Big      `json:"number"`
	Hash              *common.Hash      `json:"hash"`
	ParentHash        common.Hash       `json:"parentHash"`
	Nonce             *types.BlockNonce `json:"nonce"`
	Sha3Uncles        common.Hash       `json:"sha3Uncles"`
	LogsBloom         *types.Bloom      `json:"logsBloom"`
	TransactionsRoot  common.Hash       `json:"transactionsRoot"`
	StateRoot         common.Hash       `json:"stateRoot"`
	ReceiptsRoot      common.Hash       `json:"receiptsRoot"`
	Miner             common.Address    `json:"miner"`
	Difficulty        *hexutil.Big      `json:"difficulty"`
	TotalDifficulty   *hexutil.Big      `json:"totalDifficulty"`
	ExtraData         hexutil.Bytes     `json:"extraData"`
	Size              hexutil.Uint64    `json:"size"`
	GasLimit          hexutil.Uint64    `json:"gasLimit"`
	GasUsed           hexutil.Uint64    `json:"gasUsed"`
	Timestamp         hexutil.Uint64    `json:"timestamp"`
	BaseFeePerGas     *hexutil.Big      `json:"baseFeePerGas,omitempty"`
	WithdrawalsRoot   *common.Hash      `json:"withdrawalsRoot,omitempty"`
	BlobGasUsed       *hexutil.Uint64   `json:"blobGasUsed,omitempty"`
	ExcessBlobGas     *hexutil.Uint64   `json:"excessBlobGas,omitempty"`
	ParentBeaconRoot  *common.Hash      `json:"parentBeaconRoot,omitempty"`
	TransactionHashes []common.Hash     `json:"transactions"`
	Uncles            []common.Hash     `json:"uncles"`
	Withdrawals       []*Withdrawal     `json:"withdrawals,omitempty"`
}

// BlockWithFullTxs represents a complete block in the Ethereum blockchain with a list of full transactions
type BlockWithFullTxs struct {
	Number           *big.Int
	Hash             *common.Hash
	ParentHash       common.Hash
	Nonce            *types.BlockNonce
	Sha3Uncles       common.Hash
	LogsBloom        *types.Bloom
	TransactionsRoot common.Hash
	StateRoot        common.Hash
	ReceiptsRoot     common.Hash
	Miner            common.Address
	Difficulty       *big.Int
	TotalDifficulty  *big.Int
	ExtraData        []byte
	Size             uint64
	GasLimit         uint64
	GasUsed          uint64
	Timestamp        uint64
	BaseFeePerGas    *big.Int
	WithdrawalsRoot  *common.Hash
	BlobGasUsed      *uint64
	ExcessBlobGas    *uint64
	ParentBeaconRoot *common.Hash
	Transactions     []Transaction
	Uncles           []common.Hash
	Withdrawals      []*Withdrawal
}

// UnmarshalJSON implements json.Unmarshaler
func (b *BlockWithFullTxs) UnmarshalJSON(data []byte) error {
	var block blockWithFullTxsJSON
	if err := json.Unmarshal(data, &block); err != nil {
		return err
	}

	b.Number = (*big.Int)(block.Number)
	b.Hash = block.Hash
	b.ParentHash = block.ParentHash
	b.Nonce = block.Nonce
	b.Sha3Uncles = block.Sha3Uncles
	b.LogsBloom = block.LogsBloom
	b.TransactionsRoot = block.TransactionsRoot
	b.StateRoot = block.StateRoot
	b.ReceiptsRoot = block.ReceiptsRoot
	b.Miner = block.Miner
	b.Difficulty = (*big.Int)(block.Difficulty)
	b.TotalDifficulty = (*big.Int)(block.TotalDifficulty)
	b.ExtraData = []byte(block.ExtraData)
	b.Size = uint64(block.Size)
	b.GasLimit = uint64(block.GasLimit)
	b.GasUsed = uint64(block.GasUsed)
	b.Timestamp = uint64(block.Timestamp)
	b.BaseFeePerGas = (*big.Int)(block.BaseFeePerGas)
	b.WithdrawalsRoot = block.WithdrawalsRoot
	b.BlobGasUsed = (*uint64)(block.BlobGasUsed)
	b.ExcessBlobGas = (*uint64)(block.ExcessBlobGas)
	b.ParentBeaconRoot = block.ParentBeaconRoot
	b.Transactions = block.Transactions
	b.Uncles = block.Uncles
	b.Withdrawals = block.Withdrawals
	return nil
}

// MarshalJSON implements json.Marshaler
func (b *BlockWithFullTxs) MarshalJSON() ([]byte, error) {
	block := blockWithFullTxsJSON{
		Number:           (*hexutil.Big)(b.Number),
		Hash:             b.Hash,
		ParentHash:       b.ParentHash,
		Nonce:            b.Nonce,
		Sha3Uncles:       b.Sha3Uncles,
		LogsBloom:        b.LogsBloom,
		TransactionsRoot: b.TransactionsRoot,
		StateRoot:        b.StateRoot,
		ReceiptsRoot:     b.ReceiptsRoot,
		Miner:            b.Miner,
		Difficulty:       (*hexutil.Big)(b.Difficulty),
		TotalDifficulty:  (*hexutil.Big)(b.TotalDifficulty),
		ExtraData:        hexutil.Bytes(b.ExtraData),
		Size:             hexutil.Uint64(b.Size),
		GasLimit:         hexutil.Uint64(b.GasLimit),
		GasUsed:          hexutil.Uint64(b.GasUsed),
		Timestamp:        hexutil.Uint64(b.Timestamp),
		BaseFeePerGas:    (*hexutil.Big)(b.BaseFeePerGas),
		WithdrawalsRoot:  b.WithdrawalsRoot,
		BlobGasUsed:      (*hexutil.Uint64)(b.BlobGasUsed),
		ExcessBlobGas:    (*hexutil.Uint64)(b.ExcessBlobGas),
		ParentBeaconRoot: b.ParentBeaconRoot,
		Transactions:     b.Transactions,
		Uncles:           b.Uncles,
		Withdrawals:      b.Withdrawals,
	}
	return json.Marshal(block)
}

// blockWithFullTxsJSON is the internal type used for JSON marshaling/unmarshaling
type blockWithFullTxsJSON struct {
	Number           *hexutil.Big      `json:"number"`
	Hash             *common.Hash      `json:"hash"`
	ParentHash       common.Hash       `json:"parentHash"`
	Nonce            *types.BlockNonce `json:"nonce"`
	Sha3Uncles       common.Hash       `json:"sha3Uncles"`
	LogsBloom        *types.Bloom      `json:"logsBloom"`
	TransactionsRoot common.Hash       `json:"transactionsRoot"`
	StateRoot        common.Hash       `json:"stateRoot"`
	ReceiptsRoot     common.Hash       `json:"receiptsRoot"`
	Miner            common.Address    `json:"miner"`
	Difficulty       *hexutil.Big      `json:"difficulty"`
	TotalDifficulty  *hexutil.Big      `json:"totalDifficulty"`
	ExtraData        hexutil.Bytes     `json:"extraData"`
	Size             hexutil.Uint64    `json:"size"`
	GasLimit         hexutil.Uint64    `json:"gasLimit"`
	GasUsed          hexutil.Uint64    `json:"gasUsed"`
	Timestamp        hexutil.Uint64    `json:"timestamp"`
	BaseFeePerGas    *hexutil.Big      `json:"baseFeePerGas,omitempty"`
	WithdrawalsRoot  *common.Hash      `json:"withdrawalsRoot,omitempty"`
	BlobGasUsed      *hexutil.Uint64   `json:"blobGasUsed,omitempty"`
	ExcessBlobGas    *hexutil.Uint64   `json:"excessBlobGas,omitempty"`
	ParentBeaconRoot *common.Hash      `json:"parentBeaconRoot,omitempty"`
	Transactions     []Transaction     `json:"transactions"`
	Uncles           []common.Hash     `json:"uncles"`
	Withdrawals      []*Withdrawal     `json:"withdrawals,omitempty"`
}

// Withdrawal represents a withdrawal in the Ethereum blockchain
type Withdrawal struct {
	Index          uint64
	ValidatorIndex uint64
	Address        common.Address
	Amount         *big.Int
}

// UnmarshalJSON implements json.Unmarshaler
func (w *Withdrawal) UnmarshalJSON(data []byte) error {
	var withdrawal withdrawalJSON
	if err := json.Unmarshal(data, &withdrawal); err != nil {
		return err
	}
	w.Index = uint64(withdrawal.Index)
	w.ValidatorIndex = uint64(withdrawal.ValidatorIndex)
	w.Address = withdrawal.Address
	w.Amount = (*big.Int)(withdrawal.Amount)
	return nil
}

// MarshalJSON implements json.Marshaler
func (w *Withdrawal) MarshalJSON() ([]byte, error) {
	withdrawal := withdrawalJSON{
		Index:          hexutil.Uint64(w.Index),
		ValidatorIndex: hexutil.Uint64(w.ValidatorIndex),
		Address:        w.Address,
		Amount:         (*hexutil.Big)(w.Amount),
	}
	return json.Marshal(withdrawal)
}

// withdrawalJSON is the internal type used for JSON marshaling/unmarshaling
type withdrawalJSON struct {
	Index          hexutil.Uint64 `json:"index"`
	ValidatorIndex hexutil.Uint64 `json:"validatorIndex"`
	Address        common.Address `json:"address"`
	Amount         *hexutil.Big   `json:"amount"`
}

// Transaction types.
const (
	LegacyTxType     = 0x00
	AccessListTxType = 0x01
	DynamicFeeTxType = 0x02
	BlobTxType       = 0x03
	SetCodeTxType    = 0x04
)

// Transaction represents a transaction in the Ethereum blockchain
type Transaction struct {
	BlockHash            *common.Hash
	BlockNumber          *big.Int
	From                 common.Address
	Gas                  uint64
	GasPrice             *big.Int
	Hash                 common.Hash
	Input                []byte
	Nonce                uint64
	To                   *common.Address
	TransactionIndex     *uint64
	Value                *big.Int
	V                    *big.Int
	R                    *big.Int
	S                    *big.Int
	Type                 *uint64
	ChainID              *big.Int
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	AccessList           *AccessList
	BlobVersionedHashes  []common.Hash
}

// UnmarshalJSON implements json.Unmarshaler
func (t *Transaction) UnmarshalJSON(data []byte) error {
	var tx transactionJSON
	if err := json.Unmarshal(data, &tx); err != nil {
		return err
	}
	t.BlockHash = tx.BlockHash
	t.BlockNumber = (*big.Int)(tx.BlockNumber)
	t.From = tx.From
	t.Gas = uint64(tx.Gas)
	t.GasPrice = (*big.Int)(tx.GasPrice)
	t.Hash = tx.Hash
	t.Input = []byte(tx.Input)
	t.Nonce = uint64(tx.Nonce)
	t.To = tx.To
	t.TransactionIndex = (*uint64)(tx.TransactionIndex)
	t.Value = (*big.Int)(tx.Value)
	t.V = (*big.Int)(tx.V)
	t.R = (*big.Int)(tx.R)
	t.S = (*big.Int)(tx.S)
	t.Type = (*uint64)(tx.Type)
	t.ChainID = (*big.Int)(tx.ChainID)
	t.MaxFeePerGas = (*big.Int)(tx.MaxFeePerGas)
	t.MaxPriorityFeePerGas = (*big.Int)(tx.MaxPriorityFeePerGas)
	t.AccessList = tx.AccessList
	t.BlobVersionedHashes = tx.BlobVersionedHashes
	return nil
}

// MarshalJSON implements json.Marshaler
func (t *Transaction) MarshalJSON() ([]byte, error) {
	tx := transactionJSON{
		BlockHash:            t.BlockHash,
		BlockNumber:          (*hexutil.Big)(t.BlockNumber),
		From:                 t.From,
		Gas:                  hexutil.Uint64(t.Gas),
		GasPrice:             (*hexutil.Big)(t.GasPrice),
		Hash:                 t.Hash,
		Input:                hexutil.Bytes(t.Input),
		Nonce:                hexutil.Uint64(t.Nonce),
		To:                   t.To,
		TransactionIndex:     (*hexutil.Uint64)(t.TransactionIndex),
		Value:                (*hexutil.Big)(t.Value),
		V:                    (*hexutil.Big)(t.V),
		R:                    (*hexutil.Big)(t.R),
		S:                    (*hexutil.Big)(t.S),
		Type:                 (*hexutil.Uint64)(t.Type),
		ChainID:              (*hexutil.Big)(t.ChainID),
		MaxFeePerGas:         (*hexutil.Big)(t.MaxFeePerGas),
		MaxPriorityFeePerGas: (*hexutil.Big)(t.MaxPriorityFeePerGas),
		AccessList:           t.AccessList,
		BlobVersionedHashes:  t.BlobVersionedHashes,
	}
	return json.Marshal(tx)
}

// transactionJSON is the internal type used for JSON marshaling/unmarshaling
type transactionJSON struct {
	BlockHash            *common.Hash    `json:"blockHash"`
	BlockNumber          *hexutil.Big    `json:"blockNumber"`
	From                 common.Address  `json:"from"`
	Gas                  hexutil.Uint64  `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	Hash                 common.Hash     `json:"hash"`
	Input                hexutil.Bytes   `json:"input"`
	Nonce                hexutil.Uint64  `json:"nonce"`
	To                   *common.Address `json:"to"`
	TransactionIndex     *hexutil.Uint64 `json:"transactionIndex"`
	Value                *hexutil.Big    `json:"value"`
	V                    *hexutil.Big    `json:"v"`
	R                    *hexutil.Big    `json:"r"`
	S                    *hexutil.Big    `json:"s"`
	Type                 *hexutil.Uint64 `json:"type,omitempty"`
	ChainID              *hexutil.Big    `json:"chainId,omitempty"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas,omitempty"`
	AccessList           *AccessList     `json:"accessList,omitempty"`
	BlobVersionedHashes  []common.Hash   `json:"blobVersionedHashes,omitempty"`
}

// Receipt represents a transaction receipt
type Receipt struct {
	BlockHash         common.Hash
	BlockNumber       *big.Int
	ContractAddress   *common.Address
	CumulativeGasUsed uint64
	EffectiveGasPrice *big.Int
	From              common.Address
	GasUsed           uint64
	Logs              []*types.Log
	LogsBloom         types.Bloom
	Status            uint64
	To                *common.Address
	TransactionHash   common.Hash
	TransactionIndex  uint64
	Type              uint64
	BlobGasUsed       *uint64
	BlobGasPrice      *big.Int
}

// UnmarshalJSON implements json.Unmarshaler
func (r *Receipt) UnmarshalJSON(data []byte) error {
	var receipt receiptJSON
	if err := json.Unmarshal(data, &receipt); err != nil {
		return err
	}
	r.BlockHash = receipt.BlockHash
	r.BlockNumber = (*big.Int)(receipt.BlockNumber)
	r.ContractAddress = receipt.ContractAddress
	r.CumulativeGasUsed = uint64(receipt.CumulativeGasUsed)
	r.EffectiveGasPrice = (*big.Int)(receipt.EffectiveGasPrice)
	r.From = receipt.From
	r.GasUsed = uint64(receipt.GasUsed)
	r.Logs = receipt.Logs
	r.LogsBloom = receipt.LogsBloom
	r.Status = uint64(receipt.Status)
	r.To = receipt.To
	r.TransactionHash = receipt.TransactionHash
	r.TransactionIndex = uint64(receipt.TransactionIndex)
	r.Type = uint64(receipt.Type)
	r.BlobGasUsed = (*uint64)(receipt.BlobGasUsed)
	r.BlobGasPrice = (*big.Int)(receipt.BlobGasPrice)
	return nil
}

// MarshalJSON implements json.Marshaler
func (r *Receipt) MarshalJSON() ([]byte, error) {
	receipt := receiptJSON{
		BlockHash:         r.BlockHash,
		BlockNumber:       (*hexutil.Big)(r.BlockNumber),
		ContractAddress:   r.ContractAddress,
		CumulativeGasUsed: hexutil.Uint64(r.CumulativeGasUsed),
		EffectiveGasPrice: (*hexutil.Big)(r.EffectiveGasPrice),
		From:              r.From,
		GasUsed:           hexutil.Uint64(r.GasUsed),
		Logs:              r.Logs,
		LogsBloom:         r.LogsBloom,
		Status:            hexutil.Uint64(r.Status),
		To:                r.To,
		TransactionHash:   r.TransactionHash,
		TransactionIndex:  hexutil.Uint64(r.TransactionIndex),
		Type:              hexutil.Uint64(r.Type),
		BlobGasUsed:       (*hexutil.Uint64)(r.BlobGasUsed),
		BlobGasPrice:      (*hexutil.Big)(r.BlobGasPrice),
	}
	return json.Marshal(receipt)
}

// receiptJSON is the internal type used for JSON marshaling/unmarshaling
type receiptJSON struct {
	BlockHash         common.Hash     `json:"blockHash"`
	BlockNumber       *hexutil.Big    `json:"blockNumber"`
	ContractAddress   *common.Address `json:"contractAddress"`
	CumulativeGasUsed hexutil.Uint64  `json:"cumulativeGasUsed"`
	EffectiveGasPrice *hexutil.Big    `json:"effectiveGasPrice,omitempty"`
	From              common.Address  `json:"from"`
	GasUsed           hexutil.Uint64  `json:"gasUsed"`
	Logs              []*types.Log    `json:"logs"`
	LogsBloom         types.Bloom     `json:"logsBloom"`
	Status            hexutil.Uint64  `json:"status"`
	To                *common.Address `json:"to"`
	TransactionHash   common.Hash     `json:"transactionHash"`
	TransactionIndex  hexutil.Uint64  `json:"transactionIndex"`
	Type              hexutil.Uint64  `json:"type"`
	BlobGasUsed       *hexutil.Uint64 `json:"blobGasUsed,omitempty"`
	BlobGasPrice      *hexutil.Big    `json:"blobGasPrice,omitempty"`
}

// feeHistoryJSON is the internal type used for JSON marshaling/unmarshaling
type feeHistoryJSON struct {
	OldestBlock  *hexutil.Big     `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

// StorageProof represents a storage proof
type StorageProof struct {
	Key   []byte
	Value *big.Int
	Proof [][]byte
}

// UnmarshalJSON implements json.Unmarshaler
func (s *StorageProof) UnmarshalJSON(data []byte) error {
	var proof storageProofJSON
	if err := json.Unmarshal(data, &proof); err != nil {
		return err
	}
	s.Key = []byte(proof.Key)
	s.Value = (*big.Int)(proof.Value)
	s.Proof = make([][]byte, len(proof.Proof))
	for i, p := range proof.Proof {
		s.Proof[i] = []byte(p)
	}
	return nil
}

// MarshalJSON implements json.Marshaler
func (s *StorageProof) MarshalJSON() ([]byte, error) {
	proof := storageProofJSON{
		Key:   hexutil.Bytes(s.Key),
		Value: (*hexutil.Big)(s.Value),
		Proof: make([]hexutil.Bytes, len(s.Proof)),
	}
	for i, p := range s.Proof {
		proof.Proof[i] = hexutil.Bytes(p)
	}
	return json.Marshal(proof)
}

// storageProofJSON is the internal type used for JSON marshaling/unmarshaling
type storageProofJSON struct {
	Key   hexutil.Bytes   `json:"key"`
	Value *hexutil.Big    `json:"value"`
	Proof []hexutil.Bytes `json:"proof"`
}

// Account represents an account with its proof
type Account struct {
	Balance      *big.Int
	CodeHash     common.Hash
	Nonce        uint64
	StorageHash  common.Hash
	AccountProof [][]byte
	StorageProof []StorageProof
}

// UnmarshalJSON implements json.Unmarshaler
func (a *Account) UnmarshalJSON(data []byte) error {
	var account accountJSON
	if err := json.Unmarshal(data, &account); err != nil {
		return err
	}
	a.Balance = (*big.Int)(account.Balance)
	a.CodeHash = account.CodeHash
	a.Nonce = uint64(account.Nonce)
	a.StorageHash = account.StorageHash
	a.AccountProof = make([][]byte, len(account.AccountProof))
	for i, proof := range account.AccountProof {
		a.AccountProof[i] = []byte(proof)
	}
	a.StorageProof = account.StorageProof
	return nil
}

// MarshalJSON implements json.Marshaler
func (a *Account) MarshalJSON() ([]byte, error) {
	account := accountJSON{
		Balance:      (*hexutil.Big)(a.Balance),
		CodeHash:     a.CodeHash,
		Nonce:        hexutil.Uint64(a.Nonce),
		StorageHash:  a.StorageHash,
		AccountProof: make([]hexutil.Bytes, len(a.AccountProof)),
		StorageProof: a.StorageProof,
	}
	for i, proof := range a.AccountProof {
		account.AccountProof[i] = hexutil.Bytes(proof)
	}
	return json.Marshal(account)
}

// accountJSON is the internal type used for JSON marshaling/unmarshaling
type accountJSON struct {
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	AccountProof []hexutil.Bytes `json:"accountProof"`
	StorageProof []StorageProof  `json:"storageProof"`
}

// ProofResult represents a proof result
type ProofResult struct {
	Address      common.Address
	Balance      *big.Int
	CodeHash     common.Hash
	Nonce        uint64
	StorageHash  common.Hash
	AccountProof [][]byte
	StorageProof []StorageProof
}

// UnmarshalJSON implements json.Unmarshaler
func (p *ProofResult) UnmarshalJSON(data []byte) error {
	var result proofResultJSON
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	p.Address = result.Address
	p.Balance = (*big.Int)(result.Balance)
	p.CodeHash = result.CodeHash
	p.Nonce = uint64(result.Nonce)
	p.StorageHash = result.StorageHash
	p.AccountProof = make([][]byte, len(result.AccountProof))
	for i, proof := range result.AccountProof {
		p.AccountProof[i] = []byte(proof)
	}
	p.StorageProof = result.StorageProof
	return nil
}

// MarshalJSON implements json.Marshaler
func (p *ProofResult) MarshalJSON() ([]byte, error) {
	result := proofResultJSON{
		Address:      p.Address,
		Balance:      (*hexutil.Big)(p.Balance),
		CodeHash:     p.CodeHash,
		Nonce:        hexutil.Uint64(p.Nonce),
		StorageHash:  p.StorageHash,
		AccountProof: make([]hexutil.Bytes, len(p.AccountProof)),
		StorageProof: p.StorageProof,
	}
	for i, proof := range p.AccountProof {
		result.AccountProof[i] = hexutil.Bytes(proof)
	}
	return json.Marshal(result)
}

// proofResultJSON is the internal type used for JSON marshaling/unmarshaling
type proofResultJSON struct {
	Address      common.Address  `json:"address"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	AccountProof []hexutil.Bytes `json:"accountProof"`
	StorageProof []StorageProof  `json:"storageProof"`
}

// AccessList represents an access list
type AccessList []AccessTuple

// AccessTuple represents an access tuple
type AccessTuple struct {
	Address     common.Address `json:"address"`
	StorageKeys []common.Hash  `json:"storageKeys"`
}

// FilterID represents a filter ID
type FilterID string

// WorkData represents work data for mining
type WorkData [4]string
