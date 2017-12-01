package debug

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"reflect"

	"github.com/status-im/status-go/geth/api"
)

const (
	// CLIPort is the CLI port.
	CLIPort = "51515"
)

// Server provides a debug server receiving line based commands from
// a CLI via the debugging port and executing those on the Status API
// using reflection. The returned values will be rendered as
// string and returned to the CLI.
type Server struct {
	commandSetValue reflect.Value
	listener        net.Listener
}

// New creates a debug server using the passed Status API.
// It also starts the server.
func New(statusAPI *api.StatusAPI, port string) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port)) // nolint
	if err != nil {
		return nil, err
	}
	s := Server{
		commandSetValue: reflect.ValueOf(newCommandSet(statusAPI)),
		listener:        listener,
	}
	go s.backend()
	return &s, nil
}

// backend receives the commands and executes them on
// the Status API.
func (s *Server) backend() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("cannot establish debug connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

// handleConnection handles all commands of one connection.
func (s *Server) handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("error while closing debug connection: %v", err)
		}
	}()
	// Read, execute, and respond commands of a session.
	for {
		var (
			replies []string
			err     error
		)
		command, err := s.readCommandLine(reader)
		if err != nil {
			replies = []string{fmt.Sprintf("cannot read command: %v", err)}
		} else {
			replies, err = command.execute(s.commandSetValue)
			if err != nil {
				replies = []string{fmt.Sprintf("cannot execute command: %v", err)}
			}
		}
		err = s.writeReplies(writer, replies)
		if err != nil {
			log.Printf("cannot write replies: %v", err)
			return
		}
	}
}

// readCommandLine receives a command line via network and
// parses it into an executable command.
func (s *Server) readCommandLine(reader *bufio.Reader) (*command, error) {
	commandLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return newCommand(commandLine)
}

// writeReplies sends the replies back to the CLI.
func (s *Server) writeReplies(writer *bufio.Writer, replies []string) error {
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
