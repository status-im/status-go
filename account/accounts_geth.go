package account

import (
	"github.com/ethereum/go-ethereum/accounts"

	"github.com/status-im/status-go/account/generator"
)

// GethManager represents account manager interface.
type GethManager struct {
	*Manager

	gethAccManager *accounts.Manager
}

// NewGethManager returns new node account manager.
func NewGethManager() *GethManager {
	m := &GethManager{}
	m.Manager = &Manager{accountsGenerator: generator.New(m)}
	return m
}

// InitKeystore sets key manager and key store.
func (m *GethManager) InitKeystore(keydir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error
	m.gethAccManager, err = makeAccountManager(keydir)
	if err != nil {
		return err
	}

	m.keystore, err = makeKeyStore(m.gethAccManager)
	m.Keydir = keydir
	return err
}

func (m *GethManager) GetManager() *accounts.Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gethAccManager
}
