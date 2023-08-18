package protocol

import (
	"context"
	"errors"
	"math/big"

	"github.com/stretchr/testify/suite"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/wallet/bigint"
)

type CommunityEventsTestsInterface interface {
	GetControlNode() *Messenger
	GetEventSender() *Messenger
	GetMember() *Messenger
	GetSuite() *suite.Suite
}

const commmunitiesEventsTestTokenAddress = "0x0400000000000000000000000000000000000000"
const commmunitiesEventsTestChainID = 1
const commmunitiesEventsEventSenderAddress = "0x0200000000000000000000000000000000000000"

type MessageResponseValidator func(*MessengerResponse) error
type WaitResponseValidator func(*MessengerResponse) bool

func WaitCommunityCondition(r *MessengerResponse) bool {
	return len(r.Communities()) > 0
}

func WaitMessageCondition(response *MessengerResponse) bool {
	return len(response.Messages()) > 0
}

func waitOnMessengerResponse(s *suite.Suite, fnWait WaitResponseValidator, fn MessageResponseValidator, user *Messenger) {
	response, err := WaitOnMessengerResponse(
		user,
		fnWait,
		"MessengerResponse data not received",
	)
	s.Require().NoError(err)
	s.Require().NoError(fn(response))
}

func checkClientsReceivedAdminEvent(base CommunityEventsTestsInterface, fnWait WaitResponseValidator, fn MessageResponseValidator) {
	s := base.GetSuite()
	// Wait and verify Member received community event
	waitOnMessengerResponse(s, fnWait, fn, base.GetMember())
	// Wait and verify event sender received his own event
	waitOnMessengerResponse(s, fnWait, fn, base.GetEventSender())
	// Wait and verify ControlNode received community event
	// ControlNode will publish CommunityDescription update
	waitOnMessengerResponse(s, fnWait, fn, base.GetControlNode())
	// Wait and verify Member received the ControlNode CommunityDescription update
	waitOnMessengerResponse(s, fnWait, fn, base.GetMember())
	// Wait and verify event sender received the ControlNode CommunityDescription update
	waitOnMessengerResponse(s, fnWait, fn, base.GetEventSender())
	// Wait and verify ControlNode received his own CommunityDescription update
	waitOnMessengerResponse(s, fnWait, fn, base.GetControlNode())
}

func refreshMessengerResponses(base CommunityEventsTestsInterface) {
	_, err := WaitOnMessengerResponse(base.GetControlNode(), func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	base.GetSuite().Require().NoError(err)

	_, err = WaitOnMessengerResponse(base.GetEventSender(), func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	base.GetSuite().Require().NoError(err)

	_, err = WaitOnMessengerResponse(base.GetMember(), func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	base.GetSuite().Require().NoError(err)
}

func createMockedWalletBalance(s *suite.Suite) map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big {
	eventSenderAddress := gethcommon.HexToAddress(commmunitiesEventsEventSenderAddress)

	mockedBalances := make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	mockedBalances[testChainID1] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	mockedBalances[testChainID1][eventSenderAddress] = make(map[gethcommon.Address]*hexutil.Big)

	// event sender will have token with `commmunitiesEventsTestTokenAddress``
	contractAddress := gethcommon.HexToAddress(commmunitiesEventsTestTokenAddress)
	balance, ok := new(big.Int).SetString("200", 10)
	s.Require().True(ok)
	decimalsFactor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18)), nil)
	balance.Mul(balance, decimalsFactor)

	mockedBalances[commmunitiesEventsTestChainID][eventSenderAddress][contractAddress] = (*hexutil.Big)(balance)
	return mockedBalances
}

func setUpCommunityAndRoles(base CommunityEventsTestsInterface, role protobuf.CommunityMember_Roles) *communities.Community {
	tcs2, err := base.GetControlNode().communitiesManager.All()
	suite := base.GetSuite()
	suite.Require().NoError(err, "eventSender.communitiesManager.All")
	suite.Len(tcs2, 1, "Must have 1 community")

	// ControlNode creates a community and chat
	community := createTestCommunity(base, protobuf.CommunityPermissions_NO_MEMBERSHIP)
	refreshMessengerResponses(base)

	// add events sender and member to the community
	advertiseCommunityTo(suite, community, base.GetControlNode(), base.GetEventSender())
	advertiseCommunityTo(suite, community, base.GetControlNode(), base.GetMember())

	request := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{commmunitiesEventsEventSenderAddress},
		Password:          "qwerty1",
		AirdropAddress:    commmunitiesEventsEventSenderAddress,
	}
	joinCommunity(suite, community, base.GetControlNode(), base.GetEventSender(), request)
	refreshMessengerResponses(base)

	request = &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{"0x0300000000000000000000000000000000000000"},
		Password:          "qwerty2",
		AirdropAddress:    "0x0300000000000000000000000000000000000000",
	}
	joinCommunity(suite, community, base.GetControlNode(), base.GetMember(), request)
	refreshMessengerResponses(base)

	// grant permissions to the event sender
	grantPermission(suite, community, base.GetControlNode(), base.GetEventSender(), role)
	refreshMessengerResponses(base)

	return community
}

func createTestCommunity(base CommunityEventsTestsInterface, membershipType protobuf.CommunityPermissions_Access) *communities.Community {
	description := &requests.CreateCommunity{
		Membership:                  membershipType,
		Name:                        "status",
		Color:                       "#ffffff",
		Description:                 "status community description",
		PinMessageAllMembersEnabled: false,
	}
	response, err := base.GetControlNode().CreateCommunity(description, true)

	suite := base.GetSuite()
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.Communities(), 1)
	suite.Require().Len(response.Chats(), 1)

	return response.Communities()[0]
}

