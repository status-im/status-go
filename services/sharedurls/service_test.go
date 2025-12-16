package sharedurls

import (
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/status-im/status-go/internal/testutils/fake"
	"github.com/status-im/status-go/pkg/multiformat"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/contacts"
	"github.com/status-im/status-go/protocol/protobuf"
	mock_provider "github.com/status-im/status-go/services/sharedurls/mock"
)

const (
	userURL                      = "https://status.app/u#zQ3shwQPhRuDJSjVGVBnTjCdgXy5i9WQaeVPdGJD6yTarJQSj"
	userURLWithData              = "https://status.app/u/G10A4B0JdgwyRww90WXtnP1oNH1ZLQNM0yX0Ja9YyAMjrqSZIYINOHCbFhrnKRAcPGStPxCMJDSZlGCKzmZrJcimHY8BbcXlORrElv_BbQEegnMDPx1g9C5VVNl0fE4y#zQ3shwQPhRuDJSjVGVBnTjCdgXy5i9WQaeVPdGJD6yTarJQSj"
	communityURL                 = "https://status.app/c#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
	communityURLWithData         = "https://status.app/c/iyKACkQKB0Rvb2RsZXMSJ0NvbG9yaW5nIHRoZSB3b3JsZCB3aXRoIGpveSDigKIg4bSXIOKAohiYohsiByMxMzFEMkYqAwEhMwM=#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
	communityURLWithDataNoTags   = "https://status.app/c/CxCACh8KBFBJR1MSDHdlIGxvdmUgcGlncxgBIgcjRDM0NEM1Aw==#zQ3shZp9gY1FXfYkcd3CMrFLHriHQfrXvpF9XbZMwJhTcZsq8"
	communityURLWithDataWithTags = "https://status.app/c/CxKACiMKBFBJR1MSDHdlIGxvdmUgcGlncxgBIgcjRDM0NEM1KgIjHgM=#zQ3shZp9gY1FXfYkcd3CMrFLHriHQfrXvpF9XbZMwJhTcZsq8"
	channelURL                   = "https://status.app/cc/003cdcd5-e065-48f9-b166-b1a94ac75a11#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
	channelURLWithData           = "https://status.app/cc/G54AAKwObLdpiGjXnckYzRcOSq0QQAS_CURGfqVU42ceGHCObstUIknTTZDOKF3E8y2MSicncpO7fTskXnoACiPKeejvjtLTGWNxUhlT7fyQS7Jrr33UVHluxv_PLjV2ePGw5GQ33innzeK34pInIgUGs5RjdQifMVmURalxxQKwiuoY5zwIjixWWRHqjHM=#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
)

func TestShareUrlsSuite(t *testing.T) {
	suite.Run(t, new(ShareUrlsSuite))
}

type ShareUrlsSuite struct {
	suite.Suite
	service  *Service
	provider *mock_provider.MockDataProvider
}

func (s *ShareUrlsSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.provider = mock_provider.NewMockDataProvider(ctrl)
	s.service = NewService(s.provider)
}

func (s *ShareUrlsSuite) addFakeChannel(community *communities.Community) (*communities.Community, *protobuf.CommunityChat, string) {
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

	chatID := gofakeit.UUID()
	changes, err := community.CreateChat(chatID, chat)
	s.Require().NoError(err)
	s.Require().NotNil(changes)

	community = changes.Community
	s.Require().Len(community.Chats(), 1)

	channel, ok := community.Chats()[chatID]
	s.Require().True(ok)
	s.Require().NotNil(channel)

	return community, channel, chatID
}

func (s *ShareUrlsSuite) fakeContact() *contacts.Contact {
	contact := fake.Contact(s.T(), nil)
	contact.ENSVerified = true
	return contact
}

func (s *ShareUrlsSuite) fakeCommunity() *communities.Community {
	return fake.Community(s.T())
}

func (s *ShareUrlsSuite) verifyCommunityURL(url string, community *communities.Community, channel *protobuf.CommunityChat) {
	urlData, err := ParseSharedURL(url)
	s.Require().NoError(err)

	s.Require().Equal(community.Identity().DisplayName, urlData.Community.DisplayName)
	s.Require().Equal(community.DescriptionText(), urlData.Community.Description)
	s.Require().Equal(uint32(community.MembersCount()), urlData.Community.MembersCount)
	s.Require().Equal(community.Identity().GetColor(), urlData.Community.Color)
	s.Require().Equal(community.TagsIndices(), urlData.Community.TagIndices)

	if channel != nil {
		s.Require().NotNil(urlData.Channel)
		s.Require().Equal(channel.Identity.Emoji, urlData.Channel.Emoji)
		s.Require().Equal(channel.Identity.DisplayName, urlData.Channel.DisplayName)
		s.Require().Equal(channel.Identity.Color, urlData.Channel.Color)
	}
}

func (s *ShareUrlsSuite) TestDecodeEncodeDataURL() {
	ts := [][]byte{
		[]byte("test data 123"),
		[]byte("test data 123test data 123test data 123test data 123test data 123"),
	}

	for i := range ts {
		encodedData, err := encodeDataURL(ts[i])
		s.Require().NoError(err)

		decodedData, err := decodeDataURL(encodedData)
		s.Require().NoError(err)
		s.Require().Equal(ts[i], decodedData)
	}
}

