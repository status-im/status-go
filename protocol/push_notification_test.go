package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/pushnotificationclient"
	"github.com/status-im/status-go/protocol/pushnotificationserver"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

const (
	bob1DeviceToken = "token-1"
	bob2DeviceToken = "token-2"
	testAPNTopic    = "topic"
)

func TestMessengerPushNotificationSuite(t *testing.T) {
	suite.Run(t, new(MessengerPushNotificationSuite))
}

type MessengerPushNotificationSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerPushNotificationSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start(nil))

	s.m = s.newMessenger(s.shh)
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerPushNotificationSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
	_ = s.logger.Sync()
}

func (s *MessengerPushNotificationSuite) newMessenger(shh types.Waku) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, []Option{WithPushNotifications()})
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerPushNotificationSuite) newPushNotificationServer(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {

	serverConfig := &pushnotificationserver.Config{
		Enabled:  true,
		Logger:   s.logger,
		Identity: privateKey,
	}

	options := []Option{
		WithPushNotificationServerConfig(serverConfig),
	}
	messenger, err := newMessengerWithKey(shh, privateKey, s.logger, options)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerPushNotificationSuite) TestReceivePushNotification() {

	bob1 := s.m
	bob2, err := newMessengerWithKey(s.shh, s.m.identity, s.logger, []Option{WithPushNotifications()})
	s.Require().NoError(err)

	serverKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	server := s.newPushNotificationServer(s.shh, serverKey)

	alice := s.newMessenger(s.shh)
	// start alice and enable sending push notifications
	_, err = alice.Start()
	s.Require().NoError(err)
	s.Require().NoError(alice.EnableSendingPushNotifications())
	bobInstallationIDs := []string{bob1.installationID, bob2.installationID}

	// Register bob1
	err = bob1.AddPushNotificationsServer(context.Background(), &server.identity.PublicKey, pushnotificationclient.ServerTypeCustom)
	s.Require().NoError(err)

	err = bob1.RegisterForPushNotifications(context.Background(), bob1DeviceToken, testAPNTopic, protobuf.PushNotificationRegistration_APN_TOKEN)

	// Pull servers  and check we registered
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = bob1.RetrieveAll()
		if err != nil {
			return err
		}
		registered, err := bob1.RegisteredForPushNotifications()
		if err != nil {
			return err
		}
		if !registered {
			return errors.New("not registered")
		}
		bobServers, err := bob1.GetPushNotificationsServers()
		if err != nil {
			return err
		}

		if len(bobServers) == 0 {
			return errors.New("not registered")
		}

		return nil
	})
	// Make sure we receive it
	s.Require().NoError(err)
	bob1Servers, err := bob1.GetPushNotificationsServers()
	s.Require().NoError(err)

	// Register bob2
	err = bob2.AddPushNotificationsServer(context.Background(), &server.identity.PublicKey, pushnotificationclient.ServerTypeCustom)
	s.Require().NoError(err)

	err = bob2.RegisterForPushNotifications(context.Background(), bob2DeviceToken, testAPNTopic, protobuf.PushNotificationRegistration_APN_TOKEN)
	s.Require().NoError(err)

	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = bob2.RetrieveAll()
		if err != nil {
			return err
		}

		registered, err := bob2.RegisteredForPushNotifications()
		if err != nil {
			return err
		}
		if !registered {
			return errors.New("not registered")
		}
		bobServers, err := bob2.GetPushNotificationsServers()
		if err != nil {
			return err
		}

		if len(bobServers) == 0 {
			return errors.New("not registered")
		}

		return nil
	})
	// Make sure we receive it
	s.Require().NoError(err)
	bob2Servers, err := bob2.GetPushNotificationsServers()
	s.Require().NoError(err)

	// Create one to one chat & send message
	pkString := hex.EncodeToString(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	chat := CreateOneToOneChat(pkString, &s.m.identity.PublicKey, alice.transport)
	s.Require().NoError(alice.SaveChat(chat))
	inputMessage := buildTestMessage(*chat)
	response, err := alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	messageIDString := response.Messages[0].ID
	messageID, err := hex.DecodeString(messageIDString[2:])
	s.Require().NoError(err)

	infoMap := make(map[string]*pushnotificationclient.PushNotificationInfo)
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}

		info, err := alice.pushNotificationClient.GetPushNotificationInfo(&bob1.identity.PublicKey, bobInstallationIDs)
		if err != nil {
			return err
		}
		for _, i := range info {
			infoMap[i.AccessToken] = i
		}

		// Check we have replies for both bob1 and bob2
		if len(infoMap) != 2 {
			return errors.New("info not fetched")
		}
		return nil

	})

	s.Require().Len(infoMap, 2)

	// Check we have replies for both bob1 and bob2
	var bob1Info, bob2Info *pushnotificationclient.PushNotificationInfo

	bob1Info = infoMap[bob1Servers[0].AccessToken]
	bob2Info = infoMap[bob2Servers[0].AccessToken]

	s.Require().NotNil(bob1Info)
	s.Require().Equal(bob1.installationID, bob1Info.InstallationID)
	s.Require().Equal(bob1Servers[0].AccessToken, bob1Info.AccessToken)
	s.Require().Equal(&bob1.identity.PublicKey, bob1Info.PublicKey)

	s.Require().NotNil(bob2Info)
	s.Require().Equal(bob2.installationID, bob2Info.InstallationID)
	s.Require().Equal(bob2Servers[0].AccessToken, bob2Info.AccessToken)
	s.Require().Equal(&bob2.identity.PublicKey, bob2Info.PublicKey)

	retrievedNotificationInfo, err := alice.pushNotificationClient.GetPushNotificationInfo(&bob1.identity.PublicKey, bobInstallationIDs)

	s.Require().NoError(err)
	s.Require().NotNil(retrievedNotificationInfo)
	s.Require().Len(retrievedNotificationInfo, 2)

	var sentNotification *pushnotificationclient.SentNotification
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}
		sentNotification, err = alice.pushNotificationClient.GetSentNotification(common.HashPublicKey(&bob1.identity.PublicKey), bob1.installationID, messageID)
		if err != nil {
			return err
		}
		if sentNotification == nil {
			return errors.New("sent notification not found")
		}
		if !sentNotification.Success {
			return errors.New("sent notification not successul")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().NoError(bob2.Shutdown())
	s.Require().NoError(alice.Shutdown())
	s.Require().NoError(server.Shutdown())
}

