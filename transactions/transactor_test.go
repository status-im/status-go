package transactions

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions/fake"
)

func TestTransactorSuite(t *testing.T) {
	utils.Init()
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

	// expected by simulated backend
	chainID := gethparams.AllEthashProtocolChanges.ChainID.Uint64()
	rpcClient, _ := rpc.NewClient(s.client, chainID, params.UpstreamRPCConfig{}, nil, nil)
	rpcClient.UpstreamChainID = chainID
	nodeConfig, err := utils.MakeTestNodeConfigWithDataDir("", "/tmp", chainID)
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
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), gomock.Eq(common.Address(account.Address)), gethrpc.PendingBlockNumber).Return(&returnNonce, nil)
	if !args.IsDynamicFeeTx() {
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
	}
	// Prepare the transaction and RLP encode it.
	data := s.rlpEncodeTx(args, s.nodeConfig, account, &resultNonce, usedGas, usedGasPrice)
	// Expect the RLP encoded transaction.
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), data).Return(common.Hash{}, txErr)
}

func (s *TransactorSuite) rlpEncodeTx(args SendTxArgs, config *params.NodeConfig, account *account.SelectedExtKey, nonce *hexutil.Uint64, gas hexutil.Uint64, gasPrice *big.Int) hexutil.Bytes {
	var txData gethtypes.TxData
	to := common.Address(*args.To)
	if args.IsDynamicFeeTx() {
		gasTipCap := (*big.Int)(args.MaxPriorityFeePerGas)
		gasFeeCap := (*big.Int)(args.MaxFeePerGas)

		txData = &gethtypes.DynamicFeeTx{
			Nonce:     uint64(*nonce),
			Gas:       uint64(gas),
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			To:        &to,
			Value:     args.Value.ToInt(),
			Data:      args.GetInput(),
		}
	} else {
		txData = &gethtypes.LegacyTx{
			Nonce:    uint64(*nonce),
			GasPrice: gasPrice,
			Gas:      uint64(gas),
			To:       &to,
			Value:    args.Value.ToInt(),
			Data:     args.GetInput(),
		}
	}

	newTx := gethtypes.NewTx(txData)
	chainID := big.NewInt(int64(s.nodeConfig.NetworkID))

	signedTx, err := gethtypes.SignTx(newTx, gethtypes.NewLondonSigner(chainID), account.AccountKey.PrivateKey)
	s.NoError(err)
	data, err := signedTx.MarshalBinary()
	s.NoError(err)
	return hexutil.Bytes(data)
}

func (s *TransactorSuite) TestGasValues() {
	key, _ := gethcrypto.GenerateKey()
	selectedAccount := &account.SelectedExtKey{
		Address:    account.FromAddress(utils.TestConfig.Account1.WalletAddress),
		AccountKey: &types.Key{PrivateKey: key},
	}
	testCases := []struct {
		name                 string
		gas                  *hexutil.Uint64
		gasPrice             *hexutil.Big
		maxFeePerGas         *hexutil.Big
		maxPriorityFeePerGas *hexutil.Big
	}{
		{
			"noGasDef",
			nil,
			nil,
			nil,
			nil,
		},
		{
			"gasDefined",
			&testGas,
			nil,
			nil,
			nil,
		},
		{
			"gasPriceDefined",
			nil,
			testGasPrice,
			nil,
			nil,
		},
		{
			"nilSignTransactionSpecificArgs",
			nil,
			nil,
			nil,
			nil,
		},

		{
			"maxFeeAndPriorityset",
			nil,
			nil,
			testGasPrice,
			testGasPrice,
		},
	}

	for _, testCase := range testCases {
		s.T().Run(testCase.name, func(t *testing.T) {
			s.SetupTest()
			args := SendTxArgs{
				From:                 account.FromAddress(utils.TestConfig.Account1.WalletAddress),
				To:                   account.ToAddress(utils.TestConfig.Account2.WalletAddress),
				Gas:                  testCase.gas,
				GasPrice:             testCase.gasPrice,
				MaxFeePerGas:         testCase.maxFeePerGas,
				MaxPriorityFeePerGas: testCase.maxPriorityFeePerGas,
			}
			s.setupTransactionPoolAPI(args, testNonce, testNonce, selectedAccount, nil)

			hash, err := s.manager.SendTransaction(args, selectedAccount)
			s.NoError(err)
			s.False(reflect.DeepEqual(hash, common.Hash{}))
		})
	}
}

