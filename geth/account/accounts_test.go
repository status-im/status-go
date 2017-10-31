package account

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/extkeys"
	stcm "github.com/status-im/status-go/geth/common"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

var testErr = fmt.Errorf("error")

func TestManager_Logout_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	whisperMock := NewMockwhisperService(ctrl)
	nodeMock.EXPECT().WhisperService().Return(whisperMock, nil)
	whisperMock.EXPECT().DeleteKeyPairs().Times(1).Return(nil)

	m := Manager{node: nodeMock}
	err := m.Logout()
	require.Empty(t, err)
}

func TestManager_AddressToDecryptedAccount_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	address := "0x0000000000000000000000000000000000000001"
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	acc := accounts.Account{Address: addressBytes}
	password := "123"

	accountKeyStoreMock := NewMockaccountKeyStorer(ctrl)
	accountKeyStoreMock.EXPECT().AccountDecryptedKey(acc, password).Times(1).Return(acc, nil, nil)
	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Return(accountKeyStoreMock, nil)

	m := Manager{node: nodeMock}
	_, _, err := m.AddressToDecryptedAccount(address, password)
	require.Empty(t, err)
}

func TestManager_AddressToDecryptedAccount_NotHex_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	address := "not hex"
	password := "123"

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	address := "some addr"
	password := "123"

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Return(nil, testErr)

	m := Manager{node: nodeMock}
	_, _, err := m.AddressToDecryptedAccount(address, password)
	require.Equal(t, testErr, err)
}

func TestManager_CreateAccount_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	password := "123"

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(gomock.Any(), password).Times(1).Return("", "", nil)

	m := Manager{extKeyImporter: importMock}
	_, _, mnemonic, err := m.CreateAccount(password)
	require.Empty(t, err)
	require.Equal(t, true, len(strings.Split(mnemonic, " ")) > 0)
}

func TestManager_CreateAccount_Import_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	password := "123"

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(gomock.Any(), password).Times(1).Return("", "", testErr)

	m := Manager{extKeyImporter: importMock}
	_, _, mnemonic, err := m.CreateAccount(password)
	require.Equal(t, testErr, err)
	require.Equal(t, "", mnemonic)
}

func TestManager_CreateChildAccouAccountKeyStorent_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	address := "0x0000000000000000000000000000000000000001"
	password := "123"
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	extKey, err := extkeys.NewKeyFromString("xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi")
	if err != nil {
		t.Fatal(err)
	}
	key := keystore.Key{ExtendedKey: extKey}
	acc := accounts.Account{Address: addressBytes}

	accountKeyStoreMock := NewMockaccountKeyStorer(ctrl)
	accountKeyStoreMock.EXPECT().AccountDecryptedKey(acc, password).Times(1).Return(acc, &key, nil)
	accountKeyStoreMock.EXPECT().IncSubAccountIndex(acc, password).Times(1).Return(nil)
	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(accountKeyStoreMock, nil)

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(gomock.Any(), password).Times(1).Return("", "", nil)

	m := Manager{node: nodeMock, extKeyImporter: importMock}
	_, _, err = m.CreateChildAccount(address, password)
	t.Log(err)
	//addr,pubkey, err:=m.CreateChildAccount(address,password)
	require.Empty(t, err)
	t.Log(m.selectedAccount)
}

func TestManager_CreateChildAccouAccountKeyStorent_WithSelectedAccount_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	//address:="0x0000000000000000000000000000000000000001"
	password := "123"
	addressBytes := common.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	extKey, err := extkeys.NewKeyFromString("xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi")
	if err != nil {
		t.Fatal(err)
	}
	key := keystore.Key{ExtendedKey: extKey}
	acc := accounts.Account{Address: addressBytes}

	accountKeyStoreMock := NewMockaccountKeyStorer(ctrl)
	accountKeyStoreMock.EXPECT().AccountDecryptedKey(acc, password).Times(1).Return(acc, &key, nil)
	accountKeyStoreMock.EXPECT().IncSubAccountIndex(acc, password).Times(1).Return(nil)
	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(accountKeyStoreMock, nil)

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(gomock.Any(), password).Times(1).Return("", "", nil)

	m := Manager{
		selectedAccount: &stcm.SelectedExtKey{
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	password := "123"

	importMock := NewMockextendedKeyImporter(ctrl)
	importMock.EXPECT().Import(gomock.Any(), password).Times(1).Return("", "", nil)

	m := Manager{extKeyImporter: importMock}
	_, _, err := m.RecoverAccount(password, "some string")
	require.Empty(t, err)
}
