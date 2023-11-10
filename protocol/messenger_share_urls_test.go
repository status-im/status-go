package protocol

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/api/multiformat"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/urls"
)

const (
	userURL              = "https://status.app/u#zQ3shwQPhRuDJSjVGVBnTjCdgXy5i9WQaeVPdGJD6yTarJQSj"
	userURLWithData      = "https://status.app/u/G10A4B0JdgwyRww90WXtnP1oNH1ZLQNM0yX0Ja9YyAMjrqSZIYINOHCbFhrnKRAcPGStPxCMJDSZlGCKzmZrJcimHY8BbcXlORrElv_BbQEegnMDPx1g9C5VVNl0fE4y#zQ3shwQPhRuDJSjVGVBnTjCdgXy5i9WQaeVPdGJD6yTarJQSj"
	communityURL         = "https://status.app/c#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
	communityURLWithData = "https://status.app/c/iyKACkQKB0Rvb2RsZXMSJ0NvbG9yaW5nIHRoZSB3b3JsZCB3aXRoIGpveSDigKIg4bSXIOKAohiYohsiByMxMzFEMkYqAwEhMwM=#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
	channelURL           = "https://status.app/cc/003cdcd5-e065-48f9-b166-b1a94ac75a11#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
	channelURLWithData   = "https://status.app/cc/G54AAKwObLdpiGjXnckYzRcOSq0QQAS_CURGfqVU42ceGHCObstUIknTTZDOKF3E8y2MSicncpO7fTskXnoACiPKeejvjtLTGWNxUhlT7fyQS7Jrr33UVHluxv_PLjV2ePGw5GQ33innzeK34pInIgUGs5RjdQifMVmURalxxQKwiuoY5zwIjixWWRHqjHM=#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
)

func TestMessengerShareUrlsSuite(t *testing.T) {
	suite.Run(t, new(MessengerShareUrlsSuite))
}

type MessengerShareUrlsSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerShareUrlsSuite) createCommunity() *communities.Community {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.m.CreateCommunity(description, false)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	return response.Communities()[0]
}

func (s *MessengerShareUrlsSuite) createContact() (*Messenger, *Contact) {
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	ensName := "blah.stateofus.eth"

	s.Require().NoError(s.m.ENSVerified(contactID, ensName))

	response, err := s.m.AddContact(context.Background(), &requests.AddContact{ID: contactID})
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Contacts, 1)

	contact := response.Contacts[0]
	s.Require().Equal(ensName, contact.EnsName)
	s.Require().True(contact.ENSVerified)

	return theirMessenger, contact
}

func (s *MessengerShareUrlsSuite) createCommunityWithChannel() (*communities.Community, *protobuf.CommunityChat, string) {
	community := s.createCommunity()

	chat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "😎",
			Description: "status-core community chat",
		},
	}
	response, err := s.m.CreateCommunityChat(community.ID(), chat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	community = response.Communities()[0]
	s.Require().Len(community.Chats(), 1)

	var channelID string
	var channel *protobuf.CommunityChat

	for key, value := range community.Chats() {
		channelID = key
		channel = value
		break
	}
	s.Require().NotNil(channel)
	return community, channel, channelID
}

func (s *MessengerShareUrlsSuite) TestDecodeEncodeDataURL() {
	ts := [][]byte{
		[]byte("test data 123"),
		[]byte("test data 123test data 123test data 123test data 123test data 123"),
	}

	for i := range ts {
		encodedData, err := urls.EncodeDataURL(ts[i])
		s.Require().NoError(err)

		decodedData, err := urls.DecodeDataURL(encodedData)
		s.Require().NoError(err)
		s.Require().Equal(ts[i], decodedData)
	}
}

