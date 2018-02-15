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

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/transactions/fake"
	"github.com/status-im/status-go/geth/transactions/queue"
	. "github.com/status-im/status-go/t/utils"
)

func TestTxQueueTestSuite(t *testing.T) {
	suite.Run(t, new(TxQueueTestSuite))
}

type TxQueueTestSuite struct {
	suite.Suite
	nodeManagerMockCtrl    *gomock.Controller
	nodeManagerMock        *common.MockNodeManager
	accountManagerMockCtrl *gomock.Controller
	accountManagerMock     *common.MockAccountManager
	server                 *gethrpc.Server
	client                 *gethrpc.Client
	txServiceMockCtrl      *gomock.Controller
	txServiceMock          *fake.MockPublicTransactionPoolAPI
	nodeConfig             *params.NodeConfig

	manager *Manager
}

func (s *TxQueueTestSuite) SetupTest() {
	s.nodeManagerMockCtrl = gomock.NewController(s.T())
	s.accountManagerMockCtrl = gomock.NewController(s.T())
	s.txServiceMockCtrl = gomock.NewController(s.T())

	s.nodeManagerMock = common.NewMockNodeManager(s.nodeManagerMockCtrl)
	s.accountManagerMock = common.NewMockAccountManager(s.accountManagerMockCtrl)

	s.server, s.txServiceMock = fake.NewTestServer(s.txServiceMockCtrl)
	s.client = gethrpc.DialInProc(s.server)
	rpclient, _ := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	s.nodeManagerMock.EXPECT().RPCClient().Return(rpclient)
	nodeConfig, err := params.NewNodeConfig("/tmp", params.RopstenNetworkID, true)
	s.Require().NoError(err)
	s.nodeConfig = nodeConfig

	s.manager = NewManager(s.nodeManagerMock, s.accountManagerMock)
	s.manager.DisableNotificactions()
	s.manager.completionTimeout = time.Second
	s.manager.rpcCallTimeout = time.Second
	s.manager.Start()
}

func (s *TxQueueTestSuite) TearDownTest() {
	s.manager.Stop()
	s.nodeManagerMockCtrl.Finish()
	s.accountManagerMockCtrl.Finish()
	s.txServiceMockCtrl.Finish()
	s.server.Stop()
	s.client.Close()
}

var (
	testGas      = hexutil.Uint64(defaultGas + 1)
	testGasPrice = (*hexutil.Big)(big.NewInt(10))
	testNonce    = hexutil.Uint64(10)
)

func (s *TxQueueTestSuite) setupTransactionPoolAPI(tx *common.QueuedTx, returnNonce, resultNonce hexutil.Uint64, account *common.SelectedExtKey, txErr error) {
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

func (s *TxQueueTestSuite) rlpEncodeTx(tx *common.QueuedTx, config *params.NodeConfig, account *common.SelectedExtKey, nonce *hexutil.Uint64, gas hexutil.Uint64, gasPrice *big.Int) hexutil.Bytes {
	input := tx.Args.Input
	if input == nil {
		input = tx.Args.Data
	}
	newTx := types.NewTransaction(
		uint64(*nonce),
		gethcommon.Address(*tx.Args.To),
		tx.Args.Value.ToInt(),
		uint64(gas),
		gasPrice,
		[]byte(*input),
	)
	chainID := big.NewInt(int64(config.NetworkID))
	signedTx, err := types.SignTx(newTx, types.NewEIP155Signer(chainID), account.AccountKey.PrivateKey)
	s.NoError(err)
	data, err := rlp.EncodeToBytes(signedTx)
	s.NoError(err)
	return hexutil.Bytes(data)
}

func (s *TxQueueTestSuite) setupStatusBackend(account *common.SelectedExtKey, password string, passwordErr error) {
	s.nodeManagerMock.EXPECT().NodeConfig().Return(s.nodeConfig, nil)
	s.accountManagerMock.EXPECT().SelectedAccount().Return(account, nil)
	s.accountManagerMock.EXPECT().VerifyAccountPassword(s.nodeConfig.KeyStoreDir, account.Address.String(), password).Return(
		nil, passwordErr)
}

func (s *TxQueueTestSuite) TestCompleteTransaction() {
	password := TestConfig.Account1.Password
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	input := hexutil.Bytes(make([]byte, 0))
	testCases := []struct {
		name     string
		gas      *hexutil.Uint64
		gasPrice *hexutil.Big
		data     *hexutil.Bytes
		input    *hexutil.Bytes
	}{
		{
			"noGasDef",
			nil,
			nil,
			nil,
			&input,
		},
		{
			"gasDefined",
			&testGas,
			nil,
			nil,
			&input,
		},
		{
			"gasPriceDefined",
			nil,
			testGasPrice,
			nil,
			&input,
		},
		{
			"inputPassedInLegacyDataField",
			nil,
			testGasPrice,
			&input,
			nil,
		},
	}

	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			s.SetupTest()
			s.setupStatusBackend(account, password, nil)
			tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
				From:     common.FromAddress(TestConfig.Account1.Address),
				To:       common.ToAddress(TestConfig.Account2.Address),
				Gas:      testCase.gas,
				GasPrice: testCase.gasPrice,
				Data:     testCase.data,
				Input:    testCase.input,
			})
			s.setupTransactionPoolAPI(tx, testNonce, testNonce, account, nil)

			s.NoError(s.manager.QueueTransaction(tx))
			w := make(chan struct{})
			var (
				hash gethcommon.Hash
				err  error
			)
			go func() {
				hash, err = s.manager.CompleteTransaction(tx.ID, password)
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
	password := TestConfig.Account1.Password
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	s.setupStatusBackend(account, password, nil)

	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	s.setupTransactionPoolAPI(tx, testNonce, testNonce, account, nil)

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
			_, err := s.manager.CompleteTransaction(tx.ID, password)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				completedTx++
			} else if err == queue.ErrQueuedTxInProgress {
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
	s.nodeManagerMock.EXPECT().NodeConfig().Return(s.nodeConfig, nil)
	s.accountManagerMock.EXPECT().SelectedAccount().Return(&common.SelectedExtKey{
		Address: common.FromAddress(TestConfig.Account2.Address),
	}, nil)

	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	s.NoError(s.manager.QueueTransaction(tx))

	_, err := s.manager.CompleteTransaction(tx.ID, TestConfig.Account1.Password)
	s.Equal(err, queue.ErrInvalidCompleteTxSender)

	// Transaction should stay in the queue as mismatched accounts
	// is a recoverable error.
	s.True(s.manager.TransactionQueue().Has(tx.ID))
}

