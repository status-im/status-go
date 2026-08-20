package backupsync

import (
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/protocol/identity"
	"github.com/status-im/status-go/services/ens"
)

type BackedUpProfile struct {
	DisplayName                string                              `json:"displayName,omitempty"`
	Images                     []images.IdentityImage              `json:"images,omitempty"`
	EnsUsernameDetails         []*ens.UsernameDetail               `json:"ensUsernameDetails,omitempty"`
	ProfileShowcasePreferences identity.ProfileShowcasePreferences `json:"profile_showcase_preferences,omitempty"`
}

func (sfwr *BackedUpDataResponse) SetDisplayName(displayName string) {
	sfwr.Profile.DisplayName = displayName
}

func (sfwr *BackedUpDataResponse) SetImages(images []images.IdentityImage) {
	sfwr.Profile.Images = images
}

func (sfwr *BackedUpDataResponse) SetEnsUsernameDetails(ensUsernameDetails []*ens.UsernameDetail) {
	sfwr.Profile.EnsUsernameDetails = ensUsernameDetails
}

func (sfwr *BackedUpDataResponse) SetProfileShowcasePreferences(profileShowcasePreferences *identity.ProfileShowcasePreferences) {
	sfwr.Profile.ProfileShowcasePreferences = *profileShowcasePreferences
}
