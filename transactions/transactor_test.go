package transactions

import (
	"context"
	"errors"
	"math"
	"math/big"
	"reflect"
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

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/transactions/fake"

	. "github.com/status-im/status-go/t/utils"
)

func TestTransactorSuite(t *testing.T) {
	suite.Run(t, new(TransactorSuite))
}

type TransactorSuite struct {
	suite.Suite
	server            *gethrpc.Server
	client            *gethrpc.Client
	txServiceMockCtrl *gomock.Controller
	txServiceMock     *fake.MockPublicTransactionPoolAPI
	nodeConfig        *params.NodeConfig

	manager *Transactor
}

func (s *TransactorSuite) SetupTest() {
	s.txServiceMockCtrl = gomock.NewController(s.T())

	s.server, s.txServiceMock = fake.NewTestServer(s.txServiceMockCtrl)
	s.client = gethrpc.DialInProc(s.server)
	rpcClient, _ := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	// expected by simulated backend
	chainID := gethparams.AllEthashProtocolChanges.ChainID.Uint64()
	nodeConfig, err := MakeTestNodeConfigWithDataDir("", "/tmp", params.FleetBeta, chainID)
	s.Require().NoError(err)
	s.nodeConfig = nodeConfig

	s.manager = NewTransactor()
	s.manager.sendTxTimeout = time.Second
	s.manager.SetNetworkID(chainID)
	s.manager.SetRPC(rpcClient, time.Second)
}

func (s *TransactorSuite) TearDownTest() {
	s.txServiceMockCtrl.Finish()
	s.server.Stop()
	s.client.Close()
}

var (
	testGas      = hexutil.Uint64(defaultGas + 1)
	testGasPrice = (*hexutil.Big)(big.NewInt(10))
	testNonce    = hexutil.Uint64(10)
)

func (s *TransactorSuite) setupTransactionPoolAPI(args SendTxArgs, returnNonce, resultNonce hexutil.Uint64, account *account.SelectedExtKey, txErr error) {
	// Expect calls to gas functions only if there are no user defined values.
	// And also set the expected gas and gas price for RLP encoding the expected tx.
	var usedGas hexutil.Uint64
	var usedGasPrice *big.Int
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), account.Address, gethrpc.PendingBlockNumber).Return(&returnNonce, nil)
	if args.GasPrice == nil {
		usedGasPrice = (*big.Int)(testGasPrice)
		s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(testGasPrice, nil)
	} else {
		usedGasPrice = (*big.Int)(args.GasPrice)
	}
	if args.Gas == nil {
		s.txServiceMock.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(testGas, nil)
		usedGas = testGas
	} else {
		usedGas = *args.Gas
	}
	// Prepare the transaction and RLP encode it.
	data := s.rlpEncodeTx(args, s.nodeConfig, account, &resultNonce, usedGas, usedGasPrice)
	// Expect the RLP encoded transaction.
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), data).Return(gethcommon.Hash{}, txErr)
}

func (s *TransactorSuite) rlpEncodeTx(args SendTxArgs, config *params.NodeConfig, account *account.SelectedExtKey, nonce *hexutil.Uint64, gas hexutil.Uint64, gasPrice *big.Int) hexutil.Bytes {
	newTx := types.NewTransaction(
		uint64(*nonce),
		*args.To,
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

func (s *TransactorSuite) TestGasValues() {
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
			"nilSignTransactionSpecificArgs",
			nil,
			nil,
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

			hash, err := s.manager.SendTransaction(args, selectedAccount)
			s.NoError(err)
			s.False(reflect.DeepEqual(hash, gethcommon.Hash{}))
		})
	}
}

func (s *TransactorSuite) TestArgsValidation() {
	args := SendTxArgs{
		From:  account.FromAddress(TestConfig.Account1.Address),
		To:    account.ToAddress(TestConfig.Account2.Address),
		Data:  hexutil.Bytes([]byte{0x01, 0x02}),
		Input: hexutil.Bytes([]byte{0x02, 0x01}),
	}
	s.False(args.Valid())
	selectedAccount := &account.SelectedExtKey{
		Address: account.FromAddress(TestConfig.Account1.Address),
	}
	_, err := s.manager.SendTransaction(args, selectedAccount)
	s.EqualError(err, ErrInvalidSendTxArgs.Error())
}

func (s *TransactorSuite) TestAccountMismatch() {
	args := SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	}

	var err error

	// missing account
	_, err = s.manager.SendTransaction(args, nil)
	s.EqualError(err, account.ErrNoAccountSelected.Error())

	// mismatched accounts
	selectedAccount := &account.SelectedExtKey{
		Address: account.FromAddress(TestConfig.Account2.Address),
	}
	_, err = s.manager.SendTransaction(args, selectedAccount)
	s.EqualError(err, ErrInvalidTxSender.Error())
}

// TestLocalNonce verifies that local nonce will be used unless
// upstream nonce is updated and higher than a local
// in test we will run 3 transaction with nonce zero returned by upstream
// node, after each call local nonce will be incremented
// then, we return higher nonce, as if another node was used to send 2 transactions
// upstream nonce will be equal to 5, we update our local counter to 5+1
// as the last step, we verify that if tx failed nonce is not updated
func (s *TransactorSuite) TestLocalNonce() {
	txCount := 3
	key, _ := crypto.GenerateKey()
	selectedAccount := &account.SelectedExtKey{
		Address:    account.FromAddress(TestConfig.Account1.Address),
		AccountKey: &keystore.Key{PrivateKey: key},
	}
	nonce := hexutil.Uint64(0)

	for i := 0; i < txCount; i++ {
		args := SendTxArgs{
			From: account.FromAddress(TestConfig.Account1.Address),
			To:   account.ToAddress(TestConfig.Account2.Address),
		}
		s.setupTransactionPoolAPI(args, nonce, hexutil.Uint64(i), selectedAccount, nil)

		_, err := s.manager.SendTransaction(args, selectedAccount)
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

	_, err := s.manager.SendTransaction(args, selectedAccount)
	s.NoError(err)

	resultNonce, _ := s.manager.localNonce.Load(args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))

	testErr := errors.New("test")
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), selectedAccount.Address, gethrpc.PendingBlockNumber).Return(nil, testErr)
	args = SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   account.ToAddress(TestConfig.Account2.Address),
	}

	_, err = s.manager.SendTransaction(args, selectedAccount)
	s.EqualError(err, testErr.Error())
	resultNonce, _ = s.manager.localNonce.Load(args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))
}

func (s *TransactorSuite) TestContractCreation() {
	key, _ := crypto.GenerateKey()
	testaddr := crypto.PubkeyToAddress(key.PublicKey)
	genesis := core.GenesisAlloc{
		testaddr: {Balance: big.NewInt(100000000000)},
	}
	backend := backends.NewSimulatedBackend(genesis, math.MaxInt64)
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

	hash, err := s.manager.SendTransaction(tx, selectedAccount)
	s.NoError(err)
	backend.Commit()
	receipt, err := backend.TransactionReceipt(context.TODO(), hash)
	s.NoError(err)
	s.Equal(crypto.CreateAddress(testaddr, 0), receipt.ContractAddress)
}
