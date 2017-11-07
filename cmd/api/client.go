package api

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// Client can be used for communication with the API, e.g. by the CLI.
type Client struct {
	serverAddress string
	port          string
	conn          net.Conn
	client        *rpc.Client
}

// NewClient establishes a connection to the given server.
func NewClient(serverAddress, port string) (*Client, error) {
	c := &Client{
		serverAddress: serverAddress,
		port:          port,
	}
	conn, err := net.Dial("tcp", serverAddress+":"+port)
	if err != nil {
		return nil, fmt.Errorf("cannot establish connection to '%s:%s': %v", serverAddress, port, err)
	}
	c.conn = conn
	c.client = jsonrpc.NewClient(conn)
	return c, nil
}

// APIStartNode loads the configuration out of the passed string and
// starts a node with it.
func (c *Client) APIStartNode(config string) error {
	args := ConfigArgs{
		Config: config,
	}
	var reply NoReply
	return c.client.Call("API.StartNode", args, &reply)
}

// APIStopNode starts the stopped node.
func (c *Client) APIStopNode() error {
	var args NoArgs
	var reply NoReply
	return c.client.Call("API.StopNode", args, &reply)
}
