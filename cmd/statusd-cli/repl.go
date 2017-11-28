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
	host string
}

// NewREPL creates a REPL instance communicating with the
// addressed statusd.
func NewREPL(host string) *REPL {
	return &REPL{
		host: host,
	}
}

// Run operates the loop to read a command and its arguments,
// execute it via the client, and print the result.
func (r *REPL) Run() error {
	var conn net.Conn
	var reader *bufio.Reader
	var writer *bufio.Writer
	var err error
	input := bufio.NewReader(os.Stdin)
	connect := true
	for {
		// Connect first time and after connection errors.
		if connect {
			conn, err = net.Dial("tcp", r.host)
			if err != nil {
				return fmt.Errorf("error connecting to statusd: %v", err)
			}
			connect = false
			reader = bufio.NewReader(conn)
			writer = bufio.NewWriter(conn)
		}
		// Read command line.
		fmt.Print(">>> ")
		command, err := input.ReadString('\n')
		if err != nil {
			fmt.Printf("ERR %v\n", err)
			continue
		}
		// Check for possible end.
		if strings.ToLower(command) == "quit\n" {
			return nil
		}
		// Execute on statusd.
		_, err = writer.WriteString(command)
		if err != nil {
			fmt.Printf("ERR %v\n", err)
			connect = true
			continue
		}
		err = writer.Flush()
		if err != nil {
			fmt.Printf("ERR %v\n", err)
			connect = true
			continue
		}
		// Read number of expected result lines.
		countStr, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("ERR %v\n", err)
			connect = true
			continue
		}
		count, err := strconv.Atoi(strings.TrimSuffix(countStr, "\n"))
		if err != nil {
			fmt.Printf("ERR %v\n", err)
			continue
		}
		// Read and print result lines.
		for i := 0; i < count; i++ {
			reply, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("ERR %v\n", err)
				continue
			}
			fmt.Print("<<< ")
			fmt.Print(reply)
		}
	}
}
