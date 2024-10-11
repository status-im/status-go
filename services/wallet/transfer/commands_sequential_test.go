package transfer

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/status-im/status-go/contracts"
	"github.com/status-im/status-go/services/wallet/blockchainstate"
	"github.com/status-im/status-go/t/utils"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/mock/gomock"
	"golang.org/x/exp/slices" // since 1.21, this is in the standard library

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/contracts/balancechecker"
	"github.com/status-im/status-go/contracts/ethscan"
	"github.com/status-im/status-go/contracts/ierc20"
	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	multicommon "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/params"
	statusRpc "github.com/status-im/status-go/rpc"
	ethclient "github.com/status-im/status-go/rpc/chain/ethclient"
	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/balance"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/community"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/transactions"
	"github.com/status-im/status-go/walletdatabase"
)

type TestClient struct {
	t *testing.T
	// [][block, newBalance, nonceDiff]
	balances                       map[common.Address][][]int
	outgoingERC20Transfers         map[common.Address][]testERC20Transfer
	incomingERC20Transfers         map[common.Address][]testERC20Transfer
	outgoingERC1155SingleTransfers map[common.Address][]testERC20Transfer
	incomingERC1155SingleTransfers map[common.Address][]testERC20Transfer
	balanceHistory                 map[common.Address]map[uint64]*big.Int
	tokenBalanceHistory            map[common.Address]map[common.Address]map[uint64]*big.Int
	nonceHistory                   map[common.Address]map[uint64]uint64
	traceAPICalls                  bool
	printPreparedData              bool
	rw                             sync.RWMutex
	callsCounter                   map[string]int
	currentBlock                   uint64
	limiter                        rpclimiter.RequestLimiter
	tag                            string
	groupTag                       string
}

var countAndlog = func(tc *TestClient, method string, params ...interface{}) error {
	tc.incCounter(method)
	if tc.traceAPICalls {
		if len(params) > 0 {
			tc.t.Log(method, params)
		} else {
			tc.t.Log(method)
		}
	}

	return nil
}

func (tc *TestClient) countAndlog(method string, params ...interface{}) error {
	return countAndlog(tc, method, params...)
}

func (tc *TestClient) incCounter(method string) {
	tc.rw.Lock()
	defer tc.rw.Unlock()
	tc.callsCounter[method] = tc.callsCounter[method] + 1
}

func (tc *TestClient) getCounter() int {
	tc.rw.RLock()
	defer tc.rw.RUnlock()
	cnt := 0
	for _, v := range tc.callsCounter {
		cnt += v
	}
	return cnt
}

func (tc *TestClient) printCounter() {
	total := tc.getCounter()

	tc.rw.RLock()
	defer tc.rw.RUnlock()

	tc.t.Log("========================================= Total calls", total)
	for k, v := range tc.callsCounter {
		tc.t.Log(k, v)
	}
	tc.t.Log("=========================================")
}

func (tc *TestClient) resetCounter() {
	tc.rw.Lock()
	defer tc.rw.Unlock()
	tc.callsCounter = map[string]int{}
}

func (tc *TestClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	if tc.traceAPICalls {
		tc.t.Log("BatchCallContext")
	}
	return nil
}

func (tc *TestClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	err := tc.countAndlog("HeaderByHash")
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *TestClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	err := tc.countAndlog("BlockByHash")
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *TestClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	err := tc.countAndlog("BlockByNumber")
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *TestClient) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	nonce := tc.nonceHistory[account][blockNumber.Uint64()]
	err := tc.countAndlog("NonceAt", fmt.Sprintf("result: %d", nonce))
	if err != nil {
		return nonce, err
	}
	return nonce, nil
}

func (tc *TestClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	err := tc.countAndlog("FilterLogs")
	if err != nil {
		return nil, err
	}

	// We do not verify addresses for now
	allTransfers := []testERC20Transfer{}
	signatures := q.Topics[0]
	erc20TransferSignature := walletcommon.GetEventSignatureHash(walletcommon.Erc20_721TransferEventSignature)
	erc1155TransferSingleSignature := walletcommon.GetEventSignatureHash(walletcommon.Erc1155TransferSingleEventSignature)

	var address common.Hash
	for i := 1; i < len(q.Topics); i++ {
		if len(q.Topics[i]) > 0 {
			address = q.Topics[i][0]
			break
		}
	}

	if slices.Contains(signatures, erc1155TransferSingleSignature) {
		from := q.Topics[2]
		var to []common.Hash
		if len(q.Topics) > 3 {
			to = q.Topics[3]
		}

		if len(to) > 0 {
			for _, addressHash := range to {
				address := &common.Address{}
				address.SetBytes(addressHash.Bytes())
				allTransfers = append(allTransfers, tc.incomingERC1155SingleTransfers[*address]...)
			}
		}
		if len(from) > 0 {
			for _, addressHash := range from {
				address := &common.Address{}
				address.SetBytes(addressHash.Bytes())
				allTransfers = append(allTransfers, tc.outgoingERC1155SingleTransfers[*address]...)
			}
		}
	}

	if slices.Contains(signatures, erc20TransferSignature) {
		from := q.Topics[1]
		to := q.Topics[2]

		if len(to) > 0 {
			for _, addressHash := range to {
				address := &common.Address{}
				address.SetBytes(addressHash.Bytes())
				allTransfers = append(allTransfers, tc.incomingERC20Transfers[*address]...)
			}
		}

		if len(from) > 0 {
			for _, addressHash := range from {
				address := &common.Address{}
				address.SetBytes(addressHash.Bytes())
				allTransfers = append(allTransfers, tc.outgoingERC20Transfers[*address]...)
			}
		}
	}

	logs := []types.Log{}
	for _, transfer := range allTransfers {
		if transfer.block.Cmp(q.FromBlock) >= 0 && transfer.block.Cmp(q.ToBlock) <= 0 {
			header := getTestHeader(transfer.block)
			log := types.Log{
				BlockNumber: header.Number.Uint64(),
				BlockHash:   header.Hash(),
			}

			// Use the address at least in one any(from/to) topic to trick the implementation
			switch transfer.eventType {
			case walletcommon.Erc20TransferEventType, walletcommon.Erc721TransferEventType:
				// To detect properly ERC721, we need a different number of topics. For now we use only ERC20 for testing
				log.Topics = []common.Hash{walletcommon.GetEventSignatureHash(walletcommon.Erc20_721TransferEventSignature), address, address}
			case walletcommon.Erc1155TransferSingleEventType:
				log.Topics = []common.Hash{walletcommon.GetEventSignatureHash(walletcommon.Erc1155TransferSingleEventSignature), address, address, address}
				log.Data = make([]byte, 2*common.HashLength)
			case walletcommon.Erc1155TransferBatchEventType:
				log.Topics = []common.Hash{walletcommon.GetEventSignatureHash(walletcommon.Erc1155TransferBatchEventSignature), address, address, address}
			}

			logs = append(logs, log)
		}
	}

	return logs, nil
}

func (tc *TestClient) getBalance(address common.Address, blockNumber *big.Int) *big.Int {
	balance := tc.balanceHistory[address][blockNumber.Uint64()]
	if balance == nil {
		balance = big.NewInt(0)
	}

	return balance
}