func (s *MessengerPushNotificationSuite) TestReceivePushNotificationFromContactOnly() {

	bob := s.m

	serverKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	server := s.newPushNotificationServer(s.shh, serverKey)

	alice := s.newMessenger(s.shh)
	// start alice and enable push notifications
	_, err = alice.Start()
	s.Require().NoError(err)
	s.Require().NoError(alice.EnableSendingPushNotifications())
	bobInstallationIDs := []string{bob.installationID}

	// Register bob
	err = bob.AddPushNotificationsServer(context.Background(), &server.identity.PublicKey, pushnotificationclient.ServerTypeCustom)
	s.Require().NoError(err)

	// Add alice has a contact
	aliceContact := &Contact{
		ID:         types.EncodeHex(crypto.FromECDSAPub(&alice.identity.PublicKey)),
		Name:       "Some Contact",
		SystemTags: []string{contactAdded},
	}

	err = bob.SaveContact(aliceContact)
	s.Require().NoError(err)

	// Enable from contacts only
	err = bob.EnablePushNotificationsFromContactsOnly()
	s.Require().NoError(err)

	err = bob.RegisterForPushNotifications(context.Background(), bob1DeviceToken, testAPNTopic, protobuf.PushNotificationRegistration_APN_TOKEN)
	s.Require().NoError(err)

	// Pull servers  and check we registered
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = bob.RetrieveAll()
		if err != nil {
			return err
		}
		registered, err := bob.RegisteredForPushNotifications()
		if err != nil {
			return err
		}
		if !registered {
			return errors.New("not registered")
		}
		bobServers, err := bob.GetPushNotificationsServers()
		if err != nil {
			return err
		}

		if len(bobServers) == 0 {
			return errors.New("not registered")
		}

		return nil
	})
	// Make sure we receive it
	s.Require().NoError(err)
	bobServers, err := bob.GetPushNotificationsServers()
	s.Require().NoError(err)

	// Create one to one chat & send message
	pkString := hex.EncodeToString(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	chat := CreateOneToOneChat(pkString, &s.m.identity.PublicKey, alice.transport)
	s.Require().NoError(alice.SaveChat(chat))
	inputMessage := buildTestMessage(*chat)
	response, err := alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	messageIDString := response.Messages[0].ID
	messageID, err := hex.DecodeString(messageIDString[2:])
	s.Require().NoError(err)

	var info []*pushnotificationclient.PushNotificationInfo
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}

		info, err = alice.pushNotificationClient.GetPushNotificationInfo(&bob.identity.PublicKey, bobInstallationIDs)
		if err != nil {
			return err
		}
		// Check we have replies for bob
		if len(info) != 1 {
			return errors.New("info not fetched")
		}
		return nil

	})
	s.Require().NoError(err)

	s.Require().NotNil(info)
	s.Require().Equal(bob.installationID, info[0].InstallationID)
	s.Require().Equal(bobServers[0].AccessToken, info[0].AccessToken)
	s.Require().Equal(&bob.identity.PublicKey, info[0].PublicKey)

	retrievedNotificationInfo, err := alice.pushNotificationClient.GetPushNotificationInfo(&bob.identity.PublicKey, bobInstallationIDs)
	s.Require().NoError(err)
	s.Require().NotNil(retrievedNotificationInfo)
	s.Require().Len(retrievedNotificationInfo, 1)

	var sentNotification *pushnotificationclient.SentNotification
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}
		sentNotification, err = alice.pushNotificationClient.GetSentNotification(common.HashPublicKey(&bob.identity.PublicKey), bob.installationID, messageID)
		if err != nil {
			return err
		}
		if sentNotification == nil {
			return errors.New("sent notification not found")
		}
		if !sentNotification.Success {
			return errors.New("sent notification not successul")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().NoError(alice.Shutdown())
	s.Require().NoError(server.Shutdown())
}

