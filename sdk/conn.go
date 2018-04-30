package sdk

import (
	"fmt"

	"github.com/valyala/gorpc"
)

// Conn : TODO ...
type Conn struct {
	rpc        *gorpc.Client
	address    string
	userName   string
	channels   []*Channel
	minimumPoW string
}

func New(address string) *Conn {
	rpc := &gorpc.Client{
		Addr: address, // "rpc.server.addr:12345",
	}
	rpc.Start()

	return &Conn{
		rpc:        rpc,
		minimumPoW: "0.01",
	}
}

func (c *Conn) Close() {
	for _, channel := range c.channels {
		channel.Close()
	}
}

// Login logs in to the network with the given credentials
func (c *Conn) Login(addr, pwd string) error {
	cmd := fmt.Sprintf(statusLoginFormat, addr, pwd)
	res, err := c.rpc.Call(cmd)
	if err != nil {
		return err
	}
	// TODO(adriacidre) unmarshall and treat the response
	println(res)

	return nil
}

// Signup creates a new account with the given credentials
func (c *Conn) Signup(pwd string) error {
	cmd := fmt.Sprintf(statusSignupFormat, pwd)
	res, err := c.rpc.Call(cmd)
	if err != nil {
		return err
	}
	// TODO(adriacidre) unmarshall and treat the response
	println(res)

	return nil
}

// SignupOrLogin will attempt to login with given credentials, in first instance
// or will sign up in case login does not work
func (c *Conn) SignupOrLogin(user, password string) error {
	if err := c.Login(user, password); err != nil {
		c.Signup(password)
		return c.Login(user, password)
	}

	return nil
}

// Join a specific channel by name
func (c *Conn) Join(channelName string) (*Channel, error) {
	ch, err := c.joinPublicChannel(channelName)
	if err != nil {
		c.channels = append(c.channels, ch)
	}

	return ch, err
}

func (c *Conn) joinPublicChannel(channelName string) (*Channel, error) {
	cmd := fmt.Sprintf(statusJoinPublicChannel, channelName)
	res, err := c.rpc.Call(cmd)
	if err != nil {
		return nil, err
	}
	// TODO(adriacidre) unmarshall and treat the response
	println(res)

	return &Channel{
		conn: c,
		/*
			channelName: channelName,
			filterID:    filterID,
			topic:       topic,
			channelKey:  key,
		*/
	}, nil
}
