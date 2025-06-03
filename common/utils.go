package common

import (
	"crypto/ecdsa"
	"errors"
	"reflect"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/pkg/sentry"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
)

var ErrInvalidDisplayNameRegExp = errors.New("only letters, numbers, underscores and hyphens allowed")
var ErrInvalidDisplayNameEthSuffix = errors.New(`usernames ending with "eth" are not allowed`)
var ErrInvalidDisplayNameNotAllowed = errors.New("name is not allowed")

var DISPLAY_NAME_EXT = []string{"_eth", ".eth", "-eth"}

const DefaultTruncateLength = 8

func RecoverKey(m *protobuf.ApplicationMetadataMessage) (*ecdsa.PublicKey, error) {
	if m.Signature == nil {
		return nil, nil
	}

	recoveredKey, err := crypto.SigToPub(
		crypto.Keccak256(m.Payload),
		m.Signature,
	)
	if err != nil {
		return nil, err
	}

	return recoveredKey, nil
}

func ValidateDisplayName(displayName *string) error {
	name := strings.TrimSpace(*displayName)
	*displayName = name

	if name == "" {
		return nil
	}

	// ^[\\w-\\s]{5,24}$ to allow spaces
	if match, _ := regexp.MatchString("^[\\w-\\s]{5,24}$", name); !match {
		return ErrInvalidDisplayNameRegExp
	}

	// .eth should not happen due to the regexp above, but let's keep it here in case the regexp is changed in the future
	for _, ext := range DISPLAY_NAME_EXT {
		if strings.HasSuffix(*displayName, ext) {
			return ErrInvalidDisplayNameEthSuffix
		}
	}

	if alias.IsAlias(name) {
		return ErrInvalidDisplayNameNotAllowed
	}

	return nil
}

// implementation referenced from https://github.com/embarklabs/embark/blob/master/packages/plugins/ens/src/index.js
func IsENSName(displayName string) bool {
	if len(displayName) == 0 {
		return false
	}

	if strings.HasSuffix(displayName, ".eth") {
		return true
	}

	return false
}

func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Interface:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

func LogOnPanic() {
	err := recover()
	if err == nil {
		return
	}

	logutils.ZapLogger().Error("panic in goroutine",
		zap.Any("error", err),
		zap.Stack("stacktrace"))

	sentry.RecoverError(err)

	panic(err)
}

// TruncateWithDotN truncates a string by keeping n characters (excluding "..") from start and end,
// replacing middle part with "..".
// The total length of returned string will be n + 2 (where 2 is the length of "..").
// For n characters total (excluding dots), characters are distributed evenly between start and end.
// If n is odd, start gets one extra character.
// Examples:
//
//	TruncateWithDotN("0x123456789", 4) // returns "0x1..89" (total length 6)
//	TruncateWithDotN("0x123456789", 3) // returns "0x..9" (total length 5)
//	TruncateWithDotN("0x123456789", 5) // returns "0x1..89" (total length 7)
//	TruncateWithDotN("acdkef099", 3) // returns "ac..9" (total length 5)
//	TruncateWithDotN("ab", 4)        // returns "ab" (no truncation needed as len <= n)
//	TruncateWithDotN("test", 0)      // returns "test" (no truncation for n <= 0)
func TruncateWithDotN(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}

	if n < 2 { // need at least 2 chars (1 at start + 1 at end)
		n = 2
	}

	const dots = ".."
	// For n characters total (excluding dots), we want to keep:
	// - ceil(n/2) characters from the start
	// - floor(n/2) characters from the end
	startChars := (n + 1) / 2 // ceil(n/2)
	endChars := n / 2         // floor(n/2)

	return s[:startChars] + dots + s[len(s)-endChars:]
}

func TruncateWithDot(s string) string {
	return TruncateWithDotN(s, DefaultTruncateLength)
}

func Ptr[T any](v T) *T {
	return &v
}
