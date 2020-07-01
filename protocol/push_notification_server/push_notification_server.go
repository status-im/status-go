package push_notification_server

import (
	"errors"
	"io"

	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/protocol/protobuf"
)

const encryptedPayloadKeyLength = 16
const nonceLength = 12

var ErrInvalidPushNotificationOptionsVersion = errors.New("invalid version")
var ErrEmptyPushNotificationOptionsPayload = errors.New("empty payload")
var ErrMalformedPushNotificationOptionsInstallationID = errors.New("invalid installationID")
var ErrEmptyPushNotificationOptionsPublicKey = errors.New("no public key")
var ErrCouldNotUnmarshalPushNotificationOptions = errors.New("could not unmarshal preferences")
var ErrInvalidCiphertextLength = errors.New("invalid cyphertext length")
var ErrMalformedPushNotificationOptionsAccessToken = errors.New("invalid access token")
var ErrMalformedPushNotificationOptionsDeviceToken = errors.New("invalid device token")

type Config struct {
	// Identity is our identity key
	Identity *ecdsa.PrivateKey
	// GorushUrl is the url for the gorush service
	GorushURL string
}

type Server struct {
	persistence Persistence
	config      *Config
}

func New(config *Config, persistence Persistence) *Server {
	return &Server{persistence: persistence, config: config}
}

func (p *Server) generateSharedKey(publicKey *ecdsa.PublicKey) ([]byte, error) {
	return ecies.ImportECDSA(p.config.Identity).GenerateShared(
		ecies.ImportECDSAPublic(publicKey),
		encryptedPayloadKeyLength,
		encryptedPayloadKeyLength,
	)
}

func (p *Server) validateUUID(u string) error {
	if len(u) == 0 {
		return errors.New("empty uuid")
	}
	_, err := uuid.Parse(u)
	return err
}

func (p *Server) decryptRegistration(publicKey *ecdsa.PublicKey, payload []byte) ([]byte, error) {
	sharedKey, err := p.generateSharedKey(publicKey)
	if err != nil {
		return nil, err
	}

	return decrypt(payload, sharedKey)
}

// ValidateRegistration validates a new message against the last one received for a given installationID and and public key
// and return the decrypted message
func (p *Server) ValidateRegistration(publicKey *ecdsa.PublicKey, payload []byte) (*protobuf.PushNotificationOptions, error) {
	if payload == nil {
		return nil, ErrEmptyPushNotificationOptionsPayload
	}

	if publicKey == nil {
		return nil, ErrEmptyPushNotificationOptionsPublicKey
	}

	decryptedPayload, err := p.decryptRegistration(publicKey, payload)
	if err != nil {
		return nil, err
	}

	options := &protobuf.PushNotificationOptions{}

	if err := proto.Unmarshal(decryptedPayload, options); err != nil {
		return nil, ErrCouldNotUnmarshalPushNotificationOptions
	}

	if options.Version < 1 {
		return nil, ErrInvalidPushNotificationOptionsVersion
	}

	if err := p.validateUUID(options.InstallationId); err != nil {
		return nil, ErrMalformedPushNotificationOptionsInstallationID
	}

	previousOptions, err := p.persistence.GetPushNotificationOptions(publicKey, options.InstallationId)
	if err != nil {
		return nil, err
	}

	if previousOptions != nil && options.Version <= previousOptions.Version {
		return nil, ErrInvalidPushNotificationOptionsVersion
	}

	// Unregistering message
	if options.Unregister {
		return options, nil
	}

	if err := p.validateUUID(options.AccessToken); err != nil {
		return nil, ErrMalformedPushNotificationOptionsAccessToken
	}

	if len(options.Token) == 0 {
		return nil, ErrMalformedPushNotificationOptionsDeviceToken
	}

	return options, nil
}

func (p *Server) HandlePushNotificationOptions(publicKey *ecdsa.PublicKey, payload []byte) error {

	_, err := p.ValidateRegistration(publicKey, payload)
	if err != nil {
		return err
	}
	return nil
}

func decrypt(cyphertext []byte, key []byte) ([]byte, error) {
	if len(cyphertext) < nonceLength {
		return nil, ErrInvalidCiphertextLength
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := cyphertext[:nonceLength]
	return gcm.Open(nil, nonce, cyphertext[nonceLength:], nil)
}

func encrypt(plaintext []byte, key []byte, reader io.Reader) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
