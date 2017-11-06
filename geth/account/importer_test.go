package account

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/extkeys"
	"github.com/stretchr/testify/require"
)

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

	e := extendedKeyImporterBase{}
	addr, pub, err := e.Import(keystoreMock, extKey, password)

	require.Equal(t, "0x0000000000000000000000000000000000000000", addr)
	require.Equal(t, "0x0", pub)
	require.Empty(t, err)

}

func TestManager_ImportExtendedKey_ImportExtendedKeyErr_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	extKey := &extkeys.ExtendedKey{}
	password := "123"

	keystoreMock := NewMockaccountKeyStorer(ctrl)
	keystoreMock.EXPECT().ImportExtendedKey(extKey, password).
		Times(1).Return(accounts.Account{}, testErr)

	i := extendedKeyImporterBase{}
	addr, _, err := i.Import(keystoreMock, extKey, password)

	require.Equal(t, "", addr)
	require.Equal(t, testErr, err)
}

func TestManager_ImportExtendedKey_AccountDecryptedKeyErr_Fail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	extKey := &extkeys.ExtendedKey{}
	password := "123"
	acc := accounts.Account{Address: ethcommon.Address{}}

	keystoreMock := NewMockaccountKeyStorer(ctrl)
	keystoreMock.EXPECT().ImportExtendedKey(extKey, password).
		Times(1).Return(acc, nil)
	keystoreMock.EXPECT().AccountDecryptedKey(acc, password).Times(1).Return(acc, nil, fmt.Errorf("error"))

	i := extendedKeyImporterBase{}
	addr, _, err := i.Import(keystoreMock, extKey, password)

	require.Equal(t, "0x0000000000000000000000000000000000000000", addr)
	require.Equal(t, testErr, err)
}
