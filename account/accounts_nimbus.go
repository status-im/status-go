// +build nimbus

package account

import (
	"github.com/status-im/status-go/account/generator"
)

// NewManager returns new node account manager.
func NewManager() *Manager {
	m := &Manager{}
	m.accountsGenerator = generator.New(m)
	return m
}

// InitKeystore sets key manager and key store.
func (m *Manager) InitKeystore(keydir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO: Wire with the Nimbus keystore
	manager, err := makeAccountManager(keydir)
	if err != nil {
		return err
	}
	m.keystore, err = makeKeyStore(manager)
	return err
}
