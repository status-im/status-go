package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"net/url"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
)

const (
	jsonrpcVersion = "2.0"
)

type jsonRequest struct {
	Method  string          `json:"method"`
	Version string          `json:"jsonrpc"`
	ID      int             `json:"id,omitempty"`
	Payload json.RawMessage `json:"params,omitempty"`
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonErrResponse struct {
	Version string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Error   jsonError   `json:"error"`
}

// RPCManager abstract RPC management API (for both client and server)
type RPCManager struct {
	sync.Mutex
	requestID   int
	nodeManager common.NodeManager
}

// errors
var (
	ErrInvalidMethod       = errors.New("method does not exist")
	ErrRPCServerTimeout    = errors.New("RPC server cancelled call due to timeout")
	ErrRPCServerCallFailed = errors.New("RPC server cannot complete request")
)

// NewRPCManager returns new instance of RPC client
func NewRPCManager(nodeManager common.NodeManager) *RPCManager {
	return &RPCManager{
		nodeManager: nodeManager,
	}
}

// Call executes RPC request on node's in-proc RPC server
func (c *RPCManager) Call(inputJSON string) string {
	config, err := c.nodeManager.NodeConfig()
	if err != nil {
		return c.makeJSONErrorResponse(err)
	}

	// allow HTTP requests to block w/o
	outputJSON := make(chan string, 1)

	go func() {
		body := bytes.NewBufferString(inputJSON)

		var err error
		var res []byte

		if config.UpstreamConfig.Enabled {
			log.Info("Making RPC JSON Request to upstream RPCServer")
			res, err = c.callUpstreamStream(config, body)
		} else {
			log.Info("Making RPC JSON Request to internal RPCServer")
			res, err = c.callNodeStream(body)
		}

		if err != nil {
			outputJSON <- c.makeJSONErrorResponse(err)
			return
		}

		outputJSON <- string(res)
		return
	}()

	// wait till call is complete
	select {
	case out := <-outputJSON:
		return out
	case <-time.After((DefaultTxSendCompletionTimeout + 10) * time.Minute): // give up eventually
		// pass
	}

	return c.makeJSONErrorResponse(ErrRPCServerTimeout)
}

// callNodeStream delivers giving request and body content to the external ethereum
// (infura) RPCServer to process the request and returns response.
func (c *RPCManager) callUpstreamStream(config *params.NodeConfig, body io.Reader) ([]byte, error) {
	upstreamURL, err := url.Parse(config.UpstreamConfig.URL)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", upstreamURL.String(), body)
	if err != nil {
		return nil, err
	}

	httpClient := http.Client{
		Timeout: 20 * time.Second,
	}

	res, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if respStatusCode := res.StatusCode; respStatusCode != http.StatusOK {
		log.Error("handler returned wrong status code", "got", respStatusCode, "want", http.StatusOK)
		return nil, ErrRPCServerCallFailed
	}

	return ioutil.ReadAll(res.Body)
}

// callNodeStream delivers giving request and body content to the internal ethereum
// RPCServer to process the request.
func (c *RPCManager) callNodeStream(body io.Reader) ([]byte, error) {
	server, err := c.nodeManager.RPCServer()
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", "/", body)
	if err != nil {
		return nil, err
	}

	rr := httptest.NewRecorder()

	server.ServeHTTP(rr, httpReq)

	// Check the status code is what we expect.
	if respStatus := rr.Code; respStatus != http.StatusOK {
		log.Error("handler returned wrong status code", "got", respStatus, "want", http.StatusOK)
		// outputJSON <- c.makeJSONErrorResponse(ErrRPCServerCallFailed)
		return nil, ErrRPCServerCallFailed
	}

	return rr.Body.Bytes(), nil
}

// makeJSONErrorResponse returns error as JSON response
func (c *RPCManager) makeJSONErrorResponse(err error) string {
	response := jsonErrResponse{
		Version: jsonrpcVersion,
		Error: jsonError{
			Message: err.Error(),
		},
	}

	outBytes, _ := json.Marshal(&response)
	return string(outBytes)
}