func getModifiedCommunity(response *MessengerResponse, communityID string) (*communities.Community, error) {
	if len(response.Communities()) == 0 {
		return nil, errors.New("community not received")
	}

	var modifiedCommmunity *communities.Community = nil
	for _, c := range response.Communities() {
		if c.IDString() == communityID {
			modifiedCommmunity = c
		}
	}

	if modifiedCommmunity == nil {
		return nil, errors.New("couldn't find community in response")
	}

	return modifiedCommmunity, nil
}

func createCommunityChannel(base CommunityEventsTestsInterface, community *communities.Community, newChannel *protobuf.CommunityChat) string {
	checkChannelCreated := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == newChannel.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find created chat in response")
	}

	response, err := base.GetEventSender().CreateCommunityChat(community.ID(), newChannel)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkChannelCreated(response))
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].ChatsAdded, 1)
	var addedChatID string
	for addedChatID = range response.CommunityChanges[0].ChatsAdded {
		break
	}

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkChannelCreated)

	return addedChatID
}

func editCommunityChannel(base CommunityEventsTestsInterface, community *communities.Community, editChannel *protobuf.CommunityChat, channelID string) {
	checkChannelEdited := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == editChannel.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find modified chat in response")
	}

	response, err := base.GetEventSender().EditCommunityChat(community.ID(), channelID, editChannel)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkChannelEdited(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkChannelEdited)
}

func deleteCommunityChannel(base CommunityEventsTestsInterface, community *communities.Community, channelID string) {
	checkChannelDeleted := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		if _, exists := modifiedCommmunity.Chats()[channelID]; exists {
			return errors.New("channel was not deleted")
		}

		return nil
	}

	response, err := base.GetEventSender().DeleteCommunityChat(community.ID(), channelID)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkChannelDeleted(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkChannelDeleted)
}

func createTestPermissionRequest(community *communities.Community, pType protobuf.CommunityTokenPermission_Type) *requests.CreateCommunityTokenPermission {
	return &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        pType,
		TokenCriteria: []*protobuf.TokenCriteria{
			{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{uint64(commmunitiesEventsTestChainID): commmunitiesEventsTestTokenAddress},
				Symbol:            "TEST",
				Amount:            "100",
				Decimals:          uint64(18),
			},
		},
	}
}

func createTokenPermission(base CommunityEventsTestsInterface, community *communities.Community, request *requests.CreateCommunityTokenPermission) (string, *requests.CreateCommunityTokenPermission) {
	checkTokenPermissionCreation := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		if len(modifiedCommmunity.TokenPermissionsByType(request.Type)) == 0 {
			return errors.New("new token permission was not found")
		}

		return nil
	}

	response, err := base.GetEventSender().CreateCommunityTokenPermission(request)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().Nil(checkTokenPermissionCreation(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkTokenPermissionCreation)

	var tokenPermissionID string
	for tokenPermissionID = range response.CommunityChanges[0].TokenPermissionsAdded {
		break
	}

	s.Require().NotEqual(tokenPermissionID, "")

	return tokenPermissionID, request
}

func createTestTokenPermission(base CommunityEventsTestsInterface, community *communities.Community, pType protobuf.CommunityTokenPermission_Type) (string, *requests.CreateCommunityTokenPermission) {
	createTokenPermissionRequest := createTestPermissionRequest(community, pType)
	return createTokenPermission(base, community, createTokenPermissionRequest)
}

func editTokenPermission(base CommunityEventsTestsInterface, community *communities.Community, request *requests.EditCommunityTokenPermission) {
	s := base.GetSuite()
	checkTokenPermissionEdit := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		assertCheckTokenPermissionEdited(s, modifiedCommmunity, request.CreateCommunityTokenPermission.Type)

		return nil
	}

	response, err := base.GetEventSender().EditCommunityTokenPermission(request)
	s.Require().NoError(err)
	s.Require().Nil(checkTokenPermissionEdit(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkTokenPermissionEdit)
}

func assertCheckTokenPermissionEdited(s *suite.Suite, community *communities.Community, pType protobuf.CommunityTokenPermission_Type) {
	permissions := community.TokenPermissionsByType(pType)
	s.Require().Len(permissions, 1)
	s.Require().Len(permissions[0].TokenCriteria, 1)
	s.Require().Equal(permissions[0].TokenCriteria[0].Type, protobuf.CommunityTokenType_ERC20)
	s.Require().Equal(permissions[0].TokenCriteria[0].Symbol, "UPDATED")
	s.Require().Equal(permissions[0].TokenCriteria[0].Amount, "200")
	s.Require().Equal(permissions[0].TokenCriteria[0].Decimals, uint64(18))
}

func deleteTokenPermission(base CommunityEventsTestsInterface, community *communities.Community, request *requests.DeleteCommunityTokenPermission) {
	checkTokenPermissionDeleted := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		if modifiedCommmunity.HasTokenPermissions() {
			return errors.New("token permission was not deleted")
		}

		return nil
	}

	response, err := base.GetEventSender().DeleteCommunityTokenPermission(request)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().Nil(checkTokenPermissionDeleted(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkTokenPermissionDeleted)
}

func assertCheckTokenPermissionCreated(s *suite.Suite, community *communities.Community, pType protobuf.CommunityTokenPermission_Type) {
	permissions := community.TokenPermissionsByType(pType)
	s.Require().Len(permissions, 1)
	s.Require().Len(permissions[0].TokenCriteria, 1)
	s.Require().Equal(permissions[0].TokenCriteria[0].Type, protobuf.CommunityTokenType_ERC20)
	s.Require().Equal(permissions[0].TokenCriteria[0].Symbol, "TEST")
	s.Require().Equal(permissions[0].TokenCriteria[0].Amount, "100")
	s.Require().Equal(permissions[0].TokenCriteria[0].Decimals, uint64(18))
}

