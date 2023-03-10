package pairing

import (
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
)

type PayloadMounterReceiver interface {
	PayloadMounter
	PayloadReceiver
}

type PayloadRepository interface {
	PayloadLoader
	PayloadStorer
}

type PayloadLocker interface {
	// LockPayload prevents future excess to outbound safe and received data
	LockPayload()
}

type PayloadResetter interface {
	// ResetPayload resets all payloads the PayloadManager has in its state
	ResetPayload()
}

type Encryptor interface {
	// EncryptPlain encrypts the given plaintext using internal key(s)
	EncryptPlain(plaintext []byte) ([]byte, error)
}

type HandlerServer interface {
	GetLogger() *zap.Logger
	GetCookieStore() *sessions.CookieStore
	DecryptPlain([]byte) ([]byte, error)
}
