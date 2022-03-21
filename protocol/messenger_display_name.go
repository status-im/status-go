package protocol

import (
	"errors"
	"regexp"
	"strings"

	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/identity/alias"
)

var ErrInvalidDisplayNameRegExp = errors.New("only letters, numbers, underscores and hyphens allowed")
var ErrInvalidDisplayNameEthSuffix = errors.New(`usernames ending with "eth" are not allowed`)
var ErrInvalidDisplayNameNotAllowed = errors.New("name is not allowed")

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
	if strings.HasSuffix(name, "_eth") || strings.HasSuffix(name, ".eth") || strings.HasSuffix(name, "-eth") {
		return ErrInvalidDisplayNameEthSuffix
	}

	if alias.IsAlias(name) {
		return ErrInvalidDisplayNameNotAllowed
	}

	return nil
}

func (m *Messenger) SetDisplayName(displayName string) error {
	currDisplayName, err := m.settings.DisplayName()
	if err != nil {
		return err
	}

	if currDisplayName == displayName {
		return nil // Do nothing
	}

	if err = ValidateDisplayName(&displayName); err != nil {
		return err
	}

	m.account.Name = displayName // We might need to do the same when syncing settings?
	err = m.multiAccounts.SaveAccount(*m.account)
	if err != nil {
		return err
	}

	err = m.settings.SaveSettingField(settings.DisplayName, displayName)
	if err != nil {
		return err
	}

	err = m.resetLastPublishedTimeForChatIdentity()
	if err != nil {
		return err
	}

	return m.publishContactCode()
}