func setUpOnRequestCommunityAndRoles(base CommunityEventsTestsInterface, role protobuf.CommunityMember_Roles, additionalEventSenders []*Messenger) *communities.Community {
	tcs2, err := base.GetControlNode().communitiesManager.All()
	s := base.GetSuite()
	s.Require().NoError(err, "eventSender.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 community")

	// control node creates a community and chat
	community := createTestCommunity(base, protobuf.CommunityPermissions_ON_REQUEST)
	refreshMessengerResponses(base)

	advertiseCommunityTo(s, community, base.GetControlNode(), base.GetEventSender())
	advertiseCommunityTo(s, community, base.GetControlNode(), base.GetMember())

	joinOnRequestCommunity(s, community, base.GetControlNode(), base.GetEventSender())
	joinOnRequestCommunity(s, community, base.GetControlNode(), base.GetMember())

	checkMemberJoined := func(response *MessengerResponse) error {
		return checkMemberJoinedToTheCommunity(response, base.GetMember().IdentityPublicKey())
	}

	waitOnMessengerResponse(s, WaitCommunityCondition, checkMemberJoined, base.GetEventSender())

	// grant permissions to event sender
	grantPermission(s, community, base.GetControlNode(), base.GetEventSender(), role)
	checkPermissionGranted := func(response *MessengerResponse) error {
		return checkRolePermissionInResponse(response, base.GetEventSender().IdentityPublicKey(), role)
	}
	waitOnMessengerResponse(s, WaitCommunityCondition, checkPermissionGranted, base.GetMember())

	for _, eventSender := range additionalEventSenders {
		advertiseCommunityTo(s, community, base.GetControlNode(), eventSender)
		joinOnRequestCommunity(s, community, base.GetControlNode(), eventSender)

		grantPermission(s, community, base.GetControlNode(), eventSender, role)
		checkPermissionGranted = func(response *MessengerResponse) error {
			return checkRolePermissionInResponse(response, eventSender.IdentityPublicKey(), role)
		}
		waitOnMessengerResponse(s, WaitCommunityCondition, checkPermissionGranted, base.GetMember())
		waitOnMessengerResponse(s, WaitCommunityCondition, checkPermissionGranted, base.GetEventSender())
	}

	return community
}

func createCommunityCategory(base CommunityEventsTestsInterface, community *communities.Community, newCategory *requests.CreateCommunityCategory) string {
	checkCategoryCreated := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, category := range modifiedCommmunity.Categories() {
			if category.GetName() == newCategory.CategoryName {
				return nil
			}
		}

		return errors.New("couldn't find created Category in the response")
	}

	response, err := base.GetEventSender().CreateCommunityCategory(newCategory)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryCreated(response))
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunityChanges[0].CategoriesAdded, 1)

	var categoryID string
	for categoryID = range response.CommunityChanges[0].CategoriesAdded {
		break
	}

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkCategoryCreated)

	return categoryID
}

func editCommunityCategory(base CommunityEventsTestsInterface, communityID string, editCategory *requests.EditCommunityCategory) {
	checkCategoryEdited := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, communityID)
		if err != nil {
			return err
		}

		for _, category := range modifiedCommmunity.Categories() {
			if category.GetName() == editCategory.CategoryName {
				return nil
			}
		}

		return errors.New("couldn't find edited Category in the response")
	}

	response, err := base.GetEventSender().EditCommunityCategory(editCategory)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryEdited(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkCategoryEdited)
}

func deleteCommunityCategory(base CommunityEventsTestsInterface, communityID string, deleteCategory *requests.DeleteCommunityCategory) {
	checkCategoryDeleted := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, communityID)
		if err != nil {
			return err
		}

		if _, exists := modifiedCommmunity.Chats()[deleteCategory.CategoryID]; exists {
			return errors.New("community was not deleted")
		}

		return nil
	}

	response, err := base.GetEventSender().DeleteCommunityCategory(deleteCategory)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryDeleted(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkCategoryDeleted)
}

func reorderCategory(base CommunityEventsTestsInterface, reorderRequest *requests.ReorderCommunityCategories) {
	checkCategoryReorder := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(reorderRequest.CommunityID))
		if err != nil {
			return err
		}

		category, exist := modifiedCommmunity.Categories()[reorderRequest.CategoryID]
		if !exist {
			return errors.New("couldn't find community category")
		}

		if int(category.Position) != reorderRequest.Position {
			return errors.New("category was not reordered")
		}

		return nil
	}

	response, err := base.GetEventSender().ReorderCommunityCategories(reorderRequest)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryReorder(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkCategoryReorder)
}

func reorderChannel(base CommunityEventsTestsInterface, reorderRequest *requests.ReorderCommunityChat) {
	checkChannelReorder := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(reorderRequest.CommunityID))
		if err != nil {
			return err
		}

		chat, exist := modifiedCommmunity.Chats()[reorderRequest.ChatID]
		if !exist {
			return errors.New("couldn't find community chat")
		}

		if int(chat.Position) != reorderRequest.Position {
			return errors.New("chat position was not reordered")
		}

		if chat.CategoryId != reorderRequest.CategoryID {
			return errors.New("chat category was not reordered")
		}

		return nil
	}

	response, err := base.GetEventSender().ReorderCommunityChat(reorderRequest)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkChannelReorder(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkChannelReorder)
}

func kickMember(base CommunityEventsTestsInterface, communityID types.HexBytes, pubkey string) {
	checkKicked := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(communityID))
		if err != nil {
			return err
		}

		if modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey) {
			return errors.New("alice was not kicked")
		}

		return nil
	}

	response, err := base.GetEventSender().RemoveUserFromCommunity(
		communityID,
		pubkey,
	)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().Nil(checkKicked(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkKicked)
}

func banMember(base CommunityEventsTestsInterface, banRequest *requests.BanUserFromCommunity) {
	checkBanned := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(banRequest.CommunityID))
		if err != nil {
			return err
		}

		if modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey) {
			return errors.New("alice was not removed from the member list")
		}

		if !modifiedCommmunity.IsBanned(&base.GetMember().identity.PublicKey) {
			return errors.New("alice was not added to the banned list")
		}

		return nil
	}

	response, err := base.GetEventSender().BanUserFromCommunity(banRequest)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().Nil(checkBanned(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkBanned)
}

