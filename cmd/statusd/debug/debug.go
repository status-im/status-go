package debug

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"reflect"

	"github.com/status-im/status-go/geth/api"
)

// Debug provides a server receiving line based commands from a
// CLI via the debugging port and executing those on the Status API
// using reflection. The returned values will be rendered as
// string and returned to the CLI.
type Debug struct {
	csv      reflect.Value
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
		csv:      reflect.ValueOf(newCommandSet(statusAPI)),
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
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	defer func() {
		if err != nil {
			log.Printf("error during debug connection: %v", err)
		}
		err = conn.Close()
		if err != nil {
			log.Printf("error whil closing debug connection: %v", err)
		}
	}()
	// Read, execute, and respond commands of a session.
	for {
		command, err := d.readCommandLine(reader)
		if err != nil {
			return
		}
		replies, err := command.execute(d.csv)
		if err != nil {
			return
		}
		err = d.writeRplies(writer, replies)
		if err != nil {
			return
		}
	}
}

// readCommandLine receives a command line via network and
// parses it into an executable command.
func (d *Debug) readCommandLine(reader *bufio.Reader) (*command, error) {
	commandLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return newCommand(commandLine)
}

// writeReplies sends the replies back to the CLI.
func (d *Debug) writeRplies(writer *bufio.Writer, replies []string) error {
	_, err := fmt.Fprintf(writer, "%d\n", len(replies))
	if err != nil {
		return err
	}
	for i, reply := range replies {
		_, err = fmt.Fprintf(writer, "[%d] %s\n", i, reply)
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}
