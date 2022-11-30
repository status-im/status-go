package wakusync

import (
	"github.com/status-im/status-go/images"
)

type BackedUpProfile struct {
	DisplayName       string                 `json:"displayName,omitempty"`
	DisplayNameStored bool                   `json:"displayNameStored,omitempty"`
	Images            []images.IdentityImage `json:"images,omitempty,omitempty"`
	ImagesStored      bool                   `json:"imagesStored,omitempty"`
}

func (sfwr *WakuBackedUpDataResponse) AddDisplayName(displayName string, stored bool) {
	sfwr.Profile.DisplayName = displayName
	sfwr.Profile.DisplayNameStored = stored
}

func (sfwr *WakuBackedUpDataResponse) AddImages(images []images.IdentityImage, stored bool) {
	sfwr.Profile.ImagesStored = stored
	sfwr.Profile.Images = images
}
