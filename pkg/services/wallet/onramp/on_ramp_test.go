package onramp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/status-im/status-go/pkg/services/wallet/onramp"
	mock_onramp "github.com/status-im/status-go/pkg/services/wallet/onramp/mock"

	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

func Test_GetCryptoOnRamps(t *testing.T) {
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

	ctx := context.Background()

	// Check returned providers
	provider0.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{ID: id0}, nil)
	provider1.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{ID: id1}, nil)

	retProviders, err := onrampManager.GetProviders(ctx)
	require.NoError(t, err)
	require.Equal(t, len(providers), len(retProviders))
	require.Equal(t, id0, retProviders[0].ID)
	require.Equal(t, id1, retProviders[1].ID)

	// Check error handling
	provider0.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{}, errors.New("error"))
	provider1.EXPECT().GetCryptoOnRamp(ctx).Return(onramp.CryptoOnRamp{ID: id1}, nil)
	retProviders, err = onrampManager.GetProviders(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(retProviders))
	require.Equal(t, id1, retProviders[0].ID)

	// Check URL retrieval
	provider1.EXPECT().GetURL(ctx, onramp.Parameters{}).Return("url", nil)
	url, err := onrampManager.GetURL(ctx, id1, onramp.Parameters{})
	require.NoError(t, err)
	require.Equal(t, "url", url)
}
