package protocol

import "github.com/status-im/status-go/protocol/identity"

func (m *Messenger) SetSocialLinks(socialLinks *identity.SocialLinks) error {
	currentSocialLinks, err := m.settings.GetSocialLinks()
	if err != nil {
		return err
	}

	if currentSocialLinks.Equals(*socialLinks) {
		return nil // Do nothing
	}

	err = m.settings.SetSocialLinks(socialLinks)
	if err != nil {
		return err
	}

	err = m.resetLastPublishedTimeForChatIdentity()
	if err != nil {
		return err
	}

	return m.publishContactCode()
}