func unbanMember(base CommunityEventsTestsInterface, unbanRequest *requests.UnbanUserFromCommunity) {
	checkUnbanned := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(unbanRequest.CommunityID))
		if err != nil {
			return err
		}

		if modifiedCommmunity.IsBanned(&base.GetMember().identity.PublicKey) {
			return errors.New("alice was not unbanned")
		}

		return nil
	}

	response, err := base.GetEventSender().UnbanUserFromCommunity(unbanRequest)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().Nil(checkUnbanned(response))

	response, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		WaitCommunityCondition,
		"MessengerResponse data not received",
	)
	s.Require().NoError(err)
	s.Require().NoError(checkUnbanned(response))
}

func controlNodeSendMessage(base CommunityEventsTestsInterface, inputMessage *common.Message) string {
	response, err := base.GetControlNode().SendChatMessage(context.Background(), inputMessage)

	s := base.GetSuite()
	s.Require().NoError(err)
	message := response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)
	messageID := message.ID

	response, err = WaitOnMessengerResponse(base.GetEventSender(), WaitMessageCondition, "messages not received")
	s.Require().NoError(err)
	message = response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)

	response, err = WaitOnMessengerResponse(base.GetMember(), WaitMessageCondition, "messages not received")
	s.Require().NoError(err)
	message = response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)

	refreshMessengerResponses(base)

	return messageID
}

func deleteControlNodeMessage(base CommunityEventsTestsInterface, messageID string) {
	checkMessageDeleted := func(response *MessengerResponse) error {
		if len(response.RemovedMessages()) > 0 {
			return nil
		}
		return errors.New("message was not deleted")
	}

	response, err := base.GetEventSender().DeleteMessageAndSend(context.Background(), messageID)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkMessageDeleted(response))

	waitMessageCondition := func(response *MessengerResponse) bool {
		return len(response.RemovedMessages()) > 0
	}
	waitOnMessengerResponse(s, waitMessageCondition, checkMessageDeleted, base.GetMember())
	waitOnMessengerResponse(s, waitMessageCondition, checkMessageDeleted, base.GetControlNode())

}

func pinControlNodeMessage(base CommunityEventsTestsInterface, pinnedMessage *common.PinMessage) {
	checkPinned := func(response *MessengerResponse) error {
		if len(response.PinMessages()) > 0 {
			return nil
		}
		return errors.New("pin messages was not added")
	}

	response, err := base.GetEventSender().SendPinMessage(context.Background(), pinnedMessage)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkPinned(response))

	waitOnMessengerResponse(s, WaitMessageCondition, checkPinned, base.GetMember())
	waitOnMessengerResponse(s, WaitMessageCondition, checkPinned, base.GetControlNode())
}

func editCommunityDescription(base CommunityEventsTestsInterface, community *communities.Community) {
	expectedName := "edited community name"
	expectedColor := "#000000"
	expectedDescr := "edited community description"

	response, err := base.GetEventSender().EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_ON_REQUEST,
			Name:        expectedName,
			Color:       expectedColor,
			Description: expectedDescr,
		},
	})

	checkCommunityEdit := func(response *MessengerResponse) error {
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}

		rCommunities := response.Communities()
		if expectedName != rCommunities[0].Name() {
			return errors.New("incorrect community name")
		}

		if expectedColor != rCommunities[0].Color() {
			return errors.New("incorrect community color")
		}

		if expectedDescr != rCommunities[0].DescriptionText() {
			return errors.New("incorrect community description")
		}

		return nil
	}

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().Nil(checkCommunityEdit(response))

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkCommunityEdit)
}

func controlNodeCreatesCommunityPermission(base CommunityEventsTestsInterface, community *communities.Community, permissionRequest *requests.CreateCommunityTokenPermission) string {
	// control node creates permission
	response, err := base.GetControlNode().CreateCommunityTokenPermission(permissionRequest)
	s := base.GetSuite()
	s.Require().NoError(err)

	var tokenPermissionID string
	for id := range response.CommunityChanges[0].TokenPermissionsAdded {
		tokenPermissionID = id
	}
	s.Require().NotEqual(tokenPermissionID, "")

	ownerCommunity, err := base.GetControlNode().communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	assertCheckTokenPermissionCreated(s, ownerCommunity, permissionRequest.Type)

	// then, ensure event sender receives updated community
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 &&
				len(r.Communities()[0].TokenPermissionsByType(permissionRequest.Type)) > 0
		},
		"event sender did not receive community token permission",
	)
	s.Require().NoError(err)
	eventSenderCommunity, err := base.GetEventSender().communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	assertCheckTokenPermissionCreated(s, eventSenderCommunity, permissionRequest.Type)
	s.Require().True(eventSenderCommunity.HasPermissionToSendCommunityEvents())

	return tokenPermissionID
}

func testCreateEditDeleteChannels(base CommunityEventsTestsInterface, community *communities.Community) {
	newChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat from the event sender",
			Emoji:       "",
			Description: "chat created by an event sender",
		},
	}

	newChatID := createCommunityChannel(base, community, newChat)

	newChat.Identity.DisplayName = "modified chat from event sender"
	editCommunityChannel(base, community, newChat, newChatID)
	deleteCommunityChannel(base, community, newChatID)
}

func testCreateEditDeleteBecomeMemberPermission(base CommunityEventsTestsInterface, community *communities.Community, pType protobuf.CommunityTokenPermission_Type) {
	// first, create token permission
	tokenPermissionID, createTokenPermission := createTestTokenPermission(base, community, pType)

	createTokenPermission.TokenCriteria[0].Symbol = "UPDATED"
	createTokenPermission.TokenCriteria[0].Amount = "200"

	editTokenPermissionRequest := &requests.EditCommunityTokenPermission{
		PermissionID:                   tokenPermissionID,
		CreateCommunityTokenPermission: *createTokenPermission,
	}

	// then, event sender edits the permission
	editTokenPermission(base, community, editTokenPermissionRequest)

	deleteTokenPermissionRequest := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: tokenPermissionID,
	}

	// then, event sender deletes previously created token permission
	deleteTokenPermission(base, community, deleteTokenPermissionRequest)
}

