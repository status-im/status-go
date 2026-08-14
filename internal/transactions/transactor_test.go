package transactions

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"

	accsmanagement "github.com/status-im/status-go/internal/accounts-management"
	"github.com/status-im/status-go/internal/accounts-management/generator"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/rpc/chain"
	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
	"github.com/status-im/status-go/internal/rpc/chain/rpclimiter"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	fake "github.com/status-im/status-go/internal/transactions/fake"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/wallettypes"
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

	// expected by simulated backend
	chainID := gethparams.AllEthashProtocolChanges.ChainID.Uint64()

	ethClients := []ethclient.RPSLimitedEthClientInterface{
		ethclient.NewRPSLimitedEthClient(s.client, rpclimiter.NewRPCRpsLimiter(nil), "local-1-chain-id-1-circuit", "local-1-chain-id-1-provider"),
	}
	localClient := chain.NewClient(ethClients, chainID, nil)
	ethClientGetter := mock_rpcclient.NewMockEthClientGetter(s.txServiceMockCtrl)
	ethClientGetter.EXPECT().EthClient(chainID).Return(localClient, nil).AnyTimes()

	var err error
	s.nodeConfig, err = params.NewNodeConfig("/tmp", chainID)
	s.Require().NoError(err)
	err = s.nodeConfig.Validate()
	s.Require().NoError(err)

	s.manager = NewTransactor()
	s.manager.SetEthClientGetter(ethClientGetter, time.Second)
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

func (s *TransactorSuite) setupTransactionPoolAPI(args wallettypes.SendTxArgs, returnNonce, resultNonce hexutil.Uint64, account *generator.Account, txErr error) {
	// Expect calls to gas functions only if there are no user defined values.
	// And also set the expected gas and gas price for RLP encoding the expected tx.
	var usedGas hexutil.Uint64
	var usedGasPrice *big.Int
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), gomock.Eq(common.Address(account.Address())), gethrpc.PendingBlockNumber).Return(&returnNonce, nil)
	if !args.IsDynamicFeeTx() {
		if args.GasPrice == nil {
			usedGasPrice = (*big.Int)(testGasPrice)
			s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(testGasPrice, nil)
		} else {
			usedGasPrice = (*big.Int)(args.GasPrice)
		}
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
	s.txServiceMock.EXPECT().SendRawTransaction(gomock.Any(), data).Return(common.Hash{}, txErr)
}

func (s *TransactorSuite) rlpEncodeTx(args wallettypes.SendTxArgs, config *params.NodeConfig, account *generator.Account, nonce *hexutil.Uint64, gas hexutil.Uint64, gasPrice *big.Int) hexutil.Bytes {
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

	signedTx, err := gethtypes.SignTx(newTx, gethtypes.NewLondonSigner(chainID), account.PrivateKey())
	s.NoError(err)
	data, err := signedTx.MarshalBinary()
	s.NoError(err)
	return hexutil.Bytes(data)
}

func (s *TransactorSuite) TestGasValues() {
	key, _ := gethcrypto.GenerateKey()
	selectedAccount := generator.NewAccount(key, nil)
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
			args := wallettypes.SendTxArgs{
				From:                 selectedAccount.Address(),
				To:                   fakeAddress(),
				Gas:                  testCase.gas,
				GasPrice:             testCase.gasPrice,
				MaxFeePerGas:         testCase.maxFeePerGas,
				MaxPriorityFeePerGas: testCase.maxPriorityFeePerGas,
			}
			s.setupTransactionPoolAPI(args, testNonce, testNonce, selectedAccount, nil)

			hash, _, err := s.manager.SendTransactionWithChainID(1337, args, -1, selectedAccount)
			s.NoError(err)
			s.False(reflect.DeepEqual(hash, common.Hash{}))
		})
	}
}

