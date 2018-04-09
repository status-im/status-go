package sdk

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/node"
)

// Conn : TODO ...
type Conn struct {
	statusNode *node.StatusNode
	address    string
	userName   string
}

func New() *Conn {
	return &Conn{}
}

func (c *Conn) Start(config *Config) error {
	statusNode := node.New()
	log.Println("Starting node...")

	err := statusNode.Start(config.NodeConfig)
	if err != nil {
		log.Fatalf("Node start failed: %v", err)
		return err
	}

	c.statusNode = statusNode

	return nil
}

// Connect will attempt to connect to the STATUS messaging system.
// The url can contain username/password semantics. e.g. http://derek:pass@localhost:4222
func Connect(user, password string) (*Conn, error) {
	var err error

	log.Println("Using default config ...")
	config := DefaultConfig()
	if err != nil {
		log.Fatalf("Making config failed: %v", err)
		return nil, err
	}

	c := New()
	c.Start(config)

	return c, c.SignupOrLogin(user, password)
}

// Login logs in to the network with the given credentials
func (c *Conn) Login(user, password string) error {
	am := account.NewManager(c.statusNode)
	w, err := c.statusNode.WhisperService()
	if err != nil {
		return err
	}

	addr := getAccountAddress()
	_, accountKey, err := am.AddressToDecryptedAccount(addr, password)
	if err != nil {
		return err
	}

	log.Println("ADDING PRIVATE KEY :", accountKey.PrivateKey)
	keyID, err := w.AddKeyPair(accountKey.PrivateKey)
	if err != nil {
		return err
	}
	c.address = keyID
	c.userName = user
	log.Println("Logging in as", c.address)

	return am.SelectAccount(addr, password)
}

// Signup creates a new account with the given credentials
func (c *Conn) Signup(user, password string) error {
	am := account.NewManager(c.statusNode)
	address, _, _, err := am.CreateAccount(password)
	if err != nil {
		log.Fatalf("could not create an account. ERR: %v", err)
	}
	saveAccountAddress(address)

	return nil
}

// SignupOrLogin will attempt to login with given credentials, in first instance
// or will sign up in case login does not work
func (c *Conn) SignupOrLogin(user, password string) error {
	if err := c.Login(user, password); err != nil {
		c.Signup(user, password)
		return c.Login(user, password)
	}

	return nil
}

// Join a specific channel by name
func (c *Conn) Join(channelName string) (*Channel, error) {
	return c.joinPublicChannel(channelName)
}

func (c *Conn) joinPublicChannel(channelName string) (*Channel, error) {
	cmd := fmt.Sprintf(generateSymKeyFromPasswordFormat, channelName)
	f := unmarshalJSON(c.statusNode.RPCClient().CallRaw(cmd))

	key := f.(map[string]interface{})["result"].(string)
	id := int(f.(map[string]interface{})["id"].(float64))

	// 	p := "0x68656c6c6f20776f726c64"
	src := []byte(channelName)
	p := "0x" + hex.EncodeToString(src)

	cmd = fmt.Sprintf(web3ShaFormat, p, id)
	f1 := unmarshalJSON(c.statusNode.RPCClient().CallRaw(cmd))
	topic := f1.(map[string]interface{})["result"].(string)
	topic = topic[0:10]

	cmd = fmt.Sprintf(newMessageFilterFormat, topic, key)
	res := c.statusNode.RPCClient().CallRaw(cmd)
	f3 := unmarshalJSON(res)
	filterID := f3.(map[string]interface{})["result"].(string)

	return &Channel{
		conn:        c,
		channelName: channelName,
		filterID:    filterID,
		topic:       topic,
		channelKey:  key,
	}, nil
}
