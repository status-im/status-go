package transactions

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/transactions/fake"
	"github.com/status-im/status-go/geth/transactions/queue"
	. "github.com/status-im/status-go/testing"
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

func (s *TxQueueTestSuite) setupTransactionPoolAPI(account *common.SelectedExtKey, nonce hexutil.Uint64, gas hexutil.Big, txErr error) {
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), account.Address, gethrpc.PendingBlockNumber).Return(&nonce, nil).AnyTimes()
	s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(big.NewInt(10), nil).AnyTimes()
	s.txServiceMock.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(&gas, nil).AnyTimes()
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), gomock.Any()).Return(gethcommon.Hash{}, txErr).AnyTimes()
}

func (s *TxQueueTestSuite) setupStatusBackend(account *common.SelectedExtKey, password string, passwordErr error) {
	nodeConfig, nodeErr := params.NewNodeConfig("/tmp", params.RopstenNetworkID, true)
	s.nodeManagerMock.EXPECT().NodeConfig().Return(nodeConfig, nodeErr).AnyTimes()
	s.accountManagerMock.EXPECT().SelectedAccount().Return(account, nil).AnyTimes()
	s.accountManagerMock.EXPECT().VerifyAccountPassword(nodeConfig.KeyStoreDir, account.Address.String(), password).Return(
		nil, passwordErr).AnyTimes()
}

func (s *TxQueueTestSuite) setupCompleteTransaction(disableNotifications bool, args common.SendTxArgs) (*Manager, *common.QueuedTx, *common.SelectedExtKey, string) {
	password := TestConfig.Account1.Password
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	s.setupStatusBackend(account, password, nil)

	nonce := hexutil.Uint64(10)
	gas := hexutil.Big(*big.NewInt(defaultGas + 1))
	s.setupTransactionPoolAPI(account, nonce, gas, nil)

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)
	if disableNotifications {
		txQueueManager.DisableNotificactions()
	}
	txQueueManager.Start()

	tx := common.CreateTransaction(context.Background(), args)

	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

	return txQueueManager, tx, account, password
}

func (s *TxQueueTestSuite) TestCompleteTransaction() {
	txQueueManager, tx, _, password := s.setupCompleteTransaction(false, common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})
	defer txQueueManager.Stop()

	w := make(chan struct{})
	go func() {
		hash, err := txQueueManager.CompleteTransaction(tx.ID, password)
		s.NoError(err)
		s.Equal(tx.Hash, hash)
		close(w)
	}()

	err := txQueueManager.WaitForTransaction(tx)
	s.NoError(err)
	// Check that error is assigned to the transaction.
	s.NoError(tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))
	s.NoError(WaitClosed(w, time.Second))
}

func (s *TxQueueTestSuite) TestCompleteTransactionMultipleTimes() {
	txQueueManager, tx, _, password := s.setupCompleteTransaction(true, common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})
	defer txQueueManager.Stop()

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
			_, err := txQueueManager.CompleteTransaction(tx.ID, password)
			mu.Lock()
			if err == nil {
				completedTx++
			} else if err == queue.ErrQueuedTxInProgress {
				inprogressTx++
			} else {
				s.Fail("tx failed with unexpected error: ", err.Error())
			}
			mu.Unlock()
		}()
	}

	err := txQueueManager.WaitForTransaction(tx)
	s.NoError(err)
	// Check that error is assigned to the transaction.
	s.NoError(tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))

	// Wait for all CompleteTransaction calls.
	wg.Wait()
	s.Equal(1, completedTx, "only 1 tx expected to be completed")
	s.Equal(txCount-1, inprogressTx, "txs expected to be reported as inprogress")
}

// testTxSender implements ethereum.TransactionSender
// for testing purposes.
type testTxSender struct {
	tx  *common.QueuedTx
	s   *TxQueueTestSuite
	acc *common.SelectedExtKey
}

// SendTransaction is redefined here to inject into
// CompleteTransaction execution via Manager. Checks
// transaction values.
func (sender *testTxSender) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	s := sender.s
	s.Equal(sender.tx.Args.Gas.ToInt(), tx.Gas())
	s.Equal(sender.tx.Args.GasPrice.ToInt(), tx.GasPrice())
	// Sign the transaction externally.
	config, err := sender.s.nodeManagerMock.NodeConfig()
	s.NoError(err)
	chainID := big.NewInt(int64(config.NetworkID))
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), sender.acc.AccountKey.PrivateKey)
	s.NoError(err)
	// Verify signature.
	v1, r1, s1 := tx.RawSignatureValues()
	v2, r2, s2 := signedTx.RawSignatureValues()
	s.Equal(v1, v2)
	s.Equal(r1, r2)
	s.Equal(s1, s2)
	return nil
}

// TestCompleteTransactionValidity tests if CompleteTransaction
// uses gas and gas price correctly and sends the expected transaction.
func (s *TxQueueTestSuite) TestCompleteTransactionValidity() {
	gas := hexutil.Big(*big.NewInt(10))
	txQueueManager, tx, acc, password := s.setupCompleteTransaction(true, common.SendTxArgs{
		From:     common.FromAddress(TestConfig.Account1.Address),
		To:       common.ToAddress(TestConfig.Account2.Address),
		Gas:      &gas,
		GasPrice: &gas,
	})
	defer txQueueManager.Stop()

	txQueueManager.txSender = &testTxSender{
		tx:  tx,
		s:   s,
		acc: acc,
	}

	hash, err := txQueueManager.CompleteTransaction(tx.ID, password)
	s.NoError(err)
	s.Equal(tx.Hash, hash)
}

func (s *TxQueueTestSuite) TestAccountMismatch() {
	s.accountManagerMock.EXPECT().SelectedAccount().Return(&common.SelectedExtKey{
		Address: common.FromAddress(TestConfig.Account2.Address),
	}, nil)

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)
	txQueueManager.DisableNotificactions()

	txQueueManager.Start()
	defer txQueueManager.Stop()

	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

	_, err = txQueueManager.CompleteTransaction(tx.ID, TestConfig.Account1.Password)
	s.Equal(err, queue.ErrInvalidCompleteTxSender)

	// Transaction should stay in the queue as mismatched accounts
	// is a recoverable error.
	s.True(txQueueManager.TransactionQueue().Has(tx.ID))
}

func (s *TxQueueTestSuite) TestInvalidPassword() {
	password := "invalid-password"
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	s.setupStatusBackend(account, password, keystore.ErrDecrypt)

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)
	txQueueManager.DisableNotificactions()
	txQueueManager.Start()
	defer txQueueManager.Stop()

	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
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
	txQueueManager.DisableNotificactions()

	txQueueManager.Start()
	defer txQueueManager.Stop()

	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

	w := make(chan struct{})
	go func() {
		s.NoError(txQueueManager.DiscardTransaction(tx.ID))
		close(w)
	}()

	err = txQueueManager.WaitForTransaction(tx)
	s.Equal(queue.ErrQueuedTxDiscarded, err)
	// Check that error is assigned to the transaction.
	s.Equal(queue.ErrQueuedTxDiscarded, tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))
	s.NoError(WaitClosed(w, time.Second))
}