func (s *TxQueueTestSuite) TestInvalidPassword() {
	password := "invalid-password"
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	s.setupStatusBackend(account, password, keystore.ErrDecrypt)
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	s.NoError(s.manager.QueueTransaction(tx))
	_, err := s.manager.CompleteTransaction(tx.ID, password)
	s.Equal(err.Error(), keystore.ErrDecrypt.Error())

	// Transaction should stay in the queue as mismatched accounts
	// is a recoverable error.
	s.True(s.manager.TransactionQueue().Has(tx.ID))
}

func (s *TxQueueTestSuite) TestDiscardTransaction() {
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
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
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
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
	password := TestConfig.Account1.Password
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	// setup call expectations for 5 transactions in total
	for i := 0; i < txCount+2; i++ {
		s.setupStatusBackend(account, password, nil)
	}
	nonce := hexutil.Uint64(0)
	for i := 0; i < txCount; i++ {
		tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
			From: common.FromAddress(TestConfig.Account1.Address),
			To:   common.ToAddress(TestConfig.Account2.Address),
		})
		s.setupTransactionPoolAPI(tx, nonce, hexutil.Uint64(i), account, nil)
		s.NoError(s.manager.QueueTransaction(tx))
		hash, err := s.manager.CompleteTransaction(tx.ID, password)
		rst := s.manager.WaitForTransaction(tx)
		// simple sanity checks
		s.NoError(err)
		s.NoError(rst.Error)
		s.Equal(rst.Hash, hash)
		resultNonce, _ := s.manager.localNonce.Load(tx.Args.From)
		s.Equal(uint64(i)+1, resultNonce.(uint64))
	}
	nonce = hexutil.Uint64(5)
	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})
	s.setupTransactionPoolAPI(tx, nonce, nonce, account, nil)
	s.NoError(s.manager.QueueTransaction(tx))
	hash, err := s.manager.CompleteTransaction(tx.ID, password)
	rst := s.manager.WaitForTransaction(tx)
	s.NoError(err)
	s.NoError(rst.Error)
	s.Equal(rst.Hash, hash)
	resultNonce, _ := s.manager.localNonce.Load(tx.Args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))

	testErr := errors.New("test")
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), account.Address, gethrpc.PendingBlockNumber).Return(nil, testErr)
	tx = common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})
	s.NoError(s.manager.QueueTransaction(tx))
	_, err = s.manager.CompleteTransaction(tx.ID, password)
	rst = s.manager.WaitForTransaction(tx)
	s.EqualError(testErr, err.Error())
	s.EqualError(testErr, rst.Error.Error())
	resultNonce, _ = s.manager.localNonce.Load(tx.Args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))
}
