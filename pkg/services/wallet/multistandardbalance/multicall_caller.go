package multistandardbalance

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/go-wallet-sdk/pkg/contracts/multicall3"
	"github.com/status-im/go-wallet-sdk/pkg/multicall"
)

var (
	tryAggregateSelector         = []byte{0xbc, 0xe3, 0x8b, 0xd7}
	tryBlockAndAggregateSelector = []byte{0x39, 0x95, 0x42, 0xe9}

	errMulticallOutputTooShort  = errors.New("multicall output: too short")
	errMulticallOutputBadOffset = errors.New("multicall output: offset or length out of range")
	errMulticallOutputBadBool   = errors.New("multicall output: improperly encoded boolean value")
	errMulticallOutputNotWords  = errors.New("multicall output: length is not a multiple of 32")
	errMulticallUnsupportedOpts = errors.New("multicall: pending and block hash calls are not supported")
)

const abiWordSize = 32

type multicallBackend interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error)
}

// multicallCaller is a multicall.Caller that encodes and decodes the Multicall3
// tryAggregate/tryBlockAndAggregate ABI by hand, without the reflection of the
// generated binding, and writes the calldata straight into the hex text of the
// eth_call argument. It sends the same eth_call request and returns the same
// values. Returned ReturnData slices alias the RPC response buffer.
type multicallCaller struct {
	address common.Address
	backend multicallBackend
}

var _ multicall.Caller = (*multicallCaller)(nil)

func newMulticallCaller(address common.Address, backend multicallBackend) *multicallCaller {
	return &multicallCaller{address: address, backend: backend}
}

func (c *multicallCaller) ViewTryBlockAndAggregate(opts *bind.CallOpts, requireSuccess bool, calls []multicall3.IMulticall3Call) (*big.Int, [32]byte, []multicall3.IMulticall3Result, error) {
	out, err := c.call(opts, tryBlockAndAggregateSelector, requireSuccess, calls)
	if err != nil {
		return nil, [32]byte{}, nil, err
	}
	return decodeTryBlockAndAggregateOutput(out)
}

func (c *multicallCaller) ViewTryAggregate(opts *bind.CallOpts, requireSuccess bool, calls []multicall3.IMulticall3Call) ([]multicall3.IMulticall3Result, error) {
	out, err := c.call(opts, tryAggregateSelector, requireSuccess, calls)
	if err != nil {
		return nil, err
	}
	return decodeTryAggregateOutput(out)
}

// call mirrors bind.BoundContract.Call: eth_call at opts.BlockNumber, and
// bind.ErrNoCode for an empty answer from an address without code.
func (c *multicallCaller) call(opts *bind.CallOpts, selector []byte, requireSuccess bool, calls []multicall3.IMulticall3Call) ([]byte, error) {
	if opts == nil {
		opts = new(bind.CallOpts)
	}
	if opts.Pending || opts.BlockHash != (common.Hash{}) {
		return nil, errMulticallUnsupportedOpts
	}
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	// Not reused across calls: the fallback client can return on a timeout while
	// a provider attempt that has yet to marshal the argument is still running.
	arg := json.RawMessage(appendEthCallArg(nil, opts.From, c.address, selector, requireSuccess, calls))
	var out hexutil.Bytes
	if err := c.backend.CallContext(ctx, &out, "eth_call", arg, toBlockNumArg(opts.BlockNumber)); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		code, err := c.backend.CodeAt(ctx, c.address, opts.BlockNumber)
		if err != nil {
			return nil, err
		}
		if len(code) == 0 {
			return nil, bind.ErrNoCode
		}
	}
	return out, nil
}

// toBlockNumArg is go-ethereum's ethclient block argument encoding.
func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	if number.Sign() >= 0 {
		return hexutil.EncodeBig(number)
	}
	return rpc.BlockNumber(number.Int64()).String()
}

// appendEthCallArg appends the eth_call transaction argument exactly as
// go-ethereum's ethclient marshals it, with the calldata hex-encoded in place.
func appendEthCallArg(buf []byte, from, to common.Address, selector []byte, requireSuccess bool, calls []multicall3.IMulticall3Call) []byte {
	const (
		fromKey  = `{"from":"0x`
		inputKey = `","input":"0x`
		toKey    = `","to":"0x`
		end      = `"}`
	)
	size := multicallInputSize(selector, calls)
	buf = slices.Grow(buf, len(fromKey)+2*common.AddressLength+len(inputKey)+2*size+len(toKey)+2*common.AddressLength+len(end))
	buf = append(buf, fromKey...)
	buf = hex.AppendEncode(buf, from[:])
	buf = append(buf, inputKey...)
	input := buf[len(buf) : len(buf)+2*size]
	putMulticallInput(input[:size], selector, requireSuccess, calls)
	// Expand to hex back to front so no byte is overwritten before it is read.
	const digits = "0123456789abcdef"
	for i := size - 1; i >= 0; i-- {
		b := input[i]
		input[2*i] = digits[b>>4]
		input[2*i+1] = digits[b&0x0f]
	}
	buf = buf[:len(buf)+2*size]
	buf = append(buf, toKey...)
	buf = hex.AppendEncode(buf, to[:])
	return append(buf, end...)
}

func paddedLen(n int) int {
	return (n + abiWordSize - 1) &^ (abiWordSize - 1)
}

