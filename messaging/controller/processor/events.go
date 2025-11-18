package processor

import "crypto/ecdsa"

// SenderUnawareOfInstallation indicates that an encrypted message was sent to us,
// but our installation target is not yet known to the sender.
// This might happen when a user adds a new device and the sender is not aware of it.
type SenderUnawareOfInstallation struct {
	PublicKey *ecdsa.PublicKey
}
