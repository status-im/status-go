package identity

import "errors"

var ErrorNoAccountProvidedWithTokenOrCollectible = errors.New("no account provided with tokens or collectible")
var ErrorDublicateAccountAddress = errors.New("duplicate account address")
var ErrorAccountVisibilityLowerThanCollectible = errors.New("account visibility lower than collectible")

type ProfileShowcaseVisibility int

const (
	ProfileShowcaseVisibilityNoOne ProfileShowcaseVisibility = iota
	ProfileShowcaseVisibilityIDVerifiedContacts
	ProfileShowcaseVisibilityContacts
	ProfileShowcaseVisibilityEveryone
)

type ProfileShowcaseMembershipStatus int

const (
	ProfileShowcaseMembershipStatusUnproven ProfileShowcaseMembershipStatus = iota
	ProfileShowcaseMembershipStatusProvenMember
	ProfileShowcaseMembershipStatusNotAMember
)

type ProfileShowcaseCommunityPreference struct {
	CommunityID        string                    `json:"communityId"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

type ProfileShowcaseAccountPreference struct {
	Address            string                    `json:"address"`
	Name               string                    `json:"name"`
	ColorID            string                    `json:"colorId"`
	Emoji              string                    `json:"emoji"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

type ProfileShowcaseCollectiblePreference struct {
	ContractAddress    string                    `json:"contractAddress"`
	ChainID            uint64                    `json:"chainId"`
	TokenID            string                    `json:"tokenId"`
	CommunityID        string                    `json:"communityId"`
	AccountAddress     string                    `json:"accountAddress"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

type ProfileShowcaseVerifiedTokenPreference struct {
	Symbol             string                    `json:"symbol"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

type ProfileShowcaseUnverifiedTokenPreference struct {
	ContractAddress    string                    `json:"contractAddress"`
	ChainID            uint64                    `json:"chainId"`
	CommunityID        string                    `json:"communityId"`
	ShowcaseVisibility ProfileShowcaseVisibility `json:"showcaseVisibility"`
	Order              int                       `json:"order"`
}

type ProfileShowcasePreferences struct {
	Communities      []*ProfileShowcaseCommunityPreference       `json:"communities"`
	Accounts         []*ProfileShowcaseAccountPreference         `json:"accounts"`
	Collectibles     []*ProfileShowcaseCollectiblePreference     `json:"collectibles"`
	VerifiedTokens   []*ProfileShowcaseVerifiedTokenPreference   `json:"verifiedTokens"`
	UnverifiedTokens []*ProfileShowcaseUnverifiedTokenPreference `json:"unverifiedTokens"`
}

type ProfileShowcaseCommunity struct {
	CommunityID      string                          `json:"communityId"`
	Order            int                             `json:"order"`
	MembershipStatus ProfileShowcaseMembershipStatus `json:"membershipStatus"`
}

type ProfileShowcaseAccount struct {
	ContactID string `json:"contactId"`
	Address   string `json:"address"`
	Name      string `json:"name"`
	ColorID   string `json:"colorId"`
	Emoji     string `json:"emoji"`
	Order     int    `json:"order"`
}

type ProfileShowcaseCollectible struct {
	ContractAddress string `json:"contractAddress"`
	ChainID         uint64 `json:"chainId"`
	TokenID         string `json:"tokenId"`
	CommunityID     string `json:"communityId"`
	AccountAddress  string `json:"accountAddress"`
	Order           int    `json:"order"`
}

type ProfileShowcaseVerifiedToken struct {
	Symbol string `json:"symbol"`
	Order  int    `json:"order"`
}

type ProfileShowcaseUnverifiedToken struct {
	ContractAddress string `json:"contractAddress"`
	ChainID         uint64 `json:"chainId"`
	CommunityID     string `json:"communityId"`
	Order           int    `json:"order"`
}

type ProfileShowcase struct {
	ContactID        string                            `json:"contactId"`
	Communities      []*ProfileShowcaseCommunity       `json:"communities"`
	Accounts         []*ProfileShowcaseAccount         `json:"accounts"`
	Collectibles     []*ProfileShowcaseCollectible     `json:"collectibles"`
	VerifiedTokens   []*ProfileShowcaseVerifiedToken   `json:"verifiedTokens"`
	UnverifiedTokens []*ProfileShowcaseUnverifiedToken `json:"unverifiedTokens"`
}

func Validate(preferences *ProfileShowcasePreferences) error {
	if (len(preferences.VerifiedTokens) > 0 || len(preferences.UnverifiedTokens) > 0 || len(preferences.Collectibles) > 0) &&
		len(preferences.Accounts) == 0 {
		return ErrorNoAccountProvidedWithTokenOrCollectible
	}

	accountsMap := make(map[string]*ProfileShowcaseAccountPreference)
	for _, account := range preferences.Accounts {
		if _, ok := accountsMap[account.Address]; ok {
			return ErrorDublicateAccountAddress
		}
		accountsMap[account.Address] = account
	}

	for _, collectible := range preferences.Collectibles {
		account, ok := accountsMap[collectible.AccountAddress]
		if !ok {
			return nil
			// NOTE: with current wallet collectible implementation we don't know account on this stage
			// return errorNoAccountAddressForCollectible
		}
		if account.ShowcaseVisibility < collectible.ShowcaseVisibility {
			return ErrorAccountVisibilityLowerThanCollectible
		}
	}

	return nil
}
