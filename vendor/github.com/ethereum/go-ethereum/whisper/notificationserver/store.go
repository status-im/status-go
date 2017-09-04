package notificationserver

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/cmd/utils"
)

var (
	ErrDecrypt = errors.New("could not decrypt key with given passphrase")
)

// Store represents a session store
type Store interface {
	// GetSession reads the session file & decrypts its encrypted elements
	GetSession(filename string, auth string) (*Session, error)
	// StoreSession encrypts the session & writes the result to a file
	StoreSession(filename string, s *Session, auth string) error
}

// NewStore returns a new session store
func NewStore(dir string) (Store, error) {
	if len(dir) == 0 {
		utils.Fatalf("directory not specified")
	}
	sessionDir := filepath.Join(dir, datadirDefaultSessionStore)
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, err
	}
	return NewStorePassphrase(sessionDir, StandardScryptN, StandardScryptP)
}
