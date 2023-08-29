package activity

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"math/big"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/sqlite"
)

type ProtocolType = int

const (
	ProtocolHop ProtocolType = iota + 1
	ProtocolUniswap
)

type EntryDetails struct {
	ID           string         `json:"id"`
	MultiTxID    int            `json:"multiTxId"`
	Nonce        uint64         `json:"nonce"`
	BlockNumber  int64          `json:"blockNumber"`
	Input        string         `json:"input"`
	ProtocolType *ProtocolType  `json:"protocolType,omitempty"`
	Hash         *eth.Hash      `json:"hash,omitempty"`
	Contract     *eth.Address   `json:"contractAddress,omitempty"`
	MaxFeePerGas *hexutil.Big   `json:"maxFeePerGas"`
	GasLimit     hexutil.Uint64 `json:"gasLimit"`
	TotalFees    *hexutil.Big   `json:"totalFees,omitempty"`
}

func protocolTypeFromDBType(dbType string) (protocolType *ProtocolType) {
	protocolType = new(ProtocolType)
	switch common.Type(dbType) {
	case common.UniswapV2Swap:
		fallthrough
	case common.UniswapV3Swap:
		*protocolType = ProtocolUniswap
	case common.HopBridgeFrom:
		fallthrough
	case common.HopBridgeTo:
		*protocolType = ProtocolHop
	default:
		return nil
	}
	return protocolType
}

func getMultiTxDetails(ctx context.Context, db *sql.DB, multiTxID int) (*EntryDetails, error) {
	if multiTxID <= 0 {
		return nil, errors.New("invalid tx id")
	}
	rows, err := db.QueryContext(ctx, `
	SELECT
		tx_hash,
		blk_number,
		type,
		account_nonce,
		tx,
		contract_address
	FROM
		transfers
	WHERE
		multi_transaction_id = ?;`, multiTxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var maxFeePerGas *hexutil.Big
	var gasLimit hexutil.Uint64
	var input string
	var protocolType *ProtocolType
	var transferHash *eth.Hash
	var contractAddress *eth.Address
	var blockNumber int64
	var nonce uint64
	for rows.Next() {
		var contractTypeDB sql.NullString
		var transferHashDB, contractAddressDB sql.RawBytes
		var blockNumberDB int64
		var nonceDB uint64
		tx := &types.Transaction{}
		nullableTx := sqlite.JSONBlob{Data: tx}
		err := rows.Scan(&transferHashDB, &blockNumberDB, &contractTypeDB, &nonceDB, &nullableTx, &contractAddressDB)
		if err != nil {
			return nil, err
		}
		if len(transferHashDB) > 0 {
			transferHash = new(eth.Hash)
			*transferHash = eth.BytesToHash(transferHashDB)
		}
		if contractTypeDB.Valid && protocolType == nil {
			protocolType = protocolTypeFromDBType(contractTypeDB.String)
		}

		if blockNumberDB > 0 {
			blockNumber = blockNumberDB
		}
		if nonceDB > 0 {
			nonce = nonceDB
		}
		if len(input) == 0 && nullableTx.Valid {
			if len(input) == 0 {
				input = "0x" + hex.EncodeToString(tx.Data())
			}
			if maxFeePerGas == nil {
				maxFeePerGas = (*hexutil.Big)(tx.GasFeeCap())
				gasLimit = hexutil.Uint64(tx.Gas())
			}
		}

		if contractAddress == nil && len(contractAddressDB) > 0 {
			contractAddress = new(eth.Address)
			*contractAddress = eth.BytesToAddress(contractAddressDB)
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return &EntryDetails{
		MultiTxID:    multiTxID,
		Nonce:        nonce,
		BlockNumber:  blockNumber,
		Hash:         transferHash,
		ProtocolType: protocolType,
		Input:        input,
		Contract:     contractAddress,
		MaxFeePerGas: maxFeePerGas,
		GasLimit:     gasLimit,
	}, nil
}

func getTxDetails(ctx context.Context, db *sql.DB, id string) (*EntryDetails, error) {
	if len(id) == 0 {
		return nil, errors.New("invalid tx id")
	}
	rows, err := db.QueryContext(ctx, `
	SELECT
		tx_hash,
		blk_number,
		account_nonce,
		tx,
		contract_address,
		base_gas_fee
	FROM
		transfers
	WHERE
		hash = ?;`, eth.HexToHash(id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, errors.New("Entry not found")
	}

	tx := &types.Transaction{}
	nullableTx := sqlite.JSONBlob{Data: tx}
	var transferHashDB, contractAddressDB sql.RawBytes
	var blockNumber int64
	var nonce uint64
	var baseGasFees string
	err = rows.Scan(&transferHashDB, &blockNumber, &nonce, &nullableTx, &contractAddressDB, &baseGasFees)
	if err != nil {
		return nil, err
	}

	details := &EntryDetails{
		ID:          id,
		Nonce:       nonce,
		BlockNumber: blockNumber,
	}

	if len(transferHashDB) > 0 {
		details.Hash = new(eth.Hash)
		*details.Hash = eth.BytesToHash(transferHashDB)
	}

	if len(contractAddressDB) > 0 {
		details.Contract = new(eth.Address)
		*details.Contract = eth.BytesToAddress(contractAddressDB)
	}

	if nullableTx.Valid {
		details.Input = "0x" + hex.EncodeToString(tx.Data())
		details.MaxFeePerGas = (*hexutil.Big)(tx.GasFeeCap())
		details.GasLimit = hexutil.Uint64(tx.Gas())
		baseGasFees, _ := new(big.Int).SetString(baseGasFees, 0)
		details.TotalFees = (*hexutil.Big)(getTotalFees(tx, baseGasFees))
	}

	return details, nil
}

func getTotalFees(tx *types.Transaction, baseFee *big.Int) *big.Int {
	if tx.Type() == types.DynamicFeeTxType {
		// EIP-1559 transaction
		if baseFee == nil {
			return nil
		}
		tip := tx.GasTipCap()
		maxFee := tx.GasFeeCap()
		gasUsed := big.NewInt(int64(tx.Gas()))

		totalGasUsed := new(big.Int).Add(tip, baseFee)
		if totalGasUsed.Cmp(maxFee) > 0 {
			totalGasUsed.Set(maxFee)
		}

		return new(big.Int).Mul(totalGasUsed, gasUsed)
	}

	// Legacy transaction
	gasPrice := tx.GasPrice()
	gasUsed := big.NewInt(int64(tx.Gas()))

	return new(big.Int).Mul(gasPrice, gasUsed)
}
