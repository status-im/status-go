package account

import (
	"context"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/rpc"
)

// AccountsRPCHandler returns RPC Handler for the Accounts() method.
func (m *Manager) AccountsRPCHandler() rpc.Handler {
	return func(context.Context, ...interface{}) (interface{}, error) {
		return m.Accounts()
	}
}

func newAccountNodeManager(node common.NodeManager) *accountNodeManager {
	return &accountNodeManager{node: node}
}

type accountNodeManager struct {
	node common.NodeManager
}

func (a *accountNodeManager) AccountKeyStore() (accountKeyStorer, error) {
	return a.node.AccountKeyStore()
}
func (a *accountNodeManager) AccountManager() (gethAccountManager, error) {
	return a.node.AccountManager()
}
func (a *accountNodeManager) WhisperService() (whisperService, error) {
	return a.node.WhisperService()
}
