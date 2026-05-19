package universalresolver

import (
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// abiJSON is the subset of the Universal Resolver ABI used by this binding.
// Source: ensdomains/ens-contracts AbstractUniversalResolver.sol.
const abiJSON = `[
    {"inputs":[{"internalType":"bytes","name":"name","type":"bytes"},{"internalType":"bytes","name":"data","type":"bytes"}],"name":"resolve","outputs":[{"internalType":"bytes","name":"","type":"bytes"},{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
    {"inputs":[{"internalType":"bytes","name":"lookupAddress","type":"bytes"},{"internalType":"uint256","name":"coinType","type":"uint256"}],"name":"reverse","outputs":[{"internalType":"string","name":"","type":"string"},{"internalType":"address","name":"","type":"address"},{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
]`

var parsedABI abi.ABI

func init() {
	a, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic("universalresolver: invalid ABI: " + err.Error())
	}
	parsedABI = a
}

// UniversalResolver is a minimal binding for the ENSv2 Universal Resolver.
type UniversalResolver struct {
	address  common.Address
	contract *bind.BoundContract
}

// NewUniversalResolver binds to the UR at the given address using the supplied
// backend. The backend should typically be a CCIP-Read-aware caller so that
// offchain lookups resolve transparently.
func NewUniversalResolver(address common.Address, backend bind.ContractBackend) *UniversalResolver {
	return &UniversalResolver{
		address:  address,
		contract: bind.NewBoundContract(address, parsedABI, backend, backend, backend),
	}
}

// Address returns the bound contract address.
func (u *UniversalResolver) Address() common.Address { return u.address }

// ResolveResult holds the decoded return values of resolve().
type ResolveResult struct {
	Data     []byte
	Resolver common.Address
}

// Resolve calls UR.resolve(name, data). name must be DNS-wire-encoded.
func (u *UniversalResolver) Resolve(opts *bind.CallOpts, name []byte, data []byte) (*ResolveResult, error) {
	var out []interface{}
	if err := u.contract.Call(opts, &out, "resolve", name, data); err != nil {
		return nil, err
	}
	if len(out) != 2 {
		return nil, errInvalidReturn
	}
	return &ResolveResult{
		Data:     *abi.ConvertType(out[0], new([]byte)).(*[]byte),
		Resolver: *abi.ConvertType(out[1], new(common.Address)).(*common.Address),
	}, nil
}

// ReverseResult holds the decoded return values of reverse().
type ReverseResult struct {
	Primary         string
	Resolver        common.Address
	ReverseResolver common.Address
}

// Reverse calls UR.reverse(lookupAddress, coinType).
func (u *UniversalResolver) Reverse(opts *bind.CallOpts, lookupAddress []byte, coinType *big.Int) (*ReverseResult, error) {
	var out []interface{}
	if err := u.contract.Call(opts, &out, "reverse", lookupAddress, coinType); err != nil {
		return nil, err
	}
	if len(out) != 3 {
		return nil, errInvalidReturn
	}
	return &ReverseResult{
		Primary:         *abi.ConvertType(out[0], new(string)).(*string),
		Resolver:        *abi.ConvertType(out[1], new(common.Address)).(*common.Address),
		ReverseResolver: *abi.ConvertType(out[2], new(common.Address)).(*common.Address),
	}, nil
}

var errInvalidReturn = errors.New("universalresolver: unexpected number of return values")