func testAcceptMemberRequestToJoin(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger) {
	// set up additional user that will send request to join
	_, err := user.Start()

	s := base.GetSuite()

	s.Require().NoError(err)
	defer user.Shutdown() // nolint: errcheck

	advertiseCommunityTo(s, community, base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// control node receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// event sender has not accepted request yet
	eventSenderCommunity, err := base.GetEventSender().GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().False(eventSenderCommunity.HasMember(&user.identity.PublicKey))

	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	response, err = base.GetEventSender().AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	// we don't expect `user` to be a member already, because `eventSender` merely
	// forwards its accept decision to the control node
	s.Require().False(response.Communities()[0].HasMember(&user.identity.PublicKey))

	// at this point, the request to join is marked as accepted by GetEventSender node
	acceptedRequestsPending, err := base.GetEventSender().AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedRequestsPending, 1)
	s.Require().Equal(acceptedRequestsPending[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// user should not receive community admin event without being a member yet
	_, err = WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"user did not receive community request to join response",
	)
	s.Require().Error(err)

	// control node receives community event with accepted membership request
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"control node did not receive community request to join response",
	)
	s.Require().NoError(err)

	// at this point, the request to join is marked as accepted by control node
	acceptedRequests, err := base.GetControlNode().AcceptedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	// we expect 3 here (1 event senders, 1 member + 1 from user)
	s.Require().Len(acceptedRequests, 3)
	s.Require().Equal(acceptedRequests[2].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// user receives updated community
	_, err = WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"alice did not receive community request to join response",
	)
	s.Require().NoError(err)

	// event sender receives updated community
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"event sender did not receive community with the new member",
	)
	s.Require().NoError(err)

	// check control node notify event sender about accepting request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			acceptedRequests, err := base.GetEventSender().AcceptedRequestsToJoinForCommunity(community.ID())
			return err == nil && len(acceptedRequests) == 2 && (acceptedRequests[1].PublicKey == common.PubkeyToHex(&user.identity.PublicKey))
		},
		"no updates from control node",
	)

	s.Require().NoError(err)

	acceptedRequestsPending, err = base.GetEventSender().AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedRequestsPending, 0)
}

func testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger, additionalEventSender *Messenger) {
	// set up additional user that will send request to join
	_, err := user.Start()

	s := base.GetSuite()

	s.Require().NoError(err)
	defer user.Shutdown() // nolint: errcheck

	advertiseCommunityTo(s, community, base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender 2 receives request to join
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender 1 accepts request
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	response, err = base.GetEventSender().AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	// event sender 2 receives decision of other event sender
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// at this point, the request to join is in accepted/pending state for event sender 2
	acceptedPendingRequests, err := additionalEventSender.AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedPendingRequests, 1)
	s.Require().Equal(acceptedPendingRequests[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))
}

func testRejectMemberRequestToJoinResponseSharedWithOtherEventSenders(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger, additionalEventSender *Messenger) {
	// set up additional user that will send request to join
	_, err := user.Start()

	s := base.GetSuite()

	s.Require().NoError(err)
	defer user.Shutdown() // nolint: errcheck

	advertiseCommunityTo(s, community, base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// event sender 2 receives request to join
	response, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
	response, err = base.GetEventSender().DeclineRequestToJoinCommunity(rejectRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// event sender 2 receives decision of other event sender
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// at this point, the request to join is in declined/pending state for event sender 2
	rejectedPendingRequests, err := additionalEventSender.DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(rejectedPendingRequests, 1)
	s.Require().Equal(rejectedPendingRequests[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))
}

func testRejectMemberRequestToJoin(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger) {
	_, err := user.Start()

	s := base.GetSuite()
	s.Require().NoError(err)
	defer user.Shutdown() // nolint: errcheck

	advertiseCommunityTo(s, community, base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// control node receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"control node did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// event sender has not accepted request yet
	eventSenderCommunity, err := base.GetEventSender().GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().False(eventSenderCommunity.HasMember(&user.identity.PublicKey))

	// event sender rejects request to join
	rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = base.GetEventSender().DeclineRequestToJoinCommunity(rejectRequestToJoin)
	s.Require().NoError(err)

	eventSenderCommunity, err = base.GetEventSender().GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().False(eventSenderCommunity.HasMember(&user.identity.PublicKey))

	requests, err := base.GetEventSender().DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().Len(requests, 1)
	s.Require().NoError(err)

	// control node receives event sender event and stores rejected request to join
	response, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"control node did not receive community request to join update from event sender",
	)
	s.Require().NoError(err)
	s.Require().False(response.Communities()[0].HasMember(&user.identity.PublicKey))

	requests, err = base.GetControlNode().DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().Len(requests, 1)
	s.Require().NoError(err)

	// event sender receives updated community
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"event sender did not receive community update",
	)
	s.Require().NoError(err)

	// check control node notify event sender about declined request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			declinedRequests, err := base.GetEventSender().DeclinedRequestsToJoinForCommunity(community.ID())
			return err == nil && len(declinedRequests) == 1
		},
		"no updates from control node",
	)

	s.Require().NoError(err)

	declinedRequestsPending, err := base.GetEventSender().DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(declinedRequestsPending, 0)
}

