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

	"github.com/stretchr/testify/mock"
	"golang.org/x/exp/slices" // since 1.21, this is in the standard library

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/contracts/ethscan"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/balance"
	"github.com/status-im/status-go/t/helpers"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	statusRpc "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
	"github.com/status-im/status-go/walletdatabase"
)

type TestClient struct {
	t *testing.T
	// [][block, newBalance, nonceDiff]
	balances                       [][]int
	outgoingERC20Transfers         []testERC20Transfer
	incomingERC20Transfers         []testERC20Transfer
	outgoingERC1155SingleTransfers []testERC20Transfer
	incomingERC1155SingleTransfers []testERC20Transfer
	balanceHistory                 map[uint64]*big.Int
	tokenBalanceHistory            map[common.Address]map[uint64]*big.Int
	nonceHistory                   map[uint64]uint64
	traceAPICalls                  bool
	printPreparedData              bool
	rw                             sync.RWMutex
	callsCounter                   map[string]int
	currentBlock                   uint64
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
	tc.incCounter("HeaderByHash")
	if tc.traceAPICalls {
		tc.t.Log("HeaderByHash")
	}
	return nil, nil
}

func (tc *TestClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	tc.incCounter("BlockByHash")
	if tc.traceAPICalls {
		tc.t.Log("BlockByHash")
	}
	return nil, nil
}

func (tc *TestClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	tc.incCounter("BlockByNumber")
	if tc.traceAPICalls {
		tc.t.Log("BlockByNumber")
	}
	return nil, nil
}

func (tc *TestClient) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	tc.incCounter("NonceAt")
	nonce := tc.nonceHistory[blockNumber.Uint64()]
	if tc.traceAPICalls {
		tc.t.Log("NonceAt", blockNumber, "result:", nonce)
	}
	return nonce, nil
}

