package node

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/common"
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
	server, err := c.nodeManager.RPCServer()
	if err != nil {
		return c.makeJSONErrorResponse(err)
	}

	// allow HTTP requests to block w/o
	outputJSON := make(chan string, 1)
	go func() {
		defer common.HaltOnPanic()
		httpReq := httptest.NewRequest("POST", "/", strings.NewReader(inputJSON))
		rr := httptest.NewRecorder()
		server.ServeHTTP(rr, httpReq)

		// Check the status code is what we expect.
		if respStatus := rr.Code; respStatus != http.StatusOK {
			log.Error("handler returned wrong status code", "got", respStatus, "want", http.StatusOK)
			outputJSON <- c.makeJSONErrorResponse(ErrRPCServerCallFailed)
			return
		}

		// everything is ok, return
		outputJSON <- rr.Body.String()
	}()

	// wait till call is complete
	select {
	case out := <-outputJSON:
		return out
	case <-time.After((status.DefaultTxSendCompletionTimeout + 10) * time.Minute): // give up eventually
		// pass
	}

	return c.makeJSONErrorResponse(ErrRPCServerTimeout)
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
