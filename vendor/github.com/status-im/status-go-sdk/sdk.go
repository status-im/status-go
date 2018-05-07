package sdk

import (
	"encoding/json"
	"log"
)

// RPCClient is a client to manage all rpc calls
type RPCClient interface {
	Call(request interface{}) (response interface{}, err error)
}

// SDK is a set of tools to interact with status node
type SDK struct {
	RPCClient  RPCClient
	accounts   []*Account
	minimumPoW float64
}

// New creates a default SDK object
func New(c RPCClient) *SDK {
	return &SDK{
		RPCClient:  c,
		minimumPoW: 0.001,
	}
}

// Login to status with the given credentials
func (c *SDK) Login(addr, pwd string) (a *Account, err error) {
	res, err := statusLoginRequest(c, addr, pwd)
	if err != nil {
		return a, err
	}
	return &Account{
		Address: res.Result.AddressKeyID,
	}, err
}

// Signup creates a new account with the given credentials
func (c *SDK) Signup(pwd string) (a *Account, err error) {
	res, err := statusSignupRequest(c, pwd)
	if err != nil {
		return a, err
	}
	return &Account{
		Address:  res.Result.Address,
		PubKey:   res.Result.Pubkey,
		Mnemonic: res.Result.Mnemonic,
	}, err

}

// SignupAndLogin sign up and login on status network
func (c *SDK) SignupAndLogin(password string) (a *Account, err error) {
	a, err = c.Signup(password)
	if err != nil {
		return
	}
	la, err := c.Login(a.Address, password)
	a.Address = la.Address
	return
}

// NewMessageFilterResponse NewMessageFilter json response
type NewMessageFilterResponse struct {
	Result string `json:"result"`
}

func (c *SDK) call(cmd string, res interface{}) error {
	log.Println("[ REQUEST ] : " + cmd)
	body, err := c.RPCClient.Call(cmd)
	if err != nil {
		return err
	}
	log.Println("[ RESPONSE ] : " + body.(string))

	return json.Unmarshal([]byte(body.(string)), &res)
}