func putWord(dst []byte, v int) {
	binary.BigEndian.PutUint64(dst[abiWordSize-8:abiWordSize], uint64(v))
}

func multicallInputSize(selector []byte, calls []multicall3.IMulticall3Call) int {
	size := len(selector) + 3*abiWordSize + len(calls)*abiWordSize
	for _, call := range calls {
		size += 3*abiWordSize + paddedLen(len(call.CallData))
	}
	return size
}

// encodeMulticallInput encodes `selector(bool requireSuccess, (address,bytes)[] calls)`,
// byte-identical to abi.Pack.
func encodeMulticallInput(selector []byte, requireSuccess bool, calls []multicall3.IMulticall3Call) []byte {
	buf := make([]byte, multicallInputSize(selector, calls))
	putMulticallInput(buf, selector, requireSuccess, calls)
	return buf
}

// putMulticallInput writes the encoding into buf, of exactly multicallInputSize bytes.
func putMulticallInput(buf []byte, selector []byte, requireSuccess bool, calls []multicall3.IMulticall3Call) {
	clear(buf)
	n := copy(buf, selector)
	if requireSuccess {
		buf[n+abiWordSize-1] = 1
	}
	putWord(buf[n+abiWordSize:], 2*abiWordSize)
	putWord(buf[n+2*abiWordSize:], len(calls))

	// Element offsets are relative to the first offset word.
	head := n + 3*abiWordSize
	tail := head + len(calls)*abiWordSize
	for i, call := range calls {
		putWord(buf[head+i*abiWordSize:], tail-head)
		copy(buf[tail+abiWordSize-common.AddressLength:], call.Target[:])
		putWord(buf[tail+abiWordSize:], 2*abiWordSize)
		putWord(buf[tail+2*abiWordSize:], len(call.CallData))
		copy(buf[tail+3*abiWordSize:], call.CallData)
		tail += 3*abiWordSize + paddedLen(len(call.CallData))
	}
}

// readSize reads the word at `at` as an offset or length, which must not exceed len(data).
func readSize(data []byte, at int) (int, error) {
	if at > len(data)-abiWordSize {
		return 0, errMulticallOutputTooShort
	}
	word := data[at : at+abiWordSize]
	for _, b := range word[:abiWordSize-8] {
		if b != 0 {
			return 0, errMulticallOutputBadOffset
		}
	}
	v := binary.BigEndian.Uint64(word[abiWordSize-8:])
	if v > uint64(len(data)) {
		return 0, errMulticallOutputBadOffset
	}
	return int(v), nil
}

func readBool(data []byte, at int) (bool, error) {
	if at > len(data)-abiWordSize {
		return false, errMulticallOutputTooShort
	}
	word := data[at : at+abiWordSize]
	for _, b := range word[:abiWordSize-1] {
		if b != 0 {
			return false, errMulticallOutputBadBool
		}
	}
	switch word[abiWordSize-1] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	}
	return false, errMulticallOutputBadBool
}

// decodeResults decodes the (bool,bytes)[] whose offset word is at `at`.
func decodeResults(data []byte, at int) ([]multicall3.IMulticall3Result, error) {
	start, err := readSize(data, at)
	if err != nil {
		return nil, err
	}
	count, err := readSize(data, start)
	if err != nil {
		return nil, err
	}
	head := start + abiWordSize
	if count > (len(data)-head)/abiWordSize {
		return nil, errMulticallOutputBadOffset
	}
	results := make([]multicall3.IMulticall3Result, count)
	for i := range results {
		offset, err := readSize(data, head+i*abiWordSize)
		if err != nil {
			return nil, err
		}
		tuple := head + offset
		success, err := readBool(data, tuple)
		if err != nil {
			return nil, err
		}
		bytesOffset, err := readSize(data, tuple+abiWordSize)
		if err != nil {
			return nil, err
		}
		lengthAt := tuple + bytesOffset
		length, err := readSize(data, lengthAt)
		if err != nil {
			return nil, err
		}
		begin := lengthAt + abiWordSize
		if length > len(data)-begin {
			return nil, errMulticallOutputBadOffset
		}
		results[i] = multicall3.IMulticall3Result{
			Success:    success,
			ReturnData: data[begin : begin+length : begin+length],
		}
	}
	return results, nil
}

// decodeTryAggregateOutput decodes `((bool,bytes)[] returnData)`.
func decodeTryAggregateOutput(out []byte) ([]multicall3.IMulticall3Result, error) {
	if len(out)%abiWordSize != 0 {
		return nil, errMulticallOutputNotWords
	}
	return decodeResults(out, 0)
}

// decodeTryBlockAndAggregateOutput decodes
// `(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)`.
func decodeTryBlockAndAggregateOutput(out []byte) (*big.Int, [32]byte, []multicall3.IMulticall3Result, error) {
	if len(out)%abiWordSize != 0 {
		return nil, [32]byte{}, nil, errMulticallOutputNotWords
	}
	if len(out) < 3*abiWordSize {
		return nil, [32]byte{}, nil, errMulticallOutputTooShort
	}
	results, err := decodeResults(out, 2*abiWordSize)
	if err != nil {
		return nil, [32]byte{}, nil, err
	}
	blockNumber := new(big.Int).SetBytes(out[:abiWordSize])
	var blockHash [32]byte
	copy(blockHash[:], out[abiWordSize:2*abiWordSize])
	return blockNumber, blockHash, results, nil
}
