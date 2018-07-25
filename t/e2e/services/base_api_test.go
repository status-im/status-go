package services

import (
	"encoding/json"
	"fmt"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/e2e"

	. "github.com/status-im/status-go/t/utils"
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
	var result string

	cmd := fmt.Sprintf(`{"jsonrpc":"2.0", "method": "%s", "params": []}`, method)
	if private {
		result = s.Backend.CallPrivateRPC(cmd)
	} else {
		result = s.Backend.CallRPC(cmd)
	}

	var response struct {
		Error *rpcError `json:"error"`
	}

	s.NoError(json.Unmarshal([]byte(result), &response))

	return !(response.Error != nil && response.Error.Code == methodNotFoundErrorCode)
}

func (s *BaseJSONRPCSuite) SetupTest(upstreamEnabled, statusServiceEnabled, debugAPIEnabled bool) error {
	s.Backend = api.NewStatusBackend()
	s.NotNil(s.Backend)

	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	nodeConfig.IPCEnabled = false
	nodeConfig.StatusServiceEnabled = statusServiceEnabled
	nodeConfig.DebugAPIEnabled = debugAPIEnabled
	if nodeConfig.DebugAPIEnabled {
		nodeConfig.AddAPIModule("debug")
	}
	nodeConfig.HTTPHost = "" // to make sure that no HTTP interface is started

	if upstreamEnabled {
		networkURL, err := GetRemoteURL()
		s.NoError(err)

		nodeConfig.UpstreamConfig.Enabled = true
		nodeConfig.UpstreamConfig.URL = networkURL
	}

	return s.Backend.StartNode(nodeConfig)
}

func (s *BaseJSONRPCSuite) notificationHandler(account string, pass string, expectedError error) func(string) {
	return func(jsonEvent string) {
		envelope := unmarshalEnvelope(jsonEvent)
		if envelope.Type == signal.EventSignRequestAdded {
			event := envelope.Event.(map[string]interface{})
			id := event["id"].(string)
			s.T().Logf("Sign request added (will be completed shortly): {id: %s}\n", id)

			//check for the correct method name
			method := event["method"].(string)
			s.Equal(params.PersonalSignMethodName, method)
			//check the event data
			args := event["args"].(map[string]interface{})
			s.Equal(signDataString, args["data"].(string))
			s.Equal(account, args["account"].(string))

			e := s.Backend.ApproveSignRequest(id, pass).Error
			s.T().Logf("Sign request approved. {id: %s, acc: %s, err: %v}", id, account, e)
			if expectedError == nil {
				s.NoError(e, "cannot complete sign reauest[%v]: %v", id, e)
			} else {
				s.EqualError(e, expectedError.Error())
			}
		}
	}
}

func unmarshalEnvelope(jsonEvent string) signal.Envelope {
	var envelope signal.Envelope
	if e := json.Unmarshal([]byte(jsonEvent), &envelope); e != nil {
		panic(e)
	}
	return envelope
}

func (s *BaseJSONRPCSuite) notificationHandlerSuccess(account string, pass string) func(string) {
	return func(jsonEvent string) {
		s.notificationHandler(account, pass, nil)(jsonEvent)
	}
}
