package transactions

import (
	"math/big"
	"reflect"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/params"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/status-im/status-go/transactions"
	"github.com/stretchr/testify/suite"
)

type initFunc func([]byte, *transactions.SendTxArgs)

func TestTransactionsTestSuite(t *testing.T) {
	suite.Run(t, new(TransactionsTestSuite))
}

type TransactionsTestSuite struct {
	e2e.BackendTestSuite
}

func (s *TransactionsTestSuite) TestCallRPCSendTransaction() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID)

	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.StatusNode().EnsureSync)

	err := s.Backend.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	result := s.Backend.CallRPC(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "eth_sendTransaction",
		"params": [{
			"from": "` + TestConfig.Account1.Address + `",
			"to": "0xd46e8dd67c5d32be8058bb8eb970870f07244567",
			"value": "0x9184e72a"
		}]
	}`)
	s.NotContains(result, "error")
}

func (s *TransactionsTestSuite) TestCallRPCSendTransactionUpstream() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID, params.StatusChainNetworkID)

	addr, err := GetRemoteURL()
	s.NoError(err)
	s.StartTestBackend(e2e.WithUpstream(addr))
	defer s.StopTestBackend()

	err = s.Backend.SelectAccount(TestConfig.Account2.Address, TestConfig.Account2.Password)
	s.NoError(err)

	result := s.Backend.CallRPC(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "eth_sendTransaction",
		"params": [{
			"from": "` + TestConfig.Account2.Address + `",
			"to": "` + TestConfig.Account1.Address + `",
			"value": "0x9184e72a"
		}]
	}`)
	s.NotContains(result, "error")
}

func (s *TransactionsTestSuite) TestEmptyToFieldPreserved() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID)

	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.StatusNode().EnsureSync)
	err := s.Backend.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	result := s.Backend.CallRPC(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "eth_sendTransaction",
		"params": [{
			"from": "` + TestConfig.Account1.Address + `"
		}]
	}`)
	s.NotContains(result, "error")
}

// TestSendContractCompat tries to send transaction using the legacy "Data"
// field, which is supported for backward compatibility reasons.
func (s *TransactionsTestSuite) TestSendContractTxCompat() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID)

	initFunc := func(byteCode []byte, args *transactions.SendTxArgs) {
		args.Data = (hexutil.Bytes)(byteCode)
	}
	s.testSendContractTx(initFunc, nil, "")
}

// TestSendContractCompat tries to send transaction using both the legacy
// "Data" and "Input" fields. Also makes sure that the error is returned if
// they have different values.
func (s *TransactionsTestSuite) TestSendContractTxCollision() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID)

	// Scenario 1: Both fields are filled and have the same value, expect success
	initFunc := func(byteCode []byte, args *transactions.SendTxArgs) {
		args.Input = (hexutil.Bytes)(byteCode)
		args.Data = (hexutil.Bytes)(byteCode)
	}
	s.testSendContractTx(initFunc, nil, "")

	// Scenario 2: Both fields are filled with different values, expect an error
	inverted := func(source []byte) []byte {
		inverse := make([]byte, len(source))
		copy(inverse, source)
		for i, b := range inverse {
			inverse[i] = b ^ 0xFF
		}
		return inverse
	}

	initFunc2 := func(byteCode []byte, args *transactions.SendTxArgs) {
		args.Input = (hexutil.Bytes)(byteCode)
		args.Data = (hexutil.Bytes)(inverted(byteCode))
	}
	s.testSendContractTx(initFunc2, transactions.ErrInvalidSendTxArgs, "expected error when invalid tx args are sent")
}

func (s *TransactionsTestSuite) TestSendContractTx() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID)

	initFunc := func(byteCode []byte, args *transactions.SendTxArgs) {
		args.Input = (hexutil.Bytes)(byteCode)
	}
	s.testSendContractTx(initFunc, nil, "")
}

func (s *TransactionsTestSuite) testSendContractTx(setInputAndDataValue initFunc, expectedError error, expectedErrorDescription string) {
	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.StatusNode().EnsureSync)

	err := s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	// this call blocks, up until Complete Transaction is called
	byteCode, err := hexutil.Decode(`0x6060604052341561000c57fe5b5b60a58061001b6000396000f30060606040526000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680636ffa1caa14603a575bfe5b3415604157fe5b60556004808035906020019091905050606b565b6040518082815260200191505060405180910390f35b60008160020290505b9190505600a165627a7a72305820ccdadd737e4ac7039963b54cee5e5afb25fa859a275252bdcf06f653155228210029`)
	s.NoError(err)

	gas := uint64(params.DefaultGas)
	args := transactions.SendTxArgs{
		From: account.FromAddress(TestConfig.Account1.Address),
		To:   nil, // marker, contract creation is expected
		//Value: (*hexutil.Big)(new(big.Int).Mul(big.NewInt(1), gethcommon.Ether)),
		Gas: (*hexutil.Uint64)(&gas),
	}

	setInputAndDataValue(byteCode, &args)
	result := s.Backend.SendTransaction(args, TestConfig.Account1.Password)

	if expectedError != nil {
		s.Equal(expectedError, result.Error, expectedErrorDescription)
		return
	}
	s.NoError(result.Error, "cannot send transaction")
	s.False(reflect.DeepEqual(result.Response.Hash(), gethcommon.Hash{}), "transaction was never queued or completed")
	s.NoError(s.Backend.Logout())
}

func (s *TransactionsTestSuite) TestSendEther() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID)

	s.StartTestBackend()
	defer s.StopTestBackend()

	EnsureNodeSync(s.Backend.StatusNode().EnsureSync)

	err := s.Backend.AccountManager().SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	result := s.Backend.SendTransaction(transactions.SendTxArgs{
		From:  account.FromAddress(TestConfig.Account1.Address),
		To:    account.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	}, TestConfig.Account1.Password)

	s.NoError(result.Error, "cannot send transaction")

	s.False(reflect.DeepEqual(result.Response.Hash(), gethcommon.Hash{}), "transaction was never queued or completed")
}

func (s *TransactionsTestSuite) TestSendEtherTxUpstream() {
	CheckTestSkipForNetworks(s.T(), params.MainNetworkID, params.StatusChainNetworkID)

	addr, err := GetRemoteURL()
	s.NoError(err)
	s.StartTestBackend(e2e.WithUpstream(addr))
	defer s.StopTestBackend()

	err = s.Backend.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	result := s.Backend.SendTransaction(transactions.SendTxArgs{
		From:     account.FromAddress(TestConfig.Account1.Address),
		To:       account.ToAddress(TestConfig.Account2.Address),
		GasPrice: (*hexutil.Big)(big.NewInt(28000000000)),
		Value:    (*hexutil.Big)(big.NewInt(1000000000000)),
	}, TestConfig.Account1.Password)

	s.NoError(result.Error, "cannot send transaction")

	s.False(reflect.DeepEqual(result.Response.Hash(), gethcommon.Hash{}), "transaction was never queued or completed")
}
