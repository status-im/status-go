package accounts

import (
	"errors"

	"github.com/status-im/status-go/server"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
)

var (
	// ErrUpdatingWrongAccount raised if caller tries to update any other account except one used for login.
	ErrUpdatingWrongAccount = errors.New("failed to update wrong account. Please login with that account first")
)

func NewMultiAccountsAPI(db *multiaccounts.Database, mediaServer *server.MediaServer) *MultiAccountsAPI {
	return &MultiAccountsAPI{db: db, mediaServer: mediaServer}
}

// MultiAccountsAPI is class with methods available over RPC.
type MultiAccountsAPI struct {
	db          *multiaccounts.Database
	mediaServer *server.MediaServer
}

func (api *MultiAccountsAPI) UpdateAccount(account multiaccounts.Account) error {
	return api.db.UpdateAccount(account)
}

//
// Profile Images
//

// GetIdentityImages returns an array of json marshalled IdentityImages assigned to the user's identity
func (api *MultiAccountsAPI) GetIdentityImages(keyUID string) ([]*images.IdentityImage, error) {
	images, err := api.db.GetIdentityImages(keyUID)
	for i, _ := range images {
		images[i].RingUrl = api.mediaServer.MakeDrawRingURL(keyUID, server.DrawRingTypeAccount, images[i].Name)
	}
	return images, err
}

// GetIdentityImage returns a json object representing the image with the given name
func (api *MultiAccountsAPI) GetIdentityImage(keyUID, name string) (*images.IdentityImage, error) {
	image, err := api.db.GetIdentityImage(keyUID, name)
	if image != nil {
		image.RingUrl = api.mediaServer.MakeDrawRingURL(keyUID, server.DrawRingTypeAccount, image.Name)
	}
	return image, err
}

// StoreIdentityImage takes the filepath of an image, crops it as per the rect coords and finally resizes the image.
// The resulting image(s) will be stored in the DB along with other user account information.
// aX and aY represent the pixel coordinates of the upper left corner of the image's cropping area
// bX and bY represent the pixel coordinates of the lower right corner of the image's cropping area
func (api *MultiAccountsAPI) StoreIdentityImage(keyUID, filepath string, aX, aY, bX, bY int) ([]images.IdentityImage, error) {
	iis, err := images.GenerateIdentityImages(filepath, aX, aY, bX, bY)
	if err != nil {
		return nil, err
	}

	err = api.db.StoreIdentityImages(keyUID, iis, true)
	if err != nil {
		return nil, err
	}

	return iis, err
}

func (api *MultiAccountsAPI) StoreIdentityImageFromURL(keyUID, url string) ([]images.IdentityImage, error) {
	iis, err := images.GenerateIdentityImagesFromURL(url)
	if err != nil {
		return nil, err
	}

	err = api.db.StoreIdentityImages(keyUID, iis, true)
	if err != nil {
		return nil, err
	}

	return iis, err
}

// DeleteIdentityImage deletes an IdentityImage from the db with the given name
func (api *MultiAccountsAPI) DeleteIdentityImage(keyUID string) error {
	return api.db.DeleteIdentityImage(keyUID)
}
