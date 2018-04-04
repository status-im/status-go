package transactions

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/transactions/fake"
	. "github.com/status-im/status-go/t/utils"
)

type TxQueueTestSuite struct {
	suite.Suite
	rpcClientMockCtrl *gomock.Controller
	rpcClientMock     *MocktestRPCClientProvider
	server            *gethrpc.Server
	client            *gethrpc.Client
	txServiceMockCtrl *gomock.Controller
	txServiceMock     *fake.MockPublicTransactionPoolAPI
	nodeConfig        *params.NodeConfig

	manager *Manager
}

func (s *TxQueueTestSuite) SetupTest() {
	s.rpcClientMockCtrl = gomock.NewController(s.T())
	s.txServiceMockCtrl = gomock.NewController(s.T())

	s.rpcClientMock = NewMocktestRPCClientProvider(s.rpcClientMockCtrl)

	s.server, s.txServiceMock = fake.NewTestServer(s.txServiceMockCtrl)
	s.client = gethrpc.DialInProc(s.server)
	rpclient, _ := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	s.rpcClientMock.EXPECT().RPCClient().Return(rpclient)
	nodeConfig, err := params.NewNodeConfig("/tmp", "", params.RopstenNetworkID, true)
	s.Require().NoError(err)
	s.nodeConfig = nodeConfig

	s.manager = NewManager(s.rpcClientMock)
	s.manager.DisableNotificactions()
	s.manager.completionTimeout = time.Second
	s.manager.rpcCallTimeout = time.Second
	s.manager.Start(params.RopstenNetworkID)
}

func (s *TxQueueTestSuite) TearDownTest() {
	s.manager.Stop()
	s.rpcClientMockCtrl.Finish()
	s.txServiceMockCtrl.Finish()
	s.server.Stop()
	s.client.Close()
}

var (
	testGas      = hexutil.Uint64(defaultGas + 1)
	testGasPrice = (*hexutil.Big)(big.NewInt(10))
	testNonce    = hexutil.Uint64(10)
)

func (s *TxQueueTestSuite) setupTransactionPoolAPI(tx *QueuedTx, returnNonce, resultNonce hexutil.Uint64, account *account.SelectedExtKey, txErr error) {
	// Expect calls to gas functions only if there are no user defined values.
	// And also set the expected gas and gas price for RLP encoding the expected tx.
	var usedGas hexutil.Uint64
	var usedGasPrice *big.Int
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), account.Address, gethrpc.PendingBlockNumber).Return(&returnNonce, nil)
	if tx.Args.GasPrice == nil {
		usedGasPrice = (*big.Int)(testGasPrice)
		s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(usedGasPrice, nil)
	} else {
		usedGasPrice = (*big.Int)(tx.Args.GasPrice)
	}
	if tx.Args.Gas == nil {
		s.txServiceMock.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(testGas, nil)
		usedGas = testGas
	} else {
		usedGas = *tx.Args.Gas
	}
	// Prepare the transaction anD RLP encode it.
	data := s.rlpEncodeTx(tx, s.nodeConfig, account, &resultNonce, usedGas, usedGasPrice)
	// Expect the RLP encoded transaction.
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), data).Return(gethcommon.Hash{}, txErr)
}

func (s *TxQueueTestSuite) rlpEncodeTx(tx *QueuedTx, config *params.NodeConfig, account *account.SelectedExtKey, nonce *hexutil.Uint64, gas hexutil.Uint64, gasPrice *big.Int) hexutil.Bytes {
	newTx := types.NewTransaction(
		uint64(*nonce),
		gethcommon.Address(*tx.Args.To),
		tx.Args.Value.ToInt(),
		uint64(gas),
		gasPrice,
		[]byte(tx.Args.Input),
	)
	chainID := big.NewInt(int64(config.NetworkID))
	signedTx, err := types.SignTx(newTx, types.NewEIP155Signer(chainID), account.AccountKey.PrivateKey)
	s.NoError(err)
	data, err := rlp.EncodeToBytes(signedTx)
	s.NoError(err)
	return hexutil.Bytes(data)
}

func (s *TxQueueTestSuite) TestCompleteTransaction() {
	key, _ := crypto.GenerateKey()
	selectedAccount := &account.SelectedExtKey{
		Address:    account.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	testCases := []struct {
		name     string
		gas      *hexutil.Uint64
		gasPrice *hexutil.Big
	}{
		{
			"noGasDef",
			nil,
			nil,
		},
		{
			"gasDefined",
			&testGas,
			nil,
		},
		{
			"gasPriceDefined",
			nil,
			testGasPrice,
		},
		{
			"inputPassedInLegacyDataField",
			nil,
			testGasPrice,
		},
	}

	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			s.SetupTest()
			tx := Create(context.Background(), SendTxArgs{
				From:     account.FromAddress(TestConfig.Account1.Address),
				To:       account.ToAddress(TestConfig.Account2.Address),
				Gas:      testCase.gas,
				GasPrice: testCase.gasPrice,
			})
			s.setupTransactionPoolAPI(tx, testNonce, testNonce, selectedAccount, nil)

			s.NoError(s.manager.QueueTransaction(tx))
			w := make(chan struct{})
			var (
				hash gethcommon.Hash
				err  error
			)
			go func() {
				hash, err = s.manager.CompleteTransaction(tx.ID, selectedAccount)
				s.NoError(err)
				close(w)
			}()

			rst := s.manager.WaitForTransaction(tx)
			// Check that error is assigned to the transaction.
			s.NoError(rst.Error)
			// Transaction should be already removed from the queue.
			s.False(s.manager.TransactionQueue().Has(tx.ID))
			s.NoError(WaitClosed(w, time.Second))
			s.Equal(hash, rst.Hash)
		})
	}
}