func (s *MessengerPushNotificationSuite) TestReceivePushNotificationRetries() {

	bob := s.m

	serverKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	server := s.newPushNotificationServer(s.shh, serverKey)

	alice := s.newMessenger(s.shh)
	// another contact to invalidate the token
	frank := s.newMessenger(s.shh)
	_, err = frank.Start()
	s.Require().NoError(err)
	// start alice and enable push notifications
	_, err = alice.Start()
	s.Require().NoError(err)
	s.Require().NoError(alice.EnableSendingPushNotifications())
	bobInstallationIDs := []string{bob.installationID}

	// Register bob
	err = bob.AddPushNotificationsServer(context.Background(), &server.identity.PublicKey, pushnotificationclient.ServerTypeCustom)
	s.Require().NoError(err)

	// Add alice has a contact
	aliceContact := &Contact{
		ID:         types.EncodeHex(crypto.FromECDSAPub(&alice.identity.PublicKey)),
		Name:       "Some Contact",
		SystemTags: []string{contactAdded},
	}

	err = bob.SaveContact(aliceContact)
	s.Require().NoError(err)

	// Add frank has a contact
	frankContact := &Contact{
		ID:         types.EncodeHex(crypto.FromECDSAPub(&frank.identity.PublicKey)),
		Name:       "Some Contact",
		SystemTags: []string{contactAdded},
	}

	err = bob.SaveContact(frankContact)
	s.Require().NoError(err)

	// Enable from contacts only
	err = bob.EnablePushNotificationsFromContactsOnly()
	s.Require().NoError(err)

	err = bob.RegisterForPushNotifications(context.Background(), bob1DeviceToken, testAPNTopic, protobuf.PushNotificationRegistration_APN_TOKEN)
	s.Require().NoError(err)

	// Pull servers  and check we registered
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = bob.RetrieveAll()
		if err != nil {
			return err
		}
		registered, err := bob.RegisteredForPushNotifications()
		if err != nil {
			return err
		}
		if !registered {
			return errors.New("not registered")
		}
		bobServers, err := bob.GetPushNotificationsServers()
		if err != nil {
			return err
		}

		if len(bobServers) == 0 {
			return errors.New("not registered")
		}

		return nil
	})
	// Make sure we receive it
	s.Require().NoError(err)
	bobServers, err := bob.GetPushNotificationsServers()
	s.Require().NoError(err)

	// Create one to one chat & send message
	pkString := hex.EncodeToString(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	chat := CreateOneToOneChat(pkString, &s.m.identity.PublicKey, alice.transport)
	s.Require().NoError(alice.SaveChat(chat))
	inputMessage := buildTestMessage(*chat)
	_, err = alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// We check that alice retrieves the info from the server
	var info []*pushnotificationclient.PushNotificationInfo
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}

		info, err = alice.pushNotificationClient.GetPushNotificationInfo(&bob.identity.PublicKey, bobInstallationIDs)
		if err != nil {
			return err
		}
		// Check we have replies for bob
		if len(info) != 1 {
			return errors.New("info not fetched")
		}
		return nil

	})
	s.Require().NoError(err)

	s.Require().NotNil(info)
	s.Require().Equal(bob.installationID, info[0].InstallationID)
	s.Require().Equal(bobServers[0].AccessToken, info[0].AccessToken)
	s.Require().Equal(&bob.identity.PublicKey, info[0].PublicKey)

	// The message has been sent, but not received, now we remove a contact so that the token is invalidated
	frankContact = &Contact{
		ID:         types.EncodeHex(crypto.FromECDSAPub(&frank.identity.PublicKey)),
		Name:       "Some Contact",
		SystemTags: []string{},
	}
	err = bob.SaveContact(frankContact)
	s.Require().NoError(err)

	// Re-registration should be triggered, pull from server and bob to check we are correctly registered
	// Pull servers  and check we registered
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = bob.RetrieveAll()
		if err != nil {
			return err
		}
		registered, err := bob.RegisteredForPushNotifications()
		if err != nil {
			return err
		}
		if !registered {
			return errors.New("not registered")
		}
		return nil
	})

	newBobServers, err := bob.GetPushNotificationsServers()
	s.Require().NoError(err)
	// Make sure access token is not the same
	s.Require().NotEqual(newBobServers[0].AccessToken, bobServers[0].AccessToken)

	// Send another message, here the token will not be valid
	inputMessage = buildTestMessage(*chat)
	response, err := alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	messageIDString := response.Messages[0].ID
	messageID, err := hex.DecodeString(messageIDString[2:])
	s.Require().NoError(err)

	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}

		info, err = alice.pushNotificationClient.GetPushNotificationInfo(&bob.identity.PublicKey, bobInstallationIDs)
		if err != nil {
			return err
		}
		// Check we have replies for bob
		if len(info) != 1 {
			return errors.New("info not fetched")
		}
		if newBobServers[0].AccessToken != info[0].AccessToken {
			return errors.New("still using the old access token")
		}
		return nil

	})
	s.Require().NoError(err)

	s.Require().NotNil(info)
	s.Require().Equal(bob.installationID, info[0].InstallationID)
	s.Require().Equal(newBobServers[0].AccessToken, info[0].AccessToken)
	s.Require().Equal(&bob.identity.PublicKey, info[0].PublicKey)

	retrievedNotificationInfo, err := alice.pushNotificationClient.GetPushNotificationInfo(&bob.identity.PublicKey, bobInstallationIDs)
	s.Require().NoError(err)
	s.Require().NotNil(retrievedNotificationInfo)
	s.Require().Len(retrievedNotificationInfo, 1)

	var sentNotification *pushnotificationclient.SentNotification
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}
		sentNotification, err = alice.pushNotificationClient.GetSentNotification(common.HashPublicKey(&bob.identity.PublicKey), bob.installationID, messageID)
		if err != nil {
			return err
		}
		if sentNotification == nil {
			return errors.New("sent notification not found")
		}
		if !sentNotification.Success {
			return errors.New("sent notification not successul")
		}
		return nil
	})

	s.Require().NoError(err)
	s.Require().NoError(alice.Shutdown())
	s.Require().NoError(server.Shutdown())
}

