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
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
)

var ErrInvalidDisplayNameRegExp = errors.New("only letters, numbers, underscores and hyphens allowed")
var ErrInvalidDisplayNameEthSuffix = errors.New(`usernames ending with "eth" are not allowed`)
var ErrInvalidDisplayNameNotAllowed = errors.New("name is not allowed")

var DISPLAY_NAME_EXT = []string{"_eth", ".eth", "-eth"}

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
	if err := recover(); err != nil {
		logutils.ZapLogger().Error("panic in goroutine", zap.Any("error", err), zap.Stack("stacktrace"))
		panic(err)
	}
}
