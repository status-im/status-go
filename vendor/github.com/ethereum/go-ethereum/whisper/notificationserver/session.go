package notificationserver

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// SessionType represents the different types of sessions available
type SessionType int

const (
	SessionServer SessionType = iota
	SessionClient
	SessionChat
)

var (
	Sessions = []SessionType{SessionServer, SessionClient, SessionChat}
)

// Session represents a whisper session
type Session struct {
	Type   SessionType
	Key    []byte
	Values map[string]interface{}
}

// plainSession represents a raw session
type plainSession struct {
	Type   int                    `json:"type"`
	Key    string                 `json:"key"`
	Values map[string]interface{} `json:"values"`
}

// encryptedSession represents an encrypted session
type encryptedSession struct {
	Type   int                    `json:"type"`
	Crypto cryptoJSON             `json:"crypto"`
	Values map[string]interface{} `json:"values"`
}

// cryptoJSON contains the crypto information used to decrypt the encrypted session
type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

// cipherparamsJSON
type cipherparamsJSON struct {
	IV string `json:"iv"`
}

func writeSessionFile(file string, content []byte) error {
	// create session store directory with the appropriate permissions
	// in case is not present yet
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

// sessionFileName implements the naming convention for sessionfiles:
// UTC--<created_at UTC ISO8601>-<address hex>
func sessionFileName(keyAddr common.Address) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-128 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}