func (s *MessengerPushNotificationSuite) TestContactCode() {

	bob1 := s.m

	serverKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	server := s.newPushNotificationServer(s.shh, serverKey)

	alice := s.newMessenger(s.shh)
	// start alice and enable sending push notifications
	_, err = alice.Start()
	s.Require().NoError(err)
	s.Require().NoError(alice.EnableSendingPushNotifications())

	// Register bob1
	err = bob1.AddPushNotificationsServer(context.Background(), &server.identity.PublicKey, pushnotificationclient.ServerTypeCustom)
	s.Require().NoError(err)

	err = bob1.RegisterForPushNotifications(context.Background(), bob1DeviceToken, testAPNTopic, protobuf.PushNotificationRegistration_APN_TOKEN)

	// Pull servers  and check we registered
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = bob1.RetrieveAll()
		if err != nil {
			return err
		}
		registered, err := bob1.RegisteredForPushNotifications()
		if err != nil {
			return err
		}
		if !registered {
			return errors.New("not registered")
		}
		bobServers, err := bob1.GetPushNotificationsServers()
		if err != nil {
			return err
		}

		if len(bobServers) == 0 {
			return errors.New("not registered")
		}

		return nil
	})
	// Make sure we receive it
	s.Require().NoError(err)

	contactCodeAdvertisement, err := bob1.buildContactCodeAdvertisement()
	s.Require().NoError(err)
	s.Require().NotNil(contactCodeAdvertisement)

	s.Require().NoError(alice.pushNotificationClient.HandleContactCodeAdvertisement(&bob1.identity.PublicKey, *contactCodeAdvertisement))

	s.Require().NoError(alice.Shutdown())
	s.Require().NoError(server.Shutdown())
}

