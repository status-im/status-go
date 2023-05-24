//go:build nimbus_light_client
// +build nimbus_light_client

package rpc

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"net/http"
	"net/http/httptest"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/params"
)

type ProxySuite struct {
	suite.Suite
}

func TestProxySuite(t *testing.T) {
	suite.Run(t, new(ProxySuite))
}

func (s *ProxySuite) startRpcClient(infuraURL string) *Client {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{
			"id": 1,
			"jsonrpc": "2.0",
			"result": "0x234234e22b9ffc2387e18636e0534534a3d0c56b0243567432453264c16e78a2adc"
		}`)
	}))
	defer ts.Close()

	gethRPCClient, err := gethrpc.Dial(ts.URL)
	require.NoError(s.T(), err)

	db, close := setupTestNetworkDB(s.T())
	defer close()
	c, err := NewClient(gethRPCClient, 1, params.UpstreamRPCConfig{Enabled: true, URL: infuraURL}, []params.Network{}, db)
	require.NoError(s.T(), err)

	return c
}

func (s *ProxySuite) TestRun() {
	infuraURL := "https://mainnet.infura.io/v3/800c641949d64d768a5070a1b0511938"
	client := s.startRpcClient(infuraURL)

	// Run light client proxy
	ctx, cancel := context.WithCancel(context.Background())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Before range signals")

	// Invoke eth_getBalance
	var result hexutil.Big
	var addr common.Address
	addr = common.HexToAddress("0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5")
	chainID := uint64(1)

	time.Sleep(200 * time.Second)
	err := client.CallContext(ctx, &result, chainID, "eth_getBalance", addr, "latest")
	require.NoError(s.T(), err)

	client.UnregisterHandler("eth_getBalance")
	var resultRaw hexutil.Big
	err = client.CallContext(ctx, &resultRaw, chainID, "eth_getBalance", addr, "latest")
	s.Require().Equal(result, resultRaw)
	for range signals {
		fmt.Println("Signal caught, exiting")
		cancel()
	}
	fmt.Println("Exiting")

}
