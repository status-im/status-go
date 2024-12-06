package balancefetcher

import (
	"context"
	"errors"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/contracts/ethscan"
	"github.com/status-im/status-go/params"

	mock_contracts "github.com/status-im/status-go/contracts/mock"
	"github.com/status-im/status-go/rpc/chain"
	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	mock_network "github.com/status-im/status-go/rpc/network/mock"
	w_common "github.com/status-im/status-go/services/wallet/common"
)

type FakeBalanceScanner struct {
	etherBalances map[common.Address]*big.Int
	tokenBalances map[common.Address]map[common.Address]*big.Int
}

func (f *FakeBalanceScanner) EtherBalances(opts *bind.CallOpts, addresses []common.Address) ([]ethscan.BalanceScannerResult, error) {
	result := make([]ethscan.BalanceScannerResult, 0, len(addresses))

	for _, address := range addresses {
		balance, ok := f.etherBalances[address]
		if !ok {
			result = append(result, ethscan.BalanceScannerResult{
				Success: false,
				Data:    []byte{},
			})
		} else {
			result = append(result, ethscan.BalanceScannerResult{
				Success: true,
				Data:    balance.Bytes(),
			})
		}
	}

	return result, nil
}

func (f *FakeBalanceScanner) TokenBalances(opts *bind.CallOpts, addresses []common.Address, tokenAddress common.Address) ([]ethscan.BalanceScannerResult, error) {
	result := make([]ethscan.BalanceScannerResult, 0, len(addresses))

	for _, address := range addresses {
		balances, ok := f.tokenBalances[address]
		if !ok {
			result = append(result, ethscan.BalanceScannerResult{
				Success: false,
				Data:    []byte{},
			})
		} else {
			balance, ok := balances[tokenAddress]
			if !ok {
				result = append(result, ethscan.BalanceScannerResult{
					Success: false,
					Data:    []byte{},
				})
			} else {
				result = append(result, ethscan.BalanceScannerResult{
					Success: true,
					Data:    balance.Bytes(),
				})
			}
		}
	}

	return result, nil
}

func (f *FakeBalanceScanner) TokensBalance(opts *bind.CallOpts, owner common.Address, contracts []common.Address) ([]ethscan.BalanceScannerResult, error) {
	result := make([]ethscan.BalanceScannerResult, 0, len(contracts))

	for _, contract := range contracts {
		balances, ok := f.tokenBalances[owner]
		if !ok {
			result = append(result, ethscan.BalanceScannerResult{
				Success: false,
				Data:    []byte{},
			})
		} else {
			balance, ok := balances[contract]
			if !ok {
				result = append(result, ethscan.BalanceScannerResult{
					Success: false,
					Data:    []byte{},
				})
			} else {
				result = append(result, ethscan.BalanceScannerResult{
					Success: true,
					Data:    balance.Bytes(),
				})
			}
		}
	}

	return result, nil
}

type FakeERC20Caller struct {
	accountBalances map[common.Address]*big.Int
}

func (f *FakeERC20Caller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	balance, ok := f.accountBalances[account]
	if !ok {
		return nil, errors.New("account not found")
	}

	return balance, nil
}

func (f *FakeERC20Caller) Name(opts *bind.CallOpts) (string, error) {
	return "TestToken", nil
}

func (f *FakeERC20Caller) Symbol(opts *bind.CallOpts) (string, error) {
	return "TT", nil
}

func (f *FakeERC20Caller) Decimals(opts *bind.CallOpts) (uint8, error) {
	return 18, nil
}

