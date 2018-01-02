package txqueue

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/txqueue/fake"
	. "github.com/status-im/status-go/testing"
)

var errTxAssumedSent = errors.New("assume tx is done")

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
	txServiceMock          *fake.MockFakePublicTxApi
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
}

func (s *TxQueueTestSuite) TearDownTest() {
	s.nodeManagerMockCtrl.Finish()
	s.accountManagerMockCtrl.Finish()
	s.txServiceMockCtrl.Finish()
	s.server.Stop()
	s.client.Close()
}

func (s *TxQueueTestSuite) TestCompleteTransaction() {
	nodeConfig, nodeErr := params.NewNodeConfig("/tmp", params.RopstenNetworkID, true)
	password := TestConfig.Account1.Password
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	s.accountManagerMock.EXPECT().SelectedAccount().Return(account, nil)
	s.accountManagerMock.EXPECT().VerifyAccountPassword(nodeConfig.KeyStoreDir, account.Address.String(), password).Return(
		nil, nil)
	s.nodeManagerMock.EXPECT().NodeConfig().Return(nodeConfig, nodeErr)

	nonce := hexutil.Uint64(10)
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), account.Address, gethrpc.PendingBlockNumber).Return(&nonce, nil)
	s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(big.NewInt(10), nil)
	gas := hexutil.Big(*big.NewInt(defaultGas + 1))
	s.txServiceMock.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(&gas, nil)
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), gomock.Any()).Return(gethcommon.Hash{}, nil)

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)

	txQueueManager.Start()
	defer txQueueManager.Stop()

	tx := txQueueManager.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	// TransactionQueueHandler is required to enqueue a transaction.
	txQueueManager.SetTransactionQueueHandler(func(queuedTx *common.QueuedTx) {
		s.Equal(tx.ID, queuedTx.ID)
	})

	txQueueManager.SetTransactionReturnHandler(func(queuedTx *common.QueuedTx, err error) {
		s.Equal(tx.ID, queuedTx.ID)
		s.Equal(errTxAssumedSent, err)
	})

	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

	go func() {
		_, errCompleteTransaction := txQueueManager.CompleteTransaction(tx.ID, password)
		s.NoError(errCompleteTransaction)
	}()

	err = txQueueManager.WaitForTransaction(tx)
	s.NoError(err)
	// Check that error is assigned to the transaction.
	s.NoError(tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))
}

func (s *TxQueueTestSuite) TestCompleteTransactionMultipleTimes() {
	nodeConfig, nodeErr := params.NewNodeConfig("/tmp", params.RopstenNetworkID, true)
	password := TestConfig.Account1.Password
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	s.accountManagerMock.EXPECT().SelectedAccount().Return(account, nil)
	s.accountManagerMock.EXPECT().VerifyAccountPassword(nodeConfig.KeyStoreDir, account.Address.String(), password).Return(
		nil, nil)
	s.nodeManagerMock.EXPECT().NodeConfig().Return(nodeConfig, nodeErr)

	nonce := hexutil.Uint64(10)
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), account.Address, gethrpc.PendingBlockNumber).Return(&nonce, nil)
	s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(big.NewInt(10), nil)
	gas := hexutil.Big(*big.NewInt(defaultGas + 1))
	s.txServiceMock.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(&gas, nil)
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), gomock.Any()).Return(gethcommon.Hash{}, nil)

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)

	txQueueManager.Start()
	defer txQueueManager.Stop()

	tx := txQueueManager.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	// TransactionQueueHandler is required to enqueue a transaction.
	txQueueManager.SetTransactionQueueHandler(func(queuedTx *common.QueuedTx) {
		s.Equal(tx.ID, queuedTx.ID)
	})

	txQueueManager.SetTransactionReturnHandler(func(queuedTx *common.QueuedTx, err error) {
		s.Equal(tx.ID, queuedTx.ID)
		s.NoError(err)
	})

	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

	var wg sync.WaitGroup
	var mu sync.Mutex
	completeTxErrors := map[error]int{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errCompleteTransaction := txQueueManager.CompleteTransaction(tx.ID, password)
			mu.Lock()
			completeTxErrors[errCompleteTransaction]++
			mu.Unlock()
		}()
	}

	err = txQueueManager.WaitForTransaction(tx)
	s.NoError(err)
	// Check that error is assigned to the transaction.
	s.NoError(tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))

	// Wait for all CompleteTransaction calls.
	wg.Wait()
	s.Equal(completeTxErrors[nil], 1)
}

