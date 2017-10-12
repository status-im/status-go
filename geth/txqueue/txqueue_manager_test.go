package txqueue

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/stretchr/testify/suite"

	"github.com/golang/mock/gomock"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
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
}

func (s *TxQueueTestSuite) SetupTest() {
	s.nodeManagerMockCtrl = gomock.NewController(s.T())
	s.accountManagerMockCtrl = gomock.NewController(s.T())

	s.nodeManagerMock = common.NewMockNodeManager(s.nodeManagerMockCtrl)
	s.accountManagerMock = common.NewMockAccountManager(s.accountManagerMockCtrl)
}

func (s *TxQueueTestSuite) TearDownTest() {
	s.nodeManagerMockCtrl.Finish()
	s.accountManagerMockCtrl.Finish()
}

func (s *TxQueueTestSuite) TestCompleteTransaction() {
	s.accountManagerMock.EXPECT().SelectedAccount().Return(&common.SelectedExtKey{
		Address: common.FromAddress(TestConfig.Account1.Address),
	}, nil)

	s.nodeManagerMock.EXPECT().NodeConfig().Return(
		params.NewNodeConfig("/tmp", params.RopstenNetworkID, true),
	)

	// TODO(adam): StatusBackend as an interface would allow a better solution.
	// As we want to avoid network connection, we mock LES with a known error
	// and treat as success.
	s.nodeManagerMock.EXPECT().LightEthereumService().Return(nil, errTxAssumedSent)

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
		_, err := txQueueManager.CompleteTransaction(tx.ID, TestConfig.Account1.Password)
		s.Equal(errTxAssumedSent, err)
	}()

	err = txQueueManager.WaitForTransaction(tx)
	s.Equal(errTxAssumedSent, err)
	// Check that error is assigned to the transaction.
	s.Equal(errTxAssumedSent, tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))
}

func (s *TxQueueTestSuite) TestCompleteTransactionMultipleTimes() {
	s.accountManagerMock.EXPECT().SelectedAccount().Return(&common.SelectedExtKey{
		Address: common.FromAddress(TestConfig.Account1.Address),
	}, nil)

	s.nodeManagerMock.EXPECT().NodeConfig().Return(
		params.NewNodeConfig("/tmp", params.RopstenNetworkID, true),
	)

	// TODO(adam): StatusBackend as an interface would allow a better solution.
	// As we want to avoid network connection, we mock LES with a known error
	// and treat as success.
	s.nodeManagerMock.EXPECT().LightEthereumService().Return(nil, errTxAssumedSent)

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

	var wg sync.WaitGroup
	var mu sync.Mutex
	completeTxErrors := make(map[error]int)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := txQueueManager.CompleteTransaction(tx.ID, TestConfig.Account1.Password)
			mu.Lock()
			completeTxErrors[err]++
			mu.Unlock()
		}()
	}

	err = txQueueManager.WaitForTransaction(tx)
	s.Equal(errTxAssumedSent, err)
	// Check that error is assigned to the transaction.
	s.Equal(errTxAssumedSent, tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))

	// Wait for all CompleteTransaction calls.
	wg.Wait()
	s.Equal(completeTxErrors[errTxAssumedSent], 1)
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
	s.accountManagerMock.EXPECT().SelectedAccount().Return(&common.SelectedExtKey{
		Address: common.FromAddress(TestConfig.Account1.Address),
	}, nil)

	s.nodeManagerMock.EXPECT().NodeConfig().Return(
		params.NewNodeConfig("/tmp", params.RopstenNetworkID, true),
	)

	// Set ErrDecrypt error response as expected with a wrong password.
	s.nodeManagerMock.EXPECT().LightEthereumService().Return(nil, keystore.ErrDecrypt)

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

	_, err = txQueueManager.CompleteTransaction(tx.ID, "invalid-password")
	s.Equal(err, keystore.ErrDecrypt)

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
		err := txQueueManager.DiscardTransaction(tx.ID)
		s.NoError(err)
	}()

	err = txQueueManager.WaitForTransaction(tx)
	s.Equal(ErrQueuedTxDiscarded, err)
	// Check that error is assigned to the transaction.
	s.Equal(ErrQueuedTxDiscarded, tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))
}
