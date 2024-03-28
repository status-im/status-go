package protocol

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	utils "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/multiaccounts/settings"
	sociallinkssettings "github.com/status-im/status-go/multiaccounts/settings_social_links"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/server"
)

const (
	maxBioLength            = 240
	maxSocialLinkTextLength = 24
)

var ErrInvalidBioLength = errors.New("invalid bio length")
var ErrInvalidSocialLinkTextLength = errors.New("invalid social link text length")
var ErrDisplayNameDupeOfCommunityMember = errors.New("display name duplicates on of community members")

func (m *Messenger) SetDisplayName(displayName string) error {
	currDisplayName, err := m.settings.DisplayName()
	if err != nil {
		return err
	}

	if currDisplayName == displayName {
		return nil // Do nothing
	}

	if err = utils.ValidateDisplayName(&displayName); err != nil {
		return err
	}

	isDupe, err := m.IsDisplayNameDupeOfCommunityMember(displayName)
	if err != nil {
		return err
	}

	if isDupe {
		return ErrDisplayNameDupeOfCommunityMember
	}

	m.account.Name = displayName
	err = m.multiAccounts.UpdateDisplayName(m.account.KeyUID, displayName)
	if err != nil {
		return err
	}

	err = m.settings.SaveSettingField(settings.DisplayName, displayName)
	if err != nil {
		return err
	}

	err = m.UpdateKeypairName(m.account.KeyUID, displayName)
	if err != nil {
		return err
	}

	err = m.resetLastPublishedTimeForChatIdentity()
	if err != nil {
		return err
	}

	return m.publishContactCode()
}

func (m *Messenger) SaveSyncDisplayName(displayName string, clock uint64) error {
	err := m.settings.SaveSyncSetting(settings.DisplayName, displayName, clock)
	if err != nil {
		return err
	}

	preferredName, err := m.settings.GetPreferredUsername()
	if err != nil {
		return err
	}

	preferredNameClock, err := m.settings.GetSettingLastSynced(settings.PreferredName)
	if err != nil {
		return err
	}
	// When either the display name or preferred name changes, m.account.Name should be updated.
	// However, a race condition can occur during BackupData, where m.account.Name could be incorrectly updated.
	// The final value of m.account.Name depends on which backup message(BackedUpProfile/BackedUpSettings) arrives later.
	// So we should check the clock of the preferred name and only update m.account.Name if it's older than the display name.
	// Yet even if the preferred name clock is older, but the preferred name was empty, we should still update m.account.Name.

	if preferredNameClock < clock || preferredName == "" {
		m.account.Name = displayName
		return m.multiAccounts.SaveAccount(*m.account)
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

func ValidateSocialLinks(socialLinks identity.SocialLinks) error {
	for _, link := range socialLinks {
		l := link
		if err := ValidateSocialLink(l); err != nil {
			return err
		}
	}
	return nil
}

func ValidateSocialLink(link *identity.SocialLink) error {
	if len(link.Text) > maxSocialLinkTextLength {
		return ErrInvalidSocialLinkTextLength
	}
	return nil
}

func (m *Messenger) AddOrReplaceSocialLinks(socialLinks identity.SocialLinks) error {
	if len(socialLinks) > sociallinkssettings.MaxNumOfSocialLinks {
		return errors.New("exceeded maximum number of social links")
	}

	currentSocialLinks, err := m.settings.GetSocialLinks()
	if err != nil {
		return err
	}

	if currentSocialLinks.Equal(socialLinks) {
		return nil // Do nothing
	}

	err = ValidateSocialLinks(socialLinks)
	if err != nil {
		return err
	}

	err = m.withChatClock(func(chatID string, clock uint64) error {
		err = m.settings.AddOrReplaceSocialLinksIfNewer(socialLinks, clock)
		if err != nil {
			return err
		}
		m.selfContact.SocialLinks = socialLinks
		m.publishSelfContactSubscriptions(&SelfContactChangeEvent{
			SocialLinksChanged: true,
		})

		err = m.syncSocialLinks(context.Background(), m.dispatchMessage)
		return err
	})
	if err != nil {
		return err
	}

	if err = m.resetLastPublishedTimeForChatIdentity(); err != nil {
		return err
	}

	return m.publishContactCode()
}

func (m *Messenger) GetSocialLinks() (identity.SocialLinks, error) {
	return m.settings.GetSocialLinks()
}

func (m *Messenger) setInstallationHostname() error {
	imd, err := m.getOurInstallationMetadata()
	if err != nil {
		return err
	}

	// If the name and device are already set, don't do anything
	if len(imd.Name) != 0 && len(imd.DeviceType) != 0 {
		return nil
	}

	if len(imd.Name) == 0 {
		deviceName, err := m.settings.DeviceName()
		if err != nil {
			return err
		}
		if deviceName != "" {
			imd.Name = deviceName
		} else {
			hn, err := server.GetDeviceName()
			if err != nil {
				return err
			}
			imd.Name = fmt.Sprintf("%s %s", hn, imd.Name)
		}
	}

	if len(imd.DeviceType) == 0 {
		imd.DeviceType = runtime.GOOS
	}

	return m.setInstallationMetadata(m.installationID, imd)

}

func (m *Messenger) getOurInstallationMetadata() (*multidevice.InstallationMetadata, error) {
	ourInstallation, ok := m.allInstallations.Load(m.installationID)
	if !ok {
		return nil, fmt.Errorf("messenger's installationID is not set or not loadable")
	}

	if ourInstallation.InstallationMetadata == nil {
		return new(multidevice.InstallationMetadata), nil
	}

	return ourInstallation.InstallationMetadata, nil
}

func (m *Messenger) SetInstallationDeviceType(deviceType string) error {
	if strings.TrimSpace(deviceType) == "" {
		return errors.New("device type is empty")
	}

	imd, err := m.getOurInstallationMetadata()
	if err != nil {
		return err
	}

	// If the name is already set, don't do anything
	if len(imd.DeviceType) != 0 {
		return nil
	}

	imd.DeviceType = deviceType
	return m.setInstallationMetadata(m.installationID, imd)
}