func (s *MessengerPushNotificationSuite) TestReceivePushNotificationMention() {

	bob := s.m

	serverKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	server := s.newPushNotificationServer(s.shh, serverKey)

	alice := s.newMessenger(s.shh)
	// start alice and enable sending push notifications
	_, err = alice.Start()
	s.Require().NoError(err)
	s.Require().NoError(alice.EnableSendingPushNotifications())
	bobInstallationIDs := []string{bob.installationID}

	// Create public chat and join for both alice and bob
	chat := CreatePublicChat("status", s.m.transport)
	err = bob.SaveChat(chat)
	s.Require().NoError(err)

	_, err = bob.Join(chat)
	s.Require().NoError(err)

	err = alice.SaveChat(chat)
	s.Require().NoError(err)

	_, err = alice.Join(chat)
	s.Require().NoError(err)

	// Register bob
	err = bob.AddPushNotificationsServer(context.Background(), &server.identity.PublicKey, pushnotificationclient.ServerTypeCustom)
	s.Require().NoError(err)

	err = bob.RegisterForPushNotifications(context.Background(), bob1DeviceToken, testAPNTopic, protobuf.PushNotificationRegistration_APN_TOKEN)

	// Pull servers  and check we registered
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = bob.RetrieveAll()
		if err != nil {
			return err
		}
		registered, err := bob.RegisteredForPushNotifications()
		if err != nil {
			return err
		}
		if !registered {
			return errors.New("not registered")
		}

		bobServers, err := bob.GetPushNotificationsServers()
		if err != nil {
			return err
		}

		if len(bobServers) == 0 {
			return errors.New("not registered")
		}

		return nil
	})
	// Make sure we receive it
	s.Require().NoError(err)
	bobServers, err := bob.GetPushNotificationsServers()
	s.Require().NoError(err)

	inputMessage := buildTestMessage(*chat)
	// message contains a mention
	inputMessage.Text = "Hey @" + types.EncodeHex(crypto.FromECDSAPub(&bob.identity.PublicKey))
	response, err := alice.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	messageIDString := response.Messages[0].ID
	messageID, err := hex.DecodeString(messageIDString[2:])
	s.Require().NoError(err)

	var bobInfo []*pushnotificationclient.PushNotificationInfo
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}

		bobInfo, err = alice.pushNotificationClient.GetPushNotificationInfo(&bob.identity.PublicKey, bobInstallationIDs)
		if err != nil {
			return err
		}
		// Check we have replies for bob
		if len(bobInfo) != 1 {
			return errors.New("info not fetched")
		}
		return nil

	})

	s.Require().NoError(err)

	s.Require().NotEmpty(bobInfo)
	s.Require().Equal(bob.installationID, bobInfo[0].InstallationID)
	s.Require().Equal(bobServers[0].AccessToken, bobInfo[0].AccessToken)
	s.Require().Equal(&bob.identity.PublicKey, bobInfo[0].PublicKey)

	retrievedNotificationInfo, err := alice.pushNotificationClient.GetPushNotificationInfo(&bob.identity.PublicKey, bobInstallationIDs)

	s.Require().NoError(err)
	s.Require().NotNil(retrievedNotificationInfo)
	s.Require().Len(retrievedNotificationInfo, 1)

	var sentNotification *pushnotificationclient.SentNotification
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}
		sentNotification, err = alice.pushNotificationClient.GetSentNotification(common.HashPublicKey(&bob.identity.PublicKey), bob.installationID, messageID)
		if err != nil {
			return err
		}
		if sentNotification == nil {
			return errors.New("sent notification not found")
		}
		if !sentNotification.Success {
			return errors.New("sent notification not successul")
		}
		return nil
	})
	s.Require().NoError(err)
	s.Require().NoError(alice.Shutdown())
	s.Require().NoError(server.Shutdown())
}

