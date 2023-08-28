//go:build nimbus_light_client
// +build nimbus_light_client

package rpc

import (
	"context"
	"fmt"
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

func (s *ProxySuite) startRpcClient(infuraURL string) (*Client, func()) {
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
	c, err := NewClient(gethRPCClient, 1, params.UpstreamRPCConfig{Enabled: true, URL: infuraURL}, []params.Network{}, true, db)
	require.NoError(s.T(), err)

	return c, close
}

func (s *ProxySuite) TestRun() {
	infuraURL := "https://mainnet.infura.io/v3/800c641949d64d768a5070a1b0511938"
	client, closeDb := s.startRpcClient(infuraURL)

	defer closeDb()

	fmt.Println("Before waitForProxyHeaders")
	ctxTimeout, _ := context.WithTimeout(context.Background(), 600*time.Second)
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctxTimeout.Done():
			s.Require().Fail("Timeout reached")
		case <-ticker.C:
			// Let's check if handlers have been installed
			_, found := client.handler("eth_getBalance")
			if found {
				fmt.Println("Proceed")
				ticker.Stop()
				goto proceed
			}
		}
	}

proceed:
	fmt.Println("after waitForProxyHeaders")

	// Invoke eth_getBalance
	var result hexutil.Big
	var addr common.Address
	addr = common.HexToAddress("0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5")
	chainID := uint64(1)

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := client.CallContext(ctx, &result, chainID, "eth_getBalance", addr, "latest")
	s.Require().NoError(err)

	client.UnregisterHandler("eth_getBalance")
	var resultRaw hexutil.Big
	ctx1, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.CallContext(ctx1, &resultRaw, chainID, "eth_getBalance", addr, "latest")
	s.Require().Equal(result, resultRaw)

}
