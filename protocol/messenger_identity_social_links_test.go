package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"testing"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/types"
)

func TestMessengerSocialLinksHandlerSuite(t *testing.T) {
	suite.Run(t, new(MessengerSocialLinksHandlerSuite))
}

type MessengerSocialLinksHandlerSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh types.Waku

	logger *zap.Logger
}

func (s *MessengerSocialLinksHandlerSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
	// We start the messenger in order to receive installations
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerSocialLinksHandlerSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerSocialLinksHandlerSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)

	return messenger
}

func profileSocialLinks() identity.SocialLinks {
	return identity.SocialLinks{
		{
			Text: identity.TwitterID,
			URL:  "https://twitter.com/ethstatus",
		},
		{
			Text: identity.TwitterID,
			URL:  "https://twitter.com/StatusIMBlog",
		},
		{
			Text: identity.TelegramID,
			URL:  "dummy.telegram",
		},
		{
			Text: identity.YoutubeID,
			URL:  "https://www.youtube.com/@Statusim",
		},
		{
			Text: identity.YoutubeID,
			URL:  "https://www.youtube.com/@EthereumProtocol",
		},
		{
			Text: "customLink",
			URL:  "customLink.com",
		},
	}
}

func (s *MessengerSocialLinksHandlerSuite) TestSocialLinks() {
	// db is empty for alice
	links, err := s.m.settings.GetSocialLinks()
	s.Require().NoError(err)
	s.Require().Len(links, 0)

	// Create new device
	alicesOtherDevice, err := newMessengerWithKey(s.shh, s.m.identity, s.logger, nil)
	s.Require().NoError(err)

	// db is empty for alice's other device
	links, err = alicesOtherDevice.settings.GetSocialLinks()
	s.Require().NoError(err)
	s.Require().Len(links, 0)

	// Pair devices
	im1 := &multidevice.InstallationMetadata{
		Name:       "alice's-other-device",
		DeviceType: "alice's-other-device-type",
	}
	err = alicesOtherDevice.SetInstallationMetadata(alicesOtherDevice.installationID, im1)
	s.Require().NoError(err)
	response, err := alicesOtherDevice.SendPairInstallation(context.Background(), nil)
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
	s.Require().Equal(alicesOtherDevice.installationID, actualInstallation.ID)
	s.Require().NotNil(actualInstallation.InstallationMetadata)
	s.Require().Equal("alice's-other-device", actualInstallation.InstallationMetadata.Name)
	s.Require().Equal("alice's-other-device-type", actualInstallation.InstallationMetadata.DeviceType)

	err = s.m.EnableInstallation(alicesOtherDevice.installationID)
	s.Require().NoError(err)

	// Set social links on alice's device
	profileSocialLinks := profileSocialLinks()
	err = s.m.AddOrReplaceSocialLinks(profileSocialLinks)
	s.Require().NoError(err)

	err = tt.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if response.SocialLinksInfo == nil ||
			len(response.SocialLinksInfo.Links) != len(profileSocialLinks) {
			return errors.New("no sync data received")
		}
		return nil
	})
	s.Require().NoError(err)

	// Check social links on alice's other device
	links, err = alicesOtherDevice.settings.GetSocialLinks()
	s.Require().NoError(err)
	s.Require().Len(links, len(profileSocialLinks))

	// Delete some social links on alice's device
	removeLinkAtIndex := 2
	removedLink := profileSocialLinks[removeLinkAtIndex]
	profileSocialLinks = append(profileSocialLinks[:removeLinkAtIndex], profileSocialLinks[removeLinkAtIndex+1:]...)

	err = s.m.AddOrReplaceSocialLinks(profileSocialLinks)
	s.Require().NoError(err)

	err = tt.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if response.SocialLinksInfo == nil ||
			len(response.SocialLinksInfo.Links) != len(profileSocialLinks) {
			return errors.New("no sync data received")
		}
		return nil
	})
	s.Require().NoError(err)

	// Check social links on alice's other device
	links, err = alicesOtherDevice.settings.GetSocialLinks()
	s.Require().NoError(err)
	s.Require().Len(links, len(profileSocialLinks))
	s.Require().False(links.Contains(removedLink))

	// Delete all social links on alice's device
	profileSocialLinks = identity.SocialLinks{}
	err = s.m.AddOrReplaceSocialLinks(profileSocialLinks)
	s.Require().NoError(err)

	err = tt.RetryWithBackOff(func() error {
		response, err := alicesOtherDevice.RetrieveAll()
		if err != nil {
			return err
		}

		if response.SocialLinksInfo == nil ||
			!response.SocialLinksInfo.Removed {
			return errors.New("no sync data received")
		}
		return nil
	})
	s.Require().NoError(err)

	// Check social links on alice's other device
	links, err = alicesOtherDevice.settings.GetSocialLinks()
	s.Require().NoError(err)
	s.Require().Len(links, 0)
}