func (s *TransactorSuite) setupBuildTransactionMocks(args wallettypes.SendTxArgs, account *accsmanagementtypes.SelectedExtKey) {
	s.txServiceMock.EXPECT().GetTransactionCount(gomock.Any(), gomock.Eq(common.Address(account.Address)), gethrpc.PendingBlockNumber).Return(&testNonce, nil)

	if !args.IsDynamicFeeTx() && args.GasPrice == nil {
		s.txServiceMock.EXPECT().GasPrice(gomock.Any()).Return(testGasPrice, nil)
	}

	if args.Gas == nil {
		s.txServiceMock.EXPECT().EstimateGas(gomock.Any(), gomock.Any()).Return(testGas, nil)
	}
}

func (s *TransactorSuite) TestBuildAndValidateTransaction() {
	address1 := fakeAddress()
	address2 := fakeAddress()

	key, _ := gethcrypto.GenerateKey()
	selectedAccount := &accsmanagementtypes.SelectedExtKey{
		Address:    *address1,
		AccountKey: &accsmanagementtypes.Key{PrivateKey: key},
	}

	chainID := s.nodeConfig.NetworkID
	fromAddress := *address1
	toAddress := *address2
	value := (*hexutil.Big)(big.NewInt(10))

	expectedGasPrice := (*big.Int)(testGasPrice)
	expectedGas := uint64(testGas)
	expectedNonce := uint64(testNonce)

	s.T().Run("DynamicFeeTransaction", func(t *testing.T) {
		s.SetupTest()

		gas := hexutil.Uint64(21000)
		args := wallettypes.SendTxArgs{
			From:                 fromAddress,
			To:                   &toAddress,
			Gas:                  &gas,
			Value:                value,
			MaxFeePerGas:         testGasPrice,
			MaxPriorityFeePerGas: testGasPrice,
		}
		s.setupBuildTransactionMocks(args, selectedAccount)

		tx, _, err := s.manager.ValidateAndBuildTransaction(chainID, args, -1)
		s.NoError(err)
		s.Equal(tx.Gas(), uint64(gas), "The gas shouldn't be estimated, but should use the gas from the Tx")
		s.Equal(tx.GasFeeCap(), expectedGasPrice, "The maxFeePerGas should be the same as in the original Tx")
		s.Equal(tx.GasTipCap(), expectedGasPrice, "The maxPriorityFeePerGas should be the same as in the original Tx")
		s.Equal(tx.Type(), uint8(gethtypes.DynamicFeeTxType), "The transaction type should be DynamicFeeTxType")
	})

	s.T().Run("DynamicFeeTransaction with gas estimation", func(t *testing.T) {
		s.SetupTest()
		args := wallettypes.SendTxArgs{
			From:                 fromAddress,
			To:                   &toAddress,
			Value:                value,
			MaxFeePerGas:         testGasPrice,
			MaxPriorityFeePerGas: testGasPrice,
		}
		s.setupBuildTransactionMocks(args, selectedAccount)

		tx, _, err := s.manager.ValidateAndBuildTransaction(chainID, args, -1)
		s.NoError(err)
		s.Equal(tx.Gas(), expectedGas, "The gas should be estimated if not present in the original Tx")
		s.Equal(tx.Nonce(), expectedNonce, "The nonce should be added if not present in the original Tx")
		s.Equal(tx.GasFeeCap(), expectedGasPrice, "The maxFeePerGas should be the same as in the original Tx")
		s.Equal(tx.GasTipCap(), expectedGasPrice, "The maxPriorityFeePerGas should be the same as in the original Tx")
		s.Equal(tx.Type(), uint8(gethtypes.DynamicFeeTxType), "The transaction type should be DynamicFeeTxType")
	})

	s.T().Run("LegacyTransaction", func(t *testing.T) {
		s.SetupTest()

		gas := hexutil.Uint64(21000)
		gasPrice := (*hexutil.Big)(big.NewInt(10))
		args := wallettypes.SendTxArgs{
			From:     fromAddress,
			To:       &toAddress,
			Value:    value,
			Gas:      &gas,
			GasPrice: gasPrice,
		}
		s.setupBuildTransactionMocks(args, selectedAccount)

		tx, _, err := s.manager.ValidateAndBuildTransaction(chainID, args, -1)
		s.NoError(err)
		s.Equal(tx.Gas(), uint64(gas), "The gas shouldn't be estimated, but should use the gas from the Tx")
		s.Equal(tx.GasPrice(), expectedGasPrice, "The gasPrice should be the same as in the original Tx")
		s.Equal(tx.Type(), uint8(gethtypes.LegacyTxType), "The transaction type should be LegacyTxType")
	})
	s.T().Run("LegacyTransaction without gas estimation", func(t *testing.T) {
		s.SetupTest()

		args := wallettypes.SendTxArgs{
			From:  fromAddress,
			To:    &toAddress,
			Value: value,
		}
		s.setupBuildTransactionMocks(args, selectedAccount)

		tx, _, err := s.manager.ValidateAndBuildTransaction(chainID, args, -1)
		s.NoError(err)
		s.Equal(tx.Gas(), expectedGas, "The gas should be estimated if not present in the original Tx")
		s.Equal(tx.GasPrice(), expectedGasPrice, "The gasPrice should be estimated if not present in the original Tx")
		s.Equal(tx.Type(), uint8(gethtypes.LegacyTxType), "The transaction type should be LegacyTxType")
	})
}