func (s *TxQueueTestSuite) TestCompleteTransactionMultipleTimes() {
	key, _ := crypto.GenerateKey()
	selectedAccount := &account.SelectedExtKey{
		Address:    account.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}

	tx := Create(context.Background(), SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	})

	s.setupTransactionPoolAPI(tx, testNonce, testNonce, selectedAccount, nil)

	err := s.manager.QueueTransaction(tx)
	s.NoError(err)

	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		completedTx  int
		inprogressTx int
		txCount      = 3
	)
	for i := 0; i < txCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.manager.CompleteTransaction(tx.ID, selectedAccount)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				completedTx++
			} else if err == ErrQueuedTxInProgress {
				inprogressTx++
			} else {
				s.Fail("tx failed with unexpected error: ", err.Error())
			}
		}()
	}

	rst := s.manager.WaitForTransaction(tx)
	// Check that error is assigned to the transaction.
	s.NoError(rst.Error)
	// Transaction should be already removed from the queue.
	s.False(s.manager.TransactionQueue().Has(tx.ID))

	// Wait for all CompleteTransaction calls.
	wg.Wait()
	s.Equal(1, completedTx, "only 1 tx expected to be completed")
	s.Equal(txCount-1, inprogressTx, "txs expected to be reported as inprogress")
}

func (s *TxQueueTestSuite) TestAccountMismatch() {
	selectedAccount := &account.SelectedExtKey{
		Address: account.FromAddress(TestConfig.Account2.Address),
	}

	tx := Create(context.Background(), SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	})

	s.NoError(s.manager.QueueTransaction(tx))

	_, err := s.manager.CompleteTransaction(tx.ID, selectedAccount)
	s.Equal(err, ErrInvalidCompleteTxSender)

	// Transaction should stay in the queue as mismatched accounts
	// is a recoverable error.
	s.True(s.manager.TransactionQueue().Has(tx.ID))
}

func (s *TxQueueTestSuite) TestDiscardTransaction() {
	tx := Create(context.Background(), SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	})

	s.NoError(s.manager.QueueTransaction(tx))
	w := make(chan struct{})
	go func() {
		s.NoError(s.manager.DiscardTransaction(tx.ID))
		close(w)
	}()

	rst := s.manager.WaitForTransaction(tx)
	s.Equal(ErrQueuedTxDiscarded, rst.Error)
	// Transaction should be already removed from the queue.
	s.False(s.manager.TransactionQueue().Has(tx.ID))
	s.NoError(WaitClosed(w, time.Second))
}

func (s *TxQueueTestSuite) TestCompletionTimedOut() {
	tx := Create(context.Background(), SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	})

	s.NoError(s.manager.QueueTransaction(tx))
	rst := s.manager.WaitForTransaction(tx)
	s.Equal(ErrQueuedTxTimedOut, rst.Error)
}

// TestLocalNonce verifies that local nonce will be used unless
// upstream nonce is updated and higher than a local
// in test we will run 3 transaction with nonce zero returned by upstream
// node, after each call local nonce will be incremented
// then, we return higher nonce, as if another node was used to send 2 transactions
// upstream nonce will be equal to 5, we update our local counter to 5+1
// as the last step, we verify that if tx failed nonce is not updated
func (s *TxQueueTestSuite) TestLocalNonce() {
	txCount := 3
	key, _ := crypto.GenerateKey()
	selectedAccount := &account.SelectedExtKey{
		Address:    account.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	nonce := hexutil.Uint64(0)
	for i := 0; i < txCount; i++ {
		tx := Create(context.Background(), SendTxArgs{
			From: account.FromAddress(TestConfig.Account1.Address),
			To:   account.ToAddress(TestConfig.Account2.Address),
		})
		s.setupTransactionPoolAPI(tx, nonce, hexutil.Uint64(i), selectedAccount, nil)
		s.NoError(s.manager.QueueTransaction(tx))
		hash, err := s.manager.CompleteTransaction(tx.ID, selectedAccount)
		rst := s.manager.WaitForTransaction(tx)
		// simple sanity checks
		s.NoError(err)
		s.NoError(rst.Error)
		s.Equal(rst.Hash, hash)
		resultNonce, _ := s.manager.localNonce.Load(tx.Args.From)
		s.Equal(uint64(i)+1, resultNonce.(uint64))
	}
	nonce = hexutil.Uint64(5)
	tx := Create(context.Background(), SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	})
	s.setupTransactionPoolAPI(tx, nonce, nonce, selectedAccount, nil)
	s.NoError(s.manager.QueueTransaction(tx))
	hash, err := s.manager.CompleteTransaction(tx.ID, selectedAccount)
	rst := s.manager.WaitForTransaction(tx)
	s.NoError(err)
	s.NoError(rst.Error)
	s.Equal(rst.Hash, hash)
	resultNonce, _ := s.manager.localNonce.Load(tx.Args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))

	testErr := errors.New("test")
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), selectedAccount.Address, gethrpc.PendingBlockNumber).Return(nil, testErr)
	tx = Create(context.Background(), SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	})
	s.NoError(s.manager.QueueTransaction(tx))
	_, err = s.manager.CompleteTransaction(tx.ID, selectedAccount)
	rst = s.manager.WaitForTransaction(tx)
	s.EqualError(testErr, err.Error())
	s.EqualError(testErr, rst.Error.Error())
	resultNonce, _ = s.manager.localNonce.Load(tx.Args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))
}