func (s *MessengerShareUrlsSuite) TestParseWrongUrls() {
	const notStatusSharedURLError = "not a status shared url"
	badURLs := map[string]string{
		"https://status.appc/#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11":  notStatusSharedURLError,
		"https://status.app/cc#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11": notStatusSharedURLError,
		"https://status.app/a#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11":  notStatusSharedURLError,
		"https://status.im/u#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11":   notStatusSharedURLError,
		"https://status.app/u/": "url should contain at least one `#` separator",
	}

	for url, expectedError := range badURLs {
		urlData, err := ParseSharedURL(url)
		s.Require().Error(err)
		s.Require().Equal(err.Error(), expectedError)
		s.Require().Nil(urlData)
	}
}

func (s *MessengerShareUrlsSuite) TestIsStatusSharedUrl() {
	testCases := []struct {
		Name   string
		URL    string
		Result bool
	}{
		{
			Name:   "Direct website link",
			URL:    "https://status.app",
			Result: false,
		},
		{
			Name:   "Website page link",
			URL:    "https://status.app/features/messenger",
			Result: false,
		},
		{
			// starts with `/c`, but no `#` after
			Name:   "Website page link",
			URL:    "https://status.app/communities",
			Result: false,
		},
		{
			Name:   "User link",
			URL:    userURL,
			Result: true,
		},
		{
			Name:   "User link with data",
			URL:    userURLWithData,
			Result: true,
		},
		{
			Name:   "Community link",
			URL:    communityURL,
			Result: true,
		},
		{
			Name:   "Community link with data",
			URL:    communityURLWithData,
			Result: true,
		},
		{
			Name:   "Channel link",
			URL:    channelURL,
			Result: true,
		},
		{
			Name:   "Channel link with data",
			URL:    channelURLWithData,
			Result: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			result := IsStatusSharedURL(tc.URL)
			s.Require().Equal(tc.Result, result)
		})
	}
}

func (s *MessengerShareUrlsSuite) TestShareCommunityURLWithChatKey() {
	community := s.createCommunity()

	url, err := s.m.ShareCommunityURLWithChatKey(community.ID())
	s.Require().NoError(err)

	shortID, err := community.SerializedID()
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/c#%s", baseShareURL, shortID)
	s.Require().Equal(expectedURL, url)
}

func (s *MessengerShareUrlsSuite) TestParseCommunityURLWithChatKey() {
	community := s.createCommunity()

	shortID, err := community.SerializedID()
	s.Require().NoError(err)

	url := fmt.Sprintf("%s/c#%s", baseShareURL, shortID)

	urlData, err := ParseSharedURL(url)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Community)
	s.Require().Equal(community.IDString(), urlData.Community.CommunityID)
	s.Require().Equal("", urlData.Community.DisplayName)
	s.Require().Equal("", urlData.Community.Description)
	s.Require().Equal(uint32(0), urlData.Community.MembersCount)
	s.Require().Equal("", urlData.Community.Color)
	s.Require().Equal([]uint32{}, urlData.Community.TagIndices)
}

func (s *MessengerShareUrlsSuite) TestShareCommunityURLWithData() {
	community := s.createCommunity()

	url, err := s.m.ShareCommunityURLWithData(community.ID())
	s.Require().NoError(err)

	communityData, chatKey, err := s.m.prepareEncodedCommunityData(community)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/c/%s#%s", baseShareURL, communityData, chatKey)
	s.Require().Equal(expectedURL, url)
}

func (s *MessengerShareUrlsSuite) TestParseCommunityURLWithData() {
	urlData, err := ParseSharedURL(communityURLWithData)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Community)
	s.Require().Equal("0x02a3d2fdb9ac335917bf9d46b38d7496c00bbfadbaf832e8aa61d13ac2b4452084", urlData.Community.CommunityID)
	s.Require().Equal("Doodles", urlData.Community.DisplayName)
	s.Require().Equal("Coloring the world with joy • ᴗ •", urlData.Community.Description)
	s.Require().Equal(uint32(446744), urlData.Community.MembersCount)
	s.Require().Equal("#131D2F", urlData.Community.Color)
	s.Require().Equal([]uint32{1, 33, 51}, urlData.Community.TagIndices)
}

