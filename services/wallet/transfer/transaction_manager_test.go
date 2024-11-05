package transfer

import (
	"context"
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
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	mock_transactor "github.com/status-im/status-go/transactions/mock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type dummyAccountsStorage struct {
	keypair *accounts.Keypair
	account *accounts.Account
}

func (d *dummyAccountsStorage) GetAccountByAddress(address types.Address) (*accounts.Account, error) {
	if address != d.account.Address {
		return nil, fmt.Errorf("address not found")
	}
	return d.account, nil
}

func (d *dummyAccountsStorage) GetKeypairByKeyUID(keyUID string) (*accounts.Keypair, error) {
	if keyUID != d.keypair.KeyUID {
		return nil, fmt.Errorf("keyUID not found")
	}
	return d.keypair, nil
}

func (d *dummyAccountsStorage) AddressExists(address types.Address) (bool, error) {
	return d.account.Address == address, nil
}

type dummySigner struct{}

func (d *dummySigner) Hash(tx *gethtypes.Transaction) common.Hash {
	return common.HexToHash("0xc8e7a34af766c4ba9dc9b3d49939806fbf41fa01250c5a26afa5659e87b2020b")
}

func setupTestSuite(t *testing.T) (*TransactionManager, *mock_transactor.MockTransactorIface) {
	SetMultiTransactionIDGenerator(StaticIDCounter()) // to have different multi-transaction IDs even with fast execution
	accountsDB := setupAccountsStorage()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transactor := mock_transactor.NewMockTransactorIface(ctrl)
	return &TransactionManager{
		storage:    NewInMemMultiTransactionStorage(),
		accountsDB: accountsDB,
		transactor: transactor,
	}, transactor
}

func setupAccountsStorage() *dummyAccountsStorage {
	return &dummyAccountsStorage{
		keypair: &accounts.Keypair{
			KeyUID: "keyUid",
		},
		account: &accounts.Account{
			KeyUID:  "keyUid",
			Address: types.Address{1},
		},
	}
}

func areMultiTransactionsEqual(mt1, mt2 *MultiTransaction) bool {
	return mt1.Timestamp == mt2.Timestamp &&
		mt1.FromNetworkID == mt2.FromNetworkID &&
		mt1.ToNetworkID == mt2.ToNetworkID &&
		mt1.FromTxHash == mt2.FromTxHash &&
		mt1.ToTxHash == mt2.ToTxHash &&
		mt1.FromAddress == mt2.FromAddress &&
		mt1.ToAddress == mt2.ToAddress &&
		mt1.FromAsset == mt2.FromAsset &&
		mt1.ToAsset == mt2.ToAsset &&
		mt1.FromAmount.String() == mt2.FromAmount.String() &&
		mt1.ToAmount.String() == mt2.ToAmount.String() &&
		mt1.Type == mt2.Type &&
		mt1.CrossTxID == mt2.CrossTxID
}

func TestBridgeMultiTransactions(t *testing.T) {
	manager, _ := setupTestSuite(t)

	trx1 := NewMultiTransaction(
		/* Timestamp:		*/ 123,
		/* FromNetworkID:	*/ 0,
		/* ToNetworkID:		*/ 1,
		/* FromTxHash: 		*/ common.Hash{5},
		/* // Empty ToTxHash */ common.Hash{},
		/* FromAddress:	 	*/ common.Address{1},
		/* ToAddress:   	*/ common.Address{2},
		/* FromAsset:   	*/ "fromAsset",
		/* ToAsset:     	*/ "toAsset",
		/* FromAmount:  	*/ (*hexutil.Big)(big.NewInt(123)),
		/* ToAmount:    	*/ (*hexutil.Big)(big.NewInt(234)),
		/* Type:        	*/ MultiTransactionBridge,
		/* CrossTxID:   	*/ "crossTxD1",
	)

	trx2 := NewMultiTransaction(
		/* Timestamp:     */ 321,
		/* FromNetworkID: */ 1,
		/* ToNetworkID:   */ 0,
		/* //Empty FromTxHash */ common.Hash{},
		/* ToTxHash:    */ common.Hash{6},
		/* FromAddress: */ common.Address{2},
		/* ToAddress:   */ common.Address{1},
		/* FromAsset:   */ "fromAsset",
		/* ToAsset:     */ "toAsset",
		/* FromAmount:  */ (*hexutil.Big)(big.NewInt(123)),
		/* ToAmount:    */ (*hexutil.Big)(big.NewInt(234)),
		/* Type:        */ MultiTransactionBridge,
		/* CrossTxID:   */ "crossTxD2",
	)

	trxs := []*MultiTransaction{trx1, trx2}

	var err error
	ids := make([]wallet_common.MultiTransactionIDType, len(trxs))
	for i, trx := range trxs {
		ids[i], err = manager.InsertMultiTransaction(trx)
		require.NoError(t, err)
	}

	rst, err := manager.GetBridgeOriginMultiTransaction(context.Background(), trx1.ToNetworkID, trx1.CrossTxID)
	require.NoError(t, err)
	require.NotEmpty(t, rst)
	require.True(t, areMultiTransactionsEqual(trx1, rst))

	rst, err = manager.GetBridgeDestinationMultiTransaction(context.Background(), trx1.ToNetworkID, trx1.CrossTxID)
	require.NoError(t, err)
	require.Empty(t, rst)

	rst, err = manager.GetBridgeOriginMultiTransaction(context.Background(), trx2.ToNetworkID, trx2.CrossTxID)
	require.NoError(t, err)
	require.Empty(t, rst)

	rst, err = manager.GetBridgeDestinationMultiTransaction(context.Background(), trx2.ToNetworkID, trx2.CrossTxID)
	require.NoError(t, err)
	require.NotEmpty(t, rst)
	require.True(t, areMultiTransactionsEqual(trx2, rst))
}