func testEventSenderCannotOverrideRequestToJoinState(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger, additionalEventSender *Messenger) {
	_, err := user.Start()

	s := base.GetSuite()
	s.Require().NoError(err)
	defer user.Shutdown() // nolint: errcheck

	advertiseCommunityTo(s, community, base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender 2 receives request to join
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// request is pending for event sener 2
	pendingRequests, err := additionalEventSender.PendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(pendingRequests)
	s.Require().Len(pendingRequests, 1)

	// event sender 1 rejects request to join
	rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = base.GetEventSender().DeclineRequestToJoinCommunity(rejectRequestToJoin)
	s.Require().NoError(err)

	// request to join is now marked as rejected pending for event sender 1
	rejectedPendingRequests, err := base.GetEventSender().DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(rejectedPendingRequests)
	s.Require().Len(rejectedPendingRequests, 1)

	// event sender 2 receives event sender 1's decision
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// request to join is now marked as rejected pending for event sender 2
	rejectedPendingRequests, err = additionalEventSender.DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(rejectedPendingRequests)
	s.Require().Len(rejectedPendingRequests, 1)

	// event sender 2 should not be able to override that pending state
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = additionalEventSender.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().Error(err)
}

func testControlNodeHandlesMultipleEventSenderRequestToJoinDecisions(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger, additionalEventSender *Messenger) {
	_, err := user.Start()

	s := base.GetSuite()
	s.Require().NoError(err)
	defer user.Shutdown() // nolint: errcheck

	advertiseCommunityTo(s, community, base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender 2 receives request to join
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// control node receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender 1 rejects request to join
	rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = base.GetEventSender().DeclineRequestToJoinCommunity(rejectRequestToJoin)
	s.Require().NoError(err)
	// request to join is now marked as rejected pending for event sender 1
	rejectedPendingRequests, err := base.GetEventSender().DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(rejectedPendingRequests)
	s.Require().Len(rejectedPendingRequests, 1)

	// control node receives event sender 1's and 2's decision
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"control node did not receive event senders decision",
	)
	s.Require().NoError(err)
	// request to join is now marked as rejected
	rejectedRequests, err := base.GetControlNode().DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(rejectedRequests)
	s.Require().Len(rejectedRequests, 1)

	// event sender 2 accepts request to join
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = additionalEventSender.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	// request to join is now marked as accepted pending for event sender 2
	acceptedPendingRequests, err := additionalEventSender.AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(acceptedPendingRequests)
	s.Require().Len(acceptedPendingRequests, 1)

	// control node now receives event sender 2's decision
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"control node did not receive event senders decision",
	)
	s.Require().NoError(err)
	rejectedRequests, err = base.GetControlNode().DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(rejectedRequests)
	s.Require().Len(rejectedRequests, 1)
	// we expect user's request to join still to be rejected
	s.Require().Equal(rejectedRequests[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))
}

func testCreateEditDeleteCategories(base CommunityEventsTestsInterface, community *communities.Community) {
	newCategory := &requests.CreateCommunityCategory{
		CommunityID:  community.ID(),
		CategoryName: "event-sender-category-name",
	}
	categoryID := createCommunityCategory(base, community, newCategory)

	editCategory := &requests.EditCommunityCategory{
		CommunityID:  community.ID(),
		CategoryID:   categoryID,
		CategoryName: "edited-event-sender-category-name",
	}

	editCommunityCategory(base, community.IDString(), editCategory)

	deleteCategory := &requests.DeleteCommunityCategory{
		CommunityID: community.ID(),
		CategoryID:  categoryID,
	}

	deleteCommunityCategory(base, community.IDString(), deleteCategory)
}

func testReorderChannelsAndCategories(base CommunityEventsTestsInterface, community *communities.Community) {
	newCategory := &requests.CreateCommunityCategory{
		CommunityID:  community.ID(),
		CategoryName: "event-sender-category-name",
	}
	_ = createCommunityCategory(base, community, newCategory)

	newCategory.CategoryName = "event-sender-category-name2"
	categoryID2 := createCommunityCategory(base, community, newCategory)

	chat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat from event-sender",
			Emoji:       "",
			Description: "chat created by an event-sender",
		},
	}

	chatID := createCommunityChannel(base, community, chat)

	reorderCommunityRequest := requests.ReorderCommunityCategories{
		CommunityID: community.ID(),
		CategoryID:  categoryID2,
		Position:    0,
	}

	reorderCategory(base, &reorderCommunityRequest)

	reorderChatRequest := requests.ReorderCommunityChat{
		CommunityID: community.ID(),
		CategoryID:  categoryID2,
		ChatID:      chatID,
		Position:    0,
	}

	reorderChannel(base, &reorderChatRequest)
}

func testEventSenderKickTheSameRole(base CommunityEventsTestsInterface, community *communities.Community) {
	// event sender tries to kick the member with the same role
	_, err := base.GetEventSender().RemoveUserFromCommunity(
		community.ID(),
		common.PubkeyToHex(&base.GetEventSender().identity.PublicKey),
	)

	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().EqualError(err, "not allowed to remove admin or owner")
}

func testEventSenderKickControlNode(base CommunityEventsTestsInterface, community *communities.Community) {
	// event sender tries to kick the control node
	_, err := base.GetEventSender().RemoveUserFromCommunity(
		community.ID(),
		common.PubkeyToHex(&base.GetControlNode().identity.PublicKey),
	)

	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().EqualError(err, "not allowed to remove admin or owner")
}

func testOwnerBanTheSameRole(base CommunityEventsTestsInterface, community *communities.Community) {
	_, err := base.GetEventSender().BanUserFromCommunity(
		&requests.BanUserFromCommunity{
			CommunityID: community.ID(),
			User:        common.PubkeyToHexBytes(&base.GetEventSender().identity.PublicKey),
		},
	)

	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().EqualError(err, "not allowed to ban admin or owner")
}

func testOwnerBanControlNode(base CommunityEventsTestsInterface, community *communities.Community) {
	_, err := base.GetEventSender().BanUserFromCommunity(
		&requests.BanUserFromCommunity{
			CommunityID: community.ID(),
			User:        common.PubkeyToHexBytes(&base.GetControlNode().identity.PublicKey),
		},
	)

	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().EqualError(err, "not allowed to ban admin or owner")
}

