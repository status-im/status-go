package wallettypes

import (
	"bytes"
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	types2 "github.com/status-im/status-go/internal/crypto/types"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

type SendTxArgsVersion uint

const (
	SendTxArgsVersion0 SendTxArgsVersion = 0
	SendTxArgsVersion1 SendTxArgsVersion = 1
)

var (
	// ErrInvalidSendTxArgs is returned when the structure of SendTxArgs is ambigious.
	ErrInvalidSendTxArgs = errors.New("transaction arguments are invalid")
	// ErrUnexpectedArgs is returned when args are of unexpected length.
	ErrUnexpectedArgs = errors.New("unexpected args")
	//ErrInvalidTxSender is returned when selected account is different than From field.
	ErrInvalidTxSender = errors.New("transaction can only be send by its creator")
)

// PendingNonceProvider provides information about nonces.
type PendingNonceProvider interface {
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
}

// GasCalculator provides methods for estimating and pricing gas.
type GasCalculator interface {
	ethereum.GasEstimator
	ethereum.GasPricer
}

// SendTxArgs represents the arguments to submit a new transaction into the transaction pool.
// This struct is based on go-ethereum's type in internal/ethapi/api.go, but we have freedom
// over the exact layout of this struct.
type SendTxArgs struct {
	Version SendTxArgsVersion `json:"version"`

	From                 types2.Address  `json:"from"`
	To                   *types2.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	// We keep both "input" and "data" for backward compatibility.
	// "input" is a preferred field.
	// see `vendor/github.com/ethereum/go-ethereum/internal/ethapi/api.go:1107`
	Input types2.HexBytes `json:"input"`
	Data  types2.HexBytes `json:"data"`

	// additional data - version SendTxArgsVersion1
	FromChainID        uint64            `json:"fromChainID"`
	ToChainID          uint64            `json:"toChainID"`
	ValueIn            *hexutil.Big      `json:"valueIn"`
	ValueOut           *hexutil.Big      `json:"valueOut"`
	FromToken          *tokentypes.Token `json:"fromToken"`
	ToToken            *tokentypes.Token `json:"toToken"`
	ToContractAddress  types2.Address    `json:"toContractAddress"` // represents address of the contract that needs to be used in order to send assets, like ERC721 or ERC1155 tx
	SlippagePercentage float32           `json:"slippagePercentage"`
}

// Valid checks whether this structure is filled in correctly.
func (args SendTxArgs) Valid() bool {
	// if at least one of the fields is empty, it is a valid struct
	if isNilOrEmpty(args.Input) || isNilOrEmpty(args.Data) {
		return true
	}

	// we only allow both fields to present if they have the same data
	return bytes.Equal(args.Input, args.Data)
}

// IsDynamicFeeTx checks whether dynamic fee parameters are set for the tx
func (args SendTxArgs) IsDynamicFeeTx() bool {
	return args.MaxFeePerGas != nil && args.MaxPriorityFeePerGas != nil
}

// GetInput returns either Input or Data field's value dependent on what is filled.
func (args SendTxArgs) GetInput() types2.HexBytes {
	if !isNilOrEmpty(args.Input) {
		return args.Input
	}

	return args.Data
}

func (args SendTxArgs) ToTransactOpts(signerFn bind.SignerFn) *bind.TransactOpts {
	var gasFeeCap *big.Int
	if args.MaxFeePerGas != nil {
		gasFeeCap = (*big.Int)(args.MaxFeePerGas)
	}

	var gasTipCap *big.Int
	if args.MaxPriorityFeePerGas != nil {
		gasTipCap = (*big.Int)(args.MaxPriorityFeePerGas)
	}

	var nonce *big.Int
	if args.Nonce != nil {
		nonce = new(big.Int).SetUint64((uint64)(*args.Nonce))
	}

	var gasPrice *big.Int
	if args.GasPrice != nil {
		gasPrice = (*big.Int)(args.GasPrice)
	}

	var gasLimit uint64
	if args.Gas != nil {
		gasLimit = uint64(*args.Gas)
	}

	var noSend = false
	if signerFn == nil {
		// Dummy signerFn that returns the rawTx
		signerFn = func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
			return tx, nil
		}
		noSend = true
	}

	return &bind.TransactOpts{
		From:      common.Address(args.From),
		Signer:    signerFn,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Nonce:     nonce,
		NoSend:    noSend,
	}
}

func isNilOrEmpty(bytes types2.HexBytes) bool {
	return len(bytes) == 0
}
