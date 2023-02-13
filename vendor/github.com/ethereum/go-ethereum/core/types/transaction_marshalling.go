// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// txJSON is the JSON representation of transactions.
type txJSON struct {
	Type hexutil.Uint64 `json:"type"`

	// Common transaction fields:
	Nonce                *hexutil.Uint64 `json:"nonce"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	Value                *hexutil.Big    `json:"value"`
	Data                 *hexutil.Bytes  `json:"input"`
	V                    *hexutil.Big    `json:"v"`
	R                    *hexutil.Big    `json:"r"`
	S                    *hexutil.Big    `json:"s"`
	To                   *common.Address `json:"to"`

	// Access list transaction fields:
	ChainID    *hexutil.Big `json:"chainId,omitempty"`
	AccessList *AccessList  `json:"accessList,omitempty"`

	// Arbitrum fields:
	From                *common.Address `json:"from,omitempty"`                // Contract SubmitRetryable Unsigned Retry
	RequestId           *common.Hash    `json:"requestId,omitempty"`           // Contract SubmitRetryable Deposit
	TicketId            *common.Hash    `json:"ticketId,omitempty"`            // Retry
	MaxRefund           *hexutil.Big    `json:"maxRefund,omitempty"`           // Retry
	SubmissionFeeRefund *hexutil.Big    `json:"submissionFeeRefund,omitempty"` // Retry
	RefundTo            *common.Address `json:"refundTo,omitempty"`            // SubmitRetryable Retry
	L1BaseFee           *hexutil.Big    `json:"l1BaseFee,omitempty"`           // SubmitRetryable
	DepositValue        *hexutil.Big    `json:"depositValue,omitempty"`        // SubmitRetryable
	RetryTo             *common.Address `json:"retryTo,omitempty"`             // SubmitRetryable
	RetryValue          *hexutil.Big    `json:"retryValue,omitempty"`          // SubmitRetryable
	RetryData           *hexutil.Bytes  `json:"retryData,omitempty"`           // SubmitRetryable
	Beneficiary         *common.Address `json:"beneficiary,omitempty"`         // SubmitRetryable
	MaxSubmissionFee    *hexutil.Big    `json:"maxSubmissionFee,omitempty"`    // SubmitRetryable
	EffectiveGasPrice   *hexutil.Uint64 `json:"effectiveGasPrice,omitempty"`   // ArbLegacy
	L1BlockNumber       *hexutil.Uint64 `json:"l1BlockNumber,omitempty"`       // ArbLegacy

	// Only used for encoding - and for ArbLegacy
	Hash common.Hash `json:"hash"`
}

// MarshalJSON marshals as JSON with a hash.
func (t *Transaction) MarshalJSON() ([]byte, error) {
	var enc txJSON
	// These are set for all tx types.
	enc.Hash = t.Hash()
	enc.Type = hexutil.Uint64(t.Type())

	// Arbitrum: set to 0 for compatibility
	var zero uint64
	enc.Nonce = (*hexutil.Uint64)(&zero)
	enc.Gas = (*hexutil.Uint64)(&zero)
	enc.GasPrice = (*hexutil.Big)(common.Big0)
	enc.Value = (*hexutil.Big)(common.Big0)
	enc.Data = (*hexutil.Bytes)(&[]byte{})
	enc.V = (*hexutil.Big)(common.Big0)
	enc.R = (*hexutil.Big)(common.Big0)
	enc.S = (*hexutil.Big)(common.Big0)

	// Other fields are set conditionally depending on tx type.
	switch tx := t.inner.(type) {
	case *LegacyTx:
		enc.Nonce = (*hexutil.Uint64)(&tx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.GasPrice = (*hexutil.Big)(tx.GasPrice)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.To = t.To()
		enc.V = (*hexutil.Big)(tx.V)
		enc.R = (*hexutil.Big)(tx.R)
		enc.S = (*hexutil.Big)(tx.S)
	case *AccessListTx:
		enc.ChainID = (*hexutil.Big)(tx.ChainID)
		enc.AccessList = &tx.AccessList
		enc.Nonce = (*hexutil.Uint64)(&tx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.GasPrice = (*hexutil.Big)(tx.GasPrice)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.To = t.To()
		enc.V = (*hexutil.Big)(tx.V)
		enc.R = (*hexutil.Big)(tx.R)
		enc.S = (*hexutil.Big)(tx.S)
	case *DynamicFeeTx:
		enc.ChainID = (*hexutil.Big)(tx.ChainID)
		enc.AccessList = &tx.AccessList
		enc.Nonce = (*hexutil.Uint64)(&tx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.MaxFeePerGas = (*hexutil.Big)(tx.GasFeeCap)
		enc.MaxPriorityFeePerGas = (*hexutil.Big)(tx.GasTipCap)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.To = t.To()
		enc.V = (*hexutil.Big)(tx.V)
		enc.R = (*hexutil.Big)(tx.R)
		enc.S = (*hexutil.Big)(tx.S)
	case *ArbitrumLegacyTxData:
		enc.Nonce = (*hexutil.Uint64)(&tx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.GasPrice = (*hexutil.Big)(tx.GasPrice)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.To = t.To()
		enc.V = (*hexutil.Big)(tx.V)
		enc.R = (*hexutil.Big)(tx.R)
		enc.S = (*hexutil.Big)(tx.S)
		enc.EffectiveGasPrice = (*hexutil.Uint64)(&tx.EffectiveGasPrice)
		enc.L1BlockNumber = (*hexutil.Uint64)(&tx.L1BlockNumber)
		enc.From = tx.Sender
	case *ArbitrumInternalTx:
		enc.ChainID = (*hexutil.Big)(tx.ChainId)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
	case *ArbitrumDepositTx:
		enc.RequestId = &tx.L1RequestId
		enc.From = &tx.From
		enc.ChainID = (*hexutil.Big)(tx.ChainId)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.To = t.To()
	case *ArbitrumUnsignedTx:
		enc.From = (*common.Address)(&tx.From)
		enc.ChainID = (*hexutil.Big)(tx.ChainId)
		enc.Nonce = (*hexutil.Uint64)(&tx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.MaxFeePerGas = (*hexutil.Big)(tx.GasFeeCap)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.To = t.To()
	case *ArbitrumContractTx:
		enc.RequestId = &tx.RequestId
		enc.From = (*common.Address)(&tx.From)
		enc.ChainID = (*hexutil.Big)(tx.ChainId)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.MaxFeePerGas = (*hexutil.Big)(tx.GasFeeCap)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.To = t.To()
	case *ArbitrumRetryTx:
		enc.From = (*common.Address)(&tx.From)
		enc.TicketId = &tx.TicketId
		enc.RefundTo = &tx.RefundTo
		enc.ChainID = (*hexutil.Big)(tx.ChainId)
		enc.Nonce = (*hexutil.Uint64)(&tx.Nonce)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.MaxFeePerGas = (*hexutil.Big)(tx.GasFeeCap)
		enc.Value = (*hexutil.Big)(tx.Value)
		enc.Data = (*hexutil.Bytes)(&tx.Data)
		enc.MaxRefund = (*hexutil.Big)(tx.MaxRefund)
		enc.SubmissionFeeRefund = (*hexutil.Big)(tx.SubmissionFeeRefund)
		enc.To = t.To()
	case *ArbitrumSubmitRetryableTx:
		enc.RequestId = &tx.RequestId
		enc.From = &tx.From
		enc.L1BaseFee = (*hexutil.Big)(tx.L1BaseFee)
		enc.DepositValue = (*hexutil.Big)(tx.DepositValue)
		enc.Beneficiary = &tx.Beneficiary
		enc.RefundTo = &tx.FeeRefundAddr
		enc.MaxSubmissionFee = (*hexutil.Big)(tx.MaxSubmissionFee)
		enc.ChainID = (*hexutil.Big)(tx.ChainId)
		enc.Gas = (*hexutil.Uint64)(&tx.Gas)
		enc.MaxFeePerGas = (*hexutil.Big)(tx.GasFeeCap)
		enc.RetryTo = tx.RetryTo
		enc.RetryValue = (*hexutil.Big)(tx.RetryValue)
		enc.RetryData = (*hexutil.Bytes)(&tx.RetryData)
		data := tx.data()
		enc.Data = (*hexutil.Bytes)(&data)
		enc.To = t.To()
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *Transaction) UnmarshalJSON(input []byte) error {
	var dec txJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}

	// Decode / verify fields according to transaction type.
	var inner TxData
	switch dec.Type {
	case LegacyTxType:
		var itx LegacyTx
		inner = &itx
		if dec.To != nil {
			itx.To = dec.To
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.Nonce = uint64(*dec.Nonce)
		if dec.GasPrice == nil {
			return errors.New("missing required field 'gasPrice' in transaction")
		}
		itx.GasPrice = (*big.Int)(dec.GasPrice)
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' in transaction")
		}
		itx.Gas = uint64(*dec.Gas)
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Value = (*big.Int)(dec.Value)
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Data = *dec.Data
		if dec.V == nil {
			return errors.New("missing required field 'v' in transaction")
		}
		itx.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' in transaction")
		}
		itx.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' in transaction")
		}
		itx.S = (*big.Int)(dec.S)
		withSignature := itx.V.Sign() != 0 || itx.R.Sign() != 0 || itx.S.Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(itx.V, itx.R, itx.S, true); err != nil {
				return err
			}
		}

	case AccessListTxType:
		var itx AccessListTx
		inner = &itx
		// Access list is optional for now.
		if dec.AccessList != nil {
			itx.AccessList = *dec.AccessList
		}
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		itx.ChainID = (*big.Int)(dec.ChainID)
		if dec.To != nil {
			itx.To = dec.To
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.Nonce = uint64(*dec.Nonce)
		if dec.GasPrice == nil {
			return errors.New("missing required field 'gasPrice' in transaction")
		}
		itx.GasPrice = (*big.Int)(dec.GasPrice)
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' in transaction")
		}
		itx.Gas = uint64(*dec.Gas)
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Value = (*big.Int)(dec.Value)
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Data = *dec.Data
		if dec.V == nil {
			return errors.New("missing required field 'v' in transaction")
		}
		itx.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' in transaction")
		}
		itx.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' in transaction")
		}
		itx.S = (*big.Int)(dec.S)
		withSignature := itx.V.Sign() != 0 || itx.R.Sign() != 0 || itx.S.Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(itx.V, itx.R, itx.S, false); err != nil {
				return err
			}
		}

	case DynamicFeeTxType:
		var itx DynamicFeeTx
		inner = &itx
		// Access list is optional for now.
		if dec.AccessList != nil {
			itx.AccessList = *dec.AccessList
		}
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		itx.ChainID = (*big.Int)(dec.ChainID)
		if dec.To != nil {
			itx.To = dec.To
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.Nonce = uint64(*dec.Nonce)
		if dec.MaxPriorityFeePerGas == nil {
			return errors.New("missing required field 'maxPriorityFeePerGas' for txdata")
		}
		itx.GasTipCap = (*big.Int)(dec.MaxPriorityFeePerGas)
		if dec.MaxFeePerGas == nil {
			return errors.New("missing required field 'maxFeePerGas' for txdata")
		}
		itx.GasFeeCap = (*big.Int)(dec.MaxFeePerGas)
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' for txdata")
		}
		itx.Gas = uint64(*dec.Gas)
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Value = (*big.Int)(dec.Value)
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Data = *dec.Data
		if dec.V == nil {
			return errors.New("missing required field 'v' in transaction")
		}
		itx.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' in transaction")
		}
		itx.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' in transaction")
		}
		itx.S = (*big.Int)(dec.S)
		withSignature := itx.V.Sign() != 0 || itx.R.Sign() != 0 || itx.S.Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(itx.V, itx.R, itx.S, false); err != nil {
				return err
			}
		}

	case ArbitrumLegacyTxType:
		var itx LegacyTx
		if dec.To != nil {
			itx.To = dec.To
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		itx.Nonce = uint64(*dec.Nonce)
		if dec.GasPrice == nil {
			return errors.New("missing required field 'gasPrice' in transaction")
		}
		itx.GasPrice = (*big.Int)(dec.GasPrice)
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' in transaction")
		}
		itx.Gas = uint64(*dec.Gas)
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		itx.Value = (*big.Int)(dec.Value)
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		itx.Data = *dec.Data
		if dec.V == nil {
			return errors.New("missing required field 'v' in transaction")
		}
		itx.V = (*big.Int)(dec.V)
		if dec.R == nil {
			return errors.New("missing required field 'r' in transaction")
		}
		itx.R = (*big.Int)(dec.R)
		if dec.S == nil {
			return errors.New("missing required field 's' in transaction")
		}
		itx.S = (*big.Int)(dec.S)
		withSignature := itx.V.Sign() != 0 || itx.R.Sign() != 0 || itx.S.Sign() != 0
		if withSignature {
			if err := sanityCheckSignature(itx.V, itx.R, itx.S, true); err != nil {
				return err
			}
		}
		if dec.EffectiveGasPrice == nil {
			return errors.New("missing required field 'EffectiveGasPrice' in transaction")
		}
		if dec.L1BlockNumber == nil {
			return errors.New("missing required field 'L1BlockNumber' in transaction")
		}
		inner = &ArbitrumLegacyTxData{
			LegacyTx:          itx,
			HashOverride:      dec.Hash,
			EffectiveGasPrice: uint64(*dec.EffectiveGasPrice),
			L1BlockNumber:     uint64(*dec.L1BlockNumber),
			Sender:            dec.From,
		}

	case ArbitrumInternalTxType:
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		inner = &ArbitrumInternalTx{
			ChainId: (*big.Int)(dec.ChainID),
			Data:    *dec.Data,
		}

	case ArbitrumDepositTxType:
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		if dec.RequestId == nil {
			return errors.New("missing required field 'requestId' in transaction")
		}
		if dec.To == nil {
			return errors.New("missing required field 'to' in transaction")
		}
		if dec.From == nil {
			return errors.New("missing required field 'from' in transaction")
		}
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		inner = &ArbitrumDepositTx{
			ChainId:     (*big.Int)(dec.ChainID),
			L1RequestId: *dec.RequestId,
			To:          *dec.To,
			From:        *dec.From,
			Value:       (*big.Int)(dec.Value),
		}

	case ArbitrumUnsignedTxType:
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		if dec.From == nil {
			return errors.New("missing required field 'from' in transaction")
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		if dec.MaxFeePerGas == nil {
			return errors.New("missing required field 'maxFeePerGas' for txdata")
		}
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' in txdata")
		}
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		inner = &ArbitrumUnsignedTx{
			ChainId:   (*big.Int)(dec.ChainID),
			From:      *dec.From,
			Nonce:     uint64(*dec.Nonce),
			GasFeeCap: (*big.Int)(dec.MaxFeePerGas),
			Gas:       uint64(*dec.Gas),
			To:        dec.To,
			Value:     (*big.Int)(dec.Value),
			Data:      *dec.Data,
		}

	case ArbitrumContractTxType:
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		if dec.RequestId == nil {
			return errors.New("missing required field 'requestId' in transaction")
		}
		if dec.From == nil {
			return errors.New("missing required field 'from' in transaction")
		}
		if dec.MaxFeePerGas == nil {
			return errors.New("missing required field 'maxFeePerGas' for txdata")
		}
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' in txdata")
		}
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		inner = &ArbitrumContractTx{
			ChainId:   (*big.Int)(dec.ChainID),
			RequestId: *dec.RequestId,
			From:      *dec.From,
			GasFeeCap: (*big.Int)(dec.MaxFeePerGas),
			Gas:       uint64(*dec.Gas),
			To:        dec.To,
			Value:     (*big.Int)(dec.Value),
			Data:      *dec.Data,
		}

	case ArbitrumRetryTxType:
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		if dec.Nonce == nil {
			return errors.New("missing required field 'nonce' in transaction")
		}
		if dec.From == nil {
			return errors.New("missing required field 'from' in transaction")
		}
		if dec.MaxFeePerGas == nil {
			return errors.New("missing required field 'maxFeePerGas' for txdata")
		}
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' in txdata")
		}
		if dec.Value == nil {
			return errors.New("missing required field 'value' in transaction")
		}
		if dec.Data == nil {
			return errors.New("missing required field 'input' in transaction")
		}
		if dec.TicketId == nil {
			return errors.New("missing required field 'ticketId' in transaction")
		}
		if dec.RefundTo == nil {
			return errors.New("missing required field 'refundTo' in transaction")
		}
		if dec.MaxRefund == nil {
			return errors.New("missing required field 'maxRefund' in transaction")
		}
		if dec.SubmissionFeeRefund == nil {
			return errors.New("missing required field 'submissionFeeRefund' in transaction")
		}
		inner = &ArbitrumRetryTx{
			ChainId:             (*big.Int)(dec.ChainID),
			Nonce:               uint64(*dec.Nonce),
			From:                *dec.From,
			GasFeeCap:           (*big.Int)(dec.MaxFeePerGas),
			Gas:                 uint64(*dec.Gas),
			To:                  dec.To,
			Value:               (*big.Int)(dec.Value),
			Data:                *dec.Data,
			TicketId:            *dec.TicketId,
			RefundTo:            *dec.RefundTo,
			MaxRefund:           (*big.Int)(dec.MaxRefund),
			SubmissionFeeRefund: (*big.Int)(dec.SubmissionFeeRefund),
		}

	case ArbitrumSubmitRetryableTxType:
		if dec.ChainID == nil {
			return errors.New("missing required field 'chainId' in transaction")
		}
		if dec.RequestId == nil {
			return errors.New("missing required field 'requestId' in transaction")
		}
		if dec.From == nil {
			return errors.New("missing required field 'from' in transaction")
		}
		if dec.L1BaseFee == nil {
			return errors.New("missing required field 'l1BaseFee' in transaction")
		}
		if dec.DepositValue == nil {
			return errors.New("missing required field 'depositValue' in transaction")
		}
		if dec.MaxFeePerGas == nil {
			return errors.New("missing required field 'maxFeePerGas' for txdata")
		}
		if dec.Gas == nil {
			return errors.New("missing required field 'gas' in txdata")
		}
		if dec.RetryTo == nil {
			return errors.New("missing required field 'retryTo' in txdata")
		}
		if dec.Beneficiary == nil {
			return errors.New("missing required field 'beneficiary' in transaction")
		}
		if dec.MaxSubmissionFee == nil {
			return errors.New("missing required field 'maxSubmissionFee' in transaction")
		}
		if dec.RefundTo == nil {
			return errors.New("missing required field 'refundTo' in transaction")
		}
		if dec.RetryValue == nil {
			return errors.New("missing required field 'retryValue' in transaction")
		}
		if dec.RetryData == nil {
			return errors.New("missing required field 'retryData' in transaction")
		}
		inner = &ArbitrumSubmitRetryableTx{
			ChainId:          (*big.Int)(dec.ChainID),
			RequestId:        *dec.RequestId,
			From:             *dec.From,
			L1BaseFee:        (*big.Int)(dec.L1BaseFee),
			DepositValue:     (*big.Int)(dec.DepositValue),
			GasFeeCap:        (*big.Int)(dec.MaxFeePerGas),
			Gas:              uint64(*dec.Gas),
			RetryTo:          dec.RetryTo,
			RetryValue:       (*big.Int)(dec.RetryValue),
			Beneficiary:      *dec.Beneficiary,
			MaxSubmissionFee: (*big.Int)(dec.MaxSubmissionFee),
			FeeRefundAddr:    *dec.RefundTo,
			RetryData:        *dec.RetryData,
		}
	case OptimismDepositTxType:
		inner = &OptimismDepositTx{
			ChainID: big.NewInt(0),
			From:    *dec.From,
			To:      *dec.To,
			Mint:    new(big.Int),
			Value:   (*big.Int)(dec.Value),
			Data:    *dec.Data,
		}
	default:
		return ErrTxTypeNotSupported
	}

	// Now set the inner transaction.
	t.setDecoded(inner, 0)

	// TODO: check hash here?
	return nil
}
