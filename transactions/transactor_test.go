package transactions

import (
	"fmt"
	"math/big"
	"reflect"
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
	rpcClient, _ := rpc.NewClient(s.client, chainID, params.UpstreamRPCConfig{}, nil, false, nil)
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

func (s *TransactorSuite) TestSendTransactionWithSignature() {
	privKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	address := crypto.PubkeyToAddress(privKey.PublicKey)

	scenarios := []struct {
		nonceFromNetwork hexutil.Uint64
		txNonce          hexutil.Uint64
		expectError      bool
	}{
		{
			nonceFromNetwork: hexutil.Uint64(0),
			txNonce:          hexutil.Uint64(0),
			expectError:      false,
		},
		{
			nonceFromNetwork: hexutil.Uint64(0),
			txNonce:          hexutil.Uint64(1),
			expectError:      true,
		},
	}

	for _, scenario := range scenarios {
		desc := fmt.Sprintf("nonceFromNetwork: %d, tx nonce: %d, expect error: %v", scenario.nonceFromNetwork, scenario.txNonce, scenario.expectError)
		s.T().Run(desc, func(t *testing.T) {
			nonce := scenario.txNonce
			from := address
			to := address
			value := (*hexutil.Big)(big.NewInt(10))
			gas := hexutil.Uint64(21000)
			gasPrice := (*hexutil.Big)(big.NewInt(2000000000))
			data := []byte{}
			chainID := big.NewInt(int64(s.nodeConfig.NetworkID))
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
				Return(&scenario.nonceFromNetwork, nil)

			if !scenario.expectError {
				s.txServiceMock.EXPECT().
					SendRawTransaction(gomock.Any(), hexutil.Bytes(expectedEncodedTx)).
					Return(common.Hash{}, nil)
			}

			_, err = s.manager.BuildTransactionAndSendWithSignature(s.nodeConfig.NetworkID, args, sig)
			if scenario.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
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
