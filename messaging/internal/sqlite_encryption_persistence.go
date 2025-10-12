package internal

import (
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/types"
)

// Implements the EncryptionPersistence interface using SQLite.
type SQLiteEncryptionPersistence struct {
	*encryption.SQLitePersistence
}

var _ types.EncryptionPersistence = (*SQLiteEncryptionPersistence)(nil)

func (e *SQLiteEncryptionPersistence) X3DHBundlesStorage() types.X3DHBundlesPersistence {
	return nil
}

func (e *SQLiteEncryptionPersistence) DRKeysStorage() types.DRKeysPersistence {
	return nil
}

func (e *SQLiteEncryptionPersistence) DRSessionStorage() types.DRSessionPersistence {
	return nil
}

func (e *SQLiteEncryptionPersistence) SharedSecretStorage() types.SharedSecretPersistence {
	return nil
}

func (e *SQLiteEncryptionPersistence) MultideviceStorage() types.MultidevicePersistence {
	return nil
}

func (e *SQLiteEncryptionPersistence) HashRatchetStorage() types.HashRatchetPersistence {
	return nil
}
