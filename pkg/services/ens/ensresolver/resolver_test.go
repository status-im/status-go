package ensresolver

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/contracts"
	"github.com/status-im/status-go/internal/contracts/universalresolver"
	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

const (
	testChainID        = walletCommon.EthereumSepolia
	testStatusUsername = "alice.stateofus.eth"
)

var (
	errUnreachableName = errors.New("execution reverted")

	testPublicResolver = common.HexToAddress("0x8FADE66B79cC9f707aB26799354482EB93a5B7dD")
	testAccount        = common.HexToAddress("0x44ddd47A0c7681A5b0fa080a56CBB7701db4BB43")
	testPubkeyX        = common.HexToHash("0x68c2095d750b905f1e6482190748db0b8695dd840b0a6ce4040ca7dea9dae406")
	testPubkeyY        = common.HexToHash("0xe379dffb98963e540582ac48b9368a37ad66d9adfcd2993f9958097efe1be6bc")
)

// fakeChain answers eth_call the way a chain does when the Universal Resolver
// cannot reach names that the ENS v1 registry still serves.
type fakeChain struct {
	ethclient.EthClientInterface

	universalResolver func(data []byte) ([]byte, error)
	registryResolvers map[common.Hash]common.Address
	registryCalls     int
}

func (f *fakeChain) EthClient(uint64) (ethclient.EthClientInterface, error) {
	return f, nil
}

func (f *fakeChain) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return []byte{0x60}, nil
}

func (f *fakeChain) CallContract(_ context.Context, call ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	universalResolverAddr, err := universalresolver.ContractAddress(testChainID)
	if err != nil {
		return nil, err
	}

	switch *call.To {
	case universalResolverAddr:
		return f.universalResolver(call.Data)
	case ensRegistryAddress:
		f.registryCalls++
		node := common.BytesToHash(call.Data[4:])
		return common.LeftPadBytes(f.registryResolvers[node].Bytes(), 32), nil
	case testPublicResolver:
		return publicResolverAnswer(call.Data)
	}
	return nil, errors.New("unexpected call to " + call.To.Hex())
}

func publicResolverAnswer(data []byte) ([]byte, error) {
	switch {
	case bytes.HasPrefix(data, ensRecordMethods.Methods["addr"].ID):
		return ensRecordMethods.Methods["addr"].Outputs.Pack(testAccount)
	case bytes.HasPrefix(data, ensRecordMethods.Methods["pubkey"].ID):
		return ensRecordMethods.Methods["pubkey"].Outputs.Pack([32]byte(testPubkeyX), [32]byte(testPubkeyY))
	}
	return nil, errors.New("unexpected resolver call")
}

func unreachable([]byte) ([]byte, error) {
	return nil, errUnreachableName
}

func universalResolverABI(t *testing.T) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(universalresolver.UniversalResolverABI))
	require.NoError(t, err)
	return parsed
}

func newTestResolver(chain *fakeChain) *EnsResolver {
	return &EnsResolver{
		contractMaker:   contracts.NewContractMaker(chain),
		ethClientGetter: chain,
		quit:            make(chan struct{}),
	}
}

func registryWith(username string) map[common.Hash]common.Address {
	return map[common.Hash]common.Address{walletCommon.NameHash(username): testPublicResolver}
}

func TestStatusUsernameResolvesThroughRegistryWhenUniversalResolverFails(t *testing.T) {
	chain := &fakeChain{universalResolver: unreachable, registryResolvers: registryWith(testStatusUsername)}
	resolver := newTestResolver(chain)
	ctx := context.Background()

	addr, err := resolver.AddressOf(ctx, testChainID, testStatusUsername)
	require.NoError(t, err)
	require.Equal(t, testAccount, *addr)

	pubkey, err := resolver.PublicKeyOf(ctx, testChainID, testStatusUsername)
	require.NoError(t, err)
	require.Equal(t, "0x04"+testPubkeyX.Hex()[2:]+testPubkeyY.Hex()[2:], pubkey)

	resolverAddr, err := resolver.Resolver(ctx, testChainID, testStatusUsername)
	require.NoError(t, err)
	require.Equal(t, testPublicResolver, *resolverAddr)
}

func TestUniversalResolverAnswerIsUsedWhenItSucceeds(t *testing.T) {
	universalAccount := common.HexToAddress("0x1111111111111111111111111111111111111111")
	chain := &fakeChain{
		universalResolver: func([]byte) ([]byte, error) {
			record, err := ensRecordMethods.Methods["addr"].Outputs.Pack(universalAccount)
			if err != nil {
				return nil, err
			}
			return universalResolverABI(t).Methods["resolve"].Outputs.Pack(record, testPublicResolver)
		},
		registryResolvers: registryWith(testStatusUsername),
	}

	addr, err := newTestResolver(chain).AddressOf(context.Background(), testChainID, testStatusUsername)
	require.NoError(t, err)
	require.Equal(t, universalAccount, *addr)
	require.Zero(t, chain.registryCalls)
}

func TestOtherNamesDoNotFallBackToRegistry(t *testing.T) {
	const username = "alice.eth"
	chain := &fakeChain{universalResolver: unreachable, registryResolvers: registryWith(username)}

	_, err := newTestResolver(chain).AddressOf(context.Background(), testChainID, username)
	require.ErrorIs(t, err, errUnreachableName)
	require.Zero(t, chain.registryCalls)
}

func TestStatusUsernameWithoutRegistryResolverKeepsUniversalResolverError(t *testing.T) {
	chain := &fakeChain{universalResolver: unreachable}

	_, err := newTestResolver(chain).AddressOf(context.Background(), testChainID, testStatusUsername)
	require.ErrorIs(t, err, errUnreachableName)
}