func (tc *TestClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	balance := tc.getBalance(account, blockNumber)
	err := tc.countAndlog("BalanceAt", fmt.Sprintf("account: %s, result: %d", account, balance))
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func (tc *TestClient) tokenBalanceAt(account common.Address, token common.Address, blockNumber *big.Int) *big.Int {
	balance := tc.tokenBalanceHistory[account][token][blockNumber.Uint64()]
	if balance == nil {
		balance = big.NewInt(0)
	}

	if tc.traceAPICalls {
		tc.t.Log("tokenBalanceAt", token, blockNumber, "account:", account, "result:", balance)
	}
	return balance
}

func getTestHeader(number *big.Int) *types.Header {
	return &types.Header{
		Number:     big.NewInt(0).Set(number),
		Time:       0,
		Difficulty: big.NewInt(0),
		ParentHash: common.Hash{},
		Nonce:      types.BlockNonce{},
		MixDigest:  common.Hash{},
		Extra:      make([]byte, 0),
	}
}

func (tc *TestClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if number == nil {
		number = big.NewInt(int64(tc.currentBlock))
	}

	err := tc.countAndlog("HeaderByNumber", fmt.Sprintf("number: %d", number))
	if err != nil {
		return nil, err
	}

	header := getTestHeader(number)

	return header, nil
}

func (tc *TestClient) GetBaseFeeFromBlock(ctx context.Context, blockNumber *big.Int) (string, error) {
	err := tc.countAndlog("GetBaseFeeFromBlock")
	return "", err
}

func (tc *TestClient) NetworkID() uint64 {
	return 777333
}

func (tc *TestClient) ToBigInt() *big.Int {
	if tc.traceAPICalls {
		tc.t.Log("ToBigInt")
	}
	return nil
}

var ethscanAddress = common.HexToAddress("0x0000000000000000000000000000000000777333")
var balanceCheckAddress = common.HexToAddress("0x0000000000000000000000000000000010777333")

func (tc *TestClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	err := tc.countAndlog("CodeAt", fmt.Sprintf("contract: %s, blockNumber: %d", contract, blockNumber))
	if err != nil {
		return nil, err
	}

	if ethscanAddress == contract || balanceCheckAddress == contract {
		return []byte{1}, nil
	}

	return nil, nil
}

func (tc *TestClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	err := tc.countAndlog("CallContract", fmt.Sprintf("call: %v, blockNumber: %d, to: %s", call, blockNumber, call.To))
	if err != nil {
		return nil, err
	}

	if *call.To == ethscanAddress {
		parsed, err := abi.JSON(strings.NewReader(ethscan.BalanceScannerABI))
		if err != nil {
			return nil, err
		}
		method := parsed.Methods["tokensBalance"]
		params := call.Data[len(method.ID):]
		args, err := method.Inputs.Unpack(params)

		if err != nil {
			tc.t.Log("ERROR on unpacking", err)
			return nil, err
		}

		account := args[0].(common.Address)
		tokens := args[1].([]common.Address)
		balances := []*big.Int{}
		for _, token := range tokens {
			balances = append(balances, tc.tokenBalanceAt(account, token, blockNumber))
		}
		results := []ethscan.BalanceScannerResult{}
		for _, balance := range balances {
			results = append(results, ethscan.BalanceScannerResult{
				Success: true,
				Data:    balance.Bytes(),
			})
		}

		output, err := method.Outputs.Pack(results)
		if err != nil {
			tc.t.Log("ERROR on packing", err)
			return nil, err
		}

		return output, nil
	}

	if *call.To == tokenTXXAddress || *call.To == tokenTXYAddress {
		parsed, err := abi.JSON(strings.NewReader(ierc20.IERC20ABI))
		if err != nil {
			return nil, err
		}

		method := parsed.Methods["balanceOf"]
		params := call.Data[len(method.ID):]
		args, err := method.Inputs.Unpack(params)

		if err != nil {
			tc.t.Log("ERROR on unpacking", err)
			return nil, err
		}

		account := args[0].(common.Address)

		balance := tc.tokenBalanceAt(account, *call.To, blockNumber)

		output, err := method.Outputs.Pack(balance)
		if err != nil {
			tc.t.Log("ERROR on packing ERC20 balance", err)
			return nil, err
		}

		return output, nil
	}

	if *call.To == balanceCheckAddress {
		parsed, err := abi.JSON(strings.NewReader(balancechecker.BalanceCheckerABI))
		if err != nil {
			return nil, err
		}

		method := parsed.Methods["balancesHash"]
		params := call.Data[len(method.ID):]
		args, err := method.Inputs.Unpack(params)

		if err != nil {
			tc.t.Log("ERROR on unpacking", err)
			return nil, err
		}

		addresses := args[0].([]common.Address)
		tokens := args[1].([]common.Address)
		bn := big.NewInt(int64(tc.currentBlock))
		hashes := [][32]byte{}

		for _, address := range addresses {
			balance := tc.getBalance(address, big.NewInt(int64(tc.currentBlock)))
			balanceBytes := balance.Bytes()
			for _, token := range tokens {
				balance := tc.tokenBalanceAt(address, token, bn)
				balanceBytes = append(balanceBytes, balance.Bytes()...)
			}

			hash := [32]byte{}
			for i, b := range ethtypes.BytesToHash(balanceBytes).Bytes() {
				hash[i] = b
			}

			hashes = append(hashes, hash)
		}

		output, err := method.Outputs.Pack(bn, hashes)
		if err != nil {
			tc.t.Log("ERROR on packing", err)
			return nil, err
		}

		return output, nil
	}

	return nil, nil
}

func (tc *TestClient) prepareBalanceHistory(toBlock int) {
	tc.balanceHistory = map[common.Address]map[uint64]*big.Int{}
	tc.nonceHistory = map[common.Address]map[uint64]uint64{}

	for address, balances := range tc.balances {
		var currentBlock, currentBalance, currentNonce int

		tc.balanceHistory[address] = map[uint64]*big.Int{}
		tc.nonceHistory[address] = map[uint64]uint64{}

		if len(balances) == 0 {
			balances = append(balances, []int{toBlock + 1, 0, 0})
		} else {
			lastBlock := balances[len(balances)-1]
			balances = append(balances, []int{toBlock + 1, lastBlock[1], 0})
		}
		for _, change := range balances {
			for blockN := currentBlock; blockN < change[0]; blockN++ {
				tc.balanceHistory[address][uint64(blockN)] = big.NewInt(int64(currentBalance))
				tc.nonceHistory[address][uint64(blockN)] = uint64(currentNonce)
			}
			currentBlock = change[0]
			currentBalance = change[1]
			currentNonce += change[2]
		}
	}

	if tc.printPreparedData {
		tc.t.Log("========================================= ETH BALANCES")
		tc.t.Log(tc.balanceHistory)
		tc.t.Log(tc.nonceHistory)
		tc.t.Log(tc.tokenBalanceHistory)
		tc.t.Log("=========================================")
	}
}

func (tc *TestClient) prepareTokenBalanceHistory(toBlock int) {
	transfersPerAddress := map[common.Address]map[common.Address][]testERC20Transfer{}
	for account, transfers := range tc.outgoingERC20Transfers {
		if _, ok := transfersPerAddress[account]; !ok {
			transfersPerAddress[account] = map[common.Address][]testERC20Transfer{}
		}
		for _, transfer := range transfers {
			transfer.amount = new(big.Int).Neg(transfer.amount)
			transfer.eventType = walletcommon.Erc20TransferEventType
			transfersPerAddress[account][transfer.address] = append(transfersPerAddress[account][transfer.address], transfer)
		}
	}

	for account, transfers := range tc.incomingERC20Transfers {
		if _, ok := transfersPerAddress[account]; !ok {
			transfersPerAddress[account] = map[common.Address][]testERC20Transfer{}
		}
		for _, transfer := range transfers {
			transfer.amount = new(big.Int).Neg(transfer.amount)
			transfer.eventType = walletcommon.Erc20TransferEventType
			transfersPerAddress[account][transfer.address] = append(transfersPerAddress[account][transfer.address], transfer)
		}
	}

	for account, transfers := range tc.outgoingERC1155SingleTransfers {
		if _, ok := transfersPerAddress[account]; !ok {
			transfersPerAddress[account] = map[common.Address][]testERC20Transfer{}
		}
		for _, transfer := range transfers {
			transfer.amount = new(big.Int).Neg(transfer.amount)
			transfer.eventType = walletcommon.Erc1155TransferSingleEventType
			transfersPerAddress[account][transfer.address] = append(transfersPerAddress[account][transfer.address], transfer)
		}
	}

	for account, transfers := range tc.incomingERC1155SingleTransfers {
		if _, ok := transfersPerAddress[account]; !ok {
			transfersPerAddress[account] = map[common.Address][]testERC20Transfer{}
		}
		for _, transfer := range transfers {
			transfer.amount = new(big.Int).Neg(transfer.amount)
			transfer.eventType = walletcommon.Erc1155TransferSingleEventType
			transfersPerAddress[account][transfer.address] = append(transfersPerAddress[account][transfer.address], transfer)
		}
	}

	tc.tokenBalanceHistory = map[common.Address]map[common.Address]map[uint64]*big.Int{}

	for account, transfersPerToken := range transfersPerAddress {
		tc.tokenBalanceHistory[account] = map[common.Address]map[uint64]*big.Int{}
		for token, transfers := range transfersPerToken {
			sort.Slice(transfers, func(i, j int) bool {
				return transfers[i].block.Cmp(transfers[j].block) < 0
			})

			currentBlock := uint64(0)
			currentBalance := big.NewInt(0)

			tc.tokenBalanceHistory[token] = map[common.Address]map[uint64]*big.Int{}
			transfers = append(transfers, testERC20Transfer{big.NewInt(int64(toBlock + 1)), token, big.NewInt(0), walletcommon.Erc20TransferEventType})

			tc.tokenBalanceHistory[account][token] = map[uint64]*big.Int{}
			for _, transfer := range transfers {
				for blockN := currentBlock; blockN < transfer.block.Uint64(); blockN++ {
					tc.tokenBalanceHistory[account][token][blockN] = new(big.Int).Set(currentBalance)
				}
				currentBlock = transfer.block.Uint64()
				currentBalance = new(big.Int).Add(currentBalance, transfer.amount)
			}
		}
	}
	if tc.printPreparedData {
		tc.t.Log("========================================= ERC20 BALANCES")
		tc.t.Log(tc.tokenBalanceHistory)
		tc.t.Log("=========================================")
	}
}

func (tc *TestClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	err := tc.countAndlog("CallContext")
	return err
}

func (tc *TestClient) GetWalletNotifier() func(chainId uint64, message string) {
	if tc.traceAPICalls {
		tc.t.Log("GetWalletNotifier")
	}
	return nil
}

func (tc *TestClient) SetWalletNotifier(notifier func(chainId uint64, message string)) {
	if tc.traceAPICalls {
		tc.t.Log("SetWalletNotifier")
	}
}

func (tc *TestClient) EstimateGas(ctx context.Context, call ethereum.CallMsg) (gas uint64, err error) {
	err = tc.countAndlog("EstimateGas")
	return 0, err
}

func (tc *TestClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	err := tc.countAndlog("PendingCodeAt")
	return nil, err
}

func (tc *TestClient) PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error) {
	err := tc.countAndlog("PendingCallContract")
	return nil, err
}