func testBanUnbanMember(base CommunityEventsTestsInterface, community *communities.Community) {
	// verify that event sender can't ban a control node
	_, err := base.GetEventSender().BanUserFromCommunity(
		&requests.BanUserFromCommunity{
			CommunityID: community.ID(),
			User:        common.PubkeyToHexBytes(&base.GetControlNode().identity.PublicKey),
		},
	)
	s := base.GetSuite()
	s.Require().Error(err)

	banRequest := &requests.BanUserFromCommunity{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(&base.GetMember().identity.PublicKey),
	}

	banMember(base, banRequest)

	unbanRequest := &requests.UnbanUserFromCommunity{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(&base.GetMember().identity.PublicKey),
	}

	unbanMember(base, unbanRequest)
}

func testDeleteAnyMessageInTheCommunity(base CommunityEventsTestsInterface, community *communities.Community) {
	chatID := community.ChatIDs()[0]

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "control node text"

	messageID := controlNodeSendMessage(base, inputMessage)

	deleteControlNodeMessage(base, messageID)
}

func testEventSenderPinMessage(base CommunityEventsTestsInterface, community *communities.Community) {
	s := base.GetSuite()
	s.Require().False(community.AllowsAllMembersToPinMessage())
	chatID := community.ChatIDs()[0]

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "control node text"

	messageID := controlNodeSendMessage(base, inputMessage)

	pinnedMessage := common.NewPinMessage()
	pinnedMessage.MessageId = messageID
	pinnedMessage.ChatId = chatID
	pinnedMessage.Pinned = true

	pinControlNodeMessage(base, pinnedMessage)
}

func testMemberReceiveEventsWhenControlNodeOffline(base CommunityEventsTestsInterface, community *communities.Community) {
	// To simulate behavior when control node is offline, we will not use control node for listening new events
	// In this scenario member will reveive list of events

	s := base.GetSuite()
	member := base.GetMember()
	eventSender := base.GetEventSender()

	newAdminChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat from event sender",
			Emoji:       "",
			Description: "chat created by an event sender",
		},
	}

	checkChannelCreated := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == newAdminChat.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find created chat in response")
	}

	response, err := eventSender.CreateCommunityChat(community.ID(), newAdminChat)
	s.Require().NoError(err)
	s.Require().NoError(checkChannelCreated(response))
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].ChatsAdded, 1)
	var addedChatID string
	for addedChatID = range response.CommunityChanges[0].ChatsAdded {
		break
	}

	waitOnMessengerResponse(s, WaitCommunityCondition, checkChannelCreated, member)
	waitOnMessengerResponse(s, WaitCommunityCondition, checkChannelCreated, eventSender)

	newAdminChat.Identity.DisplayName = "modified chat from event sender"

	checkChannelEdited := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == newAdminChat.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find modified chat in response")
	}

	response, err = eventSender.EditCommunityChat(community.ID(), addedChatID, newAdminChat)
	s.Require().NoError(err)
	s.Require().NoError(checkChannelEdited(response))

	waitOnMessengerResponse(s, WaitCommunityCondition, checkChannelEdited, member)
	waitOnMessengerResponse(s, WaitCommunityCondition, checkChannelEdited, eventSender)

	checkChannelDeleted := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		if _, exists := modifiedCommmunity.Chats()[addedChatID]; exists {
			return errors.New("channel was not deleted")
		}

		return nil
	}

	response, err = eventSender.DeleteCommunityChat(community.ID(), addedChatID)
	s.Require().NoError(err)
	s.Require().NoError(checkChannelDeleted(response))

	waitOnMessengerResponse(s, WaitCommunityCondition, checkChannelDeleted, member)
	waitOnMessengerResponse(s, WaitCommunityCondition, checkChannelDeleted, eventSender)
}

func testEventSenderCannotDeletePrivilegedCommunityPermission(base CommunityEventsTestsInterface, community *communities.Community,
	testPermissionType protobuf.CommunityTokenPermission_Type, rolePermissionType protobuf.CommunityTokenPermission_Type) {
	// Community should have eventSenderRole permission or eventSender will loose his role
	// after control node create a new community permission
	if testPermissionType != rolePermissionType {
		rolePermission := createTestPermissionRequest(community, rolePermissionType)
		controlNodeCreatesCommunityPermission(base, community, rolePermission)
	}

	permissionRequest := createTestPermissionRequest(community, testPermissionType)
	tokenPermissionID := controlNodeCreatesCommunityPermission(base, community, permissionRequest)

	deleteTokenPermission := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: tokenPermissionID,
	}

	// then event sender tries to delete permission which should fail
	response, err := base.GetEventSender().DeleteCommunityTokenPermission(deleteTokenPermission)
	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().Nil(response)
}

func testEventSenderCannotEditPrivilegedCommunityPermission(base CommunityEventsTestsInterface, community *communities.Community,
	testPermissionType protobuf.CommunityTokenPermission_Type, rolePermissionType protobuf.CommunityTokenPermission_Type) {

	// Community should have eventSenderRole permission or eventSender will loose his role
	// after control node create a new community permission
	if testPermissionType != rolePermissionType {
		rolePermission := createTestPermissionRequest(community, rolePermissionType)
		controlNodeCreatesCommunityPermission(base, community, rolePermission)
	}

	permissionRequest := createTestPermissionRequest(community, testPermissionType)
	tokenPermissionID := controlNodeCreatesCommunityPermission(base, community, permissionRequest)

	permissionRequest.TokenCriteria[0].Symbol = "UPDATED"
	permissionRequest.TokenCriteria[0].Amount = "200"

	permissionEditRequest := &requests.EditCommunityTokenPermission{
		PermissionID:                   tokenPermissionID,
		CreateCommunityTokenPermission: *permissionRequest,
	}

	// then, event sender tries to edit permission
	response, err := base.GetEventSender().EditCommunityTokenPermission(permissionEditRequest)
	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().Nil(response)
}

