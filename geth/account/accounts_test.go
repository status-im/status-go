package account

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/extkeys"
	"github.com/stretchr/testify/require"
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

	m := Manager{nodeManager: nodeMock}
	err := m.Logout()
	require.Empty(t, err)
}

func TestManager_ImportExtendedKey_Success(t *testing.T) {
	extKey := &extkeys.ExtendedKey{}
	password := "123"
	acc := accounts.Account{Address: ethcommon.Address{}}
	key := keystore.Key{
		PrivateKey: &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	keystoreMock := NewMockaccountKeyStorer(ctrl)
	keystoreMock.EXPECT().ImportExtendedKey(extKey, password).
		Times(1).Return(acc, nil)

	keystoreMock.EXPECT().AccountDecryptedKey(acc, password).
		Times(1).Return(acc, &key, nil)

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(keystoreMock, nil)

	m := Manager{nodeManager: nodeMock}
	addr, pub, err := m.importExtendedKey(extKey, password)

	require.Equal(t, "0x0000000000000000000000000000000000000000", addr)
	require.Equal(t, "0x0", pub)
	require.Empty(t, err)

}

func TestManager_ImportExtendedKey_AccountKeyStoreErr_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(nil, testErr)

	extKey := &extkeys.ExtendedKey{}
	password := "123"
	m := Manager{nodeManager: nodeMock}
	_, _, err := m.importExtendedKey(extKey, password)
	require.Equal(t, testErr, err)
}

func TestManager_ImportExtendedKey_ImportExtendedKeyErr_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	extKey := &extkeys.ExtendedKey{}
	password := "123"
	acc := accounts.Account{Address: ethcommon.Address{}}

	keystoreMock := NewMockaccountKeyStorer(ctrl)
	keystoreMock.EXPECT().ImportExtendedKey(extKey, password).
		Times(1).Return(acc, nil)
	keystoreMock.EXPECT().AccountDecryptedKey(acc, password).Times(1).Return(acc, nil, fmt.Errorf("error"))

	nodeMock := NewMockaccountNode(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(keystoreMock, nil)

	m := Manager{nodeManager: nodeMock}

	addr, _, err := m.importExtendedKey(extKey, password)

	require.Equal(t, "0x0000000000000000000000000000000000000000", addr)
	require.Equal(t, testErr, err)
}
