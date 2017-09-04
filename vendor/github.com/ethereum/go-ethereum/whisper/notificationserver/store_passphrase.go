package notificationserver

import (
	"bytes"
	"crypto/aes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/randentropy"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

const (
	keyHeaderKDF = "scrypt"

	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = 1 << 18

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = 1

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = 1 << 12

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
)

// StorePassphrase -
type StorePassphrase struct {
	sessionsDirPath string
	scryptN         int
	scryptP         int
}

// NewStorePassphrase -
func NewStorePassphrase(sessionDir string, scryptN int, scryptP int) (*StorePassphrase, error) {
	sessionDir, err := filepath.Abs(sessionDir)
	if err != nil {
		return nil, err
	}
	return &StorePassphrase{sessionDir, scryptN, scryptP}, err
}

// GetSession loads the session from the sessionstore & decrypts its contents
func (s *StorePassphrase) GetSession(filename, auth string) (*Session, error) {
	sessionjson, err := ioutil.ReadFile(s.joinPath(filename))
	if err != nil {
		return nil, err
	}
	return DecryptSession(sessionjson, auth)
}

// StoreSession encrypts the session & stores it in the sessionstore
func (s *StorePassphrase) StoreSession(filename string, session *Session, auth string) error {
	sessionjson, err := EncryptSession(session, auth, s.scryptN, s.scryptP)
	if err != nil {
		return err
	}
	return writeSessionFile(s.joinPath(filename), sessionjson)
}

// joinPath -
func (s *StorePassphrase) joinPath(filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	}
	return filepath.Join(s.sessionsDirPath, filename)
}

// EncryptSession encrypts a session key using the specified scrypt parameters into a json
// blob that can be decrypted later on.
func EncryptSession(session *Session, auth string, scryptN, scryptP int) ([]byte, error) {
	authArray := []byte(auth)
	salt := randentropy.GetEntropyCSPRNG(32)
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	keyBytes := session.Key

	iv := randentropy.GetEntropyCSPRNG(aes.BlockSize) // 16
	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := cryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}

	encryptedSession := encryptedSession{
		int(session.Type),
		cryptoStruct,
		session.Values,
	}

	return json.Marshal(encryptedSession)
}

// DecryptSession -
func DecryptSession(sessionjson []byte, auth string) (*Session, error) {
	encrypted := new(encryptedSession)
	if err := json.Unmarshal(sessionjson, encrypted); err != nil {
		return nil, err
	}

	if encrypted.Crypto.Cipher != "aes-128-ctr" {
		return nil, fmt.Errorf("Cipher not supported: %v", encrypted.Crypto.Cipher)
	}

	mac, err := hex.DecodeString(encrypted.Crypto.MAC)
	if err != nil {
		return nil, err
	}

	iv, err := hex.DecodeString(encrypted.Crypto.CipherParams.IV)
	if err != nil {
		return nil, err
	}

	cipherText, err := hex.DecodeString(encrypted.Crypto.CipherText)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getKDFKey(encrypted.Crypto, auth)
	if err != nil {
		return nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, ErrDecrypt
	}

	rawKey, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, err
	}

	return &Session{
		Key:    rawKey,
		Type:   SessionType(encrypted.Type),
		Values: encrypted.Values,
	}, nil
}

func getKDFKey(cryptoJSON cryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	salt, err := hex.DecodeString(cryptoJSON.KDFParams["salt"].(string))
	if err != nil {
		return nil, err
	}
	dkLen := ensureInt(cryptoJSON.KDFParams["dklen"])

	if cryptoJSON.KDF == keyHeaderKDF {
		n := ensureInt(cryptoJSON.KDFParams["n"])
		r := ensureInt(cryptoJSON.KDFParams["r"])
		p := ensureInt(cryptoJSON.KDFParams["p"])
		return scrypt.Key(authArray, salt, n, r, p, dkLen)

	} else if cryptoJSON.KDF == "pbkdf2" {
		c := ensureInt(cryptoJSON.KDFParams["c"])
		prf := cryptoJSON.KDFParams["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	}

	return nil, fmt.Errorf("Unsupported KDF: %s", cryptoJSON.KDF)
}

// TODO: can we do without this when unmarshalling dynamic JSON?
// why do integers in KDF params end up as float64 and not int after
// unmarshal?
func ensureInt(x interface{}) int {
	res, ok := x.(int)
	if !ok {
		res = int(x.(float64))
	}
	return res
}