func (tc *TestClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	err := tc.countAndlog("PendingNonceAt")
	return 0, err
}

func (tc *TestClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	err := tc.countAndlog("SuggestGasPrice")
	return nil, err
}

func (tc *TestClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	err := tc.countAndlog("SendTransaction")
	return err
}

func (tc *TestClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	err := tc.countAndlog("SuggestGasTipCap")
	return nil, err
}

func (tc *TestClient) BatchCallContextIgnoringLocalHandlers(ctx context.Context, b []rpc.BatchElem) error {
	err := tc.countAndlog("BatchCallContextIgnoringLocalHandlers")
	return err
}

func (tc *TestClient) CallContextIgnoringLocalHandlers(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	err := tc.countAndlog("CallContextIgnoringLocalHandlers")
	return err
}

func (tc *TestClient) CallRaw(data string) string {
	_ = tc.countAndlog("CallRaw")
	return ""
}

func (tc *TestClient) GetChainID() *big.Int {
	return big.NewInt(1)
}

func (tc *TestClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	err := tc.countAndlog("SubscribeFilterLogs")
	return nil, err
}

func (tc *TestClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	err := tc.countAndlog("TransactionReceipt")
	return nil, err
}

func (tc *TestClient) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	err := tc.countAndlog("TransactionByHash")
	return nil, false, err
}

func (tc *TestClient) BlockNumber(ctx context.Context) (uint64, error) {
	err := tc.countAndlog("BlockNumber")
	return 0, err
}

func (tc *TestClient) FeeHistory(ctx context.Context, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error) {
	err := tc.countAndlog("FeeHistory")
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (tc *TestClient) PendingBalanceAt(ctx context.Context, account common.Address) (*big.Int, error) {
	err := tc.countAndlog("PendingBalanceAt")
	return nil, err
}

func (tc *TestClient) PendingStorageAt(ctx context.Context, account common.Address, key common.Hash) ([]byte, error) {
	err := tc.countAndlog("PendingStorageAt")
	return nil, err
}

func (tc *TestClient) PendingTransactionCount(ctx context.Context) (uint, error) {
	err := tc.countAndlog("PendingTransactionCount")
	return 0, err
}

func (tc *TestClient) StorageAt(ctx context.Context, account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	err := tc.countAndlog("StorageAt")
	return nil, err
}

func (tc *TestClient) SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	err := tc.countAndlog("SyncProgress")
	return nil, err
}

func (tc *TestClient) TransactionSender(ctx context.Context, tx *types.Transaction, block common.Hash, index uint) (common.Address, error) {
	err := tc.countAndlog("TransactionSender")
	return common.Address{}, err
}

func (tc *TestClient) SetIsConnected(value bool) {
	if tc.traceAPICalls {
		tc.t.Log("SetIsConnected")
	}
}

func (tc *TestClient) IsConnected() bool {
	if tc.traceAPICalls {
		tc.t.Log("GetIsConnected")
	}

	return true
}

func (tc *TestClient) GetLimiter() rpclimiter.RequestLimiter {
	return tc.limiter
}

func (tc *TestClient) SetLimiter(limiter rpclimiter.RequestLimiter) {
	tc.limiter = limiter
}

func (tc *TestClient) Close() {
	if tc.traceAPICalls {
		tc.t.Log("Close")
	}
}

type testERC20Transfer struct {
	block     *big.Int
	address   common.Address
	amount    *big.Int
	eventType walletcommon.EventType
}

type findBlockCase struct {
	balanceChanges                 [][]int
	ERC20BalanceChanges            [][]int
	fromBlock                      int64
	toBlock                        int64
	rangeSize                      int
	expectedBlocksFound            int
	outgoingERC20Transfers         []testERC20Transfer
	incomingERC20Transfers         []testERC20Transfer
	outgoingERC1155SingleTransfers []testERC20Transfer
	incomingERC1155SingleTransfers []testERC20Transfer
	label                          string
	expectedCalls                  map[string]int
}

func transferInEachBlock() [][]int {
	res := [][]int{}

	for i := 1; i < 101; i++ {
		res = append(res, []int{i, i, i})
	}

	return res
}

