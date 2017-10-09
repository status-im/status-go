package jail

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/static"
)

const (
	web3InstanceCode = `
		var Web3 = require('web3');
		var web3 = new Web3(jeth);
		var Bignumber = require("bignumber.js");
		function bn(val) {
			return new Bignumber(val);
		}
	`
)

var (
	web3Code = string(static.MustAsset("scripts/web3.js"))
	// ErrNoRPCClient is returned when an RPC client is required but it's nil.
	ErrNoRPCClient = errors.New("RPC client is not available")
)

// RPCClientProvider is a function that provides an rpc.Client.
type RPCClientProvider func() *rpc.Client

// Jail manages multiple JavaScript execution contexts (JavaScript VMs) called cells.
// Each cell is a separate VM with web3.js set up.
//
// As rpc.Client might not be available during Jail initialization,
// a provider function is used.
type Jail struct {
	rpcClientProvider RPCClientProvider
	baseJS            string
	cellsMx           sync.RWMutex
	cells             map[string]*Cell
}

// New returns a new Jail.
func New(provider RPCClientProvider) *Jail {
	return NewWithBaseJS(provider, "")
}

// NewWithBaseJS returns a new Jail with base JS configured.
func NewWithBaseJS(provider RPCClientProvider, code string) *Jail {
	if provider == nil {
		provider = func() *rpc.Client {
			return nil
		}
	}

	return &Jail{
		rpcClientProvider: provider,
		baseJS:            code,
		cells:             make(map[string]*Cell),
	}
}

// SetBaseJS sets initial JavaScript code loaded to each new cell.
func (j *Jail) SetBaseJS(js string) {
	j.baseJS = js
}

// Stop stops jail and all assosiacted cells.
func (j *Jail) Stop() {
	j.cellsMx.Lock()
	defer j.cellsMx.Unlock()

	for _, cell := range j.cells {
		cell.Stop()
	}

	// TODO(tiabc): Move this initialisation to a proper place.
	j.cells = make(map[string]*Cell)
}

// CreateCell creates a new cell if it does not exists. Otherwise, it returns an error.
func (j *Jail) CreateCell(chatID string) (*Cell, error) {
	j.cellsMx.Lock()
	defer j.cellsMx.Unlock()

	if cell, ok := j.cells[chatID]; ok {
		return cell, fmt.Errorf("cell with id '%s' already exists", chatID)
	}

	cell := NewCell(chatID)
	j.cells[chatID] = cell

	return cell, nil
}

// InitCell initializes a cell with JavaScript code. In case of a successful result,
// it returns {"result": any}, otherwise an error: {"error": "some error"}.
func (j *Jail) InitCell(chatID, code string) string {
	cell, err := j.getCell(chatID)
	if err != nil {
		return newJailErrorResponse(err)
	}

	// Register objects being a bridge between Go and JavaScript.
	err = registerWeb3Provider(j, cell)
	if err != nil {
		return newJailErrorResponse(err)
	}
	err = registerStatusSignals(j, cell)
	if err != nil {
		return newJailErrorResponse(err)
	}

	// Run some initial JS code to provide some global objects.
	c := []string{
		j.baseJS,
		web3Code,
		web3InstanceCode,
		code,
	}

	_, err = cell.Run(strings.Join(c, ";"))
	if err != nil {
		return newJailErrorResponse(err)
	}

	// Provide `catalog` as a global variable.
	// TODO(adam): does it need to be a global var as _status_catalog
	// already is?
	_, err = cell.Run(`var catalog = JSON.stringify(_status_catalog)`)
	if err != nil {
		return newJailErrorResponse(err)
	}

	value, err := cell.Get("catalog")
	if err != nil {
		return newJailErrorResponse(err)
	}

	return newJailResultResponse(value, err)
}

// CreateAndInitCell performs CreateCell and InitCell methods
// and returns a string.
// TODO(adam): fix API so that this becomes obsolete.
func (j *Jail) CreateAndInitCell(chatID, code string) string {
	_, err := j.CreateCell(chatID)
	if err != nil {
		return newJailErrorResponse(err)
	}

	return j.InitCell(chatID, code)
}

func (j *Jail) getCell(chatID string) (*Cell, error) {
	j.cellsMx.RLock()
	defer j.cellsMx.RUnlock()

	cell, ok := j.cells[chatID]
	if !ok {
		return nil, fmt.Errorf("cell '%s' not found", chatID)
	}

	return cell, nil
}

// GetCell returns a cell by chatID. If it does not exist, error is returned.
// Required by the Backend.
func (j *Jail) GetCell(chatID string) (common.JailCell, error) {
	return j.getCell(chatID)
}

// Call executes the `call` function within a cell with chatID.
// Returns a string being a valid JS code. In case of a successful result,
// it's {"result": any}. In case of an error: {"error": "some error"}.
//
// Call calls commands from `_status_catalog`.
// commandPath is an array of properties to retrieve a function.
// For instance:
//   `["prop1", "prop2"]` is translated to `_status_catalog["prop1"["prop2"]`.
func (j *Jail) Call(chatID, commandPath, args string) string {
	cell, err := j.getCell(chatID)
	if err != nil {
		return newJailErrorResponse(err)
	}

	value, err := cell.Call("call", nil, commandPath, args)
	if err != nil {
		return newJailErrorResponse(err)
	}

	return newJailResultResponse(value, err)
}

// sendRPCCall executes a raw JSON-RPC request.
func (j *Jail) sendRPCCall(cell *Cell, request string) (interface{}, error) {
	client := j.rpcClientProvider()
	if client == nil {
		return nil, ErrNoRPCClient
	}

	rawResponse := client.CallRaw(request)

	var response interface{}
	if err := json.Unmarshal([]byte(rawResponse), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %s", err)
	}

	return response, nil
}

// newJailErrorResponse returns an error.
func newJailErrorResponse(err error) string {
	response := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}

	rawResponse, err := json.Marshal(response)
	if err != nil {
		return `{"error": "` + err.Error() + `"}`
	}

	return string(rawResponse)
}

// newJailResultResponse returns a string that is a valid JavaScript code.
// Marshaling is not required as result.String() produces an string
// that is a valid JavaScript code.
func newJailResultResponse(result otto.Value, err error) string {
	if err != nil {
		return newJailErrorResponse(err)
	}

	return `{"result": ` + result.String() + `}`
}