func TestBalanceFetcherFetchBalancesForChainNativeAndTokensWithScanContract(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	ctx := context.Background()
	accounts := []common.Address{
		common.HexToAddress("0x1234567890abcdef"),
		common.HexToAddress("0xabcdef1234567890"),
	}
	tokens := []common.Address{
		NativeChainAddress,
		common.HexToAddress("0xabcdef1234567890"),
		common.HexToAddress("0x0987654321fedcba"),
	}
	var atBlock *big.Int // nil triggers using a scan contract

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	networkManager := mock_network.NewMockManagerInterface(ctrl)
	networkManager.EXPECT().GetAll().Return([]*params.Network{
		{
			ChainID: w_common.EthereumMainnet,
		},
	}, nil).AnyTimes()

	chainClient := mock_client.NewMockClientInterface(ctrl)
	chainClient.EXPECT().NetworkID().Return(w_common.EthereumMainnet).AnyTimes()
	expectedEthBalances := map[common.Address]*big.Int{
		accounts[0]: big.NewInt(100),
		accounts[1]: big.NewInt(200),
	}

	expectedTokenBalances := map[common.Address]map[common.Address]*big.Int{
		accounts[0]: {
			tokens[1]: big.NewInt(1000),
			tokens[2]: big.NewInt(2000),
		},
		accounts[1]: {
			tokens[1]: big.NewInt(3000),
			tokens[2]: big.NewInt(4000),
		},
	}

	expectedBalances := map[common.Address]map[common.Address]*hexutil.Big{
		accounts[0]: {
			tokens[0]: (*hexutil.Big)(expectedEthBalances[accounts[0]]),
			tokens[1]: (*hexutil.Big)(expectedTokenBalances[accounts[0]][tokens[1]]),
			tokens[2]: (*hexutil.Big)(expectedTokenBalances[accounts[0]][tokens[2]]),
		},
		accounts[1]: {
			tokens[0]: (*hexutil.Big)(expectedEthBalances[accounts[1]]),
			tokens[1]: (*hexutil.Big)(expectedTokenBalances[accounts[1]][tokens[1]]),
			tokens[2]: (*hexutil.Big)(expectedTokenBalances[accounts[1]][tokens[2]]),
		},
	}

	contractMaker := mock_contracts.NewMockContractMakerIface(ctrl)
	contractMaker.EXPECT().NewEthScan(w_common.EthereumMainnet).Return(&FakeBalanceScanner{
		etherBalances: expectedEthBalances,
		tokenBalances: expectedTokenBalances,
	}, uint(0), nil).AnyTimes()
	bf := NewDefaultBalanceFetcher(contractMaker)

	// Fetch native balances and token balances using scan contract
	balances, err := bf.fetchBalancesForChain(ctx, chainClient, accounts, tokens, atBlock)

	require.NoError(t, err)
	require.Equal(t, expectedBalances, balances)
}

func TestBalanceFetcherFetchBalancesForChainTokensWithTokenContracts(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	ctx := context.Background()
	accounts := []common.Address{
		common.HexToAddress("0x1234567890abcdef"),
		common.HexToAddress("0xabcdef1234567890"),
	}
	tokens := []common.Address{
		common.HexToAddress("0xabcdef1234567890"),
		common.HexToAddress("0x0987654321fedcba"),
	}
	atBlock := big.NewInt(0) // will trigger using a token contract

	ctrl := gomock.NewController(t)
	networkManager := mock_network.NewMockManagerInterface(ctrl)
	networkManager.EXPECT().GetAll().Return([]*params.Network{
		{
			ChainID: w_common.EthereumMainnet,
		},
	}, nil).AnyTimes()

	chainClient := mock_client.NewMockClientInterface(ctrl)
	chainClient.EXPECT().NetworkID().Return(w_common.EthereumMainnet).AnyTimes()
	chainClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), atBlock).Return([]byte{}, nil).AnyTimes()

	expectedTokenBalances := map[common.Address]map[common.Address]*big.Int{
		tokens[0]: {
			accounts[0]: big.NewInt(1000),
			accounts[1]: big.NewInt(2000),
		},
		tokens[1]: {
			accounts[0]: big.NewInt(3000),
			accounts[1]: big.NewInt(4000),
		},
	}

	expectedBalances := map[common.Address]map[common.Address]*hexutil.Big{
		accounts[0]: {
			tokens[0]: (*hexutil.Big)(expectedTokenBalances[tokens[0]][accounts[0]]),
			tokens[1]: (*hexutil.Big)(expectedTokenBalances[tokens[1]][accounts[0]]),
		},
		accounts[1]: {
			tokens[0]: (*hexutil.Big)(expectedTokenBalances[tokens[0]][accounts[1]]),
			tokens[1]: (*hexutil.Big)(expectedTokenBalances[tokens[1]][accounts[1]]),
		},
	}

	contractMaker := mock_contracts.NewMockContractMakerIface(ctrl)
	contractMaker.EXPECT().NewEthScan(w_common.EthereumMainnet).Return(&FakeBalanceScanner{}, uint(0), nil).Times(1)
	for _, token := range tokens {
		contractMaker.EXPECT().NewERC20Caller(w_common.EthereumMainnet, token).Return(&FakeERC20Caller{
			accountBalances: expectedTokenBalances[token],
		}, nil).AnyTimes()
	}
	bf := NewDefaultBalanceFetcher(contractMaker)

	// Fetch token balances using tokens contracts
	balances, err := bf.fetchBalancesForChain(ctx, chainClient, accounts, tokens, atBlock)

	require.NoError(t, err)
	require.Equal(t, expectedBalances, balances)
}

