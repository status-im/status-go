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

// AdminGetAddresses retrieves the internet addresses of the
// server.
func (c *Client) AdminGetAddresses() ([]string, error) {
	var args NoArgs
	var reply StringsReply
	err := c.client.Call("Admin.GetAddresses", args, &reply)
	return reply.Strings, err
}

// StatusStartNode loads the configuration out of the passed string and
// starts a node with it.
func (c *Client) StatusStartNode(config string) error {
	args := ConfigArgs{
		Config: config,
	}
	var reply NoReply
	return c.client.Call("Status.StartNode", args, &reply)
}

// StatusStopNode starts the stopped node.
func (c *Client) StatusStopNode() error {
	var args NoArgs
	var reply NoReply
	return c.client.Call("Status.StopNode", args, &reply)
}

// StatusCreateAccount creates an internal geth account.
func (c *Client) StatusCreateAccount(password string) (address, publicKey, mnemonic string, err error) {
	args := AccountArgs{
		Password: password,
	}
	var reply AccountReply
	err = c.client.Call("Status.CreateAccount", args, &reply)
	if err != nil {
		return "", "", "", err
	}
	return reply.Address, reply.PublicKey, reply.Mnemonic, nil
}

// StatusSelectAccount logs in to the given address.
func (c *Client) StatusSelectAccount(address, password string) error {
	args := AccountArgs{
		Address:  address,
		Password: password,
	}
	var reply NoReply
	return c.client.Call("Status.SelectAccount", args, &reply)
}

// StatusLogout logs the current user out.
func (c *Client) StatusLogout() error {
	var args NoArgs
	var reply NoReply
	return c.client.Call("Status.Logout", args, &reply)
}
