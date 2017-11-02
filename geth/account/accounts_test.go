package account

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/extkeys"
	stcommon "github.com/status-im/status-go/geth/common"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

var testErr = fmt.Errorf("error")

func TestManager_Logout_Success(t *testing.T) {
	acc := &stcommon.SelectedExtKey{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	whisperMock := NewMockwhisperService(ctrl)
	nodeMock.EXPECT().WhisperService().Return(whisperMock, nil)
	whisperMock.EXPECT().DeleteKeyPairs().Times(1).Return(nil)

	m := Manager{node: nodeMock, selectedAccount: acc}
	err := m.Logout()
	require.Empty(t, err)
	require.Empty(t, m.selectedAccount)
}

func TestManager_Logout_WhisperServiceErr_Fail(t *testing.T) {
	acc := &stcommon.SelectedExtKey{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().WhisperService().Return(nil, testErr)

	m := Manager{node: nodeMock, selectedAccount: acc}
	err := m.Logout()
	require.Equal(t, testErr, err)
	require.Equal(t, acc, m.selectedAccount)
}

func TestManager_Logout_DeleteKeyPairsErr_Fail(t *testing.T) {
	acc := &stcommon.SelectedExtKey{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	whisperMock := NewMockwhisperService(ctrl)
	nodeMock.EXPECT().WhisperService().Return(whisperMock, nil)
	whisperMock.EXPECT().DeleteKeyPairs().Times(1).Return(testErr)

	m := Manager{node: nodeMock, selectedAccount: acc}
	err := m.Logout()
	require.Equal(t, fmt.Sprintf("%s: %s", ErrWhisperClearIdentitiesFailure.Error(), testErr.Error()), err.Error())
	require.Equal(t, acc, m.selectedAccount)
}

func TestManager_AddressToDecryptedAccount_Success(t *testing.T) {
	address := "0x0000000000000000000000000000000000000001"
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	acc := accounts.Account{Address: addressBytes}
	password := "123"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accountKeyStoreMock := NewMockaccountKeyStorer(ctrl)
	accountKeyStoreMock.EXPECT().AccountDecryptedKey(acc, password).Times(1).Return(acc, nil, nil)
	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Return(accountKeyStoreMock, nil)

	m := Manager{node: nodeMock}
	_, _, err := m.AddressToDecryptedAccount(address, password)
	require.Empty(t, err)
}

func TestManager_AddressToDecryptedAccount_NotHex_Fail(t *testing.T) {
	address := "not hex"
	password := "123"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accountKeyStoreMock := NewMockaccountKeyStorer(ctrl)
	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Return(accountKeyStoreMock, nil)

	m := Manager{node: nodeMock}
	acc, key, err := m.AddressToDecryptedAccount(address, password)
	require.Empty(t, key)
	require.Equal(t, accounts.Account{}, acc)
	require.Equal(t, ErrAddressToAccountMappingFailure, err)
}

func TestManager_AddressToDecryptedAccount_AccountKeyStoreErr_Fail(t *testing.T) {
	address := "some addr"
	password := "123"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Return(nil, testErr)

	m := Manager{node: nodeMock}
	_, _, err := m.AddressToDecryptedAccount(address, password)
	require.Equal(t, testErr, err)
}

func TestManager_CreateAccount_Success(t *testing.T) {
	password := "123"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(nil, nil)

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(nil, gomock.Any(), password).Times(1).Return("", "", nil)

	m := Manager{extKeyImporter: importMock, node: nodeMock}
	_, _, mnemonic, err := m.CreateAccount(password)
	require.Empty(t, err)
	require.Equal(t, true, len(strings.Split(mnemonic, " ")) > 0)
}

func TestManager_CreateAccount_Import_Fail(t *testing.T) {
	password := "123"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(nil, nil)

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(nil, gomock.Any(), password).Times(1).Return("", "", testErr)

	m := Manager{extKeyImporter: importMock, node: nodeMock}
	_, _, mnemonic, err := m.CreateAccount(password)

	require.Equal(t, testErr, err)
	require.Equal(t, "", mnemonic)
}

func TestManager_CreateChildAccouAccountKeyStorent_Success(t *testing.T) {
	address := "0x0000000000000000000000000000000000000001"
	password := "123"
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	extKey, err := extkeys.NewKeyFromString("xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi")
	if err != nil {
		t.Fatal(err)
	}
	key := keystore.Key{ExtendedKey: extKey}
	acc := accounts.Account{Address: addressBytes}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accountKeyStoreMock := NewMockaccountKeyStorer(ctrl)
	accountKeyStoreMock.EXPECT().AccountDecryptedKey(acc, password).Times(1).Return(acc, &key, nil)
	accountKeyStoreMock.EXPECT().IncSubAccountIndex(acc, password).Times(1).Return(nil)
	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(accountKeyStoreMock, nil)

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(accountKeyStoreMock, gomock.Any(), password).Times(1).Return("", "", nil)

	m := Manager{node: nodeMock, extKeyImporter: importMock}
	_, _, err = m.CreateChildAccount(address, password)
	require.Empty(t, err)
}

func TestManager_CreateChildAccouAccountKeyStorent_WithSelectedAccount_Success(t *testing.T) {
	password := "123"
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	extKey, err := extkeys.NewKeyFromString("xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi")
	if err != nil {
		t.Fatal(err)
	}
	key := keystore.Key{ExtendedKey: extKey}
	acc := accounts.Account{Address: addressBytes}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accountKeyStoreMock := NewMockaccountKeyStorer(ctrl)
	accountKeyStoreMock.EXPECT().AccountDecryptedKey(acc, password).Times(1).Return(acc, &key, nil)
	accountKeyStoreMock.EXPECT().IncSubAccountIndex(acc, password).Times(1).Return(nil)
	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(accountKeyStoreMock, nil)

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(accountKeyStoreMock, gomock.Any(), password).Times(1).Return("", "", nil)

	m := Manager{
		selectedAccount: &stcommon.SelectedExtKey{
			Address:    addressBytes,
			AccountKey: &key,
		},
		node:           nodeMock,
		extKeyImporter: importMock,
	}
	_, _, err = m.CreateChildAccount("", password)

	require.Empty(t, err)
	require.Equal(t, uint32(1), m.selectedAccount.AccountKey.SubAccountIndex)
}

func TestManager_RecoverAccount_Success(t *testing.T) {
	password := "123"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(nil, nil)

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(nil, gomock.Any(), password).Times(1).Return("", "", nil)

	m := Manager{extKeyImporter: importMock, node: nodeMock}
	_, _, err := m.RecoverAccount(password, "some string")
	require.Empty(t, err)
}

func TestManager_SelectAccount(t *testing.T) {
	acc := &stcommon.SelectedExtKey{}

	m := Manager{selectedAccount: acc}
	_, err := m.SelectedAccount()

	require.Empty(t, err)
}
func TestManager_SelectAccount_Fail(t *testing.T) {
	m := Manager{selectedAccount: nil}
	_, err := m.SelectedAccount()

	require.Equal(t, ErrNoAccountSelected, err)
}

func TestManager_ReSelectAccount_Success(t *testing.T) {
	acc := &stcommon.SelectedExtKey{
		AccountKey: &keystore.Key{
			PrivateKey: &ecdsa.PrivateKey{},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whisperServiceMock := NewMockwhisperService(ctrl)
	whisperServiceMock.EXPECT().SelectKeyPair(acc.AccountKey.PrivateKey).Times(1).Return(nil)
	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().WhisperService().Times(1).Return(whisperServiceMock, nil)

	m := Manager{selectedAccount: acc, node: nodeMock}
	err := m.ReSelectAccount()

	require.Empty(t, err)
}

func TestManager_ReSelectAccount_SelectedAccountEmpty_Success(t *testing.T) {
	m := Manager{selectedAccount: nil}
	err := m.ReSelectAccount()

	require.Equal(t, nil, err)
}

func TestManager_ReSelectAccount_WhisperServiceErr_Fail(t *testing.T) {
	acc := &stcommon.SelectedExtKey{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().WhisperService().Times(1).Return(nil, testErr)

	m := Manager{selectedAccount: acc, node: nodeMock}
	err := m.ReSelectAccount()

	require.Equal(t, testErr, err)
}

func TestManager_ReSelectAccount_SelectKeyPairErr_Fail(t *testing.T) {
	acc := &stcommon.SelectedExtKey{
		AccountKey: &keystore.Key{
			PrivateKey: &ecdsa.PrivateKey{},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	whisperServiceMock := NewMockwhisperService(ctrl)
	whisperServiceMock.EXPECT().SelectKeyPair(acc.AccountKey.PrivateKey).Times(1).Return(testErr)

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().WhisperService().Times(1).Return(whisperServiceMock, nil)

	m := Manager{selectedAccount: acc, node: nodeMock}
	err := m.ReSelectAccount()

	require.Equal(t, ErrWhisperIdentityInjectionFailure, err)
}

func TestManager_Accounts_SelectedAccountWithoutUpdateFromKeystore_Success(t *testing.T) {
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	addressBytesSubAccount := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

	selectedAcc := &stcommon.SelectedExtKey{
		Address: addressBytes,
		SubAccounts: []accounts.Account{
			{
				Address: addressBytesSubAccount,
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	walletMock := NewMockWallet(ctrl)
	walletMock.EXPECT().Accounts().Times(1).
		Return([]accounts.Account{{Address: addressBytes}, {Address: addressBytesSubAccount}})

	wallets := []accounts.Wallet{accounts.Wallet(walletMock)}

	gethAccMangerMock := NewMockgethAccountManager(ctrl)
	gethAccMangerMock.EXPECT().Wallets().Return(wallets)

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountManager().Times(1).Return(gethAccMangerMock, nil)

	m := Manager{node: nodeMock, selectedAccount: selectedAcc}
	acc, err := m.Accounts()
	require.Equal(t, []common.Address{addressBytes, addressBytesSubAccount}, acc)
	require.Empty(t, err)
}

func TestManager_Accounts_WithUnselectedAccount_Success(t *testing.T) {
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	walletMock := NewMockWallet(ctrl)
	walletMock.EXPECT().Accounts().Times(1).
		Return([]accounts.Account{{Address: addressBytes}})

	wallets := []accounts.Wallet{accounts.Wallet(walletMock)}

	gethAccMangerMock := NewMockgethAccountManager(ctrl)
	gethAccMangerMock.EXPECT().Wallets().Return(wallets)

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountManager().Times(1).Return(gethAccMangerMock, nil)

	m := Manager{node: nodeMock, selectedAccount: nil}
	acc, err := m.Accounts()
	require.Equal(t, []common.Address{}, acc)
	require.Empty(t, err)
}

func TestManager_SelectAccount_Success(t *testing.T) {
	addr := "0x0000000000000000000000000000000000000001"
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	password := "123"
	acc := accounts.Account{Address: addressBytes}
	key := &keystore.Key{
		PrivateKey:      &ecdsa.PrivateKey{},
		ExtendedKey:     &extkeys.ExtendedKey{},
		SubAccountIndex: 1,
	}
	subAccounts := []accounts.Account{{Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}}}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)

	accountNodeMock := NewMockaccountKeyStorer(ctrl)
	accountNodeMock.EXPECT().AccountDecryptedKey(acc, password).Times(1).Return(acc, key, nil)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(accountNodeMock, nil)

	whisperServiceMock := NewMockwhisperService(ctrl)
	whisperServiceMock.EXPECT().SelectKeyPair(key.PrivateKey).Times(1).Return(nil)
	nodeMock.EXPECT().WhisperService().Times(1).Return(whisperServiceMock, nil)

	finderMock := NewMocksubAccountFinder(ctrl)
	finderMock.EXPECT().Find(accountNodeMock, key.ExtendedKey, key.SubAccountIndex).Times(1).Return(subAccounts, nil)

	m := Manager{selectedAccount: nil, node: nodeMock, subAccountFinder: finderMock}
	err := m.SelectAccount(addr, password)

	require.Empty(t, err)
	require.NotNil(t, m.selectedAccount)
	require.Equal(t, stcommon.SelectedExtKey{
		Address:     addressBytes,
		AccountKey:  key,
		SubAccounts: subAccounts,
	}, *m.selectedAccount)
}

func TestManager_SelectAccount_BadAddress_Fail(t *testing.T) {
	addr := "bad address"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	accountNodeMock := NewMockaccountKeyStorer(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(accountNodeMock, nil)

	m := Manager{selectedAccount: nil, node: nodeMock}
	err := m.SelectAccount(addr, "")

	require.Equal(t, ErrAddressToAccountMappingFailure, err)
	require.Empty(t, m.selectedAccount)
}

func TestManager_SelectAccount_AccountDecryptedKeyErr_Fail(t *testing.T) {
	addr := "0x0000000000000000000000000000000000000001"
	passwrod := "123s"
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	acc := accounts.Account{Address: addressBytes}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)

	accountNodeMock := NewMockaccountKeyStorer(ctrl)
	accountNodeMock.EXPECT().AccountDecryptedKey(acc, passwrod).Times(1).Return(accounts.Account{}, nil, testErr)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(accountNodeMock, nil)

	m := Manager{selectedAccount: nil, node: nodeMock}
	err := m.SelectAccount(addr, passwrod)

	require.Equal(t, fmt.Sprintf("%s: %v", ErrAccountToKeyMappingFailure, testErr), err.Error(), err)
	require.Empty(t, m.selectedAccount)
}

func TestManager_SelectAccount_SelectKeyPairErr_Fail(t *testing.T) {
	addr := "0x0000000000000000000000000000000000000001"
	passwrod := "123"
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	acc := accounts.Account{Address: addressBytes}
	key := &keystore.Key{
		PrivateKey: &ecdsa.PrivateKey{},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)

	accountNodeMock := NewMockaccountKeyStorer(ctrl)
	accountNodeMock.EXPECT().AccountDecryptedKey(acc, passwrod).Times(1).Return(acc, key, nil)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(accountNodeMock, nil)

	whisperServiceMock := NewMockwhisperService(ctrl)
	whisperServiceMock.EXPECT().SelectKeyPair(key.PrivateKey).Times(1).Return(testErr)
	nodeMock.EXPECT().WhisperService().Times(1).Return(whisperServiceMock, nil)

	m := Manager{selectedAccount: nil, node: nodeMock}
	err := m.SelectAccount(addr, passwrod)

	require.Equal(t, ErrWhisperIdentityInjectionFailure, err)
}

func TestManager_refreshSelectedAccount_Success(t *testing.T) {
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	selectedAccount := &stcommon.SelectedExtKey{
		Address: addressBytes,
		AccountKey: &keystore.Key{
			ExtendedKey:     &extkeys.ExtendedKey{},
			SubAccountIndex: 1,
		},
	}
	subAccounts := []accounts.Account{{Address: common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}}}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(accountKeyStorer(nil), nil)

	subAccountFinderMock := NewMocksubAccountFinder(ctrl)
	subAccountFinderMock.EXPECT().Find(accountKeyStorer(nil), selectedAccount.AccountKey.ExtendedKey, selectedAccount.AccountKey.SubAccountIndex).
		Times(1).Return(subAccounts, nil)

	m := Manager{selectedAccount: selectedAccount, node: nodeMock, subAccountFinder: subAccountFinderMock}
	m.refreshSelectedAccount()

	require.Equal(t, &stcommon.SelectedExtKey{
		Address: addressBytes,
		AccountKey: &keystore.Key{
			ExtendedKey:     &extkeys.ExtendedKey{},
			SubAccountIndex: 1,
		},
		SubAccounts: subAccounts,
	}, m.selectedAccount)
}

func TestManager_refreshSelectedAccount_SelectedAccountEmpty_Fail(t *testing.T) {
	m := Manager{selectedAccount: nil}
	m.refreshSelectedAccount()
	require.Empty(t, m.selectedAccount)
}

func TestManager_VerifyAccountPassword_Success(t *testing.T) {
	foundKeyFile := []byte(`{"address":"45dea0fb0bba44f4fcf290bba71fd57d7117cbb8","crypto":{"cipher":"aes-128-ctr","ciphertext":"b87781948a1befd247bff51ef4063f716cf6c2d3481163e9a8f42e1f9bb74145","cipherparams":{"iv":"dc4926b48a105133d2f16b96833abf1e"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":2,"p":1,"r":8,"salt":"004244bbdc51cadda545b1cfa43cff9ed2ae88e08c61f1479dbb45410722f8f0"},"mac":"39990c1684557447940d4c69e06b1b82b2aceacb43f284df65c956daf3046b85"},"id":"ce541d8d-c79b-40f8-9f8c-20f59616faba","version":3}`)
	dir := "./somedir"
	addr := "0x45DeA0FB0bBA44f4fcF290bbA71Fd57d7117Cbb8"
	passwrod := ""
	addressBytes := common.BytesToAddress(common.FromHex(addr))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	keyFinderMock := NewMockkeyFileFinder(ctrl)
	keyFinderMock.EXPECT().Find(dir, addressBytes).Times(1).Return(foundKeyFile, nil)
	keyfileFinder = keyFinderMock
	m := Manager{}
	key, err := m.VerifyAccountPassword(dir, addr, passwrod)

	require.Empty(t, err)
	require.Equal(t, addressBytes, key.Address)
}

func TestManager_VerifyAccountPassword_SwapAttack_Fail(t *testing.T) {
	foundKeyFile := []byte(`{"address":"45dea0fb0bba44f4fcf290bba71fd57d7117cbb8","crypto":{"cipher":"aes-128-ctr","ciphertext":"b87781948a1befd247bff51ef4063f716cf6c2d3481163e9a8f42e1f9bb74145","cipherparams":{"iv":"dc4926b48a105133d2f16b96833abf1e"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":2,"p":1,"r":8,"salt":"004244bbdc51cadda545b1cfa43cff9ed2ae88e08c61f1479dbb45410722f8f0"},"mac":"39990c1684557447940d4c69e06b1b82b2aceacb43f284df65c956daf3046b85"},"id":"ce541d8d-c79b-40f8-9f8c-20f59616faba","version":3}`)
	dir := "./somedir"
	addr := "0x0000000000000000000000000000000000000001"
	passwrod := ""
	addressBytes := common.BytesToAddress(common.FromHex(addr))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	keyFinderMock := NewMockkeyFileFinder(ctrl)
	keyFinderMock.EXPECT().Find(dir, addressBytes).Times(1).Return(foundKeyFile, nil)
	keyfileFinder = keyFinderMock
	m := Manager{}
	key, err := m.VerifyAccountPassword(dir, addr, passwrod)

	require.Empty(t, key)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "account mismatch")
}

func TestManager_VerifyAccountPassword_WithEmptyKeyfile_Fail(t *testing.T) {
	foundKeyFile := []byte{}
	dir := "./somedir"
	addr := "0x0000000000000000000000000000000000000001"
	passwrod := ""
	addressBytes := common.BytesToAddress(common.FromHex(addr))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	keyFinderMock := NewMockkeyFileFinder(ctrl)
	keyFinderMock.EXPECT().Find(dir, addressBytes).Times(1).Return(foundKeyFile, nil)
	keyfileFinder = keyFinderMock
	m := Manager{}
	key, err := m.VerifyAccountPassword(dir, addr, passwrod)

	require.Empty(t, key)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "cannot locate account")
}

func TestManager_VerifyAccountPassword_FindErr_Fail(t *testing.T) {
	dir := "./somedir"
	addr := "0x0000000000000000000000000000000000000001"
	passwrod := ""
	addressBytes := common.BytesToAddress(common.FromHex(addr))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	keyFinderMock := NewMockkeyFileFinder(ctrl)
	keyFinderMock.EXPECT().Find(dir, addressBytes).Times(1).Return([]byte{}, testErr)
	keyfileFinder = keyFinderMock
	m := Manager{}
	key, err := m.VerifyAccountPassword(dir, addr, passwrod)

	require.Empty(t, key)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "cannot traverse key store folder")
}
