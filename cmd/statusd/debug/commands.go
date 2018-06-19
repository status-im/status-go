package debug

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/params"
)

// command contains the result of a parsed command line and
// is able to execute it on the command set.
type command struct {
	funcName string
	args     []interface{}
}

// newCommand parses a command line and returns the command
// for further usage.
func newCommand(commandLine string) (*command, error) {
	expr, err := parser.ParseExpr(commandLine)
	if err != nil {
		return nil, err
	}
	switch expr := expr.(type) {
	case *ast.CallExpr:
		f, ok := expr.Fun.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("invalid expression: %q", commandLine)
		}
		return &command{
			funcName: f.Name,
			args:     exprsToArgs(expr.Args),
		}, nil
	default:
		return nil, fmt.Errorf("invalid command line: %q", commandLine)
	}
}

// execute calls the method on the passed command set value.
func (c *command) execute(commandSetValue reflect.Value) (replies []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid API call: %v", r)
		}
	}()
	method := commandSetValue.MethodByName(c.funcName)
	if !method.IsValid() {
		return nil, fmt.Errorf("command %q not found", c.funcName)
	}
	argsV := make([]reflect.Value, len(c.args))
	for i, arg := range c.args {
		argsV[i] = reflect.ValueOf(arg)
	}
	repliesV := method.Call(argsV)
	replies = make([]string, len(repliesV))
	for i, replyV := range repliesV {
		replies[i] = fmt.Sprintf("%v", replyV)
	}
	return replies, nil
}

// exprsToArgs converts the argument expressions to arguments.
func exprsToArgs(exprs []ast.Expr) []interface{} {
	args := make([]interface{}, len(exprs))
	for i, expr := range exprs {
		switch expr := expr.(type) {
		case *ast.BasicLit:
			switch expr.Kind {
			case token.INT:
				args[i], _ = strconv.ParseInt(expr.Value, 10, 64) // nolint: gas
			case token.FLOAT:
				args[i], _ = strconv.ParseFloat(expr.Value, 64) // nolint: gas
			case token.CHAR:
				args[i] = expr.Value[1]
			case token.STRING:
				r := strings.NewReplacer("\\n", "\n", "\\t", "\t", "\\\"", "\"")
				args[i] = strings.Trim(r.Replace(expr.Value), `"`)
			}
		case *ast.Ident:
			switch expr.Name {
			case "true":
				args[i] = true
			case "false":
				args[i] = false
			default:
				args[i] = expr.Name
			}
		default:
			args[i] = fmt.Sprintf("[unknown: %#v]", expr)
		}
	}
	return args
}

// commandSet implements the set of commands the debugger unterstands.
// In the beginning a subset of the Status API, may later grow to
// utility commands.
//
// Direct invocation of commands on the Status API sometimes sadly
// is not possible due to non-low-level arguments. Here this wrapper
// helps.
type commandSet struct {
	statusBackend *api.StatusBackend
}

// newCommandSet creates the command set for the passed Status API
// instance.
func newCommandSet(statusBackend *api.StatusBackend) *commandSet {
	return &commandSet{
		statusBackend: statusBackend,
	}
}

// StartNode loads the configuration out of the passed string and
// starts a node with it.
func (cs *commandSet) StartNode(config string) error {
	nodeConfig, err := params.LoadNodeConfig(config)
	if err != nil {
		return err
	}
	return cs.statusBackend.StartNode(nodeConfig)
}

// StopNode starts the stopped node.
func (cs *commandSet) StopNode() error {
	return cs.statusBackend.StopNode()
}

// ResetChainData removes chain data from data directory.
func (cs *commandSet) ResetChainData() error {
	return cs.statusBackend.ResetChainData()
}

// CallRPC calls status node via RPC.
func (cs *commandSet) CallRPC(inputJSON string) string {
	return cs.statusBackend.CallRPC(inputJSON)
}

// CreateAccount creates an internal geth account.
func (cs *commandSet) CreateAccount(password string) (string, string, string, error) {
	return cs.statusBackend.AccountManager().CreateAccount(password)
}

// CreateChildAccount creates a sub-account.
func (cs *commandSet) CreateChildAccount(parentAddress, password string) (string, string, error) {
	return cs.statusBackend.AccountManager().CreateChildAccount(parentAddress, password)
}

// RecoverAccount re-creates the master key using the given details.
func (cs *commandSet) RecoverAccount(password, mnemonic string) (string, string, error) {
	return cs.statusBackend.AccountManager().RecoverAccount(password, mnemonic)
}

// SelectAccount selects the addressed account.
func (cs *commandSet) SelectAccount(address, password string) error {
	return cs.statusBackend.SelectAccount(address, password)
}

// Logout clears the Whisper identities.
func (cs *commandSet) Logout() error {
	return cs.statusBackend.Logout()
}

// ApproveSignRequest instructs API to complete sending of a given transaction.
func (cs *commandSet) ApproveSignRequest(id, password string) (string, error) {
	result := cs.statusBackend.ApproveSignRequest(id, password)
	if result.Error != nil {
		return "", result.Error
	}
	return result.Response.Hex(), nil
}

// ApproveSignRequest instructs API to complete sending of a given transaction.
// gas and gasPrice will be overrided with the given values before signing the
// transaction.
func (cs *commandSet) ApproveSignRequestWithArgs(id, password string, gas, gasPrice int64) (string, error) {
	result := cs.statusBackend.ApproveSignRequestWithArgs(id, password, gas, gasPrice)
	if result.Error != nil {
		return "", result.Error
	}
	return result.Response.Hex(), nil
}
