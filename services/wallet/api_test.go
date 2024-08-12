package wallet

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	gomock "go.uber.org/mock/gomock"

	"github.com/status-im/status-go/services/wallet/onramp"
	mock_onramp "github.com/status-im/status-go/services/wallet/onramp/mock"
	"github.com/status-im/status-go/services/wallet/walletconnect"
)

// TestAPI_GetWalletConnectActiveSessions tames coverage
func TestAPI_GetWalletConnectActiveSessions(t *testing.T) {
	db, close := walletconnect.SetupTestDB(t)
	defer close()
	api := &API{
		s: &Service{db: db},
	}

	sessions, err := api.GetWalletConnectActiveSessions(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, 0, len(sessions))
}

// TestAPI_HashMessageEIP191
func TestAPI_HashMessageEIP191(t *testing.T) {
	api := &API{}

	res := api.HashMessageEIP191(context.Background(), []byte("test"))
	require.Equal(t, "0x4a5c5d454721bbbb25540c3317521e71c373ae36458f960d2ad46ef088110e95", res.String())
}

func TestAPI_IsChecksumValidForAddress(t *testing.T) {
	api := &API{}

	res, err := api.IsChecksumValidForAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.NoError(t, err)
	require.False(t, res)

	res, err = api.IsChecksumValidForAddress("0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa")
	require.NoError(t, err)
	require.True(t, res)
}

func TestAPI_GetCryptoOnRamps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provider0 := mock_onramp.NewMockProvider(ctrl)
	id0 := "provider0"
	provider0.EXPECT().ID().Return(id0).AnyTimes()
	provider1 := mock_onramp.NewMockProvider(ctrl)
	id1 := "provider1"
	provider1.EXPECT().ID().Return(id1).AnyTimes()
	providers := []onramp.Provider{provider0, provider1}
	onrampManager := onramp.NewManager(providers)

	api := &API{
		s: &Service{cryptoOnRampManager: onrampManager},
	}

	ctx := context.Background()

	// Check returned providers
	provider0.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{ID: id0}, nil)
	provider1.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{ID: id1}, nil)

	retProviders, err := api.GetCryptoOnRamps(ctx)
	require.NoError(t, err)
	require.Equal(t, len(providers), len(retProviders))
	require.Equal(t, id0, retProviders[0].ID)
	require.Equal(t, id1, retProviders[1].ID)

	// Check error handling
	provider0.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{}, errors.New("error"))
	provider1.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{ID: id1}, nil)
	retProviders, err = api.GetCryptoOnRamps(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(retProviders))
	require.Equal(t, id1, retProviders[0].ID)

	// Check URL retrieval
	provider1.EXPECT().GetURL(ctx, onramp.Parameters{}).Return("url", nil)
	url, err := api.GetCryptoOnRampURL(ctx, id1, onramp.Parameters{})
	require.NoError(t, err)
	require.Equal(t, "url", url)
}
