// Package urlookup performs ENSv2 name resolutions through the Universal
// Resolver.
//
// The functions in this package take a bind.ContractBackend so the caller
// controls the chain context and any transport wrapping. In practice the
// backend should be wrapped with ccipread.New so OffchainLookup reverts are
// handled transparently.
package urlookup

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/internal/contracts/universalresolver"
	"github.com/status-im/status-go/services/ens/dnsname"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

// ethCoinType is the SLIP-44 coinType for Ethereum mainnet.
const ethCoinType = 60

// ErrEmptyResolverData is returned when the underlying resolver returned no
// bytes for a record (e.g. an unset addr / contenthash / pubkey).
var ErrEmptyResolverData = errors.New("urlookup: resolver returned no data")

// Resolver returns the resolver address that the Universal Resolver picks for
// the given ENS name. It works by issuing addr(node, 60) and reading the
// resolver field of UR.resolve()'s return.
func Resolver(ctx context.Context, backend bind.ContractBackend, urAddress common.Address, name string) (common.Address, error) {
	dns, node, err := prepareName(name)
	if err != nil {
		return common.Address{}, err
	}
	callData, err := packAddrMulticoin(node, ethCoinType)
	if err != nil {
		return common.Address{}, err
	}
	result, err := newUR(urAddress, backend).Resolve(callOpts(ctx), dns, callData)
	if err != nil {
		return common.Address{}, err
	}
	return result.Resolver, nil
}

// Address resolves the given ENS name to its Ethereum mainnet address via
// the multicoin addr(bytes32,uint256) profile with coinType=60.
func Address(ctx context.Context, backend bind.ContractBackend, urAddress common.Address, name string) (common.Address, error) {
	dns, node, err := prepareName(name)
	if err != nil {
		return common.Address{}, err
	}
	callData, err := packAddrMulticoin(node, ethCoinType)
	if err != nil {
		return common.Address{}, err
	}
	result, err := newUR(urAddress, backend).Resolve(callOpts(ctx), dns, callData)
	if err != nil {
		return common.Address{}, err
	}
	if len(result.Data) == 0 {
		return common.Address{}, ErrEmptyResolverData
	}
	raw, err := unpackBytes(result.Data)
	if err != nil {
		return common.Address{}, err
	}
	if len(raw) != common.AddressLength {
		return common.Address{}, fmt.Errorf("urlookup: unexpected address length %d", len(raw))
	}
	return common.BytesToAddress(raw), nil
}

// Contenthash resolves the EIP-1577 contenthash record for the given name.
// Returns (nil, nil) when the record is unset; returns an error only on
// transport / decoding failure.
func Contenthash(ctx context.Context, backend bind.ContractBackend, urAddress common.Address, name string) ([]byte, error) {
	dns, node, err := prepareName(name)
	if err != nil {
		return nil, err
	}
	callData, err := packBytes32(selContenthash, node)
	if err != nil {
		return nil, err
	}
	result, err := newUR(urAddress, backend).Resolve(callOpts(ctx), dns, callData)
	if err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, nil
	}
	return unpackBytes(result.Data)
}

// PubkeyHex resolves the secp256k1 pubkey record and returns it in
// uncompressed hex form prefixed with "0x04".
func PubkeyHex(ctx context.Context, backend bind.ContractBackend, urAddress common.Address, name string) (string, error) {
	x, y, err := Pubkey(ctx, backend, urAddress, name)
	if err != nil {
		return "", err
	}
	return "0x04" + hex.EncodeToString(x[:]) + hex.EncodeToString(y[:]), nil
}

// Pubkey resolves the secp256k1 pubkey record and returns the raw (x, y)
// coordinates.
func Pubkey(ctx context.Context, backend bind.ContractBackend, urAddress common.Address, name string) ([32]byte, [32]byte, error) {
	dns, node, err := prepareName(name)
	if err != nil {
		return [32]byte{}, [32]byte{}, err
	}
	callData, err := packBytes32(selPubkey, node)
	if err != nil {
		return [32]byte{}, [32]byte{}, err
	}
	result, err := newUR(urAddress, backend).Resolve(callOpts(ctx), dns, callData)
	if err != nil {
		return [32]byte{}, [32]byte{}, err
	}
	if len(result.Data) == 0 {
		return [32]byte{}, [32]byte{}, ErrEmptyResolverData
	}
	return unpackPubkey(result.Data)
}

