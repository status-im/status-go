package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
)

func newCommunitiesTestMessenger(shh types.Waku, privateKey *ecdsa.PrivateKey, logger *zap.Logger, accountsManager account.Manager, tokenManager communities.TokenManager) (*Messenger, error) {
	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	if err != nil {
		return nil, err
	}
	madb, err := multiaccounts.InitializeDB(tmpfile.Name())
	if err != nil {
		return nil, err
	}

	acc := generator.NewAccount(privateKey, nil)
	iai := acc.ToIdentifiedAccountInfo("")

	options := []Option{
		WithCustomLogger(logger),
		WithDatabaseConfig(":memory:", "somekey", sqlite.ReducedKDFIterationsNumber),
		WithMultiAccounts(madb),
		WithAccount(iai.ToMultiAccount()),
		WithDatasync(),
		WithTokenManager(tokenManager),
	}

	m, err := NewMessenger(
		"Test",
		privateKey,
		&testNode{shh: shh},
		uuid.New().String(),
		nil,
		accountsManager,
		options...,
	)
	if err != nil {
		return nil, err
	}

	err = m.Init()
	if err != nil {
		return nil, err
	}

	config := params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}

	networks := json.RawMessage("{}")
	setting := settings.Settings{
		Address:                   types.HexToAddress("0x1122334455667788990011223344556677889900"),
		AnonMetricsShouldSend:     false,
		CurrentNetwork:            "mainnet_rpc",
		DappsAddress:              types.HexToAddress("0x1122334455667788990011223344556677889900"),
		InstallationID:            "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:                    "0x1122334455667788990011223344556677889900",
		Name:                      "Test",
		Networks:                  &networks,
		LatestDerivedPath:         0,
		PhotoPath:                 "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreviewPrivacy:            false,
		PublicKey:                 "0x04112233445566778899001122334455667788990011223344556677889900112233445566778899001122334455667788990011223344556677889900",
		SigningPhrase:             "yurt joey vibe",
		SendPushNotifications:     true,
		ProfilePicturesVisibility: 1,
		DefaultSyncPeriod:         777600,
		UseMailservers:            true,
		LinkPreviewRequestEnabled: true,
		SendStatusUpdates:         true,
		WalletRootAddress:         types.HexToAddress("0x1122334455667788990011223344556677889900")}

	_ = m.settings.CreateSettings(setting, config)

	return m, nil
}

func createCommunity(s *suite.Suite, owner *Messenger) (*communities.Community, *Chat) {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := owner.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	community := response.Communities()[0]
	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "😎",
			Description: "status-core community chat",
		},
	}

	response, err = owner.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)

	return community, response.Chats()[0]
}

func advertiseCommunityTo(s *suite.Suite, community *communities.Community, owner *Messenger, user *Messenger) {
	chat := CreateOneToOneChat(common.PubkeyToHex(&user.identity.PublicKey), &user.identity.PublicKey, user.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err := owner.SaveChat(chat)
	s.Require().NoError(err)
	_, err = owner.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Ensure community is received
	err = tt.RetryWithBackOff(func() error {
		response, err := user.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})
	s.Require().NoError(err)
}

func joinCommunity(s *suite.Suite, community *communities.Community, owner *Messenger, user *Messenger, request *requests.RequestToJoinCommunity) {
	response, err := user.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

	// Retrieve and accept join request
	err = tt.RetryWithBackOff(func() error {
		response, err := owner.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities in response (accept join request)")
		}
		if !response.Communities()[0].HasMember(&user.identity.PublicKey) {
			return errors.New("user not accepted")
		}
		return nil
	})
	s.Require().NoError(err)

	// Retrieve join request response
	err = tt.RetryWithBackOff(func() error {
		response, err := user.RetrieveAll()

		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities in response (join request response)")
		}
		if !response.Communities()[0].HasMember(&user.identity.PublicKey) {
			return errors.New("user not a member")
		}
		return nil
	})
	s.Require().NoError(err)
}

func joinOnRequestCommunity(s *suite.Suite, community *communities.Community, controlNode *Messenger, user *Messenger) {
	// Request to join the community
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin := response.RequestsToJoinCommunity[0]
	s.Require().Equal(requestToJoin.PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	response, err = WaitOnMessengerResponse(
		controlNode,
		func(r *MessengerResponse) bool {
			return len(r.RequestsToJoinCommunity) > 0
		},
		"control node did not receive community request to join",
	)
	s.Require().NoError(err)

	userRequestToJoin := response.RequestsToJoinCommunity[0]
	s.Require().Equal(userRequestToJoin.PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// accept join request
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoin.ID}
	response, err = controlNode.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	updatedCommunity := response.Communities()[0]
	s.Require().NotNil(updatedCommunity)
	s.Require().True(updatedCommunity.HasMember(&user.identity.PublicKey))

	// receive request to join response
	_, err = WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0
		},
		"user did not receive request to join response",
	)
	s.Require().NoError(err)

	userCommunity, err := user.GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(userCommunity.HasMember(&user.identity.PublicKey))

	// We can't identify which owner is a control node, so owner will receive twice request to join event
	_, err = WaitOnMessengerResponse(
		controlNode,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0
		},
		"user did not receive request to join response",
	)
	s.Require().NoError(err)
}

func sendChatMessage(s *suite.Suite, sender *Messenger, chatID string, text string) *common.Message {
	msg := &common.Message{
		ChatMessage: protobuf.ChatMessage{
			ChatId:      chatID,
			ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			Text:        text,
		},
	}

	_, err := sender.SendChatMessage(context.Background(), msg)
	s.Require().NoError(err)

	return msg
}

func grantPermission(s *suite.Suite, community *communities.Community, controlNode *Messenger, target *Messenger, role protobuf.CommunityMember_Roles) {
	responseAddRole, err := controlNode.AddRoleToMember(&requests.AddRoleToMember{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(target.IdentityPublicKey()),
		Role:        role,
	})
	s.Require().NoError(err)

	checkRole := func(response *MessengerResponse) bool {
		if len(response.Communities()) == 0 {
			return false
		}
		rCommunities := response.Communities()
		s.Require().Len(rCommunities, 1)
		switch role {
		case protobuf.CommunityMember_ROLE_OWNER:
			s.Require().True(rCommunities[0].IsMemberOwner(target.IdentityPublicKey()))
		case protobuf.CommunityMember_ROLE_ADMIN:
			s.Require().True(rCommunities[0].IsMemberAdmin(target.IdentityPublicKey()))
		case protobuf.CommunityMember_ROLE_TOKEN_MASTER:
			s.Require().True(rCommunities[0].IsMemberTokenMaster(target.IdentityPublicKey()))
		default:
			return false
		}

		return true
	}

	s.Require().True(checkRole(responseAddRole))

	response, err := WaitOnMessengerResponse(target, func(response *MessengerResponse) bool {
		return len(response.Communities()) > 0
	}, "community description changed message not received")
	s.Require().NoError(err)
	s.Require().True(checkRole(response))
}
