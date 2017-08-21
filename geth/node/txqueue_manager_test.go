package node

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/stretchr/testify/suite"

	"github.com/golang/mock/gomock"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
)

var errTxAssumedSent = errors.New("assume tx is done")

type TxQueueTestSuite struct {
	suite.Suite
	nodeManagerMockCtrl    *gomock.Controller
	nodeManagerMock        *common.MockNodeManager
	accountManagerMockCtrl *gomock.Controller
	accountManagerMock     *common.MockAccountManager
}

func (suite *TxQueueTestSuite) SetupTest() {
	suite.nodeManagerMockCtrl = gomock.NewController(suite.T())
	suite.accountManagerMockCtrl = gomock.NewController(suite.T())

	suite.nodeManagerMock = common.NewMockNodeManager(suite.nodeManagerMockCtrl)
	suite.accountManagerMock = common.NewMockAccountManager(suite.accountManagerMockCtrl)
}

func (suite *TxQueueTestSuite) TearDownTest() {
	suite.nodeManagerMockCtrl.Finish()
	suite.accountManagerMockCtrl.Finish()
}

func (suite *TxQueueTestSuite) TestCompleteTransaction() {
	suite.accountManagerMock.EXPECT().SelectedAccount().Return(&common.SelectedExtKey{
		Address: common.FromAddress(TestConfig.Account1.Address),
	}, nil)

	suite.nodeManagerMock.EXPECT().NodeConfig().Return(
		params.NewNodeConfig("/tmp", params.RopstenNetworkID, true))

	// TODO(adam): StatusBackend as an interface would allow a better solution.
	// As we want to avoid network connection, we mock LES with a known error
	// and treat as success.
	suite.nodeManagerMock.EXPECT().LightEthereumService().Return(nil, errTxAssumedSent)

	m := NewTxQueueManager(suite.nodeManagerMock, suite.accountManagerMock)

	m.Start()
	defer m.Stop()

	toAddr := common.FromAddress(TestConfig.Account2.Address)
	tx := m.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   &toAddr,
	})

	// TransactionQueueHandler is required to enqueue a transaction.
	m.SetTransactionQueueHandler(func(queuedTx common.QueuedTx) {
		suite.Equal(tx.ID, queuedTx.ID)
	})

	m.SetTransactionReturnHandler(func(queuedTx *common.QueuedTx, err error) {
		suite.Equal(tx.ID, queuedTx.ID)
		suite.Equal(errTxAssumedSent, err)
	})

	err := m.QueueTransaction(tx)
	suite.NoError(err)

	go func() {
		_, err := m.CompleteTransaction(tx.ID, TestConfig.Account1.Password)
		if err != errTxAssumedSent {
			suite.Fail("failed to complete transaction: %s", err)
		}
	}()

	err = m.WaitForTransaction(tx)
	if err != errTxAssumedSent {
		suite.Fail("unexpected transaction error: %s", err)
	}

	// Check that error is assigned to the transaction.
	suite.Equal(errTxAssumedSent, tx.Err)
	// Transaction should be already removed from the queue.
	suite.False(m.TransactionQueue().Has(tx.ID))
}

func (suite *TxQueueTestSuite) TestAccountMissmatch() {
	suite.accountManagerMock.EXPECT().SelectedAccount().Return(&common.SelectedExtKey{
		Address: common.FromAddress(TestConfig.Account2.Address),
	}, nil)

	m := NewTxQueueManager(suite.nodeManagerMock, suite.accountManagerMock)

	m.Start()
	defer m.Stop()

	toAddr := common.FromAddress(TestConfig.Account2.Address)
	tx := m.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   &toAddr,
	})

	// TransactionQueueHandler is required to enqueue a transaction.
	m.SetTransactionQueueHandler(func(queuedTx common.QueuedTx) {
		suite.Equal(tx.ID, queuedTx.ID)
	})

	// Missmatched address is a recoverable error, that's why
	// the return handler is called.
	m.SetTransactionReturnHandler(func(queuedTx *common.QueuedTx, err error) {
		suite.Equal(tx.ID, queuedTx.ID)
		suite.Equal(ErrInvalidCompleteTxSender, err)
		suite.Nil(tx.Err)
	})

	err := m.QueueTransaction(tx)
	suite.NoError(err)

	_, err = m.CompleteTransaction(tx.ID, TestConfig.Account1.Password)
	suite.Equal(err, ErrInvalidCompleteTxSender)

	// Transaction should stay in the queue as mismatched accounts
	// is a recoverable error.
	suite.True(m.TransactionQueue().Has(tx.ID))
}

