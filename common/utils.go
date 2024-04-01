package common

import (
	"crypto/ecdsa"
	"errors"
	"regexp"
	"strings"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/protobuf"
)

var ErrInvalidDisplayNameRegExp = errors.New("only letters, numbers, underscores and hyphens allowed")
var ErrInvalidDisplayNameEthSuffix = errors.New(`usernames ending with "eth" are not allowed`)
var ErrInvalidDisplayNameNotAllowed = errors.New("name is not allowed")

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

	// ^[\\w-\\s]{3,24}$ to allow spaces
	if match, _ := regexp.MatchString("^[\\w-\\s]{3,24}$", name); !match {
		return ErrInvalidDisplayNameRegExp
	}

	// .eth should not happen due to the regexp above, but let's keep it here in case the regexp is changed in the future
	if strings.HasSuffix(name, "_eth") || strings.HasSuffix(name, ".eth") || strings.HasSuffix(name, "-eth") {
		return ErrInvalidDisplayNameEthSuffix
	}

	if alias.IsAlias(name) {
		return ErrInvalidDisplayNameNotAllowed
	}

	return nil
}
