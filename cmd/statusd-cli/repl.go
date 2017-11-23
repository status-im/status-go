package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// REPL implements the read-eval-print loop for the commands
// to be sent to statusd.
type REPL struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

// NewREPL creates a REPL instance communicating with the
// addressed statusd.
func NewREPL(serverAddress string) (*REPL, error) {
	conn, err := net.Dial("tcp", serverAddress+":51515")
	if err != nil {
		return nil, fmt.Errorf("error when starting REPL: %v", err)
	}
	return &REPL{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, nil
}

// Run operates the loop to read a command and its arguments,
// execute it via the client, and print the result.
func (r *REPL) Run() error {
	defer func() {
		r.conn.Close() //nolint: errcheck
	}()
	input := bufio.NewReader(os.Stdin)
	for {
		// Read command line.
		fmt.Print(">>> ")
		command, err := input.ReadString('\n')
		if err != nil {
			return err
		}
		// Check for possible end.
		if strings.ToLower(command) == "quit" {
			return nil
		}
		// Execute on statusd.
		_, err = r.writer.WriteString(command + "\n")
		if err != nil {
			return err
		}
		err = r.writer.Flush()
		if err != nil {
			return err
		}
		// Print result.
		countStr, err := r.reader.ReadString('\n')
		if err != nil {
			return err
		}
		count, err := strconv.Atoi(strings.TrimSuffix(countStr, "\n"))
		if err != nil {
			return err
		}
		for i := 0; i < count; i++ {
			reply, err := r.reader.ReadString('\n')
			if err != nil {
				return err
			}
			fmt.Print("<<< ")
			fmt.Print(reply)
		}
	}
}