func (s *TransactorSuite) TestArgsValidation() {
	args := SendTxArgs{
		From:  account.FromAddress(utils.TestConfig.Account1.WalletAddress),
		To:    account.ToAddress(utils.TestConfig.Account2.WalletAddress),
		Data:  types.HexBytes([]byte{0x01, 0x02}),
		Input: types.HexBytes([]byte{0x02, 0x01}),
	}
	s.False(args.Valid())
	selectedAccount := &account.SelectedExtKey{
		Address: account.FromAddress(utils.TestConfig.Account1.WalletAddress),
	}
	_, err := s.manager.SendTransaction(args, selectedAccount)
	s.EqualError(err, ErrInvalidSendTxArgs.Error())
}

func (s *TransactorSuite) TestAccountMismatch() {
	args := SendTxArgs{
		From: account.FromAddress(utils.TestConfig.Account1.WalletAddress),
		To:   account.ToAddress(utils.TestConfig.Account2.WalletAddress),
	}

	var err error

	// missing account
	_, err = s.manager.SendTransaction(args, nil)
	s.EqualError(err, account.ErrNoAccountSelected.Error())

	// mismatched accounts
	selectedAccount := &account.SelectedExtKey{
		Address: account.FromAddress(utils.TestConfig.Account2.WalletAddress),
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
	chainID := s.nodeConfig.NetworkID
	key, _ := gethcrypto.GenerateKey()
	selectedAccount := &account.SelectedExtKey{
		Address:    account.FromAddress(utils.TestConfig.Account1.WalletAddress),
		AccountKey: &types.Key{PrivateKey: key},
	}
	nonce := hexutil.Uint64(0)

	for i := 0; i < txCount; i++ {
		args := SendTxArgs{
			From: account.FromAddress(utils.TestConfig.Account1.WalletAddress),
			To:   account.ToAddress(utils.TestConfig.Account2.WalletAddress),
		}
		s.setupTransactionPoolAPI(args, nonce, hexutil.Uint64(i), selectedAccount, nil)

		_, err := s.manager.SendTransaction(args, selectedAccount)
		s.NoError(err)
		resultNonce, _ := s.manager.nonce.localNonce[chainID].Load(args.From)
		s.Equal(uint64(i)+1, resultNonce.(uint64))
	}

	nonce = hexutil.Uint64(5)
	args := SendTxArgs{
		From: account.FromAddress(utils.TestConfig.Account1.WalletAddress),
		To:   account.ToAddress(utils.TestConfig.Account2.WalletAddress),
	}

	s.setupTransactionPoolAPI(args, nonce, nonce, selectedAccount, nil)

	_, err := s.manager.SendTransaction(args, selectedAccount)
	s.NoError(err)

	resultNonce, _ := s.manager.nonce.localNonce[chainID].Load(args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))

	testErr := errors.New("test")
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), gomock.Eq(common.Address(selectedAccount.Address)), gethrpc.PendingBlockNumber).Return(nil, testErr)
	args = SendTxArgs{
		From: account.FromAddress(utils.TestConfig.Account1.WalletAddress),
		To:   account.ToAddress(utils.TestConfig.Account2.WalletAddress),
	}

	_, err = s.manager.SendTransaction(args, selectedAccount)
	s.EqualError(err, testErr.Error())
	resultNonce, _ = s.manager.nonce.localNonce[chainID].Load(args.From)
	s.Equal(uint64(nonce)+1, resultNonce.(uint64))
}

