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
	"github.com/ethereum/go-ethereum/rlp"
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
	txServiceMock          *fake.MockFakePublicTransactionPoolAPI
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

func (s *TxQueueTestSuite) setupTransactionPoolAPI(tx *common.QueuedTx, config *params.NodeConfig, account *common.SelectedExtKey, nonce uint64, gas hexutil.Big, txErr error) {
	non := hexutil.Uint64(nonce)
	// Expect calls to gas functions only if there are no user defined values.
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), account.Address, gethrpc.PendingBlockNumber).Return(&non, nil)
	if tx.Args.GasPrice == nil {
		s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(big.NewInt(10), nil)
	}
	if tx.Args.Gas == nil {
		s.txServiceMock.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(&gas, nil)
	}
	// Prepare the transaction an RLP encode it.
	data := s.rlpEncodeTx(tx, config, account, nonce, gas)
	// Expect the RLP encoded transaction.
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), data).Return(gethcommon.Hash{}, txErr)
}

func (s *TxQueueTestSuite) rlpEncodeTx(tx *common.QueuedTx, config *params.NodeConfig, account *common.SelectedExtKey, nonce uint64, gas hexutil.Big) []byte {
	newTx := types.NewTransaction(
		nonce,
		gethcommon.Address(*tx.Args.To),
		tx.Args.Value.ToInt(),
		gas.ToInt(),
		tx.Args.GasPrice.ToInt(),
		tx.Args.Data,
	)
	chainID := big.NewInt(int64(config.NetworkID))
	signedTx, err := types.SignTx(newTx, types.NewEIP155Signer(chainID), account.AccountKey.PrivateKey)
	s.NoError(err)
	data, err := rlp.EncodeToBytes(signedTx)
	s.NoError(err)
	return data
}

func (s *TxQueueTestSuite) setupStatusBackend(account *common.SelectedExtKey, password string, passwordErr error) *params.NodeConfig {
	nodeConfig, nodeErr := params.NewNodeConfig("/tmp", params.RopstenNetworkID, true)
	s.nodeManagerMock.EXPECT().NodeConfig().Return(nodeConfig, nodeErr)
	s.accountManagerMock.EXPECT().SelectedAccount().Return(account, nil)
	s.accountManagerMock.EXPECT().VerifyAccountPassword(nodeConfig.KeyStoreDir, account.Address.String(), password).Return(
		nil, passwordErr)
	return nodeConfig
}

func (s *TxQueueTestSuite) TestCompleteTransaction() {
	password := TestConfig.Account1.Password
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	config := s.setupStatusBackend(account, password, nil)

	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	nonce := uint64(10)
	gas := hexutil.Big(*big.NewInt(defaultGas + 1))
	s.setupTransactionPoolAPI(tx, config, account, nonce, gas, nil)

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)

	txQueueManager.Start()
	defer txQueueManager.Stop()

	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

	w := make(chan struct{})
	go func() {
		hash, err := txQueueManager.CompleteTransaction(tx.ID, password)
		s.NoError(err)
		s.Equal(tx.Hash, hash)
		close(w)
	}()

	err = txQueueManager.WaitForTransaction(tx)
	s.NoError(err)
	// Check that error is assigned to the transaction.
	s.NoError(tx.Err)
	// Transaction should be already removed from the queue.
	s.False(txQueueManager.TransactionQueue().Has(tx.ID))
	s.NoError(WaitClosed(w, time.Second))
}

func (s *TxQueueTestSuite) TestCompleteTransactionMultipleTimes() {
	password := TestConfig.Account1.Password
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	config := s.setupStatusBackend(account, password, nil)

	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})

	nonce := uint64(10)
	gas := hexutil.Big(*big.NewInt(defaultGas + 1))
	s.setupTransactionPoolAPI(tx, config, account, nonce, gas, nil)

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)
	txQueueManager.DisableNotificactions()
	txQueueManager.Start()
	defer txQueueManager.Stop()

	err := txQueueManager.QueueTransaction(tx)
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

	err = txQueueManager.WaitForTransaction(tx)
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

/*
func (s *TxQueueTestSuite) TestCompleteTransactionSignature() {
	password := TestConfig.Account1.Password
	key, _ := crypto.GenerateKey()
	account := &common.SelectedExtKey{
		Address:    common.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	s.setupStatusBackend(account, password, nil)

	nonce := hexutil.Uint64(10)
	gas := hexutil.Big(*big.NewInt(defaultGas + 1))

	txQueueManager := NewManager(s.nodeManagerMock, s.accountManagerMock)

	txQueueManager.Start()
	defer txQueueManager.Stop()

	tx := common.CreateTransaction(context.Background(), common.SendTxArgs{
		From: common.FromAddress(TestConfig.Account1.Address),
		To:   common.ToAddress(TestConfig.Account2.Address),
	})
	err := txQueueManager.QueueTransaction(tx)
	s.NoError(err)

}
*/

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
