package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type OptimismDepositTx struct {
	ChainID  *big.Int
	From     common.Address
	To       common.Address
	Mint     *big.Int
	Value    *big.Int
	Data     []byte
	GasLimit uint64
}

func (d *OptimismDepositTx) txType() byte {
	return OptimismDepositTxType
}

func (d *OptimismDepositTx) copy() TxData {
	tx := &OptimismDepositTx{
		From:     d.From,
		To:       d.To,
		Mint:     new(big.Int),
		Value:    new(big.Int),
		ChainID:  new(big.Int),
		Data:     d.Data,
		GasLimit: d.GasLimit,
	}
	if d.Value != nil {
		tx.Value.Set(d.Value)
	}

	if d.ChainID != nil {
		tx.Value.Set(d.ChainID)
	}

	if d.Mint != nil {
		tx.Value.Set(d.Mint)
	}
	return tx
}

func (d *OptimismDepositTx) chainID() *big.Int      { return d.ChainID }
func (d *OptimismDepositTx) accessList() AccessList { return nil }
func (d *OptimismDepositTx) data() []byte           { return nil }
func (d *OptimismDepositTx) gas() uint64            { return 0 }
func (d *OptimismDepositTx) gasPrice() *big.Int     { return bigZero }
func (d *OptimismDepositTx) gasTipCap() *big.Int    { return bigZero }
func (d *OptimismDepositTx) gasFeeCap() *big.Int    { return bigZero }
func (d *OptimismDepositTx) value() *big.Int        { return d.Value }
func (d *OptimismDepositTx) nonce() uint64          { return 0 }
func (d *OptimismDepositTx) to() *common.Address    { return &d.To }
func (d *OptimismDepositTx) isFake() bool           { return true }

func (d *OptimismDepositTx) rawSignatureValues() (v, r, s *big.Int) {
	return bigZero, bigZero, bigZero
}

func (d *OptimismDepositTx) setSignatureValues(chainID, v, r, s *big.Int) {

}
