package noise

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"strings"
)

type QR struct {
	applicationName    string
	applicationVersion string
	shardID            string
	ephemeralPublicKey ed25519.PublicKey
	committedStaticKey []byte
}

func NewQR(applicationName, applicationVersion, shardID string, ephemeralKey ed25519.PublicKey, committedStaticKey []byte) QR {
	return QR{
		applicationName:    applicationName,
		applicationVersion: applicationVersion,
		shardID:            shardID,
		ephemeralPublicKey: ephemeralKey,
		committedStaticKey: committedStaticKey,
	}
}

// Serializes input parameters to a base64 string for exposure through QR code (used by WakuPairing)
func (qr QR) String() string {
	return base64.URLEncoding.EncodeToString([]byte(qr.applicationName)) + ":" +
		base64.URLEncoding.EncodeToString([]byte(qr.applicationVersion)) + ":" +
		base64.URLEncoding.EncodeToString([]byte(qr.shardID)) + ":" +
		base64.URLEncoding.EncodeToString(qr.ephemeralPublicKey) + ":" +
		base64.URLEncoding.EncodeToString(qr.committedStaticKey[:])
}

func (qr QR) Bytes() []byte {
	return []byte(qr.String())
}

func decodeBase64String(inputValue string) ([]byte, error) {
	enc := base64.StdEncoding
	if strings.ContainsAny(inputValue, "-_") {
		enc = base64.URLEncoding
	}
	if len(inputValue)%4 != 0 {
		enc = enc.WithPadding(base64.NoPadding)
	}

	return enc.DecodeString(inputValue)
}

// StringToQR deserializes input string in base64 to the corresponding (applicationName, applicationVersion, shardId, ephemeralKey, committedStaticKey)
func StringToQR(qrString string) (QR, error) {
	values := strings.Split(qrString, ":")
	if len(values) != 5 {
		return QR{}, errors.New("invalid qr string")
	}

	applicationName, err := decodeBase64String(values[0])
	if err != nil {
		return QR{}, err
	}

	applicationVersion, err := decodeBase64String(values[1])
	if err != nil {
		return QR{}, err
	}

	shardID, err := decodeBase64String(values[2])
	if err != nil {
		return QR{}, err
	}

	ephemeralKey, err := decodeBase64String(values[3])
	if err != nil {
		return QR{}, err
	}

	committedStaticKey, err := decodeBase64String(values[4])
	if err != nil {
		return QR{}, err
	}

	return QR{
		applicationName:    string(applicationName),
		applicationVersion: string(applicationVersion),
		shardID:            string(shardID),
		ephemeralPublicKey: ephemeralKey,
		committedStaticKey: committedStaticKey,
	}, nil
}
