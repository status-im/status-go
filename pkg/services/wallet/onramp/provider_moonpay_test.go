package onramp_test

import (
	"context"
	"testing"

	"github.com/status-im/status-go/pkg/services/wallet/onramp"

	"github.com/stretchr/testify/require"
)

func TestMoonPayProvider(t *testing.T) {
	provider := onramp.NewMoonPayProvider()
	ctx := context.Background()

	ramp, err := provider.GetCryptoOnRamp(ctx)
	require.NoError(t, err)
	require.Equal(t, "moonpay", ramp.ID)
	require.Equal(t, "moonpay.com", ramp.Hostname)
	require.False(t, ramp.URLsNeedParameters)

	url, err := provider.GetURL(ctx, onramp.Parameters{IsRecurrent: false})
	require.NoError(t, err)
	require.Equal(t, ramp.SiteURL, url)

	_, err = provider.GetURL(ctx, onramp.Parameters{IsRecurrent: true})
	require.Error(t, err)
}
