package relay_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/relay"
)

func TestFetchChains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/chains", r.URL.Path)
		_, err := w.Write([]byte(`{"chains":[
			{"id": 8453, "name": "base", "displayName": "Base", "depositEnabled": true, "disabled": false},
			{"id": 999, "name": "paused", "displayName": "Paused", "depositEnabled": false, "disabled": true}
		]}`))
		require.NoError(t, err)
	}))
	defer srv.Close()

	client := relay.NewClientWithBaseURL(srv.URL, walletCommon.BaseMainnet, relay.ReferrerDev, "")

	chains, err := client.FetchChains(context.Background())
	require.NoError(t, err)
	require.Len(t, chains, 2)
	require.Equal(t, uint64(8453), chains[0].ID)
	require.Equal(t, "Base", chains[0].DisplayName)
	require.True(t, chains[0].DepositEnabled)
	require.False(t, chains[0].Disabled)
	require.True(t, chains[1].Disabled)
}
