package transfer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/crypto/types"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	mock_transactor "github.com/status-im/status-go/transactions/mock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type dummyAccountsStorage struct {
	keypair *accsmanagementtypes.Keypair
	account *accsmanagementtypes.Account
}

func (d *dummyAccountsStorage) GetAccountByAddress(address types.Address) (*accsmanagementtypes.Account, error) {
	if address != d.account.Address {
		return nil, fmt.Errorf("address not found")
	}
	return d.account, nil
}

func (d *dummyAccountsStorage) GetKeypairByKeyUID(keyUID string) (*accsmanagementtypes.Keypair, error) {
	if keyUID != d.keypair.KeyUID {
		return nil, fmt.Errorf("keyUID not found")
	}
	return d.keypair, nil
}

func (d *dummyAccountsStorage) AddressExists(address types.Address) (bool, error) {
	return d.account.Address == address, nil
}

func (d *dummyAccountsStorage) GetWalletAddresses() ([]types.Address, error) {
	return []types.Address{d.account.Address}, nil
}

type dummySigner struct{}

func (d *dummySigner) Hash(tx *gethtypes.Transaction) common.Hash {
	return common.HexToHash("0xc8e7a34af766c4ba9dc9b3d49939806fbf41fa01250c5a26afa5659e87b2020b")
}

func setupTestSuite(t *testing.T) (*TransactionManager, *mock_transactor.MockTransactorIface) {
	accountsDB := setupAccountsStorage()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transactor := mock_transactor.NewMockTransactorIface(ctrl)
	return &TransactionManager{
		accountsDB: accountsDB,
		transactor: transactor,
	}, transactor
}

func setupAccountsStorage() *dummyAccountsStorage {
	return &dummyAccountsStorage{
		keypair: &accsmanagementtypes.Keypair{
			KeyUID: "keyUid",
		},
		account: &accsmanagementtypes.Account{
			KeyUID:  "keyUid",
			Address: types.Address{1},
		},
	}
}

func TestSignMessage(t *testing.T) {
	tm, _ := setupTestSuite(t)

	message := (types.HexBytes)(make([]byte, 32))
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	signature, err := tm.SignMessage(message, privateKey)
	require.NoError(t, err)
	require.NotEmpty(t, signature)
}

func TestSignMessage_InvalidAccount(t *testing.T) {
	tm, _ := setupTestSuite(t)

	message := (types.HexBytes)(make([]byte, 32))

	signature, err := tm.SignMessage(message, nil)
	require.Error(t, err)
	require.Empty(t, signature)
}

func TestSignMessage_InvalidMessage(t *testing.T) {
	tm, _ := setupTestSuite(t)

	message := types.HexBytes{}
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	signature, err := tm.SignMessage(message, privateKey)
	require.Error(t, err)
	require.Equal(t, "0x", signature)
}

func TestBuildTransaction(t *testing.T) {
	manager, transactor := setupTestSuite(t)

	chainID := uint64(1)
	nonce := uint64(1)
	gas := uint64(21000)
	sendArgs := wallettypes.SendTxArgs{
		From:                 types.Address{1},
		To:                   &types.Address{2},
		Value:                (*hexutil.Big)(big.NewInt(123)),
		Nonce:                (*hexutil.Uint64)(&nonce),
		Gas:                  (*hexutil.Uint64)(&gas),
		GasPrice:             (*hexutil.Big)(big.NewInt(1000000000)),
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000000)),
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000000)),
	}

	expectedTx := gethtypes.NewTransaction(nonce, common.Address(*sendArgs.To), sendArgs.Value.ToInt(), gas, sendArgs.GasPrice.ToInt(), nil)
	transactor.EXPECT().ValidateAndBuildTransaction(chainID, sendArgs, int64(-1)).Return(expectedTx, uint64(0), nil)

	response, err := manager.BuildTransaction(chainID, sendArgs)
	require.NoError(t, err)
	require.NotNil(t, response)

	accDB := manager.accountsDB.(*dummyAccountsStorage)
	signer := dummySigner{}
	expectedKeyUID := accDB.keypair.KeyUID
	expectedAddress := accDB.account.Address
	expectedAddressPath := ""
	expectedSignOnKeycard := false
	expectedMessageToSign := signer.Hash(expectedTx)

	require.Equal(t, expectedKeyUID, response.KeyUID)
	require.Equal(t, expectedAddress, response.Address)
	require.Equal(t, expectedAddressPath, response.AddressPath)
	require.Equal(t, expectedSignOnKeycard, response.SignOnKeycard)
	require.Equal(t, chainID, response.ChainID)
	require.Equal(t, expectedMessageToSign, response.MessageToSign)
	require.True(t, reflect.DeepEqual(sendArgs, response.TxArgs))
}

func TestBuildTransaction_AccountNotFound(t *testing.T) {
	manager, _ := setupTestSuite(t)

	chainID := uint64(1)
	nonce := uint64(1)
	gas := uint64(21000)
	sendArgs := wallettypes.SendTxArgs{
		From:                 types.Address{2},
		To:                   &types.Address{2},
		Value:                (*hexutil.Big)(big.NewInt(123)),
		Nonce:                (*hexutil.Uint64)(&nonce),
		Gas:                  (*hexutil.Uint64)(&gas),
		GasPrice:             (*hexutil.Big)(big.NewInt(1000000000)),
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000000)),
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000000)),
	}

	_, err := manager.BuildTransaction(chainID, sendArgs)
	require.Error(t, err)
}

func TestBuildTransaction_InvalidSendTxArgs(t *testing.T) {
	manager, transactor := setupTestSuite(t)

	chainID := uint64(1)
	sendArgs := wallettypes.SendTxArgs{
		From: types.Address{1},
		To:   &types.Address{2},
	}

	expectedErr := fmt.Errorf("invalid SendTxArgs")
	transactor.EXPECT().ValidateAndBuildTransaction(chainID, sendArgs, int64(-1)).Return(nil, uint64(0), expectedErr)
	tx, err := manager.BuildTransaction(chainID, sendArgs)
	require.Equal(t, expectedErr, err)
	require.Nil(t, tx)
}

func TestBuildRawTransaction(t *testing.T) {
	manager, transactor := setupTestSuite(t)

	chainID := uint64(1)
	nonce := uint64(1)
	gas := uint64(21000)
	sendArgs := wallettypes.SendTxArgs{
		From:                 types.Address{1},
		To:                   &types.Address{2},
		Value:                (*hexutil.Big)(big.NewInt(123)),
		Nonce:                (*hexutil.Uint64)(&nonce),
		Gas:                  (*hexutil.Uint64)(&gas),
		GasPrice:             (*hexutil.Big)(big.NewInt(1000000000)),
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000000)),
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000000)),
	}

	expectedTx := gethtypes.NewTransaction(1, common.Address(*sendArgs.To), sendArgs.Value.ToInt(), 21000, sendArgs.GasPrice.ToInt(), nil)
	signature := []byte("signature")
	transactor.EXPECT().BuildTransactionWithSignature(chainID, sendArgs, signature).Return(expectedTx, nil)

	response, err := manager.BuildRawTransaction(chainID, sendArgs, signature)
	require.NoError(t, err)
	require.NotNil(t, response)

	expectedData, _ := expectedTx.MarshalBinary()
	expectedHash := expectedTx.Hash()

	require.Equal(t, chainID, response.ChainID)
	require.Equal(t, sendArgs, response.TxArgs)
	require.Equal(t, types.EncodeHex(expectedData), response.RawTx)
	require.Equal(t, expectedHash, response.TxHash)
}
