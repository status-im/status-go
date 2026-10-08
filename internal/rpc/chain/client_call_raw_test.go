package chain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/rpc/chain/ethclient"
)

// CallContractRaw is accounted as eth_CallContract (rpc stats, provider health).
func TestClientWithFallback_CallContractRawLabel(t *testing.T) {
	client, ethClients, cleanup := setupClientTest(t)
	defer cleanup()

	for _, ethClient := range ethClients {
		ethClient.EXPECT().CallContractRaw(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("down")).Times(1)
	}
	_, err := client.CallContractRaw(context.Background(), json.RawMessage(`{}`), nil)
	require.ErrorContains(t, err, "(eth_CallContract)")
}

func TestClientWithFallback_CallContractRawSendsTheSameRequestAsCallContract(t *testing.T) {
	var mu sync.Mutex
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(body))
		mu.Unlock()
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.Unmarshal(body, &req)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x0102"})
	}))
	defer server.Close()

	newClient := func() *ClientWithFallback {
		rpcClient, err := rpc.Dial(server.URL)
		require.NoError(t, err)
		t.Cleanup(rpcClient.Close)
		return NewClient([]ethclient.RPSLimitedEthClientInterface{ethclient.NewRPSLimitedEthClient(rpcClient, nil, "raw_test_circuit", "raw_test_provider")}, 0, nil)
	}

	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	msg := ethereum.CallMsg{From: common.HexToAddress("0x1111111111111111111111111111111111111111"), To: &to, Data: []byte{0xde, 0xad, 0xbe, 0xef}}
	callArg := json.RawMessage(`{"from":"0x1111111111111111111111111111111111111111","input":"0xdeadbeef","to":"0x2222222222222222222222222222222222222222"}`)

	out, err := newClient().CallContract(context.Background(), msg, big.NewInt(7))
	require.NoError(t, err)
	rawOut, err := newClient().CallContractRaw(context.Background(), callArg, big.NewInt(7))
	require.NoError(t, err)

	require.Len(t, bodies, 2)
	require.Equal(t, bodies[0], bodies[1])
	require.Equal(t, out, rawOut)
}
