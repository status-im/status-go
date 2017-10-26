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

// RPCClientProvider is an interface that provides a way
// to obtain an rpc.Client.
type RPCClientProvider interface {
	RPCClient() *rpc.Client
}

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
		cell.Stop() //nolint: errcheck
	}

	// TODO(tiabc): Move this initialisation to a proper place.
	j.cells = make(map[string]*Cell)
}

// createCell creates a new cell if it does not exists.
func (j *Jail) createCell(chatID string) (*Cell, error) {
	j.cellsMx.Lock()
	defer j.cellsMx.Unlock()

	if cell, ok := j.cells[chatID]; ok {
		return cell, fmt.Errorf("cell with id '%s' already exists", chatID)
	}

	cell, err := NewCell(chatID)
	if err != nil {
		return nil, err
	}

	j.cells[chatID] = cell

	return cell, nil
}

// CreateCell creates a new cell. It returns an error
// if a cell with a given ID already exists.
func (j *Jail) CreateCell(chatID string) (common.JailCell, error) {
	return j.createCell(chatID)
}

// initCell initializes a cell with default JavaScript handlers and user code.
func (j *Jail) initCell(cell *Cell) error {
	// Register objects being a bridge between Go and JavaScript.
	if err := registerWeb3Provider(j, cell); err != nil {
		return err
	}

	if err := registerStatusSignals(cell); err != nil {
		return err
	}

	// Run some initial JS code to provide some global objects.
	c := []string{
		j.baseJS,
		web3Code,
		web3InstanceCode,
	}

	_, err := cell.Run(strings.Join(c, ";"))
	return err
}

// CreateAndInitCell creates and initializes a new Cell.
func (j *Jail) createAndInitCell(chatID string, code ...string) (*Cell, error) {
	cell, err := j.createCell(chatID)
	if err != nil {
		return nil, err
	}

	if err := j.initCell(cell); err != nil {
		return nil, err
	}

	// Run custom user code
	for _, js := range code {
		_, err := cell.Run(js)
		if err != nil {
			return nil, err
		}
	}

	return cell, nil
}

// CreateAndInitCell creates and initializes new Cell. Additionally,
// it creates a `catalog` variable in the VM.
// It returns the response as a JSON string.
func (j *Jail) CreateAndInitCell(chatID string, code ...string) string {
	cell, err := j.createAndInitCell(chatID, code...)
	if err != nil {
		return newJailErrorResponse(err)
	}

	return j.makeCatalogVariable(cell)
}

// makeCatalogVariable provides `catalog` as a global variable.
// TODO(divan): this can and should be implemented outside of jail,
// on a clojure side. Moving this into separate method to nuke it later
// easier.
func (j *Jail) makeCatalogVariable(cell *Cell) string {
	_, err := cell.Run(`var catalog = JSON.stringify(_status_catalog)`)
	if err != nil {
		return newJailErrorResponse(err)
	}

	value, err := cell.Get("catalog")
	if err != nil {
		return newJailErrorResponse(err)
	}

	return newJailResultResponse(value)
}

func (j *Jail) cell(chatID string) (*Cell, error) {
	j.cellsMx.RLock()
	defer j.cellsMx.RUnlock()

	cell, ok := j.cells[chatID]
	if !ok {
		return nil, fmt.Errorf("cell '%s' not found", chatID)
	}

	return cell, nil
}

// Cell returns a cell by chatID. If it does not exist, error is returned.
// Required by the Backend.
func (j *Jail) Cell(chatID string) (common.JailCell, error) {
	return j.cell(chatID)
}

// Call executes the `call` function within a cell with chatID.
// Returns a string being a valid JS code. In case of a successful result,
// it's {"result": any}. In case of an error: {"error": "some error"}.
//
// Call calls commands from `_status_catalog`.
// commandPath is an array of properties to retrieve a function.
// For instance:
//   `["prop1", "prop2"]` is translated to `_status_catalog["prop1"]["prop2"]`.
func (j *Jail) Call(chatID, commandPath, args string) string {
	cell, err := j.cell(chatID)
	if err != nil {
		return newJailErrorResponse(err)
	}

	value, err := cell.Call("call", nil, commandPath, args)
	if err != nil {
		return newJailErrorResponse(err)
	}

	return newJailResultResponse(value)
}

// GetRPCClient returns an rpc.Client.
func (j *Jail) GetRPCClient() *rpc.Client {
	if j.rpcClientProvider == nil {
		return nil
	}

	return j.rpcClientProvider.RPCClient()
}

// sendRPCCall executes a raw JSON-RPC request.
func (j *Jail) sendRPCCall(request string) (interface{}, error) {
	client := j.GetRPCClient()
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
// Marshaling is not required as result.String() produces a string
// that is a valid JavaScript code.
func newJailResultResponse(result otto.Value) string {
	return `{"result": ` + result.String() + `}`
}
