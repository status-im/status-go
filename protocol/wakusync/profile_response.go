package wakusync

import (
	"github.com/status-im/status-go/images"
)

type BackedUpProfile struct {
	DisplayName string                 `json:"displayName,omitempty"`
	Images      []images.IdentityImage `json:"images,omitempty"`
}

func (sfwr *WakuBackedUpDataResponse) AddDisplayName(displayName string) {
	sfwr.Profile.DisplayName = displayName
}

func (sfwr *WakuBackedUpDataResponse) AddImages(images []images.IdentityImage) {
	sfwr.Profile.Images = images
}