func (tc *TestClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	tc.incCounter("FilterLogs")
	if tc.traceAPICalls {
		tc.t.Log("FilterLogs")
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
			allTransfers = append(allTransfers, tc.incomingERC1155SingleTransfers...)
		}
		if len(from) > 0 {
			allTransfers = append(allTransfers, tc.outgoingERC1155SingleTransfers...)
		}
	}

	if slices.Contains(signatures, erc20TransferSignature) {
		from := q.Topics[1]
		to := q.Topics[2]
		if len(to) > 0 {
			allTransfers = append(allTransfers, tc.incomingERC20Transfers...)
		}
		if len(from) > 0 {
			allTransfers = append(allTransfers, tc.outgoingERC20Transfers...)
		}
	}

	logs := []types.Log{}
	for _, transfer := range allTransfers {
		if transfer.block.Cmp(q.FromBlock) >= 0 && transfer.block.Cmp(q.ToBlock) <= 0 {
			log := types.Log{
				BlockNumber: transfer.block.Uint64(),
				BlockHash:   common.BigToHash(transfer.block),
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

func (tc *TestClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	tc.incCounter("BalanceAt")
	balance := tc.balanceHistory[blockNumber.Uint64()]

	if tc.traceAPICalls {
		tc.t.Log("BalanceAt", blockNumber, "result:", balance)
	}
	return balance, nil
}

func (tc *TestClient) tokenBalanceAt(token common.Address, blockNumber *big.Int) *big.Int {
	balance := tc.tokenBalanceHistory[token][blockNumber.Uint64()]
	if balance == nil {
		balance = big.NewInt(0)
	}

	if tc.traceAPICalls {
		tc.t.Log("tokenBalanceAt", token, blockNumber, "result:", balance)
	}
	return balance
}

func (tc *TestClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	tc.incCounter("HeaderByNumber")
	if number == nil {
		number = big.NewInt(int64(tc.currentBlock))
	}

	if tc.traceAPICalls {
		tc.t.Log("HeaderByNumber", number)
	}
	header := &types.Header{
		Number: number,
		Time:   0,
	}

	return header, nil
}

func (tc *TestClient) FullTransactionByBlockNumberAndIndex(ctx context.Context, blockNumber *big.Int, index uint) (*chain.FullTransaction, error) {
	tc.incCounter("FullTransactionByBlockNumberAndIndex")
	if tc.traceAPICalls {
		tc.t.Log("FullTransactionByBlockNumberAndIndex")
	}
	blockHash := common.BigToHash(blockNumber)
	tx := &chain.FullTransaction{
		Tx: &types.Transaction{},
		TxExtraInfo: chain.TxExtraInfo{
			BlockNumber: (*hexutil.Big)(big.NewInt(0)),
			BlockHash:   &blockHash,
		},
	}

	return tx, nil
}

func (tc *TestClient) GetBaseFeeFromBlock(blockNumber *big.Int) (string, error) {
	tc.incCounter("GetBaseFeeFromBlock")
	if tc.traceAPICalls {
		tc.t.Log("GetBaseFeeFromBlock")
	}
	return "", nil
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

func (tc *TestClient) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	tc.incCounter("CodeAt")
	if tc.traceAPICalls {
		tc.t.Log("CodeAt", contract, blockNumber)
	}

	if ethscanAddress == contract {
		return []byte{1}, nil
	}

	return nil, nil
}

func (tc *TestClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	tc.incCounter("CallContract")
	if tc.traceAPICalls {
		tc.t.Log("CallContract", call, blockNumber, call.To)
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

		tokens := args[1].([]common.Address)
		balances := []*big.Int{}
		for _, token := range tokens {
			balances = append(balances, tc.tokenBalanceAt(token, blockNumber))
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
		balance := tc.tokenBalanceAt(*call.To, blockNumber)

		parsed, err := abi.JSON(strings.NewReader(ierc20.IERC20ABI))
		if err != nil {
			return nil, err
		}

		method := parsed.Methods["balanceOf"]
		output, err := method.Outputs.Pack(balance)
		if err != nil {
			tc.t.Log("ERROR on packing ERC20 balance", err)
			return nil, err
		}

		return output, nil
	}

	return nil, nil
}

func (tc *TestClient) prepareBalanceHistory(toBlock int) {
	var currentBlock, currentBalance, currentNonce int

	tc.balanceHistory = map[uint64]*big.Int{}
	tc.nonceHistory = map[uint64]uint64{}

	if len(tc.balances) == 0 {
		tc.balances = append(tc.balances, []int{toBlock + 1, 0, 0})
	} else {
		lastBlock := tc.balances[len(tc.balances)-1]
		tc.balances = append(tc.balances, []int{toBlock + 1, lastBlock[1], 0})
	}
	for _, change := range tc.balances {
		for blockN := currentBlock; blockN < change[0]; blockN++ {
			tc.balanceHistory[uint64(blockN)] = big.NewInt(int64(currentBalance))
			tc.nonceHistory[uint64(blockN)] = uint64(currentNonce)
		}
		currentBlock = change[0]
		currentBalance = change[1]
		currentNonce += change[2]
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
	transfersPerToken := map[common.Address][]testERC20Transfer{}
	for _, transfer := range tc.outgoingERC20Transfers {
		transfer.amount = new(big.Int).Neg(transfer.amount)
		transfer.eventType = walletcommon.Erc20TransferEventType
		transfersPerToken[transfer.address] = append(transfersPerToken[transfer.address], transfer)
	}

	for _, transfer := range tc.incomingERC20Transfers {
		transfer.eventType = walletcommon.Erc20TransferEventType
		transfersPerToken[transfer.address] = append(transfersPerToken[transfer.address], transfer)
	}

	for _, transfer := range tc.outgoingERC1155SingleTransfers {
		transfer.amount = new(big.Int).Neg(transfer.amount)
		transfer.eventType = walletcommon.Erc1155TransferSingleEventType
		transfersPerToken[transfer.address] = append(transfersPerToken[transfer.address], transfer)
	}

	for _, transfer := range tc.incomingERC1155SingleTransfers {
		transfer.eventType = walletcommon.Erc1155TransferSingleEventType
		transfersPerToken[transfer.address] = append(transfersPerToken[transfer.address], transfer)
	}

	tc.tokenBalanceHistory = map[common.Address]map[uint64]*big.Int{}

	for token, transfers := range transfersPerToken {
		sort.Slice(transfers, func(i, j int) bool {
			return transfers[i].block.Cmp(transfers[j].block) < 0
		})

		currentBlock := uint64(0)
		currentBalance := big.NewInt(0)

		tc.tokenBalanceHistory[token] = map[uint64]*big.Int{}
		transfers = append(transfers, testERC20Transfer{big.NewInt(int64(toBlock + 1)), token, big.NewInt(0), walletcommon.Erc20TransferEventType})

		for _, transfer := range transfers {
			for blockN := currentBlock; blockN < transfer.block.Uint64(); blockN++ {
				tc.tokenBalanceHistory[token][blockN] = new(big.Int).Set(currentBalance)
			}
			currentBlock = transfer.block.Uint64()
			currentBalance = new(big.Int).Add(currentBalance, transfer.amount)
		}
	}
	if tc.printPreparedData {
		tc.t.Log("========================================= ERC20 BALANCES")
		tc.t.Log(tc.tokenBalanceHistory)
		tc.t.Log("=========================================")
	}
}

func (tc *TestClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	tc.incCounter("CallContext")
	if tc.traceAPICalls {
		tc.t.Log("CallContext")
	}
	return nil
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
	tc.incCounter("EstimateGas")
	if tc.traceAPICalls {
		tc.t.Log("EstimateGas")
	}
	return 0, nil
}

func (tc *TestClient) PendingCodeAt(ctx context.Context, account common.Address) ([]byte, error) {
	tc.incCounter("PendingCodeAt")
	if tc.traceAPICalls {
		tc.t.Log("PendingCodeAt")
	}

	return nil, nil
}

func (tc *TestClient) PendingCallContract(ctx context.Context, call ethereum.CallMsg) ([]byte, error) {
	tc.incCounter("PendingCallContract")
	if tc.traceAPICalls {
		tc.t.Log("PendingCallContract")
	}

	return nil, nil
}

func (tc *TestClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	tc.incCounter("PendingNonceAt")
	if tc.traceAPICalls {
		tc.t.Log("PendingNonceAt")
	}

	return 0, nil
}

func (tc *TestClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	tc.incCounter("SuggestGasPrice")
	if tc.traceAPICalls {
		tc.t.Log("SuggestGasPrice")
	}

	return nil, nil
}

func (tc *TestClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	tc.incCounter("SendTransaction")
	if tc.traceAPICalls {
		tc.t.Log("SendTransaction")
	}

	return nil
}

func (tc *TestClient) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	tc.incCounter("SuggestGasTipCap")
	if tc.traceAPICalls {
		tc.t.Log("SuggestGasTipCap")
	}

	return nil, nil
}

func (tc *TestClient) BatchCallContextIgnoringLocalHandlers(ctx context.Context, b []rpc.BatchElem) error {
	tc.incCounter("BatchCallContextIgnoringLocalHandlers")
	if tc.traceAPICalls {
		tc.t.Log("BatchCallContextIgnoringLocalHandlers")
	}

	return nil
}

func (tc *TestClient) CallContextIgnoringLocalHandlers(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	tc.incCounter("CallContextIgnoringLocalHandlers")
	if tc.traceAPICalls {
		tc.t.Log("CallContextIgnoringLocalHandlers")
	}

	return nil
}

func (tc *TestClient) CallRaw(data string) string {
	tc.incCounter("CallRaw")
	if tc.traceAPICalls {
		tc.t.Log("CallRaw")
	}

	return ""
}

func (tc *TestClient) GetChainID() *big.Int {
	return big.NewInt(1)
}

func (tc *TestClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	tc.incCounter("SubscribeFilterLogs")
	if tc.traceAPICalls {
		tc.t.Log("SubscribeFilterLogs")
	}

	return nil, nil
}

func (tc *TestClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	tc.incCounter("TransactionReceipt")
	if tc.traceAPICalls {
		tc.t.Log("TransactionReceipt")
	}

	return nil, nil
}

func (tc *TestClient) TransactionByHash(ctx context.Context, txHash common.Hash) (*types.Transaction, bool, error) {
	tc.incCounter("TransactionByHash")
	if tc.traceAPICalls {
		tc.t.Log("TransactionByHash")
	}

	return nil, false, nil
}

func (tc *TestClient) BlockNumber(ctx context.Context) (uint64, error) {
	tc.incCounter("BlockNumber")
	if tc.traceAPICalls {
		tc.t.Log("BlockNumber")
	}

	return 0, nil
}
func (tc *TestClient) SetIsConnected(value bool) {
	if tc.traceAPICalls {
		tc.t.Log("SetIsConnected")
	}
}

func (tc *TestClient) GetIsConnected() bool {
	if tc.traceAPICalls {
		tc.t.Log("GetIsConnected")
	}

	return true
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

func TestFindBlocksCommand(t *testing.T) {
	for idx, testCase := range getCases() {
		t.Log("case #", idx+1)
		ctx := context.Background()
		group := async.NewGroup(ctx)

		appdb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
		require.NoError(t, err)

		db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
		require.NoError(t, err)
		tm := &TransactionManager{db, nil, nil, nil, nil, nil, nil, nil, nil, nil}

		wdb := NewDB(db)
		tc := &TestClient{
			t:                              t,
			balances:                       testCase.balanceChanges,
			outgoingERC20Transfers:         testCase.outgoingERC20Transfers,
			incomingERC20Transfers:         testCase.incomingERC20Transfers,
			outgoingERC1155SingleTransfers: testCase.outgoingERC1155SingleTransfers,
			incomingERC1155SingleTransfers: testCase.incomingERC1155SingleTransfers,
			callsCounter:                   map[string]int{},
		}
		// tc.traceAPICalls = true
		// tc.printPreparedData = true
		tc.prepareBalanceHistory(100)
		tc.prepareTokenBalanceHistory(100)
		blockChannel := make(chan []*DBHeader, 100)
		rangeSize := 20
		if testCase.rangeSize != 0 {
			rangeSize = testCase.rangeSize
		}
		client, _ := statusRpc.NewClient(nil, 1, params.UpstreamRPCConfig{Enabled: false, URL: ""}, []params.Network{}, db)
		client.SetClient(tc.NetworkID(), tc)
		tokenManager := token.NewTokenManager(db, client, network.NewManager(appdb))
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
		fbc := &findBlocksCommand{
			accounts:                  []common.Address{common.HexToAddress("0x1234")},
			db:                        wdb,
			blockRangeDAO:             &BlockRangeSequentialDAO{wdb.client},
			accountsDB:                accDB,
			chainClient:               tc,
			balanceCacher:             balance.NewCacherWithTTL(5 * time.Minute),
			feed:                      &event.Feed{},
			noLimit:                   false,
			fromBlockNumber:           big.NewInt(testCase.fromBlock),
			toBlockNumber:             big.NewInt(testCase.toBlock),
			transactionManager:        tm,
			blocksLoadedCh:            blockChannel,
			defaultNodeBlockChunkSize: rangeSize,
			tokenManager:              tokenManager,
		}
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
		}
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

	clients map[walletcommon.ChainID]*MockETHClient
}

func newMockChainClient() *MockChainClient {
	return &MockChainClient{
		clients: make(map[walletcommon.ChainID]*MockETHClient),
	}
}

func (m *MockChainClient) AbstractEthClient(chainID walletcommon.ChainID) (chain.BatchCallClient, error) {
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
	tm := &TransactionManager{db, nil, nil, nil, nil, nil, nil, nil, nil, nil}

	wdb := NewDB(db)
	blockChannel := make(chan []*DBHeader, 100)

	tc := &TestClient{
		t:                      t,
		balances:               [][]int{},
		outgoingERC20Transfers: []testERC20Transfer{},
		incomingERC20Transfers: []testERC20Transfer{},
		callsCounter:           map[string]int{},
		currentBlock:           100,
	}

	client, _ := statusRpc.NewClient(nil, 1, params.UpstreamRPCConfig{Enabled: false, URL: ""}, []params.Network{}, db)
	client.SetClient(tc.NetworkID(), tc)
	tokenManager := token.NewTokenManager(db, client, network.NewManager(appdb))

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
	tracker := transactions.NewPendingTxTracker(db, chainClient, nil, &event.Feed{}, transactions.PendingCheckInterval)
	accDB, err := accounts.NewDB(wdb.client)
	require.NoError(t, err)

	cmd := &loadBlocksAndTransfersCommand{
		accounts:           []common.Address{address},
		db:                 wdb,
		blockRangeDAO:      &BlockRangeSequentialDAO{wdb.client},
		blockDAO:           &BlockDAO{db},
		accountsDB:         accDB,
		chainClient:        tc,
		feed:               &event.Feed{},
		balanceCacher:      balance.NewCacherWithTTL(5 * time.Minute),
		transactionManager: tm,
		pendingTxManager:   tracker,
		tokenManager:       tokenManager,
		blocksLoadedCh:     blockChannel,
		omitHistory:        true,
	}

	tc.prepareBalanceHistory(int(tc.currentBlock))
	tc.prepareTokenBalanceHistory(int(tc.currentBlock))
	tc.traceAPICalls = true

	ctx := context.Background()
	group := async.NewGroup(ctx)

	fromNum := big.NewInt(0)
	toNum, err := getHeadBlockNumber(ctx, cmd.chainClient)
	require.NoError(t, err)
	err = cmd.fetchHistoryBlocks(ctx, group, address, fromNum, toNum, blockChannel)
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
	tm := &TransactionManager{db, nil, nil, nil, nil, nil, nil, nil, nil, nil}

	wdb := NewDB(db)
	blockChannel := make(chan []*DBHeader, 10)

	address := common.HexToAddress("0x1234")
	accDB, err := accounts.NewDB(wdb.client)
	require.NoError(t, err)

	for idx, testCase := range getNewBlocksCases() {
		t.Log("case #", idx+1)
		tc := &TestClient{
			t:                      t,
			balances:               testCase.balanceChanges,
			outgoingERC20Transfers: []testERC20Transfer{},
			incomingERC20Transfers: []testERC20Transfer{},
			callsCounter:           map[string]int{},
			currentBlock:           100,
		}

		client, _ := statusRpc.NewClient(nil, 1, params.UpstreamRPCConfig{Enabled: false, URL: ""}, []params.Network{}, db)
		client.SetClient(tc.NetworkID(), tc)
		tokenManager := token.NewTokenManager(db, client, network.NewManager(appdb))

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
				transactionManager:        tm,
				tokenManager:              tokenManager,
				blocksLoadedCh:            blockChannel,
				defaultNodeBlockChunkSize: DefaultNodeBlockChunkSize,
			},
		}
		tc.prepareBalanceHistory(int(tc.currentBlock))
		tc.prepareTokenBalanceHistory(int(tc.currentBlock))

		ctx := context.Background()
		blocks, _, err := cmd.findBlocksWithEthTransfers(ctx, address, big.NewInt(testCase.fromBlock), big.NewInt(testCase.toBlock))
		require.NoError(t, err)
		require.Equal(t, testCase.expectedBlocksFound, len(blocks), fmt.Sprintf("case %d: %s, blocks from %d to %d", idx+1, testCase.label, testCase.fromBlock, testCase.toBlock))
	}
}

func TestFetchNewBlocksCommand(t *testing.T) {
	appdb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	tm := &TransactionManager{db, nil, nil, nil, nil, nil, nil, nil, nil, nil}

	wdb := NewDB(db)
	blockChannel := make(chan []*DBHeader, 10)

	address1 := common.HexToAddress("0x1234")
	address2 := common.HexToAddress("0x5678")
	accDB, err := accounts.NewDB(wdb.client)
	require.NoError(t, err)

	tc := &TestClient{
		t:                      t,
		balances:               [][]int{},
		outgoingERC20Transfers: []testERC20Transfer{},
		incomingERC20Transfers: []testERC20Transfer{},
		callsCounter:           map[string]int{},
		currentBlock:           1,
	}

	client, _ := statusRpc.NewClient(nil, 1, params.UpstreamRPCConfig{Enabled: false, URL: ""}, []params.Network{}, db)
	client.SetClient(tc.NetworkID(), tc)
	tokenManager := token.NewTokenManager(db, client, network.NewManager(appdb))

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
			transactionManager:        tm,
			tokenManager:              tokenManager,
			blocksLoadedCh:            blockChannel,
			defaultNodeBlockChunkSize: DefaultNodeBlockChunkSize,
		},
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
	tc.balances = [][]int{
		{3, 1, 0},
	}
	tc.incomingERC20Transfers = []testERC20Transfer{
		{big.NewInt(3), tokenTXXAddress, big.NewInt(1), walletcommon.Erc20TransferEventType},
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
