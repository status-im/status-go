package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/status-im/status-go/cmd/api"
)

// Command contains the result of a parsed command line.
type Command struct {
	FuncName string
	Args     []interface{}
}

// NewCommand parses a command line and returns the command
// for further usage.
func NewCommand(commandLine string) (*Command, error) {
	expr, err := parser.ParseExpr(commandLine)
	if err != nil {
		return nil, err
	}
	switch expr := expr.(type) {
	case *ast.CallExpr:
		return &Command{
			FuncName: (expr.Fun.(*ast.Ident)).Name,
			Args:     exprsToArgs(expr.Args),
		}, nil
	default:
		return nil, fmt.Errorf("invalid command line: %q", commandLine)
	}
}

// exprsToArgs converts the argument expressions to arguments.
func exprsToArgs(exprs []ast.Expr) []interface{} {
	args := make([]interface{}, len(exprs))
	for i, expr := range exprs {
		switch expr := expr.(type) {
		case *ast.BasicLit:
			switch expr.Kind {
			case token.INT:
				args[i], _ = strconv.ParseInt(expr.Value, 10, 64)
			case token.FLOAT:
				args[i], _ = strconv.ParseFloat(expr.Value, 64)
			case token.CHAR:
				args[i] = expr.Value[1]
			case token.STRING:
				args[i] = strings.Trim(expr.Value, `"`)
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

// REPL implements the read-eval-print loop for the commands
// to be sent to statusd.
type REPL struct {
	client  *api.Client
	clientV reflect.Value
}

// NewREPL creates a REPL instance communicating with the
// addressed statusd.
func NewREPL(serverAddress, port string) (*REPL, error) {
	clnt, err := api.NewClient(serverAddress, port)
	if err != nil {
		return nil, err
	}
	clientV := reflect.ValueOf(clnt)
	return &REPL{
		client:  clnt,
		clientV: clientV,
	}, nil
}

// Run operates the loop to read a command and its arguments,
// execute it via the client, and print the result.
func (r *REPL) Run() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		// Read and parse command line.
		fmt.Print(">>> ")
		commandLine, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		cmd, err := NewCommand(commandLine)
		if err != nil {
			return err
		}
		// Switch based on function groups.
		switch {
		case cmd.FuncName == "Quit":
			return nil
		case strings.HasPrefix(cmd.FuncName, "Status"):
			fmt.Printf("perform Status API command: %q %v\n", cmd.FuncName, cmd.Args)
			if err := r.execCommand(cmd); err != nil {
				fmt.Printf("ERR %v\n", err)
			}
		case strings.HasPrefix(cmd.FuncName, "Admin"):
			fmt.Printf("perform administration command: %q %v\n", cmd.FuncName, cmd.Args)
			if err := r.execCommand(cmd); err != nil {
				fmt.Printf("ERR %v\n", err)
			}
		default:
			fmt.Printf("ERR invalid command: %q\n", cmd.FuncName)
		}
	}
}

// execCommand executes the entered command on the client.
func (r *REPL) execCommand(cmd *Command) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid command execution: %v", r)
		}
	}()
	method := r.clientV.MethodByName(cmd.FuncName)
	if !method.IsValid() {
		return fmt.Errorf("command %q not found", cmd.FuncName)
	}
	argsV := make([]reflect.Value, len(cmd.Args))
	for i, arg := range cmd.Args {
		argsV[i] = reflect.ValueOf(arg)
	}
	repliesV := method.Call(argsV)
	for i, replyV := range repliesV {
		fmt.Printf("<<< %d) %v\n", i, replyV)
	}
	return nil
}
