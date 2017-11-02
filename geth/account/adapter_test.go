package account

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/geth/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProxy_Base_Success(t *testing.T) {
	keyStore := &keystore.KeyStore{}
	accountManager := &accounts.Manager{}
	whisper := &whisperv5.Whisper{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	nodeMock := common.NewMockNodeManager(ctrl)
	nodeMock.EXPECT().AccountKeyStore().Times(1).Return(keyStore, nil)
	nodeMock.EXPECT().AccountManager().Times(1).Return(accountManager, nil)
	nodeMock.EXPECT().WhisperService().Times(1).Return(whisper, nil)

	m := NewManager(nodeMock)
	aks, err := m.node.AccountKeyStore()
	require.Equal(t, keyStore, aks)
	require.Empty(t, err)

	am, err := m.node.AccountManager()
	require.Equal(t, accountManager, am)
	require.Empty(t, err)

	w, err := m.node.WhisperService()
	require.Equal(t, whisper, w)
	require.Empty(t, err)
}