func (s *MessengerPushNotificationSuite) TestReceivePushNotificationCommunityRequest() {

	bob := s.m

	serverKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	server := s.newPushNotificationServer(s.shh, serverKey)

	alice := s.newMessenger(s.shh)
	// start alice and enable sending push notifications
	_, err = alice.Start()
	s.Require().NoError(err)
	s.Require().NoError(alice.EnableSendingPushNotifications())

	// Register bob
	err = bob.AddPushNotificationsServer(context.Background(), &server.identity.PublicKey, pushnotificationclient.ServerTypeCustom)
	s.Require().NoError(err)

	err = bob.RegisterForPushNotifications(context.Background(), bob1DeviceToken, testAPNTopic, protobuf.PushNotificationRegistration_APN_TOKEN)

	// Pull servers  and check we registered
	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = bob.RetrieveAll()
		if err != nil {
			return err
		}
		registered, err := bob.RegisteredForPushNotifications()
		if err != nil {
			return err
		}
		if !registered {
			return errors.New("not registered")
		}

		bobServers, err := bob.GetPushNotificationsServers()
		if err != nil {
			return err
		}

		if len(bobServers) == 0 {
			return errors.New("not registered")
		}

		return nil
	})
	// Make sure we receive it
	s.Require().NoError(err)
	_, err = bob.GetPushNotificationsServers()
	s.Require().NoError(err)

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_ON_REQUEST,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	response, err := bob.CreateCommunity(description)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	community := response.Communities()[0]

	// Send an community message
	chat := CreateOneToOneChat(common.PubkeyToHex(&alice.identity.PublicKey), &alice.identity.PublicKey, alice.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err = bob.SaveChat(chat)
	s.NoError(err)
	_, err = bob.SendChatMessage(context.Background(), inputMessage)
	s.NoError(err)

	// Pull message and make sure org is received
	err = tt.RetryWithBackOff(func() error {
		response, err = alice.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}

	// We try to join the org
	response, err = alice.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin1 := response.RequestsToJoinCommunity[0]
	s.Require().NotNil(requestToJoin1)
	s.Require().Equal(community.ID(), requestToJoin1.CommunityID)
	s.Require().True(requestToJoin1.Our)
	s.Require().NotEmpty(requestToJoin1.ID)
	s.Require().NotEmpty(requestToJoin1.Clock)
	s.Require().Equal(requestToJoin1.PublicKey, common.PubkeyToHex(&alice.identity.PublicKey))
	s.Require().Equal(communities.RequestToJoinStatePending, requestToJoin1.State)

	err = tt.RetryWithBackOff(func() error {
		_, err = server.RetrieveAll()
		if err != nil {
			return err
		}
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}

		if server.pushNotificationServer.SentRequests != 1 {
			return errors.New("request not sent")
		}

		return nil

	})

	s.Require().NoError(err)

	s.Require().NoError(alice.Shutdown())
	s.Require().NoError(server.Shutdown())
}
