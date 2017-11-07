package api

import (
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
)

// NoArgs has to be used when a remote function needs
// no concrete argument for its work.
type NoArgs bool

// NoReply has to be used when a remote function provides
// no concrete reply beside a potential error.
type NoReply bool

// ConfigArgs is used to pass a configuration as argument.
type ConfigArgs struct {
	Config string
}

// LoginArgs is used to pass the combination of address and
// password as argument.
type LoginArgs struct {
	Address  string
	Password string
}

// API exposes the functions of the StatusAPI via JSON-RPC.
type API struct {
	statusAPI *api.StatusAPI
}

// NewAPI creates an instance of the API to expose.
func NewAPI() *API {
	return &API{
		statusAPI: api.NewStatusAPI(),
	}
}

// StartNode loads the configuration out of the passed string and
// starts a node with it.
func (a *API) StartNode(args *ConfigArgs, reply *NoReply) error {
	config, err := params.LoadNodeConfig(args.Config)
	if err != nil {
		return err
	}

	_, err = a.statusAPI.StartNodeAsync(config)
	return err
}

// StopNode starts the stopped node.
func (a *API) StopNode(args *NoArgs, reply *NoReply) error {
	_, err := a.statusAPI.StopNodeAsync()
	return err
}

// Login loads the key file for the given address, tries to decrypt it
// using the password to verify the ownership. If verified it purges all
// previous identities from Whisper and injects verified key as ssh
// identity.
func (a *API) Login(args *LoginArgs, reply *NoReply) error {
	return a.statusAPI.SelectAccount(args.Address, args.Password)
}

// Logout clears the Whisper identities.
func (a *API) Logout(args *NoArgs, reply *NoReply) error {
	return a.statusAPI.Logout()
}