func fakeAddress() *cryptotypes.Address {
	var address cryptotypes.Address
	gofakeit.Slice(&address)
	return &address
}

func (s *TransactorSuite) TestArgsValidation() {
	args := wallettypes.SendTxArgs{
		From:  *fakeAddress(),
		To:    fakeAddress(),
		Data:  cryptotypes.HexBytes([]byte{0x01, 0x02}),
		Input: cryptotypes.HexBytes([]byte{0x02, 0x01}),
	}
	s.False(args.Valid())
	selectedAccount := generator.NewAccount(nil, nil)
	_, _, err := s.manager.SendTransactionWithChainID(1, args, -1, selectedAccount)
	s.EqualError(err, wallettypes.ErrInvalidSendTxArgs.Error())
}

func (s *TransactorSuite) TestAccountMismatch() {
	args := wallettypes.SendTxArgs{
		From: *fakeAddress(),
		To:   fakeAddress(),
	}

	var err error

	// missing account
	_, _, err = s.manager.SendTransactionWithChainID(1, args, -1, nil)
	s.EqualError(err, accsmanagement.ErrNoAccountSelected.Error())

	// mismatched accounts
	selectedAccount := generator.NewAccount(nil, nil)
	_, _, err = s.manager.SendTransactionWithChainID(1, args, -1, selectedAccount)
	s.EqualError(err, wallettypes.ErrInvalidTxSender.Error())
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

	for _, localScenario := range scenarios {
		// to satisfy gosec: C601 checks
		scenario := localScenario
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
			args := wallettypes.SendTxArgs{
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

			tx, err = s.manager.BuildTransactionWithSignature(s.nodeConfig.NetworkID, args, sig)
			if scenario.expectError {
				s.Error(err)
			} else {
				s.NoError(err)

				_, err = s.manager.SendTransactionWithSignature(&args, tx)
				if scenario.expectError {
					s.Error(err)
				} else {
					s.NoError(err)
				}
			}
		})
	}
}

func (s *TransactorSuite) TestSendTransactionWithSignature_InvalidSignature() {
	args := wallettypes.SendTxArgs{}
	_, err := s.manager.BuildTransactionWithSignature(1, args, []byte{})
	s.Equal(ErrInvalidSignatureSize, err)
}

func (s *TransactorSuite) TestStoreAndTrackPendingTx() {
	s.Nil(s.manager.pendingTracker)

	// Empty tracker doesn't produce error
	err := s.manager.StoreAndTrackPendingTx(&wallettypes.SendTxArgs{}, nil)
	s.NoError(err)
}
