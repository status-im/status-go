package universalresolver

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// UniversalResolverABI exposes the subset of the ENS Universal Resolver used for
// forward (`resolve`) and reverse (`reverse`) resolution.
// source: https://docs.ens.domains/resolvers/universal/
const UniversalResolverABI = `[` +
	`{"inputs":[{"internalType":"bytes","name":"name","type":"bytes"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"resolve","outputs":[{"internalType":"bytes","name":"","type":"bytes"},{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},` +
	`{"inputs":[{"internalType":"bytes","name":"lookupAddress","type":"bytes"},{"internalType":"uint256","name":"coinType","type":"uint256"}],"name":"reverse","outputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"address","name":"","type":"address"},{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}` +
	`]`

// UniversalResolver is a minimal read-only binding around the ENS Universal Resolver.
type UniversalResolver struct {
	contract *bind.BoundContract
}

// NewUniversalResolver creates a binding bound to the given address.
func NewUniversalResolver(address common.Address, backend bind.ContractBackend) (*UniversalResolver, error) {
	parsed, err := abi.JSON(strings.NewReader(UniversalResolverABI))
	if err != nil {
		return nil, err
	}
	contract := bind.NewBoundContract(address, parsed, backend, backend, backend)
	return &UniversalResolver{contract: contract}, nil
}

// Resolve performs forward resolution: it executes the encoded resolver call `data`
// (e.g. `addr(bytes32)`) for the DNS-encoded `name` and returns the raw result
// together with the resolver that answered.
func (u *UniversalResolver) Resolve(opts *bind.CallOpts, name []byte, data []byte) ([]byte, common.Address, error) {
	var out []interface{}
	if err := u.contract.Call(opts, &out, "resolve", name, data); err != nil {
		return nil, common.Address{}, err
	}
	result := *abi.ConvertType(out[0], new([]byte)).(*[]byte)
	resolverAddr := *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	return result, resolverAddr, nil
}

// Reverse performs reverse resolution for the given address bytes and coin type
// (ENSIP-11/19), returning the primary name and the resolvers that answered.
func (u *UniversalResolver) Reverse(opts *bind.CallOpts, lookupAddress []byte, coinType *big.Int) (string, common.Address, common.Address, error) {
	var out []interface{}
	if err := u.contract.Call(opts, &out, "reverse", lookupAddress, coinType); err != nil {
		return "", common.Address{}, common.Address{}, err
	}
	name := *abi.ConvertType(out[0], new(string)).(*string)
	resolverAddr := *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	reverseResolverAddr := *abi.ConvertType(out[2], new(common.Address)).(*common.Address)
	return name, resolverAddr, reverseResolverAddr, nil
}

// DNSEncode converts a dot-separated ENS name into DNS wire format, the encoding
// expected by the Universal Resolver's `resolve` method (e.g. "alice.stateofus.eth"
// -> 0x05616c696365...00).
func DNSEncode(name string) []byte {
	var buf []byte
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 {
			continue
		}
		buf = append(buf, byte(len(label)))
		buf = append(buf, []byte(label)...)
	}
	return append(buf, 0x00)
}
