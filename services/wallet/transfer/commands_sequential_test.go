package transfer

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/balance"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

type TestClient struct {
	t *testing.T
	// [][block, newBalance, nonceDiff]
	balances       [][]int
	balanceHistory map[uint64]*big.Int
	nonceHistory   map[uint64]uint64
}

func (tc TestClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	tc.t.Log("BatchCallContext")
	return nil
}

func (tc TestClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	tc.t.Log("HeaderByHash")
	return nil, nil
}

func (tc TestClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	tc.t.Log("BlockByHash")
	return nil, nil
}

func (tc TestClient) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	tc.t.Log("BlockByNumber")
	return nil, nil
}

func (tc TestClient) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	nonce := tc.nonceHistory[blockNumber.Uint64()]

	tc.t.Log("NonceAt", blockNumber, "result:", nonce)
	return nonce, nil
}

func (tc TestClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	tc.t.Log("FilterLogs")
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

	tc.t.Log("=========================================")
	tc.t.Log(tc.balanceHistory)
	tc.t.Log(tc.nonceHistory)
	tc.t.Log("=========================================")
}

func (tc TestClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	balance := tc.balanceHistory[blockNumber.Uint64()]

	tc.t.Log("BalanceAt", blockNumber, "result:", balance)
	return balance, nil
}

func (tc *TestClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	tc.t.Log("HeaderByNumber", number)
	header := &types.Header{
		Number: number,
		Time:   0,
	}

	return header, nil
}

func (tc TestClient) FullTransactionByBlockNumberAndIndex(ctx context.Context, blockNumber *big.Int, index uint) (*chain.FullTransaction, error) {
	tc.t.Log("FullTransactionByBlockNumberAndIndex")
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

func (tc TestClient) GetBaseFeeFromBlock(blockNumber *big.Int) (string, error) {
	tc.t.Log("GetBaseFeeFromBloc")
	return "", nil
}

func (tc TestClient) NetworkID() uint64 {
	return 1
}

func (tc TestClient) ToBigInt() *big.Int {
	tc.t.Log("ToBigInt")
	return nil
}

type findBlockCase struct {
	balanceChanges      [][]int
	fromBlock           int64
	toBlock             int64
	expectedBlocksFound int
}

var findBlocksCommandCases = []findBlockCase{
	{
		balanceChanges: [][]int{
			{5, 1, 0},
			{20, 2, 0},
			{45, 1, 1},
			{46, 50, 0},
			{75, 0, 1},
		},
		toBlock:             100,
		expectedBlocksFound: 5,
	},
	{
		balanceChanges:      [][]int{},
		toBlock:             100,
		expectedBlocksFound: 0,
	},
}

func TestFindBlocksCommand(t *testing.T) {
	for _, testCase := range findBlocksCommandCases {

		ctx := context.Background()
		group := async.NewGroup(ctx)

		db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
		require.NoError(t, err)
		tm := &TransactionManager{db, nil, nil, nil, nil, nil, nil}

		wdb := NewDB(db)
		tc := &TestClient{
			t:        t,
			balances: testCase.balanceChanges,
		}
		tc.prepareBalanceHistory(100)
		blockChannel := make(chan []*DBHeader, 100)
		fbc := &findBlocksCommand{
			account:            common.HexToAddress("0x1234"),
			db:                 wdb,
			blockRangeDAO:      &BlockRangeSequentialDAO{wdb.client},
			chainClient:        tc,
			balanceCacher:      balance.NewCache(),
			feed:               &event.Feed{},
			noLimit:            false,
			fromBlockNumber:    big.NewInt(testCase.fromBlock),
			toBlockNumber:      big.NewInt(testCase.toBlock),
			transactionManager: tm,
			blocksLoadedCh:     blockChannel,
		}
		group.Add(fbc.Command())

		select {
		case <-ctx.Done():
			t.Log("ERROR")
		case <-group.WaitAsync():
			close(blockChannel)
			require.Equal(t, testCase.expectedBlocksFound, len(<-blockChannel))
		}
	}
}
