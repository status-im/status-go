package processor

import (
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"
	"sync"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
)

type EphemeralKeysManager struct {
	capacity int
	keys     map[string]*ecdsa.PrivateKey
	mutex    sync.RWMutex
}

func NewEphemeralKeysManager(capacity int) *EphemeralKeysManager {
	return &EphemeralKeysManager{
		capacity: capacity,
		keys:     make(map[string]*ecdsa.PrivateKey),
	}
}

func formatKey(publicKey *ecdsa.PublicKey) string {
	return cryptotypes.EncodeHex(crypto.FromECDSAPub(publicKey))
}

func (e *EphemeralKeysManager) GetPrivateKeyFor(key *ecdsa.PublicKey) *ecdsa.PrivateKey {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return e.keys[formatKey(key)]
}

func (e *EphemeralKeysManager) GetRandom() (*ecdsa.PrivateKey, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if len(e.keys) >= e.capacity {
		return e.getRandom(), nil
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	e.keys[formatKey(&privateKey.PublicKey)] = privateKey

	return privateKey, nil
}

func (e *EphemeralKeysManager) getRandom() *ecdsa.PrivateKey {
	k := 0
	if index, err := rand.Int(rand.Reader, big.NewInt(int64(len(e.keys)))); err == nil {
		k = int(index.Int64())
	}
	for _, key := range e.keys {
		if k == 0 {
			return key
		}
		k--
	}
	return nil
}
