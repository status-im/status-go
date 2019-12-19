// +build !nimbus

package account

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/status-im/status-go/account/generator"
)

// GethManager represents account manager interface.
type GethManager struct {
	Manager

	manager *accounts.Manager
}

// NewManager returns new node account manager.
func NewManager() *GethManager {
	m := &GethManager{}
	m.accountsGenerator = generator.New(m)
	return m
}

// InitKeystore sets key manager and key store.
func (m *GethManager) InitKeystore(keydir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error
	m.manager, err = makeAccountManager(keydir)
	if err != nil {
		return err
	}
	m.keystore, err = makeKeyStore(m.manager)
	return err
}

func (m *GethManager) GetManager() *accounts.Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.manager
}