func (s *ShareUrlsSuite) TestParseWrongUrls() {
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

func (s *ShareUrlsSuite) TestIsStatusSharedUrl() {
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

func (s *ShareUrlsSuite) TestShareCommunityURLWithChatKey() {
	community := s.fakeCommunity()

	s.provider.EXPECT().GetContactByID(gomock.Any()).Times(0)
	s.provider.EXPECT().GetCommunityByID(gomock.Any()).Times(0)

	url, err := s.service.ShareCommunityURLWithChatKey(community.ID())
	s.Require().NoError(err)

	shortID, err := community.SerializedID()
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/c#%s", baseShareURL, shortID)
	s.Require().Equal(expectedURL, url)
}

func (s *ShareUrlsSuite) TestParseCommunityURLWithChatKey() {
	community := s.fakeCommunity()

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

func (s *ShareUrlsSuite) TestShareCommunityURLWithData() {
	community := s.fakeCommunity()

	s.provider.EXPECT().
		GetCommunityByID(gomock.Eq(community.ID())).
		Return(community, nil).
		Times(1)

	url, err := s.service.ShareCommunityURLWithData(community.ID())
	s.Require().NoError(err)

	communityData, chatKey, err := prepareEncodedCommunityData(community)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/c/%s#%s", baseShareURL, communityData, chatKey)
	s.Require().Equal(expectedURL, url)

	response, err := ParseSharedURL(url)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().NotNil(response.Community)
	s.Require().Equal(community.IDString(), response.Community.CommunityID)
	s.Require().Equal(community.TagsIndices(), response.Community.TagIndices)
}

func (s *ShareUrlsSuite) TestParseCommunityURLWithData() {
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

func (s *ShareUrlsSuite) TestParseCommunityURLWithDataNoTags() {
	urlData, err := ParseSharedURL(communityURLWithDataNoTags)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Community)
	s.Require().Equal("0x02b84843377a24ff498b6c37bd63e2b285c1ee2ccbab82d7a4afa25fff8c5076df", urlData.Community.CommunityID)
	s.Require().Equal("PIGS", urlData.Community.DisplayName)
	s.Require().Equal("we love pigs", urlData.Community.Description)
	s.Require().Equal(uint32(0x1), urlData.Community.MembersCount)
	s.Require().Equal("#D344C5", urlData.Community.Color)
	s.Require().Equal([]uint32{}, urlData.Community.TagIndices)
}

func (s *ShareUrlsSuite) TestParseCommunityURLWithDataWithTags() {
	urlData, err := ParseSharedURL(communityURLWithDataWithTags)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Community)
	s.Require().Equal("0x02b84843377a24ff498b6c37bd63e2b285c1ee2ccbab82d7a4afa25fff8c5076df", urlData.Community.CommunityID)
	s.Require().Equal("PIGS", urlData.Community.DisplayName)
	s.Require().Equal("we love pigs", urlData.Community.Description)
	s.Require().Equal(uint32(0x1), urlData.Community.MembersCount)
	s.Require().Equal("#D344C5", urlData.Community.Color)
	s.Require().Equal([]uint32{0x23, 0x1e}, urlData.Community.TagIndices)
}

func (s *ShareUrlsSuite) TestShareAndParseCommunityURLWithData() {
	community := s.fakeCommunity()

	s.provider.EXPECT().GetContactByID(gomock.Any()).Times(0)
	s.provider.EXPECT().GetCommunityByID(gomock.Eq(community.ID())).Return(community, nil).Times(1)

	url, err := s.service.ShareCommunityURLWithData(community.ID())
	s.Require().NoError(err)

	s.verifyCommunityURL(url, community, nil)
}

func (s *ShareUrlsSuite) TestShareCommunityChannelURLWithChatKey() {
	community := s.fakeCommunity()
	channelID := gofakeit.UUID()

	url, err := s.service.ShareCommunityChannelURLWithChatKey(community.ID(), channelID)
	s.Require().NoError(err)

	shortID, err := community.SerializedID()
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/cc/%s#%s", baseShareURL, channelID, shortID)
	s.Require().Equal(expectedURL, url)
}

func (s *ShareUrlsSuite) TestParseCommunityChannelURLWithChatKey() {
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

func (s *ShareUrlsSuite) TestShareCommunityChannelURLWithData() {
	community := s.fakeCommunity()
	community, channel, channelID := s.addFakeChannel(community)

	s.provider.EXPECT().GetContactByID(gomock.Any()).Times(0)
	s.provider.EXPECT().GetCommunityByID(gomock.Eq(community.ID())).Return(community, nil).Times(1)

	url, err := s.service.ShareCommunityChannelURLWithData(community.ID(), channelID)
	s.Require().NoError(err)

	communityChannelData, chatKey, err := s.service.prepareEncodedCommunityChannelData(community, channel, channelID)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/cc/%s#%s", baseShareURL, communityChannelData, chatKey)
	s.Require().Equal(expectedURL, url)
}

func (s *ShareUrlsSuite) TestParseCommunityChannelURLWithData() {
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

func (s *ShareUrlsSuite) TestShareAndParseCommunityChannelURLWithData() {
	community := s.fakeCommunity()
	community, channel, channelID := s.addFakeChannel(community)

	s.provider.EXPECT().GetContactByID(gomock.Any()).Times(0)
	s.provider.EXPECT().GetCommunityByID(gomock.Eq(community.ID())).Return(community, nil).Times(1)

	url, err := s.service.ShareCommunityChannelURLWithData(community.ID(), channelID)
	s.Require().NoError(err)

	s.verifyCommunityURL(url, community, channel)
}

func (s *ShareUrlsSuite) TestShareUserURLWithChatKey() {
	contact := s.fakeContact()

	s.provider.EXPECT().GetContactByID(gomock.Any()).Times(0)
	s.provider.EXPECT().GetCommunityByID(gomock.Any()).Times(0)

	url, err := s.service.ShareUserURLWithChatKey(contact.ID)
	s.Require().NoError(err)

	shortKey, err := multiformat.SerializeLegacyKey(contact.ID)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/u#%s", baseShareURL, shortKey)
	s.Require().Equal(expectedURL, url)
}

func (s *ShareUrlsSuite) TestParseUserURLWithChatKey() {
	urlData, err := ParseSharedURL(userURL)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Contact)
	s.Require().Equal("", urlData.Contact.DisplayName)
	s.Require().Equal("", urlData.Contact.Description)
}

func (s *ShareUrlsSuite) TestShareUserURLWithENS() {
	contact := s.fakeContact()

	s.provider.EXPECT().GetContactByID(contact.ID).Return(contact, nil).Times(1)
	s.provider.EXPECT().GetCommunityByID(gomock.Any()).Times(0)

	url, err := s.service.ShareUserURLWithENS(contact.ID)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/u#%s", baseShareURL, contact.EnsName)
	s.Require().Equal(expectedURL, url)
}

// TODO: ens in the next ticket
func (s *ShareUrlsSuite) TestParseUserURLWithENS() {
	s.T().Skip("ens in the next ticket")

	contact := s.fakeContact()

	url := fmt.Sprintf("%s/u#%s", baseShareURL, contact.EnsName)

	urlData, err := ParseSharedURL(url)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Contact)
	s.Require().Equal(contact.DisplayName, urlData.Contact.DisplayName)
	s.Require().Equal(contact.Bio, urlData.Contact.DisplayName)
}

func (s *ShareUrlsSuite) TestParseUserURLWithData() {
	urlData, err := ParseSharedURL(userURLWithData)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Contact)
	s.Require().Equal("Mark Cole", urlData.Contact.DisplayName)
	s.Require().Equal("Visual designer @Status, cat lover, pizza enthusiast, yoga afficionada", urlData.Contact.Description)
	s.Require().Equal("zQ3shwQPhRuDJSjVGVBnTjCdgXy5i9WQaeVPdGJD6yTarJQSj", urlData.Contact.PublicKey)
}

func (s *ShareUrlsSuite) TestShareUserURLWithData() {
	contact := s.fakeContact()

	s.provider.EXPECT().GetContactByID(contact.ID).Return(contact, nil).Times(1)
	s.provider.EXPECT().GetCommunityByID(gomock.Any()).Times(0)

	url, err := s.service.ShareUserURLWithData(contact.ID)
	s.Require().NoError(err)

	userData, chatKey, err := prepareEncodedUserData(contact)
	s.Require().NoError(err)

	expectedURL := fmt.Sprintf("%s/u/%s#%s", baseShareURL, userData, chatKey)
	s.Require().Equal(expectedURL, url)
}

func (s *ShareUrlsSuite) TestPrepareEncodedUserDataWithEmptyAccount() {
	contact := s.fakeContact()
	contact.DisplayName = ""
	contact.Bio = ""

	s.provider.EXPECT().GetContactByID(gomock.Any()).Return(contact, nil).Times(0)
	s.provider.EXPECT().GetCommunityByID(gomock.Any()).Times(0)

	userData, chatKey, err := prepareEncodedUserData(contact)
	s.Require().NoError(err)
	// The data should be empty if no display name or bio is set
	s.Require().Empty(userData)
	s.Require().NotEmpty(chatKey)
}

func (s *ShareUrlsSuite) TestShareAndParseUserURLWithData() {
	contact := s.fakeContact()

	s.provider.EXPECT().GetContactByID(contact.ID).Return(contact, nil).Times(1)
	s.provider.EXPECT().GetCommunityByID(gomock.Any()).Times(0)

	shortKey, err := multiformat.SerializeLegacyKey(contact.ID)
	s.Require().NoError(err)

	url, err := s.service.ShareUserURLWithData(contact.ID)
	s.Require().NoError(err)

	urlData, err := ParseSharedURL(url)
	s.Require().NoError(err)

	s.Require().NotNil(urlData.Contact)
	s.Require().Equal(contact.DisplayName, urlData.Contact.DisplayName)
	s.Require().Equal(shortKey, urlData.Contact.PublicKey)
}