func (suite *TxQueueTestSuite) TestInvalidPassword() {
	suite.accountManagerMock.EXPECT().SelectedAccount().Return(&common.SelectedExtKey{
		Address: common.FromAddress(TestConfig.Account1.Address),
	}, nil)

	suite.nodeManagerMock.EXPECT().NodeConfig().Return(
		params.NewNodeConfig("/tmp", params.RopstenNetworkID, true))

	// Set ErrDecrypt error response as expected with a wrong password.
	suite.nodeManagerMock.EXPECT().LightEthereumService().Return(nil, keystore.ErrDecrypt)

	m := NewTxQueueManager(suite.nodeManagerMock, suite.accountManagerMock)

	m.Start()
	defer m.Stop()

	toAddr := common.FromAddress(TestConfig.Account2.Address)
	tx := m.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   &toAddr,
	})

	// TransactionQueueHandler is required to enqueue a transaction.
	m.SetTransactionQueueHandler(func(queuedTx common.QueuedTx) {
		suite.Equal(tx.ID, queuedTx.ID)
	})

	// Missmatched address is a revocable error, that's why
	// the return handler is called.
	m.SetTransactionReturnHandler(func(queuedTx *common.QueuedTx, err error) {
		suite.Equal(tx.ID, queuedTx.ID)
		suite.Equal(keystore.ErrDecrypt, err)
		suite.Nil(tx.Err)
	})

	err := m.QueueTransaction(tx)
	suite.NoError(err)

	_, err = m.CompleteTransaction(tx.ID, "invalid-password")
	suite.Equal(err, keystore.ErrDecrypt)

	// Transaction should stay in the queue as mismatched accounts
	// is a recoverable error.
	suite.True(m.TransactionQueue().Has(tx.ID))
}

func (suite *TxQueueTestSuite) TestDiscardTransaction() {
	m := NewTxQueueManager(suite.nodeManagerMock, suite.accountManagerMock)

	m.Start()
	defer m.Stop()

	toAddr := common.FromAddress(TestConfig.Account2.Address)
	tx := m.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   &toAddr,
	})

	// TransactionQueueHandler is required to enqueue a transaction.
	m.SetTransactionQueueHandler(func(queuedTx common.QueuedTx) {
		suite.Equal(tx.ID, queuedTx.ID)
	})

	m.SetTransactionReturnHandler(func(queuedTx *common.QueuedTx, err error) {
		suite.Equal(tx.ID, queuedTx.ID)
		suite.Equal(ErrQueuedTxDiscarded, err)
	})

	err := m.QueueTransaction(tx)
	suite.NoError(err)

	go func() {
		err := m.DiscardTransaction(tx.ID)
		suite.NoError(err)
	}()

	err = m.WaitForTransaction(tx)
	if err != ErrQueuedTxDiscarded {
		suite.Fail("unexpected transaction error: %s", err)
	}

	// Check that error is assigned to the transaction.
	suite.Equal(ErrQueuedTxDiscarded, tx.Err)
	// Transaction should be already removed from the queue.
	suite.False(m.TransactionQueue().Has(tx.ID))
}

func TestTxQueueTestSuite(t *testing.T) {
	suite.Run(t, new(TxQueueTestSuite))
}
