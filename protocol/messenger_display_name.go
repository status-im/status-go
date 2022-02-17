package protocol

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

var ErrInvalidDisplayName = errors.New("invalid display name")

func ValidateDisplayName(displayName string) error {
	// ^[\\w-\\s]{5,24}$ to allow spaces
	if match, _ := regexp.MatchString("^[\\w-]{5,24}$", displayName); !match {
		return ErrInvalidDisplayName
	}

	// .eth should not happen due to the regexp above, but let's keep it here in case the regexp is changed in the future
	if strings.HasSuffix(displayName, "_eth") || strings.HasSuffix(displayName, ".eth") || strings.HasSuffix(displayName, "-eth") {
		return ErrInvalidDisplayName
	}

	// Uncomment this if spaces are allowed in a display name
	/*
		if alias.IsAlias(displayName) {
			return ErrInvalidDisplayName
		}
	*/

	return nil
}

func (m *Messenger) SetDisplayName(displayName string) error {
	logger := m.logger.Named("SetDisplayName")

	displayName = strings.TrimSpace(displayName)
	currDisplayName, err := m.settings.DisplayName()
	if err != nil {
		return err
	}

	if currDisplayName == displayName {
		return nil // Do nothing
	}

	if err = ValidateDisplayName(displayName); err != nil {
		return err
	}

	ensName, err := m.settings.ENSName()
	if err != nil {
		return err
	}

	err = m.settings.SaveSetting("display-name", displayName)
	if err != nil {
		return err
	}

	go func() {
		// We send a contact update to all contacts so they get latest displayName
		err = m.SendContactUpdates(context.Background(), displayName, ensName, "")
		if err != nil {
			logger.Error("m.SendContactUpdates error", zap.Error(err))
		}
	}()

	return nil
}