func TestBalanceFetcherGetBalancesAtByChain(t *testing.T) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	ctx := context.Background()
	accounts := []common.Address{
		common.HexToAddress("0x1234567890abcdef"),
		common.HexToAddress("0xabcdef1234567890"),
	}
	tokens := []common.Address{
		NativeChainAddress,
		common.HexToAddress("0xabcdef1234567890"),
		common.HexToAddress("0x0987654321fedcba"),
	}
	var atBlock *big.Int // nil triggers using a scan contract
	atBlocks := map[uint64]*big.Int{
		w_common.EthereumMainnet: atBlock, // nil triggers using a scan contract
		w_common.OptimismMainnet: atBlock, // nil triggers using a scan contract
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	networkManager := mock_network.NewMockManagerInterface(ctrl)
	networkManager.EXPECT().GetAll().Return([]*params.Network{
		{
			ChainID: w_common.EthereumMainnet,
		},
		{
			ChainID: w_common.OptimismMainnet,
		},
		{
			ChainID: w_common.ArbitrumMainnet,
		},
	}, nil).AnyTimes()

	chainClient := mock_client.NewMockClientInterface(ctrl)
	chainClient.EXPECT().NetworkID().Return(w_common.EthereumMainnet).AnyTimes()
	chainClientOpt := mock_client.NewMockClientInterface(ctrl)
	chainClientOpt.EXPECT().NetworkID().Return(w_common.OptimismMainnet).AnyTimes()
	chainClientArb := mock_client.NewMockClientInterface(ctrl)
	chainClientArb.EXPECT().NetworkID().Return(w_common.ArbitrumMainnet).AnyTimes()

	expectedEthBalances := map[common.Address]*big.Int{
		accounts[0]: big.NewInt(100),
		accounts[1]: big.NewInt(200),
	}

	expectedEthOptBalances := map[common.Address]*big.Int{
		accounts[0]: big.NewInt(300),
		accounts[1]: big.NewInt(400),
	}

	expectedTokenBalances := map[common.Address]map[common.Address]*big.Int{
		accounts[0]: {
			tokens[1]: big.NewInt(1000),
			tokens[2]: big.NewInt(2000),
		},
		accounts[1]: {
			tokens[1]: big.NewInt(3000),
			tokens[2]: big.NewInt(4000),
		},
	}

	expectedTokenOptBalances := map[common.Address]map[common.Address]*big.Int{
		accounts[0]: {
			tokens[1]: big.NewInt(5000),
			tokens[2]: big.NewInt(6000),
		},
	}

	expectedBalances := map[uint64]map[common.Address]map[common.Address]*hexutil.Big{
		w_common.EthereumMainnet: {
			accounts[0]: {
				tokens[0]: (*hexutil.Big)(expectedEthBalances[accounts[0]]),
				tokens[1]: (*hexutil.Big)(expectedTokenBalances[accounts[0]][tokens[1]]),
				tokens[2]: (*hexutil.Big)(expectedTokenBalances[accounts[0]][tokens[2]]),
			},
			accounts[1]: {
				tokens[0]: (*hexutil.Big)(expectedEthBalances[accounts[1]]),
				tokens[1]: (*hexutil.Big)(expectedTokenBalances[accounts[1]][tokens[1]]),
				tokens[2]: (*hexutil.Big)(expectedTokenBalances[accounts[1]][tokens[2]]),
			},
		},
		w_common.OptimismMainnet: {
			accounts[0]: {
				tokens[0]: (*hexutil.Big)(expectedEthOptBalances[accounts[0]]),
				tokens[1]: (*hexutil.Big)(expectedTokenOptBalances[accounts[0]][tokens[1]]),
				tokens[2]: (*hexutil.Big)(expectedTokenOptBalances[accounts[0]][tokens[2]]),
			},
			accounts[1]: {
				tokens[0]: (*hexutil.Big)(expectedEthOptBalances[accounts[1]]),
			},
		},
	}

	contractMaker := mock_contracts.NewMockContractMakerIface(ctrl)
	contractMaker.EXPECT().NewEthScan(w_common.EthereumMainnet).Return(&FakeBalanceScanner{
		etherBalances: expectedEthBalances,
		tokenBalances: expectedTokenBalances,
	}, uint(0), nil).AnyTimes()
	contractMaker.EXPECT().NewEthScan(w_common.OptimismMainnet).Return(&FakeBalanceScanner{
		etherBalances: expectedEthOptBalances,
		tokenBalances: expectedTokenOptBalances,
	}, uint(0), nil).AnyTimes()
	contractMaker.EXPECT().NewEthScan(w_common.ArbitrumMainnet).Return(nil, uint(0), errors.New("no scan contract")).AnyTimes()

	bf := NewDefaultBalanceFetcher(contractMaker)

	// Fetch native balances and token balances using scan contract for Ethereum Mainnet and Optimism Mainnet
	chainClients := map[uint64]chain.ClientInterface{
		w_common.EthereumMainnet: chainClient,
		w_common.OptimismMainnet: chainClientOpt,
	}
	balances, err := bf.GetBalancesAtByChain(ctx, chainClients, accounts, tokens, atBlocks)

	require.NoError(t, err)
	require.Equal(t, expectedBalances, balances)

	// Test errors
	chainClients = map[uint64]chain.ClientInterface{
		w_common.ArbitrumMainnet: chainClientArb,
	}
	balances, err = bf.GetBalancesAtByChain(ctx, chainClients, nil, nil, atBlocks)
	require.Error(t, err)
	require.ErrorContains(t, err, errScanningContract.Error())
	require.Nil(t, balances)
}
