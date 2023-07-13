package protocol

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/tt"

	"github.com/stretchr/testify/suite"
)

func TestMessengerSyncProfilePictureSuite(t *testing.T) {
	suite.Run(t, new(MessengerSyncProfilePictureSuite))
}

type MessengerSyncProfilePictureSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerSyncProfilePictureSuite) TestSyncProfilePicture() {

	// Add identity images
	keyUID := s.m.account.KeyUID

	// pair
	theirMessenger, err := newMessengerWithKey(s.shh, s.privateKey, s.logger, nil)
	s.Require().NoError(err)

	err = theirMessenger.SetInstallationMetadata(theirMessenger.installationID, &multidevice.InstallationMetadata{
		Name:       "their-name",
		DeviceType: "their-device-type",
	})
	s.Require().NoError(err)
	response, err := theirMessenger.SendPairInstallation(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)
	s.Require().False(response.Chats()[0].Active)

	// Wait for the message to reach its destination
	response, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool { return len(r.Installations) > 0 },
		"installation not received",
	)

	s.Require().NoError(err)
	actualInstallation := response.Installations[0]
	s.Require().Equal(theirMessenger.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("their-name", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("their-device-type", actualInstallation.InstallationMetadata.DeviceType)

	err = s.m.EnableInstallation(theirMessenger.installationID)
	s.Require().NoError(err)

	// Sync happens via subscription triggered from within StoreIdentityImages
	const (
		lowClock  = 5
		highClock = 10
	)

	iis := images.SampleIdentityImages()
	for i := range iis {
		iis[i].Clock = highClock
	}
	s.Require().NoError(s.m.multiAccounts.StoreIdentityImages(keyUID, iis, true))

	// Wait for the message to reach its destination
	err = tt.RetryWithBackOff(func() error {
		response, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		syncedImages, err := theirMessenger.multiAccounts.GetIdentityImages(keyUID)
		if err != nil {
			return err
		}

		if len(syncedImages) == 2 {
			return nil
		}

		return errors.New("not received all identity images")
	})

	s.Require().NoError(err)

	syncedImages, err := theirMessenger.multiAccounts.GetIdentityImages(keyUID)
	s.Require().NoError(err)
	s.Require().Equal(2, len(syncedImages))
	s.Require().Equal(2, len(response.IdentityImages))

	// Check that we don't update images with earlier clock values

	for i := range iis {
		iis[i].Clock = lowClock
	}
	iis2 := images.SampleIdentityImages()
	for i := range iis2 {
		iis2[i].Name = fmt.Sprintf("newimg%d", i)
		iis2[i].Clock = highClock
	}
	iis = append(iis, iis2...)
	s.Require().NoError(s.m.multiAccounts.StoreIdentityImages(keyUID, iis, true))

	err = tt.RetryWithBackOff(func() error {
		response, err = theirMessenger.RetrieveAll()
		if err != nil {
			return err
		}

		syncedImages, err := theirMessenger.multiAccounts.GetIdentityImages(keyUID)
		if err != nil {
			return err
		}

		if len(syncedImages) == 4 {
			return nil
		}

		return errors.New("not received all identity images")
	})

	syncedImages, err = theirMessenger.multiAccounts.GetIdentityImages(keyUID)
	s.Require().NoError(err)
	s.Require().Equal(4, len(syncedImages))
	for _, img := range syncedImages {
		s.Require().NotEqual(img.Clock, lowClock)
	}

	s.Require().NoError(theirMessenger.Shutdown())
}
