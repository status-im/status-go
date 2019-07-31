package services

import (
	"encoding/json"
	"fmt"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/t/e2e"

	"github.com/status-im/status-go/t/utils"
)

const (
	// see vendor/github.com/ethereum/go-ethereum/rpc/errors.go:L27
	methodNotFoundErrorCode = -32601
)

type rpcError struct {
	Code int `json:"code"`
}

type BaseJSONRPCSuite struct {
	e2e.BackendTestSuite
}

func (s *BaseJSONRPCSuite) AssertAPIMethodUnexported(method string) {
	exported := s.isMethodExported(method, false)
	s.False(exported,
		"method %s should be hidden, but it isn't",
		method)
}

func (s *BaseJSONRPCSuite) AssertAPIMethodExported(method string) {
	exported := s.isMethodExported(method, false)
	s.True(exported,
		"method %s should be exported, but it isn't",
		method)
}

func (s *BaseJSONRPCSuite) AssertAPIMethodExportedPrivately(method string) {
	exported := s.isMethodExported(method, true)
	s.True(exported,
		"method %s should be exported, but it isn't",
		method)
}

func (s *BaseJSONRPCSuite) isMethodExported(method string, private bool) bool {
	var (
		result string
		err    error
	)

	cmd := fmt.Sprintf(`{"jsonrpc":"2.0", "method": "%s", "params": []}`, method)
	if private {
		result, err = s.Backend.CallPrivateRPC(cmd)
	} else {
		result, err = s.Backend.CallRPC(cmd)
	}
	s.NoError(err)

	var response struct {
		Error *rpcError `json:"error"`
	}

	s.NoError(json.Unmarshal([]byte(result), &response))

	return !(response.Error != nil && response.Error.Code == methodNotFoundErrorCode)
}

func (s *BaseJSONRPCSuite) SetupTest(upstreamEnabled, statusServiceEnabled, debugAPIEnabled bool) error {
	s.Backend = api.NewStatusBackend()
	s.NotNil(s.Backend)

	nodeConfig, err := utils.MakeTestNodeConfig(utils.GetNetworkID())
	s.NoError(err)
	s.NoError(s.Backend.AccountManager().InitKeystore(nodeConfig.KeyStoreDir))

	nodeConfig.IPCEnabled = false
	nodeConfig.EnableStatusService = statusServiceEnabled
	if debugAPIEnabled {
		nodeConfig.AddAPIModule("debug")
	}
	nodeConfig.HTTPHost = "" // to make sure that no HTTP interface is started

	if upstreamEnabled {
		networkURL, err := utils.GetRemoteURL()
		s.NoError(err)

		nodeConfig.UpstreamConfig.Enabled = true
		nodeConfig.UpstreamConfig.URL = networkURL
	}

	return s.Backend.StartNode(nodeConfig)
}