func getCases() []findBlockCase {
	cases := []findBlockCase{}
	case1 := findBlockCase{
		balanceChanges: [][]int{
			{5, 1, 0},
			{20, 2, 0},
			{45, 1, 1},
			{46, 50, 0},
			{75, 0, 1},
		},
		outgoingERC20Transfers: []testERC20Transfer{
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
		},
		toBlock:             100,
		expectedBlocksFound: 6,
		expectedCalls: map[string]int{
			"FilterLogs":     15,
			"HeaderByNumber": 5,
		},
	}

	case100transfers := findBlockCase{
		balanceChanges:      transferInEachBlock(),
		toBlock:             100,
		expectedBlocksFound: 100,
		expectedCalls: map[string]int{
			"BalanceAt":      101,
			"NonceAt":        0,
			"FilterLogs":     15,
			"HeaderByNumber": 100,
		},
	}

	case3 := findBlockCase{
		balanceChanges: [][]int{
			{1, 1, 1},
			{2, 2, 2},
			{45, 1, 1},
			{46, 50, 0},
			{75, 0, 1},
		},
		toBlock:             100,
		expectedBlocksFound: 5,
	}
	case4 := findBlockCase{
		balanceChanges: [][]int{
			{20, 1, 0},
		},
		toBlock:             100,
		fromBlock:           10,
		expectedBlocksFound: 1,
		label:               "single block",
	}

	case5 := findBlockCase{
		balanceChanges:      [][]int{},
		toBlock:             100,
		fromBlock:           20,
		expectedBlocksFound: 0,
	}

	case6 := findBlockCase{
		balanceChanges: [][]int{
			{20, 1, 0},
			{45, 1, 1},
		},
		toBlock:             100,
		fromBlock:           30,
		expectedBlocksFound: 1,
		rangeSize:           20,
		label:               "single block in range",
	}

	case7emptyHistoryWithOneERC20Transfer := findBlockCase{
		balanceChanges:      [][]int{},
		toBlock:             100,
		rangeSize:           20,
		expectedBlocksFound: 1,
		incomingERC20Transfers: []testERC20Transfer{
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
		},
	}

	case8emptyHistoryWithERC20Transfers := findBlockCase{
		balanceChanges:      [][]int{},
		toBlock:             100,
		rangeSize:           20,
		expectedBlocksFound: 2,
		incomingERC20Transfers: []testERC20Transfer{
			// edge case when a regular scan will find transfer at 80,
			// but erc20 tail scan should only find transfer at block 6
			{big.NewInt(80), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
		},
		expectedCalls: map[string]int{
			"FilterLogs":   5,
			"CallContract": 3,
		},
	}

	case9emptyHistoryWithERC20Transfers := findBlockCase{
		balanceChanges: [][]int{},
		toBlock:        100,
		rangeSize:      20,
		// we expect only a single eth_getLogs to be executed here for both erc20 transfers,
		// thus only 2 blocks found
		expectedBlocksFound: 2,
		incomingERC20Transfers: []testERC20Transfer{
			{big.NewInt(7), tokenTXYAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
		},
		expectedCalls: map[string]int{
			"FilterLogs": 5,
		},
	}

	case10 := findBlockCase{
		balanceChanges:      [][]int{},
		toBlock:             100,
		fromBlock:           99,
		expectedBlocksFound: 0,
		label:               "single block range, no transactions",
		expectedCalls: map[string]int{
			// only two requests to check the range for incoming ERC20
			"FilterLogs": 3,
			// no contract calls as ERC20 is not checked
			"CallContract": 0,
		},
	}

	case11IncomingERC1155SingleTransfers := findBlockCase{
		balanceChanges: [][]int{},
		toBlock:        100,
		rangeSize:      20,
		// we expect only a single eth_getLogs to be executed here for both erc20 transfers,
		// thus only 2 blocks found
		expectedBlocksFound: 2,
		incomingERC1155SingleTransfers: []testERC20Transfer{
			{big.NewInt(7), tokenTXYAddress, big.NewInt(1), walletcommon.Erc1155TransferSingleEventType},
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc1155TransferSingleEventType},
		},
		expectedCalls: map[string]int{
			"FilterLogs":   5,
			"CallContract": 5,
		},
	}

	case12OutgoingERC1155SingleTransfers := findBlockCase{
		balanceChanges: [][]int{
			{6, 1, 0},
		},
		toBlock:             100,
		rangeSize:           20,
		expectedBlocksFound: 3,
		outgoingERC1155SingleTransfers: []testERC20Transfer{
			{big.NewInt(80), tokenTXYAddress, big.NewInt(1), walletcommon.Erc1155TransferSingleEventType},
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc1155TransferSingleEventType},
		},
		expectedCalls: map[string]int{
			"FilterLogs": 15, // 3 for each range
		},
	}

	case13outgoingERC20ERC1155SingleTransfers := findBlockCase{
		balanceChanges: [][]int{
			{63, 1, 0},
		},
		toBlock:             100,
		rangeSize:           20,
		expectedBlocksFound: 3,
		outgoingERC1155SingleTransfers: []testERC20Transfer{
			{big.NewInt(80), tokenTXYAddress, big.NewInt(1), walletcommon.Erc1155TransferSingleEventType},
		},
		outgoingERC20Transfers: []testERC20Transfer{
			{big.NewInt(63), tokenTXYAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
		},
		expectedCalls: map[string]int{
			"FilterLogs": 6, // 3 for each range, 0 for tail check becauseERC20ScanByBalance  returns no ranges
		},
	}

	case14outgoingERC20ERC1155SingleTransfersMoreFilterLogs := findBlockCase{
		balanceChanges: [][]int{
			{61, 1, 0},
		},
		toBlock:             100,
		rangeSize:           20,
		expectedBlocksFound: 3,
		outgoingERC1155SingleTransfers: []testERC20Transfer{
			{big.NewInt(80), tokenTXYAddress, big.NewInt(1), walletcommon.Erc1155TransferSingleEventType},
		},
		outgoingERC20Transfers: []testERC20Transfer{
			{big.NewInt(61), tokenTXYAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
		},
		expectedCalls: map[string]int{
			"FilterLogs": 9, // 3 for each range of [40-100], 0 for tail check because ERC20ScanByBalance returns no ranges
		},
		label: "outgoing ERC20 and ERC1155 transfers but more FilterLogs calls because startFromBlock is not detected at range [60-80] as it is in the first subrange",
	}

	case15incomingERC20outgoingERC1155SingleTransfers := findBlockCase{
		balanceChanges: [][]int{
			{85, 1, 0},
		},
		toBlock:             100,
		rangeSize:           20,
		expectedBlocksFound: 2,
		outgoingERC1155SingleTransfers: []testERC20Transfer{
			{big.NewInt(85), tokenTXYAddress, big.NewInt(1), walletcommon.Erc1155TransferSingleEventType},
		},
		incomingERC20Transfers: []testERC20Transfer{
			{big.NewInt(88), tokenTXYAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
		},
		expectedCalls: map[string]int{
			"FilterLogs": 3, // 3 for each range of [40-100], 0 for tail check because ERC20ScanByBalance returns no ranges
		},
		label: "incoming ERC20 and outgoing ERC1155 transfers are fetched with same topic",
	}

	case16 := findBlockCase{
		balanceChanges: [][]int{
			{75, 0, 1},
		},
		outgoingERC20Transfers: []testERC20Transfer{
			{big.NewInt(80), tokenTXXAddress, big.NewInt(4), walletcommon.Erc20TransferEventType},
		},
		toBlock:             100,
		rangeSize:           20,
		expectedBlocksFound: 3, // ideally we should find 2 blocks, but we will find 3 and this test shows that we are ok with that
		label: `duplicate blocks detected but we wont fix it because we want to save requests on the edges of the ranges,
		 taking balance and nonce from cache while ETH and tokens ranges searching are tightly coupled`,
	}

	cases = append(cases, case1)
	cases = append(cases, case100transfers)
	cases = append(cases, case3)
	cases = append(cases, case4)
	cases = append(cases, case5)

	cases = append(cases, case6)
	cases = append(cases, case7emptyHistoryWithOneERC20Transfer)
	cases = append(cases, case8emptyHistoryWithERC20Transfers)
	cases = append(cases, case9emptyHistoryWithERC20Transfers)
	cases = append(cases, case10)
	cases = append(cases, case11IncomingERC1155SingleTransfers)
	cases = append(cases, case12OutgoingERC1155SingleTransfers)
	cases = append(cases, case13outgoingERC20ERC1155SingleTransfers)
	cases = append(cases, case14outgoingERC20ERC1155SingleTransfersMoreFilterLogs)
	cases = append(cases, case15incomingERC20outgoingERC1155SingleTransfers)
	cases = append(cases, case16)

	//cases = append([]findBlockCase{}, case10)

	return cases
}

var tokenTXXAddress = common.HexToAddress("0x53211")
var tokenTXYAddress = common.HexToAddress("0x73211")

func setupFindBlocksCommand(t *testing.T, accountAddress common.Address, fromBlock, toBlock *big.Int, rangeSize int, balances map[common.Address][][]int, outgoingERC20Transfers, incomingERC20Transfers, outgoingERC1155SingleTransfers, incomingERC1155SingleTransfers map[common.Address][]testERC20Transfer) (*findBlocksCommand, *TestClient, chan []*DBHeader, *BlockRangeSequentialDAO) {
	appdb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	mediaServer, err := server.NewMediaServer(appdb, nil, nil, db)
	require.NoError(t, err)

	wdb := NewDB(db)
	tc := &TestClient{
		t:                              t,
		balances:                       balances,
		outgoingERC20Transfers:         outgoingERC20Transfers,
		incomingERC20Transfers:         incomingERC20Transfers,
		outgoingERC1155SingleTransfers: outgoingERC1155SingleTransfers,
		incomingERC1155SingleTransfers: incomingERC1155SingleTransfers,
		callsCounter:                   map[string]int{},
	}
	// tc.traceAPICalls = true
	// tc.printPreparedData = true
	tc.prepareBalanceHistory(100)
	tc.prepareTokenBalanceHistory(100)
	blockChannel := make(chan []*DBHeader, 100)

	// Reimplement the common function that is called from every method to check for the limit
	countAndlog = func(tc *TestClient, method string, params ...interface{}) error {
		if tc.GetLimiter() != nil {
			if allow, _ := tc.GetLimiter().Allow(tc.tag); !allow {
				t.Log("ERROR: requests over limit")
				return rpclimiter.ErrRequestsOverLimit
			}
			if allow, _ := tc.GetLimiter().Allow(tc.groupTag); !allow {
				t.Log("ERROR: requests over limit for group tag")
				return rpclimiter.ErrRequestsOverLimit
			}
		}

		tc.incCounter(method)
		if tc.traceAPICalls {
			if len(params) > 0 {
				tc.t.Log(method, params)
			} else {
				tc.t.Log(method)
			}
		}

		return nil
	}

	config := statusRpc.ClientConfig{
		Client:          nil,
		UpstreamChainID: 1,
		Networks:        []params.Network{},
		DB:              db,
		WalletFeed:      nil,
		ProviderConfigs: nil,
	}
	client, _ := statusRpc.NewClient(config)

	client.SetClient(tc.NetworkID(), tc)
	tokenManager := token.NewTokenManager(db, client, community.NewManager(appdb, nil, nil), network.NewManager(appdb), appdb, mediaServer, nil, nil, nil, token.NewPersistence(db))
	tokenManager.SetTokens([]*token.Token{
		{
			Address:  tokenTXXAddress,
			Symbol:   "TXX",
			Decimals: 18,
			ChainID:  tc.NetworkID(),
			Name:     "Test Token 1",
			Verified: true,
		},
		{
			Address:  tokenTXYAddress,
			Symbol:   "TXY",
			Decimals: 18,
			ChainID:  tc.NetworkID(),
			Name:     "Test Token 2",
			Verified: true,
		},
	})
	accDB, err := accounts.NewDB(appdb)
	require.NoError(t, err)
	blockRangeDAO := &BlockRangeSequentialDAO{wdb.client}
	fbc := &findBlocksCommand{
		accounts:                  []common.Address{accountAddress},
		db:                        wdb,
		blockRangeDAO:             blockRangeDAO,
		accountsDB:                accDB,
		chainClient:               tc,
		balanceCacher:             balance.NewCacherWithTTL(5 * time.Minute),
		feed:                      &event.Feed{},
		noLimit:                   false,
		fromBlockNumber:           fromBlock,
		toBlockNumber:             toBlock,
		blocksLoadedCh:            blockChannel,
		defaultNodeBlockChunkSize: rangeSize,
		tokenManager:              tokenManager,
	}
	return fbc, tc, blockChannel, blockRangeDAO
}

func TestFindBlocksCommand(t *testing.T) {
	for idx, testCase := range getCases() {
		t.Log("case #", idx+1)

		accountAddress := common.HexToAddress("0x1234")
		rangeSize := 20
		if testCase.rangeSize != 0 {
			rangeSize = testCase.rangeSize
		}

		balances := map[common.Address][][]int{accountAddress: testCase.balanceChanges}
		outgoingERC20Transfers := map[common.Address][]testERC20Transfer{accountAddress: testCase.outgoingERC20Transfers}
		incomingERC20Transfers := map[common.Address][]testERC20Transfer{accountAddress: testCase.incomingERC20Transfers}
		outgoingERC1155SingleTransfers := map[common.Address][]testERC20Transfer{accountAddress: testCase.outgoingERC1155SingleTransfers}
		incomingERC1155SingleTransfers := map[common.Address][]testERC20Transfer{accountAddress: testCase.incomingERC1155SingleTransfers}

		fbc, tc, blockChannel, blockRangeDAO := setupFindBlocksCommand(t, accountAddress, big.NewInt(testCase.fromBlock), big.NewInt(testCase.toBlock), rangeSize, balances, outgoingERC20Transfers, incomingERC20Transfers, outgoingERC1155SingleTransfers, incomingERC1155SingleTransfers)
		ctx := context.Background()
		group := async.NewGroup(ctx)
		group.Add(fbc.Command())

		foundBlocks := []*DBHeader{}
		select {
		case <-ctx.Done():
			t.Log("ERROR")
		case <-group.WaitAsync():
			close(blockChannel)
			for {
				bloks, ok := <-blockChannel
				if !ok {
					break
				}
				foundBlocks = append(foundBlocks, bloks...)
			}

			numbers := []int64{}
			for _, block := range foundBlocks {
				numbers = append(numbers, block.Number.Int64())
			}

			if tc.traceAPICalls {
				tc.printCounter()
			}

			for name, cnt := range testCase.expectedCalls {
				require.Equal(t, cnt, tc.callsCounter[name], "calls to "+name)
			}

			sort.Slice(numbers, func(i, j int) bool { return numbers[i] < numbers[j] })
			require.Equal(t, testCase.expectedBlocksFound, len(foundBlocks), testCase.label, "found blocks", numbers)

			blRange, _, err := blockRangeDAO.getBlockRange(tc.NetworkID(), accountAddress)
			require.NoError(t, err)
			require.NotNil(t, blRange.eth.FirstKnown)
			require.NotNil(t, blRange.tokens.FirstKnown)
			if testCase.fromBlock == 0 {
				require.Equal(t, 0, blRange.tokens.FirstKnown.Cmp(zero))
			}
		}
	}
}

func TestFindBlocksCommandWithLimiter(t *testing.T) {
	maxRequests := 1
	rangeSize := 20
	accountAddress := common.HexToAddress("0x1234")
	balances := map[common.Address][][]int{accountAddress: {{5, 1, 0}, {20, 2, 0}, {45, 1, 1}, {46, 50, 0}, {75, 0, 1}}}
	fbc, tc, blockChannel, _ := setupFindBlocksCommand(t, accountAddress, big.NewInt(0), big.NewInt(20), rangeSize, balances, nil, nil, nil, nil)

	limiter := rpclimiter.NewRequestLimiter(rpclimiter.NewInMemRequestsMapStorage())
	err := limiter.SetLimit(transferHistoryTag, maxRequests, time.Hour)
	require.NoError(t, err)
	tc.SetLimiter(limiter)
	tc.tag = transferHistoryTag

	ctx := context.Background()
	group := async.NewAtomicGroup(ctx)
	group.Add(fbc.Command(1 * time.Millisecond))

	select {
	case <-ctx.Done():
		t.Log("ERROR")
	case <-group.WaitAsync():
		close(blockChannel)
		require.Error(t, rpclimiter.ErrRequestsOverLimit, group.Error())
		require.Equal(t, maxRequests, tc.getCounter())
	}
}

func TestFindBlocksCommandWithLimiterTagDifferentThanTransfers(t *testing.T) {
	rangeSize := 20
	maxRequests := 1
	accountAddress := common.HexToAddress("0x1234")
	balances := map[common.Address][][]int{accountAddress: {{5, 1, 0}, {20, 2, 0}, {45, 1, 1}, {46, 50, 0}, {75, 0, 1}}}
	outgoingERC20Transfers := map[common.Address][]testERC20Transfer{accountAddress: {{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType}}}
	incomingERC20Transfers := map[common.Address][]testERC20Transfer{accountAddress: {{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType}}}

	fbc, tc, blockChannel, _ := setupFindBlocksCommand(t, accountAddress, big.NewInt(0), big.NewInt(20), rangeSize, balances, outgoingERC20Transfers, incomingERC20Transfers, nil, nil)
	limiter := rpclimiter.NewRequestLimiter(rpclimiter.NewInMemRequestsMapStorage())
	err := limiter.SetLimit("some-other-tag-than-transfer-history", maxRequests, time.Hour)
	require.NoError(t, err)
	tc.SetLimiter(limiter)

	ctx := context.Background()
	group := async.NewAtomicGroup(ctx)
	group.Add(fbc.Command(1 * time.Millisecond))

	select {
	case <-ctx.Done():
		t.Log("ERROR")
	case <-group.WaitAsync():
		close(blockChannel)
		require.NoError(t, group.Error())
		require.Greater(t, tc.getCounter(), maxRequests)
	}
}

func TestFindBlocksCommandWithLimiterForMultipleAccountsSameGroup(t *testing.T) {
	rangeSize := 20
	maxRequestsTotal := 5
	limit1 := 3
	limit2 := 3
	account1 := common.HexToAddress("0x1234")
	account2 := common.HexToAddress("0x5678")
	balances := map[common.Address][][]int{account1: {{5, 1, 0}, {20, 2, 0}, {45, 1, 1}, {46, 50, 0}, {75, 0, 1}}, account2: {{5, 1, 0}, {20, 2, 0}, {45, 1, 1}, {46, 50, 0}, {75, 0, 1}}}
	outgoingERC20Transfers := map[common.Address][]testERC20Transfer{account1: {{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType}}}
	incomingERC20Transfers := map[common.Address][]testERC20Transfer{account2: {{big.NewInt(6), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType}}}

	// Limiters share the same storage
	storage := rpclimiter.NewInMemRequestsMapStorage()

	// Set up the first account
	fbc, tc, blockChannel, _ := setupFindBlocksCommand(t, account1, big.NewInt(0), big.NewInt(20), rangeSize, balances, outgoingERC20Transfers, nil, nil, nil)
	tc.tag = transferHistoryTag + account1.String()
	tc.groupTag = transferHistoryTag

	limiter1 := rpclimiter.NewRequestLimiter(storage)
	err := limiter1.SetLimit(transferHistoryTag, maxRequestsTotal, time.Hour)
	require.NoError(t, err)
	err = limiter1.SetLimit(transferHistoryTag+account1.String(), limit1, time.Hour)
	require.NoError(t, err)
	tc.SetLimiter(limiter1)

	// Set up the second account
	fbc2, tc2, _, _ := setupFindBlocksCommand(t, account2, big.NewInt(0), big.NewInt(20), rangeSize, balances, nil, incomingERC20Transfers, nil, nil)
	tc2.tag = transferHistoryTag + account2.String()
	tc2.groupTag = transferHistoryTag
	limiter2 := rpclimiter.NewRequestLimiter(storage)
	err = limiter2.SetLimit(transferHistoryTag, maxRequestsTotal, time.Hour)
	require.NoError(t, err)
	err = limiter2.SetLimit(transferHistoryTag+account2.String(), limit2, time.Hour)
	require.NoError(t, err)
	tc2.SetLimiter(limiter2)
	fbc2.blocksLoadedCh = blockChannel

	ctx := context.Background()
	group := async.NewGroup(ctx)
	group.Add(fbc.Command(1 * time.Millisecond))
	group.Add(fbc2.Command(1 * time.Millisecond))

	select {
	case <-ctx.Done():
		t.Log("ERROR")
	case <-group.WaitAsync():
		close(blockChannel)
		require.LessOrEqual(t, tc.getCounter(), maxRequestsTotal)
	}
}

type MockETHClient struct {
	mock.Mock
}

func (m *MockETHClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

type MockChainClient struct {
	mock.Mock
	mock_client.MockClientInterface

	clients map[walletcommon.ChainID]*MockETHClient
}

func newMockChainClient() *MockChainClient {
	return &MockChainClient{
		clients: make(map[walletcommon.ChainID]*MockETHClient),
	}
}

func (m *MockChainClient) AbstractEthClient(chainID walletcommon.ChainID) (ethclient.BatchCallClient, error) {
	if _, ok := m.clients[chainID]; !ok {
		panic(fmt.Sprintf("no mock client for chainID %d", chainID))
	}
	return m.clients[chainID], nil
}

func TestFetchTransfersForLoadedBlocks(t *testing.T) {
	appdb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	mediaServer, err := server.NewMediaServer(appdb, nil, nil, db)
	require.NoError(t, err)

	wdb := NewDB(db)
	blockChannel := make(chan []*DBHeader, 100)

	tc := &TestClient{
		t:                      t,
		balances:               map[common.Address][][]int{},
		outgoingERC20Transfers: map[common.Address][]testERC20Transfer{},
		incomingERC20Transfers: map[common.Address][]testERC20Transfer{},
		callsCounter:           map[string]int{},
		currentBlock:           100,
	}

	config := statusRpc.ClientConfig{
		Client:          nil,
		UpstreamChainID: 1,
		Networks:        []params.Network{},
		DB:              db,
		WalletFeed:      nil,
		ProviderConfigs: nil,
	}
	client, _ := statusRpc.NewClient(config)

	client.SetClient(tc.NetworkID(), tc)
	tokenManager := token.NewTokenManager(db, client, community.NewManager(appdb, nil, nil), network.NewManager(appdb), appdb, mediaServer, nil, nil, nil, token.NewPersistence(db))

	tokenManager.SetTokens([]*token.Token{
		{
			Address:  tokenTXXAddress,
			Symbol:   "TXX",
			Decimals: 18,
			ChainID:  tc.NetworkID(),
			Name:     "Test Token 1",
			Verified: true,
		},
		{
			Address:  tokenTXYAddress,
			Symbol:   "TXY",
			Decimals: 18,
			ChainID:  tc.NetworkID(),
			Name:     "Test Token 2",
			Verified: true,
		},
	})

	address := common.HexToAddress("0x1234")
	chainClient := newMockChainClient()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rpcClient := mock_rpcclient.NewMockClientInterface(ctrl)
	rpcClient.EXPECT().AbstractEthClient(tc.NetworkID()).Return(chainClient, nil).AnyTimes()
	tracker := transactions.NewPendingTxTracker(db, rpcClient, nil, &event.Feed{}, transactions.PendingCheckInterval)
	accDB, err := accounts.NewDB(appdb)
	require.NoError(t, err)

	cmd := &loadBlocksAndTransfersCommand{
		accounts:         []common.Address{address},
		db:               wdb,
		blockRangeDAO:    &BlockRangeSequentialDAO{wdb.client},
		blockDAO:         &BlockDAO{db},
		accountsDB:       accDB,
		chainClient:      tc,
		feed:             &event.Feed{},
		balanceCacher:    balance.NewCacherWithTTL(5 * time.Minute),
		pendingTxManager: tracker,
		tokenManager:     tokenManager,
		blocksLoadedCh:   blockChannel,
		omitHistory:      true,
		contractMaker:    tokenManager.ContractMaker,
	}

	tc.prepareBalanceHistory(int(tc.currentBlock))
	tc.prepareTokenBalanceHistory(int(tc.currentBlock))
	// tc.traceAPICalls = true

	ctx := context.Background()
	group := async.NewAtomicGroup(ctx)

	fromNum := big.NewInt(0)
	toNum, err := getHeadBlockNumber(ctx, cmd.chainClient)
	require.NoError(t, err)
	err = cmd.fetchHistoryBlocksForAccount(group, address, fromNum, toNum, blockChannel)
	require.NoError(t, err)

	select {
	case <-ctx.Done():
		t.Log("ERROR")
	case <-group.WaitAsync():
		require.Equal(t, 1, tc.getCounter())
	}
}

func getNewBlocksCases() []findBlockCase {
	cases := []findBlockCase{
		findBlockCase{
			balanceChanges: [][]int{
				{20, 1, 0},
			},
			fromBlock:           0,
			toBlock:             10,
			expectedBlocksFound: 0,
			label:               "single block, but not in range",
		},
		findBlockCase{
			balanceChanges: [][]int{
				{20, 1, 0},
			},
			fromBlock:           10,
			toBlock:             20,
			expectedBlocksFound: 1,
			label:               "single block in range",
		},
	}

	return cases
}

func TestFetchNewBlocksCommand_findBlocksWithEthTransfers(t *testing.T) {
	appdb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	mediaServer, err := server.NewMediaServer(appdb, nil, nil, db)
	require.NoError(t, err)

	wdb := NewDB(db)
	blockChannel := make(chan []*DBHeader, 10)

	address := common.HexToAddress("0x1234")
	accDB, err := accounts.NewDB(appdb)
	require.NoError(t, err)

	for idx, testCase := range getNewBlocksCases() {
		t.Log("case #", idx+1)
		tc := &TestClient{
			t:                      t,
			balances:               map[common.Address][][]int{address: testCase.balanceChanges},
			outgoingERC20Transfers: map[common.Address][]testERC20Transfer{},
			incomingERC20Transfers: map[common.Address][]testERC20Transfer{},
			callsCounter:           map[string]int{},
			currentBlock:           100,
		}

		config := statusRpc.ClientConfig{
			Client:          nil,
			UpstreamChainID: 1,
			Networks:        []params.Network{},
			DB:              db,
			WalletFeed:      nil,
			ProviderConfigs: nil,
		}
		client, _ := statusRpc.NewClient(config)

		client.SetClient(tc.NetworkID(), tc)
		tokenManager := token.NewTokenManager(db, client, community.NewManager(appdb, nil, nil), network.NewManager(appdb), appdb, mediaServer, nil, nil, nil, token.NewPersistence(db))

		tokenManager.SetTokens([]*token.Token{
			{
				Address:  tokenTXXAddress,
				Symbol:   "TXX",
				Decimals: 18,
				ChainID:  tc.NetworkID(),
				Name:     "Test Token 1",
				Verified: true,
			},
			{
				Address:  tokenTXYAddress,
				Symbol:   "TXY",
				Decimals: 18,
				ChainID:  tc.NetworkID(),
				Name:     "Test Token 2",
				Verified: true,
			},
		})

		cmd := &findNewBlocksCommand{
			findBlocksCommand: &findBlocksCommand{
				accounts:                  []common.Address{address},
				db:                        wdb,
				accountsDB:                accDB,
				blockRangeDAO:             &BlockRangeSequentialDAO{wdb.client},
				chainClient:               tc,
				balanceCacher:             balance.NewCacherWithTTL(5 * time.Minute),
				feed:                      &event.Feed{},
				noLimit:                   false,
				tokenManager:              tokenManager,
				blocksLoadedCh:            blockChannel,
				defaultNodeBlockChunkSize: DefaultNodeBlockChunkSize,
			},
			nonceCheckIntervalIterations: nonceCheckIntervalIterations,
			logsCheckIntervalIterations:  logsCheckIntervalIterations,
		}
		tc.prepareBalanceHistory(int(tc.currentBlock))
		tc.prepareTokenBalanceHistory(int(tc.currentBlock))

		ctx := context.Background()
		blocks, _, err := cmd.findBlocksWithEthTransfers(ctx, address, big.NewInt(testCase.fromBlock), big.NewInt(testCase.toBlock))
		require.NoError(t, err)
		require.Equal(t, testCase.expectedBlocksFound, len(blocks), fmt.Sprintf("case %d: %s, blocks from %d to %d", idx+1, testCase.label, testCase.fromBlock, testCase.toBlock))
	}
}

func TestFetchNewBlocksCommand_nonceDetection(t *testing.T) {
	balanceChanges := [][]int{
		{5, 1, 0},
		{6, 0, 1},
	}

	scanRange := 5
	address := common.HexToAddress("0x1234")

	tc := &TestClient{
		t:                      t,
		balances:               map[common.Address][][]int{address: balanceChanges},
		outgoingERC20Transfers: map[common.Address][]testERC20Transfer{},
		incomingERC20Transfers: map[common.Address][]testERC20Transfer{},
		callsCounter:           map[string]int{},
		currentBlock:           0,
	}

	//tc.printPreparedData = true
	tc.prepareBalanceHistory(20)

	appdb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	mediaServer, err := server.NewMediaServer(appdb, nil, nil, db)
	require.NoError(t, err)

	config := statusRpc.ClientConfig{
		Client:          nil,
		UpstreamChainID: 1,
		Networks:        []params.Network{},
		DB:              db,
		WalletFeed:      nil,
		ProviderConfigs: nil,
	}
	client, _ := statusRpc.NewClient(config)

	client.SetClient(tc.NetworkID(), tc)
	tokenManager := token.NewTokenManager(db, client, community.NewManager(appdb, nil, nil), network.NewManager(appdb), appdb, mediaServer, nil, nil, nil, token.NewPersistence(db))

	wdb := NewDB(db)
	blockChannel := make(chan []*DBHeader, 10)

	accDB, err := accounts.NewDB(appdb)
	require.NoError(t, err)

	maker, _ := contracts.NewContractMaker(client)

	cmd := &findNewBlocksCommand{
		findBlocksCommand: &findBlocksCommand{
			accounts:                  []common.Address{address},
			db:                        wdb,
			accountsDB:                accDB,
			blockRangeDAO:             &BlockRangeSequentialDAO{wdb.client},
			chainClient:               tc,
			balanceCacher:             balance.NewCacherWithTTL(5 * time.Minute),
			feed:                      &event.Feed{},
			noLimit:                   false,
			tokenManager:              tokenManager,
			blocksLoadedCh:            blockChannel,
			defaultNodeBlockChunkSize: scanRange,
			fromBlockNumber:           big.NewInt(0),
		},
		blockChainState:              blockchainstate.NewBlockChainState(),
		contractMaker:                maker,
		nonceCheckIntervalIterations: 2,
		logsCheckIntervalIterations:  2,
	}

	acc := &accounts.Account{
		Address: ethtypes.BytesToAddress(address.Bytes()),
		Type:    accounts.AccountTypeWatch,
		Name:    address.String(),
		ColorID: multicommon.CustomizationColorPrimary,
		Emoji:   "emoji",
	}
	err = accDB.SaveOrUpdateAccounts([]*accounts.Account{acc}, false)
	require.NoError(t, err)

	ctx := context.Background()
	tc.currentBlock = 3
	for i := 0; i < 3; i++ {
		err := cmd.Run(ctx)
		require.NoError(t, err)
		close(blockChannel)

		foundBlocks := []*DBHeader{}
		for {
			bloks, ok := <-blockChannel
			if !ok {
				break
			}
			foundBlocks = append(foundBlocks, bloks...)
		}

		numbers := []int64{}
		for _, block := range foundBlocks {
			numbers = append(numbers, block.Number.Int64())
		}
		if i == 2 {
			require.Equal(t, 2, len(foundBlocks), "blocks", numbers)
		} else {
			require.Equal(t, 0, len(foundBlocks), "no blocks expected to be found")
		}
		blockChannel = make(chan []*DBHeader, 10)
		cmd.blocksLoadedCh = blockChannel
		tc.currentBlock += uint64(scanRange)
	}
}

func TestFetchNewBlocksCommand(t *testing.T) {
	appdb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	mediaServer, err := server.NewMediaServer(appdb, nil, nil, db)
	require.NoError(t, err)

	wdb := NewDB(db)
	blockChannel := make(chan []*DBHeader, 10)

	address1 := common.HexToAddress("0x1234")
	address2 := common.HexToAddress("0x5678")
	accDB, err := accounts.NewDB(appdb)
	require.NoError(t, err)

	for _, address := range []*common.Address{&address1, &address2} {
		acc := &accounts.Account{
			Address: ethtypes.BytesToAddress(address.Bytes()),
			Type:    accounts.AccountTypeWatch,
			Name:    address.String(),
			ColorID: multicommon.CustomizationColorPrimary,
			Emoji:   "emoji",
		}
		err = accDB.SaveOrUpdateAccounts([]*accounts.Account{acc}, false)
		require.NoError(t, err)
	}

	tc := &TestClient{
		t:                      t,
		balances:               map[common.Address][][]int{},
		outgoingERC20Transfers: map[common.Address][]testERC20Transfer{},
		incomingERC20Transfers: map[common.Address][]testERC20Transfer{},
		callsCounter:           map[string]int{},
		currentBlock:           1,
	}
	//tc.printPreparedData = true

	config := statusRpc.ClientConfig{
		Client:          nil,
		UpstreamChainID: 1,
		Networks:        []params.Network{},
		DB:              db,
		WalletFeed:      nil,
		ProviderConfigs: nil,
	}
	client, _ := statusRpc.NewClient(config)

	client.SetClient(tc.NetworkID(), tc)

	tokenManager := token.NewTokenManager(db, client, community.NewManager(appdb, nil, nil), network.NewManager(appdb), appdb, mediaServer, nil, nil, nil, token.NewPersistence(db))

	tokenManager.SetTokens([]*token.Token{
		{
			Address:  tokenTXXAddress,
			Symbol:   "TXX",
			Decimals: 18,
			ChainID:  tc.NetworkID(),
			Name:     "Test Token 1",
			Verified: true,
		},
		{
			Address:  tokenTXYAddress,
			Symbol:   "TXY",
			Decimals: 18,
			ChainID:  tc.NetworkID(),
			Name:     "Test Token 2",
			Verified: true,
		},
	})

	cmd := &findNewBlocksCommand{
		findBlocksCommand: &findBlocksCommand{
			accounts:                  []common.Address{address1, address2},
			db:                        wdb,
			accountsDB:                accDB,
			blockRangeDAO:             &BlockRangeSequentialDAO{wdb.client},
			chainClient:               tc,
			balanceCacher:             balance.NewCacherWithTTL(5 * time.Minute),
			feed:                      &event.Feed{},
			noLimit:                   false,
			fromBlockNumber:           big.NewInt(int64(tc.currentBlock)),
			tokenManager:              tokenManager,
			blocksLoadedCh:            blockChannel,
			defaultNodeBlockChunkSize: DefaultNodeBlockChunkSize,
		},
		contractMaker:                tokenManager.ContractMaker,
		blockChainState:              blockchainstate.NewBlockChainState(),
		nonceCheckIntervalIterations: nonceCheckIntervalIterations,
		logsCheckIntervalIterations:  logsCheckIntervalIterations,
	}

	ctx := context.Background()

	// I don't prepare lots of data and a loop, as I just need to verify a few cases

	// Verify that cmd.fromBlockNumber stays the same
	tc.prepareBalanceHistory(int(tc.currentBlock))
	tc.prepareTokenBalanceHistory(int(tc.currentBlock))
	err = cmd.Run(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), cmd.fromBlockNumber.Uint64())

	// Verify that cmd.fromBlockNumber is incremented, equal to the head block number
	tc.currentBlock = 2 // this is the head block number that will be returned by the mock client
	tc.prepareBalanceHistory(int(tc.currentBlock))
	tc.prepareTokenBalanceHistory(int(tc.currentBlock))
	err = cmd.Run(ctx)
	require.NoError(t, err)
	require.Equal(t, tc.currentBlock, cmd.fromBlockNumber.Uint64())

	// Verify that blocks are found and cmd.fromBlockNumber is incremented
	tc.resetCounter()
	tc.currentBlock = 3
	tc.balances = map[common.Address][][]int{
		address1: {{3, 1, 0}},
		address2: {{3, 1, 0}},
	}
	tc.incomingERC20Transfers = map[common.Address][]testERC20Transfer{
		address1: {{big.NewInt(3), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType}},
		address2: {{big.NewInt(3), tokenTXYAddress, big.NewInt(1), walletcommon.Erc20TransferEventType}},
	}
	tc.prepareBalanceHistory(int(tc.currentBlock))
	tc.prepareTokenBalanceHistory(int(tc.currentBlock))

	group := async.NewGroup(ctx)
	group.Add(cmd.Command()) // This is an infinite command, I can't use WaitAsync() here to wait for it to finish

	expectedBlocksNumber := 3 // ETH block is found twice for each account as we don't handle addresses in MockClient. A block with ERC20 transfer is found once
	blocksFound := 0
	stop := false
	for stop == false {
		select {
		case <-ctx.Done():
			require.Fail(t, "context done")
			stop = true
		case <-blockChannel:
			blocksFound++
		case <-time.After(100 * time.Millisecond):
			stop = true
		}
	}
	group.Stop()
	group.Wait()
	require.Equal(t, expectedBlocksNumber, blocksFound)
	require.Equal(t, tc.currentBlock, cmd.fromBlockNumber.Uint64())
	// We must check all the logs for all accounts with a single iteration of eth_getLogs call
	require.Equal(t, 3, tc.callsCounter["FilterLogs"], "calls to FilterLogs")
}

type TestClientWithError struct {
	*TestClient
}

func (tc *TestClientWithError) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	tc.incCounter("BlockByNumber")
	if tc.traceAPICalls {
		tc.t.Log("BlockByNumber", number)
	}

	return nil, errors.New("Network error")
}

type BlockRangeSequentialDAOMockError struct {
	*BlockRangeSequentialDAO
}

func (b *BlockRangeSequentialDAOMockError) getBlockRange(chainID uint64, address common.Address) (blockRange *ethTokensBlockRanges, exists bool, err error) {
	return nil, true, errors.New("DB error")
}

type BlockRangeSequentialDAOMockSuccess struct {
	*BlockRangeSequentialDAO
}

func (b *BlockRangeSequentialDAOMockSuccess) getBlockRange(chainID uint64, address common.Address) (blockRange *ethTokensBlockRanges, exists bool, err error) {
	return newEthTokensBlockRanges(), true, nil
}

func TestLoadBlocksAndTransfersCommand_FiniteFinishedInfiniteRunning(t *testing.T) {
	appdb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	config := statusRpc.ClientConfig{
		Client:          nil,
		UpstreamChainID: 1,
		Networks:        []params.Network{},
		DB:              db,
		WalletFeed:      nil,
		ProviderConfigs: nil,
	}
	client, _ := statusRpc.NewClient(config)

	maker, _ := contracts.NewContractMaker(client)

	wdb := NewDB(db)
	tc := &TestClient{
		t:            t,
		callsCounter: map[string]int{},
	}
	accDB, err := accounts.NewDB(appdb)
	require.NoError(t, err)

	cmd := &loadBlocksAndTransfersCommand{
		accounts:    []common.Address{common.HexToAddress("0x1234")},
		chainClient: tc,
		blockDAO:    &BlockDAO{db},
		blockRangeDAO: &BlockRangeSequentialDAOMockSuccess{
			&BlockRangeSequentialDAO{
				wdb.client,
			},
		},
		accountsDB:    accDB,
		db:            wdb,
		contractMaker: maker,
	}

	ctx, cancel := context.WithCancel(context.Background())
	group := async.NewGroup(ctx)

	group.Add(cmd.Command(1 * time.Millisecond))

	select {
	case <-ctx.Done():
		cancel() // linter is not happy if cancel is not called on all code paths
		t.Log("Done")
	case <-group.WaitAsync():
		require.True(t, cmd.isStarted())

		// Test that it stops if canceled
		cancel()
		require.NoError(t, utils.Eventually(func() error {
			if !cmd.isStarted() {
				return nil
			}
			return errors.New("command is still running")
		}, 100*time.Millisecond, 10*time.Millisecond))
	}
}

func TestTransfersCommand_RetryAndQuitOnMaxError(t *testing.T) {
	tc := &TestClientWithError{
		&TestClient{
			t:            t,
			callsCounter: map[string]int{},
		},
	}

	address := common.HexToAddress("0x1234")
	cmd := &transfersCommand{
		chainClient: tc,
		address:     address,
		eth: &ETHDownloader{
			chainClient: tc,
			accounts:    []common.Address{address},
		},
		blockNums: []*big.Int{big.NewInt(1)},
	}

	ctx := context.Background()
	group := async.NewGroup(ctx)

	runner := cmd.Runner(1 * time.Millisecond)
	group.Add(runner.Run)

	select {
	case <-ctx.Done():
		t.Log("Done")
	case <-group.WaitAsync():
		errorCounter := runner.(async.FiniteCommandWithErrorCounter).ErrorCounter
		require.Equal(t, errorCounter.MaxErrors(), tc.callsCounter["BlockByNumber"])

		_, expectedErr := tc.BlockByNumber(context.TODO(), nil)
		require.Error(t, expectedErr, errorCounter.Error())
	}
}
