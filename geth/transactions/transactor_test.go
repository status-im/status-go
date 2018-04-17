package transactions

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/contracts/ens/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/transactions/fake"
	"github.com/status-im/status-go/sign"

	. "github.com/status-im/status-go/t/utils"
)

func simpleVerifyFunc(acc *account.SelectedExtKey) func(string) (*account.SelectedExtKey, error) {
	return func(string) (*account.SelectedExtKey, error) {
		return acc, nil
	}
}

func TestTxQueueTestSuite(t *testing.T) {
	suite.Run(t, new(TxQueueTestSuite))
}

type TxQueueTestSuite struct {
	suite.Suite
	server            *gethrpc.Server
	client            *gethrpc.Client
	txServiceMockCtrl *gomock.Controller
	txServiceMock     *fake.MockPublicTransactionPoolAPI
	nodeConfig        *params.NodeConfig

	manager *Transactor
}

func (s *TxQueueTestSuite) SetupTest() {
	s.txServiceMockCtrl = gomock.NewController(s.T())

	s.server, s.txServiceMock = fake.NewTestServer(s.txServiceMockCtrl)
	s.client = gethrpc.DialInProc(s.server)
	rpcClient, _ := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	// expected by simulated backend
	chainID := gethparams.AllEthashProtocolChanges.ChainId.Uint64()
	nodeConfig, err := params.NewNodeConfig("/tmp", "", chainID, true)
	s.Require().NoError(err)
	s.nodeConfig = nodeConfig

	s.manager = NewTransactor(sign.NewPendingRequests())
	s.manager.sendTxTimeout = time.Second
	s.manager.SetNetworkID(chainID)
	s.manager.SetRPC(rpcClient, time.Second)
}

func (s *TxQueueTestSuite) TearDownTest() {
	s.txServiceMockCtrl.Finish()
	s.server.Stop()
	s.client.Close()
}

var (
	testGas      = hexutil.Uint64(defaultGas + 1)
	testGasPrice = (*hexutil.Big)(big.NewInt(10))
	testNonce    = hexutil.Uint64(10)
)

func (s *TxQueueTestSuite) setupTransactionPoolAPI(args SendTxArgs, returnNonce, resultNonce hexutil.Uint64, account *account.SelectedExtKey, txErr error) {
	// Expect calls to gas functions only if there are no user defined values.
	// And also set the expected gas and gas price for RLP encoding the expected tx.
	var usedGas hexutil.Uint64
	var usedGasPrice *big.Int
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), account.Address, gethrpc.PendingBlockNumber).Return(&returnNonce, nil)
	if args.GasPrice == nil {
		usedGasPrice = (*big.Int)(testGasPrice)
		s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(usedGasPrice, nil)
	} else {
		usedGasPrice = (*big.Int)(args.GasPrice)
	}
	if args.Gas == nil {
		s.txServiceMock.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(testGas, nil)
		usedGas = testGas
	} else {
		usedGas = *args.Gas
	}
	// Prepare the transaction anD RLP encode it.
	data := s.rlpEncodeTx(args, s.nodeConfig, account, &resultNonce, usedGas, usedGasPrice)
	// Expect the RLP encoded transaction.
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), data).Return(gethcommon.Hash{}, txErr)
}

func (s *TxQueueTestSuite) rlpEncodeTx(args SendTxArgs, config *params.NodeConfig, account *account.SelectedExtKey, nonce *hexutil.Uint64, gas hexutil.Uint64, gasPrice *big.Int) hexutil.Bytes {
	newTx := types.NewTransaction(
		uint64(*nonce),
		gethcommon.Address(*args.To),
		args.Value.ToInt(),
		uint64(gas),
		gasPrice,
		[]byte(args.Input),
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
			args := SendTxArgs{
				From:     account.FromAddress(TestConfig.Account1.Address),
				To:       account.ToAddress(TestConfig.Account2.Address),
				Gas:      testCase.gas,
				GasPrice: testCase.gasPrice,
			}
			s.setupTransactionPoolAPI(args, testNonce, testNonce, selectedAccount, nil)

			w := make(chan struct{})
			var sendHash gethcommon.Hash
			go func() {
				var sendErr error
				sendHash, sendErr = s.manager.SendTransaction(context.Background(), args)
				s.NoError(sendErr)
				close(w)
			}()

			for i := 10; i > 0; i-- {
				if s.manager.pendingSignRequests.Count() > 0 {
					break
				}
				time.Sleep(time.Millisecond)
			}

			req := s.manager.pendingSignRequests.First()
			s.NotNil(req)
			approveResult := s.manager.pendingSignRequests.Approve(req.ID, "", simpleVerifyFunc(selectedAccount))
			s.NoError(approveResult.Error)
			s.NoError(WaitClosed(w, time.Second))

			// Transaction should be already removed from the queue.
			s.False(s.manager.pendingSignRequests.Has(req.ID))
			s.Equal(sendHash.Bytes(), approveResult.Response.Bytes())
		})
	}
}