func testAddAndSyncTokenFromControlNode(base CommunityEventsTestsInterface, community *communities.Community,
	privilegesLvl token.PrivilegesLevel, expectedSync bool) {
	tokenERC721 := createCommunityToken(community.IDString(), privilegesLvl)

	s := base.GetSuite()

	_, err := base.GetControlNode().SaveCommunityToken(tokenERC721, nil)
	s.Require().NoError(err)

	err = base.GetControlNode().AddCommunityToken(tokenERC721.CommunityID, tokenERC721.ChainID, tokenERC721.Address)
	s.Require().NoError(err)

	tokens, err := base.GetEventSender().communitiesManager.GetAllCommunityTokens()
	s.Require().NoError(err)
	s.Require().Len(tokens, 0)

	checkTokenAdded := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, tokenMetadata := range modifiedCommmunity.CommunityTokensMetadata() {
			if tokenMetadata.Name == tokenERC721.Name {
				return nil
			}
		}

		return errors.New("Token was not found")
	}

	waitOnMessengerResponse(s, WaitCommunityCondition, checkTokenAdded, base.GetMember())
	waitOnMessengerResponse(s, WaitCommunityCondition, checkTokenAdded, base.GetEventSender())

	// check control node sent sync message to the event sender
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			tokens, err := base.GetEventSender().communitiesManager.GetAllCommunityTokens()
			return err == nil && len(tokens) == 1
		},
		"no token sync message from control node",
	)

	if expectedSync {
		s.Require().NoError(err)
	} else {
		s.Require().Error(err)
	}

	// check member did not receive sync message with the token
	_, err = WaitOnMessengerResponse(
		base.GetMember(),
		func(r *MessengerResponse) bool {
			tokens, err := base.GetMember().communitiesManager.GetAllCommunityTokens()
			return err == nil && len(tokens) == 1
		},
		"no token sync message from control node",
	)

	s.Require().Error(err)
}

func testEventSenderCannotCreatePrivilegedCommunityPermission(base CommunityEventsTestsInterface, community *communities.Community, pType protobuf.CommunityTokenPermission_Type) {
	permissionRequest := createTestPermissionRequest(community, pType)

	response, err := base.GetEventSender().CreateCommunityTokenPermission(permissionRequest)
	s := base.GetSuite()
	s.Require().Nil(response)
	s.Require().Error(err)
}

func createCommunityToken(communityID string, privilegesLevel token.PrivilegesLevel) *token.CommunityToken {
	return &token.CommunityToken{
		CommunityID:        communityID,
		TokenType:          protobuf.CommunityTokenType_ERC721,
		Address:            "0x123",
		Name:               "StatusToken",
		Symbol:             "STT",
		Description:        "desc",
		Supply:             &bigint.BigInt{Int: big.NewInt(123)},
		InfiniteSupply:     false,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            1,
		DeployState:        token.Deployed,
		Base64Image:        "ABCD",
		PrivilegesLevel:    privilegesLevel,
	}
}

func testAddAndSyncTokenFromEventSenderByControlNode(base CommunityEventsTestsInterface, community *communities.Community,
	privilegesLvl token.PrivilegesLevel) {
	tokenERC721 := createCommunityToken(community.IDString(), privilegesLvl)

	s := base.GetSuite()

	_, err := base.GetEventSender().SaveCommunityToken(tokenERC721, nil)
	s.Require().NoError(err)

	err = base.GetEventSender().AddCommunityToken(tokenERC721.CommunityID, tokenERC721.ChainID, tokenERC721.Address)
	s.Require().NoError(err)

	tokens, err := base.GetControlNode().communitiesManager.GetAllCommunityTokens()
	s.Require().NoError(err)
	s.Require().Len(tokens, 0)

	checkTokenAdded := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, tokenMetadata := range modifiedCommmunity.CommunityTokensMetadata() {
			if tokenMetadata.Name == tokenERC721.Name {
				return nil
			}
		}

		return errors.New("Token was not found")
	}

	checkClientsReceivedAdminEvent(base, WaitCommunityCondition, checkTokenAdded)

	// check event sender sent sync message to the control node
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool {
			tokens, err := base.GetControlNode().communitiesManager.GetAllCommunityTokens()
			return err == nil && len(tokens) == 1
		},
		"no token sync message from event sender",
	)

	s.Require().NoError(err)

	// check member did not receive sync message with the token
	_, err = WaitOnMessengerResponse(
		base.GetMember(),
		func(r *MessengerResponse) bool {
			tokens, err := base.GetMember().communitiesManager.GetAllCommunityTokens()
			return err == nil && len(tokens) == 1
		},
		"no token sync message from event sender",
	)

	s.Require().Error(err)
}

func testEventSenderAddTokenMasterAndOwnerToken(base CommunityEventsTestsInterface, community *communities.Community) {
	ownerToken := createCommunityToken(community.IDString(), token.OwnerLevel)

	s := base.GetSuite()

	_, err := base.GetEventSender().SaveCommunityToken(ownerToken, nil)
	s.Require().NoError(err)

	err = base.GetEventSender().AddCommunityToken(ownerToken.CommunityID, ownerToken.ChainID, ownerToken.Address)
	s.Require().Error(err, communities.ErrInvalidManageTokensPermission)

	tokenMasterToken := ownerToken
	tokenMasterToken.PrivilegesLevel = token.MasterLevel
	tokenMasterToken.Address = "0x124"

	_, err = base.GetEventSender().SaveCommunityToken(tokenMasterToken, nil)
	s.Require().NoError(err)

	err = base.GetEventSender().AddCommunityToken(ownerToken.CommunityID, ownerToken.ChainID, ownerToken.Address)
	s.Require().Error(err, communities.ErrInvalidManageTokensPermission)
}