func (s *TxQueueTestSuite) TestAccountMismatch() {
	s.accountManagerMock.EXPECT().SelectedAccount().Return(&common.SelectedExtKey{
		Address: common.FromAddress(TestConfig.Account2.Address),
	}, nil)

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)

	txQueueManager.Start()
	defer txQueueManager.Stop()

	tx := txQueueManager.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	// TransactionQueueHandler is required to enqueue a transaction.
	txQueueManager.SetTransactionQueueHandler(func(queuedTx *common.QueuedTx) {
		s.Equal(tx.ID, queuedTx.ID)
	})

	// Missmatched address is a recoverable error, that's why
	// the return handler is called.
	txQueueManager.SetTransactionReturnHandler(func(queuedTx *common.QueuedTx, err error) {
		s.Equal(tx.ID, queuedTx.ID)
		s.Equal(ErrInvalidCompleteTxSender, err)
		s.Nil(tx.Err)
	})

	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

	_, err = txQueueManager.CompleteTransaction(tx.ID, TestConfig.Account1.Password)
	s.Equal(err, ErrInvalidCompleteTxSender)

	// Transaction should stay in the queue as mismatched accounts
	// is a recoverable error.
	s.True(txQueueManager.TransactionQueue().Has(tx.ID))
}

func (s *TxQueueTestSuite) TestInvalidPassword() {
	nodeConfig, nodeErr := params.NewNodeConfig("/tmp", params.RopstenNetworkID, true)
	password := "invalid-password"
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	s.accountManagerMock.EXPECT().SelectedAccount().Return(account, nil)
	s.accountManagerMock.EXPECT().VerifyAccountPassword(nodeConfig.KeyStoreDir, account.Address.String(), password).Return(
		nil, nil)
	s.nodeManagerMock.EXPECT().NodeConfig().Return(nodeConfig, nodeErr)

	nonce := hexutil.Uint64(10)
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), account.Address, gethrpc.PendingBlockNumber).Return(&nonce, nil)
	s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(big.NewInt(10), nil)
	gas := hexutil.Big(*big.NewInt(defaultGas + 1))
	s.txServiceMock.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(&gas, nil)
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), gomock.Any()).Return(gethcommon.Hash{}, keystore.ErrDecrypt)

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)

	txQueueManager.Start()
	defer txQueueManager.Stop()

	tx := txQueueManager.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	// TransactionQueueHandler is required to enqueue a transaction.
	txQueueManager.SetTransactionQueueHandler(func(queuedTx *common.QueuedTx) {
		s.Equal(tx.ID, queuedTx.ID)
	})

	// Missmatched address is a revocable error, that's why
	// the return handler is called.
	txQueueManager.SetTransactionReturnHandler(func(queuedTx *common.QueuedTx, err error) {
		s.Equal(tx.ID, queuedTx.ID)
		s.Equal(keystore.ErrDecrypt, err)
		s.Nil(tx.Err)
	})

	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

	_, err = txQueueManager.CompleteTransaction(tx.ID, password)
	s.Equal(err.Error(), keystore.ErrDecrypt.Error())

	// Transaction should stay in the queue as mismatched accounts
	// is a recoverable error.
	s.True(txQueueManager.TransactionQueue().Has(tx.ID))
}

func (s *TxQueueTestSuite) TestDiscardTransaction() {
	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)

	txQueueManager.Start()
	defer txQueueManager.Stop()

	tx := txQueueManager.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	// TransactionQueueHandler is required to enqueue a transaction.
	txQueueManager.SetTransactionQueueHandler(func(queuedTx *common.QueuedTx) {
		s.Equal(tx.ID, queuedTx.ID)
	})

	txQueueManager.SetTransactionReturnHandler(func(queuedTx *common.QueuedTx, err error) {
		s.Equal(tx.ID, queuedTx.ID)
		s.Equal(ErrQueuedTxDiscarded, err)
	})

	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

	go func() {
		discardErr := txQueueManager.DiscardTransaction(tx.ID)
		s.NoError(discardErr)
	}()

	err = txQueueManager.WaitForTransaction(tx)
	s.Equal(ErrQueuedTxDiscarded, err)
	// Check that error is assigned to the transaction.
	s.Equal(ErrQueuedTxDiscarded, tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))
}