func (s *MessengerShareUrlsSuite) TestShareAndParseCommunityURLWithData() {
	community := s.createCommunity()

	url, err := s.m.ShareCommunityURLWithData(community.ID())
	s.Require().NoError(err)

	urlData, err := ParseSharedURL(url)
	s.Require().NoError(err)

	s.Require().Equal(community.Identity().DisplayName, urlData.Community.DisplayName)
	s.Require().Equal(community.DescriptionText(), urlData.Community.Description)
	s.Require().Equal(uint32(community.MembersCount()), urlData.Community.MembersCount)
	s.Require().Equal(community.Identity().GetColor(), urlData.Community.Color)
	s.Require().Equal(community.TagsIndices(), urlData.Community.TagIndices)
}

func (s *MessengerShareUrlsSuite) TestShareCommunityChannelURLWithChatKey() {
	community := s.createCommunity()
	channelID := "003cdcd5-e065-48f9-b166-b1a94ac75a11"

	request := &requests.CommunityChannelShareURL{
		CommunityID: community.ID(),
		ChannelID:   channelID,
	}
	url, err := s.m.ShareCommunityChannelURLWithChatKey(request)
	s.Require().NoError(err)

	shortID, err := community.SerializedID()
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/cc/%s#%s", baseShareURL, channelID, shortID)
	s.Require().Equal(expectedURL, url)
}

func (s *MessengerShareUrlsSuite) TestParseCommunityChannelURLWithChatKey() {
	const channelUUID = "003cdcd5-e065-48f9-b166-b1a94ac75a11"
	const communityID = "0x02a3d2fdb9ac335917bf9d46b38d7496c00bbfadbaf832e8aa61d13ac2b4452084"

	urlData, err := ParseSharedURL(channelURL)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Community)
	s.Require().Equal(communityID, urlData.Community.CommunityID)
	s.Require().Equal("", urlData.Community.DisplayName)
	s.Require().Equal("", urlData.Community.Description)
	s.Require().Equal(uint32(0), urlData.Community.MembersCount)
	s.Require().Equal("", urlData.Community.Color)
	s.Require().Equal([]uint32{}, urlData.Community.TagIndices)

	s.Require().NotNil(urlData.Channel)
	s.Require().Equal(channelUUID, urlData.Channel.ChannelUUID)
	s.Require().Equal("", urlData.Channel.Emoji)
	s.Require().Equal("", urlData.Channel.DisplayName)
	s.Require().Equal("", urlData.Channel.Color)
}

func (s *MessengerShareUrlsSuite) TestShareCommunityChannelURLWithData() {
	community, channel, channelID := s.createCommunityWithChannel()

	request := &requests.CommunityChannelShareURL{
		CommunityID: community.ID(),
		ChannelID:   channelID,
	}
	url, err := s.m.ShareCommunityChannelURLWithData(request)
	s.Require().NoError(err)

	communityChannelData, chatKey, err := s.m.prepareEncodedCommunityChannelData(community, channel, channelID)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/cc/%s#%s", baseShareURL, communityChannelData, chatKey)
	s.Require().Equal(expectedURL, url)
}

func (s *MessengerShareUrlsSuite) TestParseCommunityChannelURLWithData() {
	urlData, err := ParseSharedURL(channelURLWithData)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Community)
	s.Require().Equal("Doodles", urlData.Community.DisplayName)

	s.Require().NotNil(urlData.Channel)
	s.Require().Equal("🍿", urlData.Channel.Emoji)
	s.Require().Equal("design", urlData.Channel.DisplayName)
	s.Require().Equal("#131D2F", urlData.Channel.Color)
}

