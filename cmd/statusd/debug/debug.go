package debug

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/status-im/status-go/geth/api"
)

// command contains the result of a parsed command line.
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
		return &command{
			funcName: (expr.Fun.(*ast.Ident)).Name,
			args:     exprsToArgs(expr.Args),
		}, nil
	default:
		return nil, fmt.Errorf("invalid command line: %q", commandLine)
	}
}

// exeecute calls the method on the passed Status API value.
func (c *command) execute(apiValue reflect.Value) (replies []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid API call: %v", r)
		}
	}()
	method := apiValue.MethodByName(c.funcName)
	if !method.IsValid() {
		return nil, fmt.Errorf("API function %q not found", c.funcName)
	}
	argsV := make([]reflect.Value, len(c.args))
	for i, arg := range c.args {
		argsV[i] = reflect.ValueOf(arg)
	}
	repliesV := method.Call(argsV)

	replies = make([]string, len(repliesV))
	for i, replyV := range repliesV {
		replies[i] = fmt.Sprintf("%#v", replyV)
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

// Debug provides a server receiving line based commands from a
// CLI via the debugging port and executing those on the Status API
// using reflection. The returned values will be rendered as
// string and returned to the CLI.
type Debug struct {
	apiValue reflect.Value
	listener net.Listener
}

// New creates a debugger using the oassed Status API.
// It also starts the server.
func New(statusAPI *api.StatusAPI) (*Debug, error) {
	listener, err := net.Listen("tcp", ":51515")
	if err != nil {
		return nil, err
	}
	d := &Debug{
		apiValue: reflect.ValueOf(statusAPI),
		listener: listener,
	}
	go d.backend()
	return d, nil
}

// backend receives the commands and executes them on
// the Status API.
func (d *Debug) backend() {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			log.Printf("cannot establish debug connection: %v", err)
			continue
		}
		go d.handleConnection(conn)
	}
}

// handleConnection handles all commands of one connection.
func (d *Debug) handleConnection(conn net.Conn) {
	var err error
	buf := bufio.NewReadWriter(
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
	)
	defer func() {
		if err != nil {
			log.Printf("error during debug connection: %v", err)
		}
		err = buf.Flush()
		if err != nil {
			log.Printf("error while flushing debug connection: %v", err)
		}
		err = conn.Close()
		if err != nil {
			log.Printf("error whil closing debug connection: %v", err)
		}
	}()
	// Read, execute, and respond commands of a session.
	for {
		command, err := d.readCommandLine(buf)
		if err != nil {
			return
		}
		replies, err := command.execute(d.apiValue)
		if err != nil {
			return
		}
		err = d.writeRplies(buf, replies)
		if err != nil {
			return
		}
	}
}

// readCommandLine receives a command line via network and
// parses it into an executable command.
func (d *Debug) readCommandLine(buf *bufio.ReadWriter) (*command, error) {
	commandLine, err := buf.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return newCommand(commandLine)
}

// writeReplies sends the replies back to the CLI.
func (d *Debug) writeRplies(buf *bufio.ReadWriter, replies []string) error {
	_, err := fmt.Fprintf(buf, "%d\n", len(replies))
	if err != nil {
		return err
	}
	for i, reply := range replies {
		_, err = fmt.Fprintf(buf, "[%d] %s\n", i, reply)
		if err != nil {
			return err
		}
	}
	return nil
}