func (s *TransactorSuite) TestSendTransactionWithSignature() {
	privKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	address := crypto.PubkeyToAddress(privKey.PublicKey)

	scenarios := []struct {
		localNonce  hexutil.Uint64
		txNonce     hexutil.Uint64
		expectError bool
	}{
		{
			localNonce:  hexutil.Uint64(0),
			txNonce:     hexutil.Uint64(0),
			expectError: false,
		},
		{
			localNonce:  hexutil.Uint64(1),
			txNonce:     hexutil.Uint64(0),
			expectError: true,
		},
		{
			localNonce:  hexutil.Uint64(0),
			txNonce:     hexutil.Uint64(1),
			expectError: true,
		},
	}

	for _, scenario := range scenarios {
		desc := fmt.Sprintf("local nonce: %d, tx nonce: %d, expect error: %v", scenario.localNonce, scenario.txNonce, scenario.expectError)
		s.T().Run(desc, func(t *testing.T) {
			nonce := scenario.txNonce
			from := address
			to := address
			value := (*hexutil.Big)(big.NewInt(10))
			gas := hexutil.Uint64(21000)
			gasPrice := (*hexutil.Big)(big.NewInt(2000000000))
			data := []byte{}
			chainID := big.NewInt(int64(s.nodeConfig.NetworkID))
			s.manager.nonce.localNonce[s.nodeConfig.NetworkID] = &sync.Map{}
			s.manager.nonce.localNonce[s.nodeConfig.NetworkID].Store(address, uint64(scenario.localNonce))
			args := SendTxArgs{
				From:     from,
				To:       &to,
				Gas:      &gas,
				GasPrice: gasPrice,
				Value:    value,
				Nonce:    &nonce,
				Data:     nil,
			}

			// simulate transaction signed externally
			signer := gethtypes.NewLondonSigner(chainID)
			tx := gethtypes.NewTransaction(uint64(nonce), common.Address(to), (*big.Int)(value), uint64(gas), (*big.Int)(gasPrice), data)
			hash := signer.Hash(tx)
			sig, err := gethcrypto.Sign(hash[:], privKey)
			s.Require().NoError(err)
			txWithSig, err := tx.WithSignature(signer, sig)
			s.Require().NoError(err)
			expectedEncodedTx, err := rlp.EncodeToBytes(txWithSig)
			s.Require().NoError(err)

			s.txServiceMock.EXPECT().
				GetTransactionCount(gomock.Any(), common.Address(address), gethrpc.PendingBlockNumber).
				Return(&scenario.localNonce, nil)

			if !scenario.expectError {
				s.txServiceMock.EXPECT().
					SendRawTransaction(gomock.Any(), hexutil.Bytes(expectedEncodedTx)).
					Return(common.Hash{}, nil)
			}

			_, err = s.manager.BuildTransactionAndSendWithSignature(s.nodeConfig.NetworkID, args, sig)
			if scenario.expectError {
				s.Error(err)
				// local nonce should not be incremented
				resultNonce, _ := s.manager.nonce.localNonce[s.nodeConfig.NetworkID].Load(args.From)
				s.Equal(uint64(scenario.localNonce), resultNonce.(uint64))
			} else {
				s.NoError(err)
				// local nonce should be incremented
				resultNonce, _ := s.manager.nonce.localNonce[s.nodeConfig.NetworkID].Load(args.From)

				s.Equal(uint64(nonce)+1, resultNonce.(uint64))
			}
		})
	}
}

func (s *TransactorSuite) TestSendTransactionWithSignature_InvalidSignature() {
	args := SendTxArgs{}
	_, err := s.manager.BuildTransactionAndSendWithSignature(1, args, []byte{})
	s.Equal(ErrInvalidSignatureSize, err)
}

func (s *TransactorSuite) TestHashTransaction() {
	privKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	address := crypto.PubkeyToAddress(privKey.PublicKey)

	remoteNonce := hexutil.Uint64(1)
	txNonce := hexutil.Uint64(0)
	from := address
	to := address
	value := (*hexutil.Big)(big.NewInt(10))
	gas := hexutil.Uint64(21000)
	gasPrice := (*hexutil.Big)(big.NewInt(2000000000))

	args := SendTxArgs{
		From:     from,
		To:       &to,
		Gas:      &gas,
		GasPrice: gasPrice,
		Value:    value,
		Nonce:    &txNonce,
		Data:     nil,
	}

	s.txServiceMock.EXPECT().
		GetTransactionCount(gomock.Any(), common.Address(address), gethrpc.PendingBlockNumber).
		Return(&remoteNonce, nil)

	newArgs, hash, err := s.manager.HashTransaction(args)
	s.Require().NoError(err)
	// args should be updated with the right nonce
	s.NotEqual(*args.Nonce, *newArgs.Nonce)
	s.Equal(remoteNonce, *newArgs.Nonce)

	s.NotEqual(common.Hash{}, hash)
}