func (s *MessengerShareUrlsSuite) TestShareAndParseCommunityChannelURLWithData() {
	community, channel, channelID := s.createCommunityWithChannel()

	request := &requests.CommunityChannelShareURL{
		CommunityID: community.ID(),
		ChannelID:   channelID,
	}
	url, err := s.m.ShareCommunityChannelURLWithData(request)
	s.Require().NoError(err)

	urlData, err := ParseSharedURL(url)
	s.Require().NoError(err)

	s.Require().Equal(community.Identity().DisplayName, urlData.Community.DisplayName)
	s.Require().Equal(community.DescriptionText(), urlData.Community.Description)
	s.Require().Equal(uint32(community.MembersCount()), urlData.Community.MembersCount)
	s.Require().Equal(community.Identity().GetColor(), urlData.Community.Color)
	s.Require().Equal(community.TagsIndices(), urlData.Community.TagIndices)

	s.Require().NotNil(urlData.Channel)
	s.Require().Equal(channel.Identity.Emoji, urlData.Channel.Emoji)
	s.Require().Equal(channel.Identity.DisplayName, urlData.Channel.DisplayName)
	s.Require().Equal(channel.Identity.Color, urlData.Channel.Color)
}

func (s *MessengerShareUrlsSuite) TestShareUserURLWithChatKey() {
	_, contact := s.createContact()

	url, err := s.m.ShareUserURLWithChatKey(contact.ID)
	s.Require().NoError(err)

	shortKey, err := multiformat.SerializeLegacyKey(contact.ID)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/u#%s", baseShareURL, shortKey)
	s.Require().Equal(expectedURL, url)
}

func (s *MessengerShareUrlsSuite) TestParseUserURLWithChatKey() {
	urlData, err := ParseSharedURL(userURL)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Contact)
	s.Require().Equal("", urlData.Contact.DisplayName)
	s.Require().Equal("", urlData.Contact.Description)
}

func (s *MessengerShareUrlsSuite) TestShareUserURLWithENS() {
	_, contact := s.createContact()

	url, err := s.m.ShareUserURLWithENS(contact.ID)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/u#%s", baseShareURL, contact.EnsName)
	s.Require().Equal(expectedURL, url)
}

// TODO: ens in the next ticket
// func (s *MessengerShareUrlsSuite) TestParseUserURLWithENS() {
// 	_, contact := s.createContact()

// 	url := fmt.Sprintf("%s/u#%s", baseShareURL, contact.EnsName)

// 	urlData, err := ParseSharedURL(url)
// 	s.Require().NoError(err)
// 	s.Require().NotNil(urlData)

// 	s.Require().NotNil(urlData.Contact)
// 	s.Require().Equal(contact.DisplayName, urlData.Contact.DisplayName)
//  s.Require().Equal(contact.Bio, urlData.Contact.DisplayName)
// }

func (s *MessengerShareUrlsSuite) TestParseUserURLWithData() {
	urlData, err := ParseSharedURL(userURLWithData)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Contact)
	s.Require().Equal("Mark Cole", urlData.Contact.DisplayName)
	s.Require().Equal("Visual designer @Status, cat lover, pizza enthusiast, yoga afficionada", urlData.Contact.Description)
	s.Require().Equal("zQ3shwQPhRuDJSjVGVBnTjCdgXy5i9WQaeVPdGJD6yTarJQSj", urlData.Contact.PublicKey)
}

func (s *MessengerShareUrlsSuite) TestShareUserURLWithData() {
	_, contact := s.createContact()

	url, err := s.m.ShareUserURLWithData(contact.ID)
	s.Require().NoError(err)

	userData, chatKey, err := s.m.prepareEncodedUserData(contact)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/u/%s#%s", baseShareURL, userData, chatKey)
	s.Require().Equal(expectedURL, url)
}

func (s *MessengerShareUrlsSuite) TestShareAndParseUserURLWithData() {
	_, contact := s.createContact()

	shortKey, err := multiformat.SerializeLegacyKey(contact.ID)
	s.Require().NoError(err)

	url, err := s.m.ShareUserURLWithData(contact.ID)
	s.Require().NoError(err)

	urlData, err := ParseSharedURL(url)
	s.Require().NoError(err)

	s.Require().NotNil(urlData.Contact)
	s.Require().Equal(contact.DisplayName, urlData.Contact.DisplayName)
	s.Require().Equal(shortKey, urlData.Contact.PublicKey)
}