// Reverse resolves an address back to its primary ENS name.
func Reverse(ctx context.Context, backend bind.ContractBackend, urAddress common.Address, address common.Address) (string, error) {
	res, err := newUR(urAddress, backend).Reverse(callOpts(ctx), address.Bytes(), big.NewInt(ethCoinType))
	if err != nil {
		return "", err
	}
	return res.Primary, nil
}

// --- internal helpers ------------------------------------------------------

func prepareName(name string) ([]byte, common.Hash, error) {
	dns, err := dnsname.Encode(name)
	if err != nil {
		return nil, common.Hash{}, err
	}
	return dns, walletCommon.NameHash(name), nil
}

func newUR(addr common.Address, backend bind.ContractBackend) *universalresolver.UniversalResolver {
	return universalresolver.NewUniversalResolver(addr, backend)
}

func callOpts(ctx context.Context) *bind.CallOpts {
	return &bind.CallOpts{Context: ctx, Pending: false}
}

var (
	bytes32Ty, _ = abi.NewType("bytes32", "", nil)
	uint256Ty, _ = abi.NewType("uint256", "", nil)
	bytesTy, _   = abi.NewType("bytes", "", nil)

	argsBytes32          = abi.Arguments{{Type: bytes32Ty}}
	argsBytes32Uint256   = abi.Arguments{{Type: bytes32Ty}, {Type: uint256Ty}}
	argsReturnBytes      = abi.Arguments{{Type: bytesTy}}
	argsReturnTwoBytes32 = abi.Arguments{{Type: bytes32Ty}, {Type: bytes32Ty}}

	selAddrMulticoin = methodSelector("addr(bytes32,uint256)")
	selPubkey        = methodSelector("pubkey(bytes32)")
	selContenthash   = methodSelector("contenthash(bytes32)")
)

func methodSelector(sig string) [4]byte {
	h := crypto.Keccak256([]byte(sig))
	var out [4]byte
	copy(out[:], h[:4])
	return out
}

func packAddrMulticoin(node common.Hash, coinType uint64) ([]byte, error) {
	body, err := argsBytes32Uint256.Pack(node, new(big.Int).SetUint64(coinType))
	if err != nil {
		return nil, err
	}
	return prependSelector(selAddrMulticoin, body), nil
}

// packBytes32 packs a single-bytes32-arg call (pubkey, contenthash, addr legacy).
func packBytes32(selector [4]byte, node common.Hash) ([]byte, error) {
	body, err := argsBytes32.Pack(node)
	if err != nil {
		return nil, err
	}
	return prependSelector(selector, body), nil
}

func prependSelector(selector [4]byte, body []byte) []byte {
	out := make([]byte, 0, 4+len(body))
	out = append(out, selector[:]...)
	out = append(out, body...)
	return out
}

func unpackBytes(data []byte) ([]byte, error) {
	values, err := argsReturnBytes.Unpack(data)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, errors.New("urlookup: unexpected return arity")
	}
	b, ok := values[0].([]byte)
	if !ok {
		return nil, errors.New("urlookup: return not bytes")
	}
	return b, nil
}

func unpackPubkey(data []byte) ([32]byte, [32]byte, error) {
	values, err := argsReturnTwoBytes32.Unpack(data)
	if err != nil {
		return [32]byte{}, [32]byte{}, err
	}
	if len(values) != 2 {
		return [32]byte{}, [32]byte{}, errors.New("urlookup: pubkey arity")
	}
	x, ok := values[0].([32]byte)
	if !ok {
		return [32]byte{}, [32]byte{}, errors.New("urlookup: pubkey.x not bytes32")
	}
	y, ok := values[1].([32]byte)
	if !ok {
		return [32]byte{}, [32]byte{}, errors.New("urlookup: pubkey.y not bytes32")
	}
	return x, y, nil
}
