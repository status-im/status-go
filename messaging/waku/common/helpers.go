package common

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"errors"
	"fmt"
	mrand "math/rand"
	"regexp"
	"strings"

	"github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/common"
)

// IsPubKeyEqual checks that two public keys are equal
func IsPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	if !ValidatePublicKey(a) {
		return false
	} else if !ValidatePublicKey(b) {
		return false
	}
	// the curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}

// ValidatePublicKey checks the format of the given public key.
func ValidatePublicKey(k *ecdsa.PublicKey) bool {
	return k != nil && k.X != nil && k.Y != nil && k.X.Sign() != 0 && k.Y.Sign() != 0
}

// BytesToUintLittleEndian converts the slice to 64-bit unsigned integer.
func BytesToUintLittleEndian(b []byte) (res uint64) {
	mul := uint64(1)
	for i := 0; i < len(b); i++ {
		res += uint64(b[i]) * mul
		mul *= 256
	}
	return res
}

// BytesToUintBigEndian converts the slice to 64-bit unsigned integer.
func BytesToUintBigEndian(b []byte) (res uint64) {
	for i := 0; i < len(b); i++ {
		res *= 256
		res += uint64(b[i])
	}
	return res
}

// ContainsOnlyZeros checks if the data contain only zeros.
func ContainsOnlyZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

// GenerateSecureRandomData generates random data where extra security is required.
// The purpose of this function is to prevent some bugs in software or in hardware
// from delivering not-very-random data. This is especially useful for AES nonce,
// where true randomness does not really matter, but it is very important to have
// a unique nonce for every message.
func GenerateSecureRandomData(length int) ([]byte, error) {
	x := make([]byte, length)
	y := make([]byte, length)
	res := make([]byte, length)

	_, err := crand.Read(x)
	if err != nil {
		return nil, err
	} else if !ValidateDataIntegrity(x, length) {
		return nil, errors.New("crypto/rand failed to generate secure random data")
	}
	_, err = mrand.Read(y) // nolint: gosec
	if err != nil {
		return nil, err
	} else if !ValidateDataIntegrity(y, length) {
		return nil, errors.New("math/rand failed to generate secure random data")
	}
	for i := 0; i < length; i++ {
		res[i] = x[i] ^ y[i]
	}
	if !ValidateDataIntegrity(res, length) {
		return nil, errors.New("failed to generate secure random data")
	}
	return res, nil
}

// GenerateRandomID generates a random string, which is then returned to be used as a key id
func GenerateRandomID() (id string, err error) {
	buf, err := GenerateSecureRandomData(KeyIDSize)
	if err != nil {
		return "", err
	}
	if !ValidateDataIntegrity(buf, KeyIDSize) {
		return "", fmt.Errorf("error in generateRandomID: crypto/rand failed to generate random data")
	}
	id = common.Bytes2Hex(buf)
	return id, err
}

// ValidateDataIntegrity returns false if the data have the wrong or contains all zeros,
// which is the simplest and the most common bug.
func ValidateDataIntegrity(k []byte, expectedSize int) bool {
	if len(k) != expectedSize {
		return false
	}
	if expectedSize > 3 && ContainsOnlyZeros(k) {
		return false
	}
	return true
}

func ParseDialErrors(errMsg string) []DialError {
	// Regular expression to match the array of failed dial attempts
	re := regexp.MustCompile(`all dials failed\n((?:\s*\*\s*\[.*\].*\n?)+)`)

	match := re.FindStringSubmatch(errMsg)
	if len(match) < 2 {
		return nil
	}

	// Split the matched string into individual dial attempts
	dialAttempts := strings.Split(strings.TrimSpace(match[1]), "\n")

	// Regular expression to extract multiaddr and error message
	reAttempt := regexp.MustCompile(`\[(.*?)\]\s*(.*)`)

	var dialErrors []DialError
	for _, attempt := range dialAttempts {
		attempt = strings.TrimSpace(strings.Trim(attempt, "* "))
		matches := reAttempt.FindStringSubmatch(attempt)
		if len(matches) != 3 {
			continue
		}
		errMsg := strings.TrimSpace(matches[2])
		ma, err := multiaddr.NewMultiaddr(matches[1])
		if err != nil {
			continue
		}
		protocols := ma.Protocols()
		protocolsStr := "/"
		for i, protocol := range protocols {
			protocolsStr += protocol.Name
			if i < len(protocols)-1 {
				protocolsStr += "/"
			}
		}
		dialErrors = append(dialErrors, DialError{
			Protocols: protocolsStr,
			MultiAddr: matches[1],
			ErrMsg:    errMsg,
			ErrType:   CategorizeDialError(errMsg),
		})

	}

	return dialErrors
}

// DialErrorType represents the type of dial error
type DialErrorType int

const (
	ErrorUnknown DialErrorType = iota
	ErrorIOTimeout
	ErrorConnectionRefused
	ErrorRelayCircuitFailed
	ErrorRelayNoReservation
	ErrorSecurityNegotiationFailed
	ErrorConcurrentDialSucceeded
	ErrorConcurrentDialFailed
	ErrorConnectionsPerIPLimitExceeded
	ErrorStreamReset
	ErrorRelayResourceLimitExceeded
	ErrorOpeningHopStreamToRelay
	ErrorDialBackoff
)

func (det DialErrorType) String() string {
	return [...]string{
		"Unknown",
		"I/O Timeout",
		"Connection Refused",
		"Relay Circuit Failed",
		"Relay No Reservation",
		"Security Negotiation Failed",
		"Concurrent Dial Succeeded",
		"Concurrent Dial Failed",
		"Connections Per IP Limit Exceeded",
		"Stream Reset",
		"Relay Resource Limit Exceeded",
		"Error Opening Hop Stream to Relay",
		"Dial Backoff",
	}[det]
}

func CategorizeDialError(errMsg string) DialErrorType {
	switch {
	case strings.Contains(errMsg, "i/o timeout"):
		return ErrorIOTimeout
	case strings.Contains(errMsg, "connect: connection refused"):
		return ErrorConnectionRefused
	case strings.Contains(errMsg, "error opening relay circuit: CONNECTION_FAILED"):
		return ErrorRelayCircuitFailed
	case strings.Contains(errMsg, "error opening relay circuit: NO_RESERVATION"):
		return ErrorRelayNoReservation
	case strings.Contains(errMsg, "failed to negotiate security protocol"):
		return ErrorSecurityNegotiationFailed
	case strings.Contains(errMsg, "concurrent active dial succeeded"):
		return ErrorConcurrentDialSucceeded
	case strings.Contains(errMsg, "concurrent active dial through the same relay failed"):
		return ErrorConcurrentDialFailed
	case strings.Contains(errMsg, "connections per ip limit exceeded"):
		return ErrorConnectionsPerIPLimitExceeded
	case strings.Contains(errMsg, "stream reset"):
		return ErrorStreamReset
	case strings.Contains(errMsg, "error opening relay circuit: RESOURCE_LIMIT_EXCEEDED"):
		return ErrorRelayResourceLimitExceeded
	case strings.Contains(errMsg, "error opening hop stream to relay: connection failed"):
		return ErrorOpeningHopStreamToRelay
	case strings.Contains(errMsg, "dial backoff"):
		return ErrorDialBackoff
	default:
		return ErrorUnknown
	}
}

// DialError represents a single dial error with its multiaddr and error message
type DialError struct {
	MultiAddr string
	ErrMsg    string
	ErrType   DialErrorType
	Protocols string
}
