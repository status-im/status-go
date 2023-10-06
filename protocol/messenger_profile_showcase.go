package protocol

type ProfileShowcaseResponse struct {
	Communities  []*ProfileShowcaseEntry `json:"communities"`
	Accounts     []*ProfileShowcaseEntry `json:"accounts"`
	Collectibles []*ProfileShowcaseEntry `json:"collectibles"`
	Assets       []*ProfileShowcaseEntry `json:"assets"`
}

func (m *Messenger) SetProfileShowcasePreference(entry *ProfileShowcaseEntry) error {
	return m.persistence.InsertOrUpdateProfileShowcasePreference(entry)
}

func (m *Messenger) GetProfileShowcasePreferences() (*ProfileShowcaseResponse, error) {
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

	return &ProfileShowcaseResponse{
		Communities:  communities,
		Accounts:     accounts,
		Collectibles: collectibles,
		Assets:       assets,
	}, nil
}