func (s *TxQueueTestSuite) TestAccountMismatch() {
	selectedAccount := &account.SelectedExtKey{
		Address: account.FromAddress(TestConfig.Account2.Address),
	}

	args := SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	}

	go func() {
		s.manager.SendTransaction(context.Background(), args) // nolint: errcheck
	}()

	for i := 10; i > 0; i-- {
		if s.manager.pendingSignRequests.Count() > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	req := s.manager.pendingSignRequests.First()
	s.NotNil(req)
	result := s.manager.pendingSignRequests.Approve(req.ID, "", simpleVerifyFunc(selectedAccount))
	s.EqualError(result.Error, ErrInvalidCompleteTxSender.Error())

	// Transaction should stay in the queue as mismatched accounts
	// is a recoverable error.
	s.True(s.manager.pendingSignRequests.Has(req.ID))
}

func (s *TxQueueTestSuite) TestDiscardTransaction() {
	args := SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	}
	w := make(chan struct{})
	go func() {
		_, err := s.manager.SendTransaction(context.Background(), args)
		s.Equal(sign.ErrSignReqDiscarded, err)
		close(w)
	}()

	for i := 10; i > 0; i-- {
		if s.manager.pendingSignRequests.Count() > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	req := s.manager.pendingSignRequests.First()
	s.NotNil(req)
	err := s.manager.pendingSignRequests.Discard(req.ID)
	s.NoError(err)
	s.NoError(WaitClosed(w, time.Second))
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

	go func() {
		approved := 0
		for {
			// 3 in a cycle, then 2
			if approved >= txCount+2 {
				return
			}
			req := s.manager.pendingSignRequests.First()
			if req == nil {
				time.Sleep(time.Millisecond)
			} else {
				s.manager.pendingSignRequests.Approve(req.ID, "", simpleVerifyFunc(selectedAccount)) // nolint: errcheck
			}
		}
	}()

	for i := 0; i < txCount; i++ {
		args := SendTxArgs{
			From: account.FromAddress(TestConfig.Account1.Address),
			To:   account.ToAddress(TestConfig.Account2.Address),
		}
		s.setupTransactionPoolAPI(args, nonce, hexutil.Uint64(i), selectedAccount, nil)

		_, err := s.manager.SendTransaction(context.Background(), args)
		s.NoError(err)
		resultNonce, _ := s.manager.localNonce.Load(args.From)
		s.Equal(uint64(i)+1, resultNonce.(uint64))
	}

	nonce = hexutil.Uint64(5)
	args := SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	}

	s.setupTransactionPoolAPI(args, nonce, nonce, selectedAccount, nil)

	_, err := s.manager.SendTransaction(context.Background(), args)
	s.NoError(err)

	resultNonce, _ := s.manager.localNonce.Load(args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))

	testErr := errors.New("test")
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), selectedAccount.Address, gethrpc.PendingBlockNumber).Return(nil, testErr)
	args = SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	}

	_, err = s.manager.SendTransaction(context.Background(), args)
	s.EqualError(testErr, err.Error())
	resultNonce, _ = s.manager.localNonce.Load(args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))
}

func (s *TxQueueTestSuite) TestContractCreation() {
	key, _ := crypto.GenerateKey()
	testaddr := crypto.PubkeyToAddress(key.PublicKey)
	genesis := core.GenesisAlloc{
		testaddr: {Balance: big.NewInt(100000000000)},
	}
	backend := backends.NewSimulatedBackend(genesis)
	selectedAccount := &account.SelectedExtKey{
		Address:    testaddr,
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	s.manager.sender = backend
	s.manager.gasCalculator = backend
	s.manager.pendingNonceProvider = backend
	tx := SendTxArgs{
		From:  testaddr,
		Input: hexutil.Bytes(gethcommon.FromHex(contract.ENSBin)),
	}

	go func() {
		for i := 1000; i > 0; i-- {
			req := s.manager.pendingSignRequests.First()
			if req == nil {
				time.Sleep(time.Millisecond)
			} else {
				s.manager.pendingSignRequests.Approve(req.ID, "", simpleVerifyFunc(selectedAccount)) // nolint: errcheck
				break
			}
		}
	}()

	hash, err := s.manager.SendTransaction(context.Background(), tx)
	s.NoError(err)
	backend.Commit()
	receipt, err := backend.TransactionReceipt(context.TODO(), hash)
	s.NoError(err)
	s.Equal(crypto.CreateAddress(testaddr, 0), receipt.ContractAddress)
}
