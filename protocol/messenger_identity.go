package protocol

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/server"
)

const (
	maxBioLength            = 240
	maxSocialLinkTextLength = 24
)

var ErrInvalidDisplayNameRegExp = errors.New("only letters, numbers, underscores and hyphens allowed")
var ErrInvalidDisplayNameEthSuffix = errors.New(`usernames ending with "eth" are not allowed`)
var ErrInvalidDisplayNameNotAllowed = errors.New("name is not allowed")
var ErrInvalidBioLength = errors.New("invalid bio length")
var ErrInvalidSocialLinkTextLength = errors.New("invalid social link text length")

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

func (m *Messenger) SetDisplayName(displayName string, publishChange bool) error {
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
	err = m.multiAccounts.SaveAccount(m.account)
	if err != nil {
		return err
	}

	err = m.settings.SaveSettingField(settings.DisplayName, displayName)
	if err != nil {
		return err
	}

	if publishChange {
		err = m.resetLastPublishedTimeForChatIdentity()
		if err != nil {
			return err
		}

		return m.publishContactCode()
	}

	return nil
}

func ValidateBio(bio *string) error {
	if len(*bio) > maxBioLength {
		return ErrInvalidBioLength
	}
	return nil
}

func (m *Messenger) SetBio(bio string) error {
	currentBio, err := m.settings.Bio()
	if err != nil {
		return err
	}

	if currentBio == bio {
		return nil // Do nothing
	}

	if err = ValidateBio(&bio); err != nil {
		return err
	}

	if err = m.settings.SaveSettingField(settings.Bio, bio); err != nil {
		return err
	}

	if err = m.resetLastPublishedTimeForChatIdentity(); err != nil {
		return err
	}

	return m.publishContactCode()
}

func ValidateSocialLinks(socialLinks *identity.SocialLinks) error {
	for _, link := range *socialLinks {
		if len(link.Text) > maxSocialLinkTextLength {
			return ErrInvalidSocialLinkTextLength
		}
	}
	return nil
}

func (m *Messenger) SetSocialLinks(socialLinks *identity.SocialLinks) error {
	currentSocialLinks, err := m.settings.GetSocialLinks()
	if err != nil {
		return err
	}

	if currentSocialLinks.Equals(*socialLinks) {
		return nil // Do nothing
	}

	if err = ValidateSocialLinks(socialLinks); err != nil {
		return err
	}

	if err = m.settings.SetSocialLinks(socialLinks); err != nil {
		return err
	}

	if err = m.resetLastPublishedTimeForChatIdentity(); err != nil {
		return err
	}

	return m.publishContactCode()
}

func (m *Messenger) setInstallationHostname() error {
	ourInstallation, ok := m.allInstallations.Load(m.installationID)
	if !ok {
		m.logger.Error("Messenger's installationID is not set or not loadable")
		return nil
	}

	var imd *multidevice.InstallationMetadata
	if ourInstallation.InstallationMetadata == nil {
		imd = new(multidevice.InstallationMetadata)
	} else {
		imd = ourInstallation.InstallationMetadata
	}

	// If the name is already set, don't do anything
	if len(imd.Name) != 0 {
		return nil
	}

	hn, err := server.GetDeviceName()
	if err != nil {
		return err
	}
	imd.Name = fmt.Sprintf("%s %s", hn, imd.Name)
	return m.setInstallationMetadata(m.installationID, imd)
}
