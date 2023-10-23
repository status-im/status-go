package protocol

import "fmt"

type ProfileShowcasePreferences struct {
	Communities  []*ProfileShowcaseEntry `json:"communities"`
	Accounts     []*ProfileShowcaseEntry `json:"accounts"`
	Collectibles []*ProfileShowcaseEntry `json:"collectibles"`
	Assets       []*ProfileShowcaseEntry `json:"assets"`
}

func (p *ProfileShowcasePreferences) Validate() error {
	for _, community := range p.Communities {
		if community.EntryType != ProfileShowcaseEntryTypeCommunity {
			return fmt.Errorf("communities must contain only entriers of type ProfileShowcaseEntryTypeCommunity")
		}
	}

	for _, community := range p.Accounts {
		if community.EntryType != ProfileShowcaseEntryTypeAccount {
			return fmt.Errorf("accounts must contain only entriers of type ProfileShowcaseEntryTypeAccount")
		}
	}

	for _, community := range p.Collectibles {
		if community.EntryType != ProfileShowcaseEntryTypeCollectible {
			return fmt.Errorf("collectibles must contain only entriers of type ProfileShowcaseEntryTypeCollectible")
		}
	}

	for _, community := range p.Assets {
		if community.EntryType != ProfileShowcaseEntryTypeAsset {
			return fmt.Errorf("assets must contain only entriers of type ProfileShowcaseEntryTypeAsset")
		}
	}
	return nil
}

func (m *Messenger) SetProfileShowcasePreference(entry *ProfileShowcaseEntry) error {
	return m.persistence.InsertOrUpdateProfileShowcasePreference(entry)
}

func (m *Messenger) SetProfileShowcasePreferences(preferences ProfileShowcasePreferences) error {
	err := preferences.Validate()
	if err != nil {
		return err
	}

	allPreferences := []*ProfileShowcaseEntry{}

	allPreferences = append(allPreferences, preferences.Communities...)
	allPreferences = append(allPreferences, preferences.Accounts...)
	allPreferences = append(allPreferences, preferences.Collectibles...)
	allPreferences = append(allPreferences, preferences.Assets...)

	return m.persistence.SaveProfileShowcasePreferences(allPreferences)
}

func (m *Messenger) GetProfileShowcasePreferences() (*ProfileShowcasePreferences, error) {
	// NOTE: in the future default profile preferences should be filled in for each group according to special rules,
	// that's why they should be grouped here
	communities, err := m.persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeCommunity)
	if err != nil {
		return nil, err
	}

	accounts, err := m.persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeAccount)
	if err != nil {
		return nil, err
	}

	collectibles, err := m.persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeCollectible)
	if err != nil {
		return nil, err
	}

	assets, err := m.persistence.GetProfileShowcasePreferencesByType(ProfileShowcaseEntryTypeAsset)
	if err != nil {
		return nil, err
	}

	return &ProfileShowcasePreferences{
		Communities:  communities,
		Accounts:     accounts,
		Collectibles: collectibles,
		Assets:       assets,
	}, nil
}
