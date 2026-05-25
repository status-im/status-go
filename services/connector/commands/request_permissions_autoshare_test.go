package commands

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	types2 "github.com/status-im/status-go/internal/crypto/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/signal"
)

type overrideRequestShareHandler struct {
	*ClientSideHandler
	shareFn func(dApp signal.ConnectorDApp) (types2.Address, uint64, error)
}

func (o *overrideRequestShareHandler) RequestShareAccountForDApp(dApp signal.ConnectorDApp) (types2.Address, uint64, error) {
	if o.shareFn != nil {
		return o.shareFn(dApp)
	}
	return o.ClientSideHandler.RequestShareAccountForDApp(dApp)
}

func TestRequestPermissions_AutoShareWhenNoDApp(t *testing.T) {
	rejectErr := errors.New("user dismissed share")
	acc := types2.BytesToAddress(types2.FromHex("0x00000000000000000000000000000000000000a1"))

	tests := []struct {
		name             string
		autoURL          string
		clientID         string
		shareFn          func(dApp signal.ConnectorDApp) (types2.Address, uint64, error)
		wantGrantSignals int
		wantDApp         bool
		wantErr          error
	}{
		{
			name:     "accept creates dapp and emits one grant",
			autoURL:  "https://perm-autoshare-no-pending.test",
			clientID: "status-desktop/dapp-browser",
			shareFn: func(dApp signal.ConnectorDApp) (types2.Address, uint64, error) {
				require.Equal(t, "https://perm-autoshare-no-pending.test", dApp.URL)
				return acc, walletCommon.EthereumMainnet, nil
			},
			wantGrantSignals: 1,
			wantDApp:         true,
		},
		{
			name:     "reject propagates error and leaves no dapp",
			autoURL:  "https://perm-autoshare-reject.test",
			clientID: "c1",
			shareFn: func(dApp signal.ConnectorDApp) (types2.Address, uint64, error) {
				return types2.Address{}, 0, rejectErr
			},
			wantGrantSignals: 0,
			wantDApp:         false,
			wantErr:          rejectErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, closeFn := setupCommand(t, Method_RequestPermissions)
			t.Cleanup(closeFn)

			base := NewClientSideHandler(state.db, nil)
			h := &overrideRequestShareHandler{ClientSideHandler: base, shareFn: tt.shareFn}
			cmd := NewRequestPermissionsCommand(state.walletDb, h)

			dAppSignal := signal.ConnectorDApp{
				URL:      tt.autoURL,
				Name:     "n",
				IconURL:  "http://icon",
				ClientID: tt.clientID,
			}
			params := []interface{}{map[string]interface{}{"eth_accounts": map[string]interface{}{}}}
			req, err := ConstructRPCRequest(Method_RequestPermissions, params, &dAppSignal)
			require.NoError(t, err)

			var permissionGranted int
			signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
				var evt EventType
				if err := json.Unmarshal(s, &evt); err != nil {
					return
				}
				if evt.Type == signal.EventConnectorDAppPermissionGranted {
					permissionGranted++
				}
			}))
			t.Cleanup(signal.ResetMobileSignalHandler)

			out, err := cmd.Execute(context.Background(), req)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, "", out)
			} else {
				require.NoError(t, err)
				perms, ok := out.([]persistence.Permission)
				require.True(t, ok)
				require.Len(t, perms, 1)
				require.Equal(t, persistence.NormalizeURL(tt.autoURL), perms[0].Invoker)
				require.Equal(t, "eth_accounts", perms[0].ParentCapability)
			}

			dApp, err := persistence.SelectDApp(state.walletDb, tt.autoURL, tt.clientID)
			require.NoError(t, err)
			if tt.wantDApp {
				require.NotNil(t, dApp)
				require.Equal(t, acc, dApp.SharedAccount)
				require.Equal(t, walletCommon.EthereumMainnet, dApp.ChainID)
				dbPerms, err := persistence.SelectPermissions(state.walletDb, tt.autoURL, tt.clientID)
				require.NoError(t, err)
				require.Len(t, dbPerms, 1)
				require.Equal(t, "eth_accounts", dbPerms[0].ParentCapability)
			} else {
				require.Nil(t, dApp)
			}
			require.Equal(t, tt.wantGrantSignals, permissionGranted)
		})
	}
}
