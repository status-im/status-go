package accounts

import (
	"errors"

	"github.com/status-im/status-go/timesource"

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
	oldAcc, err := api.db.GetAccount(account.KeyUID)
	if err != nil {
		return err
	}
	if oldAcc == nil {
		return errors.New("UpdateAccount but account not found")
	}
	if oldAcc.CustomizationColor != account.CustomizationColor {
		updatedAt := timesource.GetCurrentTimeInMillis()
		account.CustomizationColorClock = updatedAt
	}
	return api.db.UpdateAccount(account)
}

//
// Profile Images
//

func (api *MultiAccountsAPI) setLocalURLOnImage(keyUID string, image *images.IdentityImage) {
	image.LocalURL = api.mediaServer.MakeAccountImageURL(keyUID, image.Name, image.Clock)
}

func (api *MultiAccountsAPI) setLocalURLOnImages(keyUID string, iis []*images.IdentityImage) {
	for k := range iis {
		if iis[k] == nil {
			continue
		}
		api.setLocalURLOnImage(keyUID, iis[k])
	}
}

func (api *MultiAccountsAPI) setLocalURLOnImages2(keyUID string, iis []images.IdentityImage) {
	for k := range iis {
		api.setLocalURLOnImage(keyUID, &iis[k])
	}
}

// GetIdentityImages returns an array of json marshalled IdentityImages assigned to the user's identity
func (api *MultiAccountsAPI) GetIdentityImages(keyUID string) ([]*images.IdentityImage, error) {
	images, err := api.db.GetIdentityImages(keyUID)
	if err != nil {
		return nil, err
	}
	api.setLocalURLOnImages(keyUID, images)
	return images, nil
}

// GetIdentityImage returns a json object representing the image with the given name
func (api *MultiAccountsAPI) GetIdentityImage(keyUID, name string) (*images.IdentityImage, error) {
	image, err := api.db.GetIdentityImage(keyUID, name)

	if err != nil {
		return nil, err
	}
	if image == nil {
		return nil, nil
	}
	api.setLocalURLOnImage(keyUID, image)
	return image, nil
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

	api.setLocalURLOnImages2(keyUID, iis)

	return iis, nil
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

	api.setLocalURLOnImages2(keyUID, iis)

	return iis, nil
}

// DeleteIdentityImage deletes an IdentityImage from the db with the given name
func (api *MultiAccountsAPI) DeleteIdentityImage(keyUID string) error {
	return api.db.DeleteIdentityImage(keyUID)
}
