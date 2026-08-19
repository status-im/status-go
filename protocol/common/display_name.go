package common

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/status-im/status-go/protocol/identity/alias"
)

var ErrInvalidDisplayNameRegExp = errors.New("only letters, numbers, underscores and hyphens allowed")
var ErrInvalidDisplayNameEthSuffix = errors.New(`usernames ending with "eth" are not allowed`)
var ErrInvalidDisplayNameNotAllowed = errors.New("name is not allowed")
var ErrInvalidDisplayNameLength = fmt.Errorf("length must be between %d and %d characters", MinDisplayNameLength, MaxDisplayNameLength)

var DISPLAY_NAME_EXT = []string{"_eth", ".eth", "-eth"}

const (
	MinDisplayNameLength = 5
	MaxDisplayNameLength = 24
)

// Compiled regex pattern for validating display names
var displayNameRegex = regexp.MustCompile(fmt.Sprintf("^[\\w-\\s]{%d,%d}$", MinDisplayNameLength, MaxDisplayNameLength))

func ValidateDisplayName(displayName *string) error {
	name := strings.TrimSpace(*displayName)
	*displayName = name

	if name == "" {
		return nil
	}

	if len(name) < MinDisplayNameLength || len(name) > MaxDisplayNameLength {
		return ErrInvalidDisplayNameLength
	}

	// Use pre-compiled regex to validate name format
	if !displayNameRegex.MatchString(name) {
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
