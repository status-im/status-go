package wakusync

import (
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/identity"
)

type BackedUpProfile struct {
	DisplayName string                 `json:"displayName,omitempty"`
	Images      []images.IdentityImage `json:"images,omitempty"`
	SocialLinks identity.SocialLinks   `json:"socialLinks,omitempty"`
}

func (sfwr *WakuBackedUpDataResponse) SetDisplayName(displayName string) {
	sfwr.Profile.DisplayName = displayName
}

func (sfwr *WakuBackedUpDataResponse) SetImages(images []images.IdentityImage) {
	sfwr.Profile.Images = images
}

func (sfwr *WakuBackedUpDataResponse) SetSocialLinks(socialLinks identity.SocialLinks) {
	sfwr.Profile.SocialLinks = socialLinks
}