func TestMultiTransactions(t *testing.T) {
	manager, _ := setupTestSuite(t)

	trx1 := *NewMultiTransaction(
		/* Timestamp:    */ 123,
		/* FromNetworkID:*/ 0,
		/* ToNetworkID:  */ 1,
		/* FromTxHash:   */ common.Hash{5},
		/* ToTxHash:     */ common.Hash{6},
		/* FromAddress:  */ common.Address{1},
		/* ToAddress:    */ common.Address{2},
		/* FromAsset:    */ "fromAsset",
		/* ToAsset:      */ "toAsset",
		/* FromAmount:   */ (*hexutil.Big)(big.NewInt(123)),
		/* ToAmount:     */ (*hexutil.Big)(big.NewInt(234)),
		/* Type:         */ MultiTransactionBridge,
		/* CrossTxID:    */ "crossTxD",
	)
	trx2 := trx1
	trx2.FromAmount = (*hexutil.Big)(big.NewInt(456))
	trx2.ToAmount = (*hexutil.Big)(big.NewInt(567))
	trx2.ID = multiTransactionIDGenerator()

	require.NotEqual(t, trx1.ID, trx2.ID)

	trxs := []*MultiTransaction{&trx1, &trx2}

	var err error
	ids := make([]wallet_common.MultiTransactionIDType, len(trxs))
	for i, trx := range trxs {
		ids[i], err = manager.InsertMultiTransaction(trx)
		require.NoError(t, err)
	}

	rst, err := manager.GetMultiTransactions(context.Background(), []wallet_common.MultiTransactionIDType{ids[0], 555})
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.True(t, areMultiTransactionsEqual(trxs[0], rst[0]))

	trx1.FromAmount = (*hexutil.Big)(big.NewInt(789))
	trx1.ToAmount = (*hexutil.Big)(big.NewInt(890))
	err = manager.UpdateMultiTransaction(&trx1)
	require.NoError(t, err)

	rst, err = manager.GetMultiTransactions(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, len(ids), len(rst))

	for i, id := range ids {
		found := false
		for _, trx := range rst {
			if id == trx.ID {
				found = true
				require.True(t, areMultiTransactionsEqual(trxs[i], trx))
				break
			}
		}
		require.True(t, found, "result contains transaction with id %d", id)
	}
}

func TestSignMessage(t *testing.T) {
	tm, _ := setupTestSuite(t)

	message := (types.HexBytes)(make([]byte, 32))
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	account := &types.Key{
		PrivateKey: privateKey,
	}

	signature, err := tm.SignMessage(message, account)
	require.NoError(t, err)
	require.NotEmpty(t, signature)
}

func TestSignMessage_InvalidAccount(t *testing.T) {
	tm, _ := setupTestSuite(t)

	message := (types.HexBytes)(make([]byte, 32))
	account := &types.Key{
		PrivateKey: nil,
	}

	signature, err := tm.SignMessage(message, account)
	require.Error(t, err)
	require.Empty(t, signature)
}

func TestSignMessage_InvalidMessage(t *testing.T) {
	tm, _ := setupTestSuite(t)

	message := types.HexBytes{}
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	account := &types.Key{
		PrivateKey: privateKey,
	}

	signature, err := tm.SignMessage(message, account)
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
