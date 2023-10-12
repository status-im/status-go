package transfer

import (
	"context"
	"math/big"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/contracts/ethscan"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/balance"
	"github.com/status-im/status-go/t/helpers"

	"github.com/status-im/status-go/params"
	statusRpc "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/walletdatabase"
)

type TestClient struct {
	t *testing.T
	// [][block, newBalance, nonceDiff]
	balances               [][]int
	outgoingERC20Transfers []testERC20Transfer
	incomingERC20Transfers []testERC20Transfer
	balanceHistory         map[uint64]*big.Int
	tokenBalanceHistory    map[common.Address]map[uint64]*big.Int
	nonceHistory           map[uint64]uint64
	traceAPICalls          bool
	printPreparedData      bool
	rw                     sync.RWMutex
	callsCounter           map[string]int
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
	//checking only ERC20 for now
	incomingAddress := q.Topics[len(q.Topics)-1]
	allTransfers := tc.incomingERC20Transfers
	if len(incomingAddress) == 0 {
		allTransfers = tc.outgoingERC20Transfers
	}

	logs := []types.Log{}
	for _, transfer := range allTransfers {
		if transfer.block.Cmp(q.FromBlock) >= 0 && transfer.block.Cmp(q.ToBlock) <= 0 {
			logs = append(logs, types.Log{
				BlockNumber: transfer.block.Uint64(),
				BlockHash:   common.BigToHash(transfer.block),
			})
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
		transfersPerToken[transfer.address] = append(transfersPerToken[transfer.address], transfer)
	}

	for _, transfer := range tc.incomingERC20Transfers {
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
		transfers = append(transfers, testERC20Transfer{big.NewInt(int64(toBlock + 1)), token, big.NewInt(0)})

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
	block   *big.Int
	address common.Address
	amount  *big.Int
}

type findBlockCase struct {
	balanceChanges         [][]int
	ERC20BalanceChanges    [][]int
	fromBlock              int64
	toBlock                int64
	rangeSize              int
	expectedBlocksFound    int
	outgoingERC20Transfers []testERC20Transfer
	incomingERC20Transfers []testERC20Transfer
	label                  string
	expectedCalls          map[string]int
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
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1)},
		},
		toBlock:             100,
		expectedBlocksFound: 6,
		expectedCalls: map[string]int{
			"BalanceAt": 27,
			//TODO(rasom) NonceAt is flaky, sometimes it's called 18 times, sometimes 17
			//to be investigated
			//"NonceAt":        18,
			"FilterLogs":     10,
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
			"FilterLogs":     10,
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
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1)},
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
			{big.NewInt(80), tokenTXXAddress, big.NewInt(1)},
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1)},
		},
		expectedCalls: map[string]int{
			"FilterLogs":   3,
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
			{big.NewInt(7), tokenTXYAddress, big.NewInt(1)},
			{big.NewInt(6), tokenTXXAddress, big.NewInt(1)},
		},
		expectedCalls: map[string]int{
			"FilterLogs":   3,
			"CallContract": 5,
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
			"FilterLogs": 2,
			// no contract calls as ERC20 is not checked
			"CallContract": 0,
		},
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

	//cases = append([]findBlockCase{}, case10)

	return cases
}

var tokenTXXAddress = common.HexToAddress("0x53211")
var tokenTXYAddress = common.HexToAddress("0x73211")

func TestFindBlocksCommand(t *testing.T) {
	for idx, testCase := range getCases() {
		t.Log("case #", idx)
		ctx := context.Background()
		group := async.NewGroup(ctx)

		db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
		require.NoError(t, err)
		tm := &TransactionManager{db, nil, nil, nil, nil, nil, nil}

		wdb := NewDB(db)
		tc := &TestClient{
			t:                      t,
			balances:               testCase.balanceChanges,
			outgoingERC20Transfers: testCase.outgoingERC20Transfers,
			incomingERC20Transfers: testCase.incomingERC20Transfers,
			callsCounter:           map[string]int{},
		}
		//tc.traceAPICalls = true
		//tc.printPreparedData = true
		tc.prepareBalanceHistory(100)
		tc.prepareTokenBalanceHistory(100)
		blockChannel := make(chan []*DBHeader, 100)
		rangeSize := 20
		if testCase.rangeSize != 0 {
			rangeSize = testCase.rangeSize
		}
		client, _ := statusRpc.NewClient(nil, 1, params.UpstreamRPCConfig{Enabled: false, URL: ""}, []params.Network{}, db)
		client.SetClient(tc.NetworkID(), tc)
		tokenManager := token.NewTokenManager(db, client, network.NewManager(db))
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
		fbc := &findBlocksCommand{
			account:                   common.HexToAddress("0x1234"),
			db:                        wdb,
			blockRangeDAO:             &BlockRangeSequentialDAO{wdb.client},
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
