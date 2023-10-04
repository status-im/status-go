package server

import (
	"database/sql"
	"encoding/json"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/t/helpers"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
)

func TestHandlersSuite(t *testing.T) {
	suite.Run(t, new(HandlersSuite))
}

type HandlersSuite struct {
	suite.Suite
	db     *sql.DB
	logger *zap.Logger
}

func (s *HandlersSuite) SetupTest() {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)

	err = sqlite.Migrate(db)
	s.Require().NoError(err)

	s.logger = tt.MustCreateTestLogger()
	s.db = db
}

func (s *HandlersSuite) saveUserMessage(msg *common.Message) {
	whisperTimestamp := 0
	source := ""
	text := ""
	contentType := 0
	timestamp := 0
	chatID := "1"
	localChatID := "1"
	responseTo := ""
	clockValue := 0

	stmt, err := s.db.Prepare(`
		INSERT INTO user_messages (
			id,
			whisper_timestamp,
			source,
			text,
			content_type,
			timestamp,
			chat_id,
			local_chat_id,
			response_to,
			clock_value,
			unfurled_links,
		    unfurled_status_links
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
	`)

	s.Require().NoError(err)

	links, err := json.Marshal(msg.UnfurledLinks)
	s.Require().NoError(err)

	statusLinks, err := proto.Marshal(msg.UnfurledStatusLinks)
	s.Require().NoError(err)

	_, err = stmt.Exec(
		msg.ID,
		whisperTimestamp,
		source,
		text,
		contentType,
		timestamp,
		chatID,
		localChatID,
		responseTo,
		clockValue,
		links,
		statusLinks,
	)
	s.Require().NoError(err)
}

func (s *HandlersSuite) httpGetReqRecorder(handler http.HandlerFunc, reqURL string) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", reqURL, nil)
	s.Require().NoError(err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

func (s *HandlersSuite) verifyHTTPResponseThumbnail(rr *httptest.ResponseRecorder, expectedPayload []byte) {
	s.Require().Equal(expectedPayload, rr.Body.Bytes())
	s.Require().Equal("image/jpeg", rr.HeaderMap.Get("Content-Type"))
	s.Require().Equal("no-store", rr.HeaderMap.Get("Cache-Control"))
}

func (s *HandlersSuite) TestHandleLinkPreviewThumbnail() {
	previewURL := "https://github.com"
	defaultPayload := []byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x50}

	msg := common.Message{
		ID: "1",
		ChatMessage: &protobuf.ChatMessage{
			UnfurledLinks: []*protobuf.UnfurledLink{
				{
					Type:            protobuf.UnfurledLink_LINK,
					Url:             previewURL,
					ThumbnailWidth:  100,
					ThumbnailHeight: 200,
				},
			},
		},
	}
	s.saveUserMessage(&msg)

	testCases := []struct {
		Name                   string
		ExpectedHTTPStatusCode int
		ThumbnailPayload       []byte
		Parameters             url.Values
		CheckFunc              func(s *HandlersSuite, rr *httptest.ResponseRecorder)
	}{
		{
			Name:                   "Test happy path",
			ExpectedHTTPStatusCode: http.StatusOK,
			ThumbnailPayload:       defaultPayload,
			Parameters: url.Values{
				"message-id": {msg.ID},
				"url":        {previewURL},
			},
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.verifyHTTPResponseThumbnail(rr, msg.UnfurledLinks[0].ThumbnailPayload)
			},
		},
		{
			Name:                   "Test request with missing 'url' parameter",
			ThumbnailPayload:       defaultPayload,
			ExpectedHTTPStatusCode: http.StatusBadRequest,
			Parameters: url.Values{
				"message-id": {msg.ID},
			},
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.Require().Equal("missing query parameter 'url'\n", rr.Body.String())
			},
		},
		{
			Name:                   "Test request with missing 'message-id' parameter",
			ThumbnailPayload:       defaultPayload,
			ExpectedHTTPStatusCode: http.StatusBadRequest,
			Parameters: url.Values{
				"url": {previewURL},
			},
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.Require().Equal("missing query parameter 'message-id'\n", rr.Body.String())
			},
		},
		{
			Name:                   "Test mime type not supported",
			ThumbnailPayload:       []byte("unsupported image"),
			ExpectedHTTPStatusCode: http.StatusNotImplemented,
			Parameters: url.Values{
				"message-id": {msg.ID},
				"url":        {previewURL},
			},
		},
	}

	handler := handleLinkPreviewThumbnail(s.db, s.logger)

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			msg.UnfurledLinks[0].ThumbnailPayload = tc.ThumbnailPayload
			s.saveUserMessage(&msg)

			requestURL := "/dummy?" + tc.Parameters.Encode()
			rr := s.httpGetReqRecorder(handler, requestURL)
			s.Require().Equal(tc.ExpectedHTTPStatusCode, rr.Code)
			if tc.CheckFunc != nil {
				tc.CheckFunc(s, rr)
			}
		})
	}
}

func (s *HandlersSuite) TestHandleStatusLinkPreviewThumbnail() {
	contact := &protobuf.UnfurledStatusContactLink{
		PublicKey: "PublicKey_1",
		Icon: &protobuf.UnfurledLinkThumbnail{
			Width:   10,
			Height:  20,
			Payload: []byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x50},
		},
	}

	contactWithUnsupportedImage := &protobuf.UnfurledStatusContactLink{
		PublicKey: "PublicKey_2",
		Icon: &protobuf.UnfurledLinkThumbnail{
			Width:   10,
			Height:  20,
			Payload: []byte("unsupported image"),
		},
	}

	community := &protobuf.UnfurledStatusCommunityLink{
		CommunityId: "CommunityId_1",
		Icon: &protobuf.UnfurledLinkThumbnail{
			Width:   30,
			Height:  40,
			Payload: []byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x51},
		},
		Banner: &protobuf.UnfurledLinkThumbnail{
			Width:   50,
			Height:  60,
			Payload: []byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x52},
		},
	}

	channel := &protobuf.UnfurledStatusChannelLink{
		ChannelUuid: "ChannelUuid_1",
		Community: &protobuf.UnfurledStatusCommunityLink{
			CommunityId: "CommunityId_2",
			Icon: &protobuf.UnfurledLinkThumbnail{
				Width:   70,
				Height:  80,
				Payload: []byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x53},
			},
			Banner: &protobuf.UnfurledLinkThumbnail{
				Width:   90,
				Height:  100,
				Payload: []byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x54},
			},
		},
	}

	unfurledContact := &protobuf.UnfurledStatusLink{
		Url: "https://status.app/u/",
		Payload: &protobuf.UnfurledStatusLink_Contact{
			Contact: contact,
		},
	}

	unfurledContactWithUnsupportedImage := &protobuf.UnfurledStatusLink{
		Url: "https://status.app/u/",
		Payload: &protobuf.UnfurledStatusLink_Contact{
			Contact: contactWithUnsupportedImage,
		},
	}

	unfurledCommunity := &protobuf.UnfurledStatusLink{
		Url: "https://status.app/c/",
		Payload: &protobuf.UnfurledStatusLink_Community{
			Community: community,
		},
	}

	unfurledChannel := &protobuf.UnfurledStatusLink{
		Url: "https://status.app/cc/",
		Payload: &protobuf.UnfurledStatusLink_Channel{
			Channel: channel,
		},
	}

	const (
		messageIDContactOnly      = "1"
		messageIDCommunityOnly    = "2"
		messageIDChannelOnly      = "3"
		messageIDAllLinks         = "4"
		messageIDUnsupportedImage = "5"
	)

	s.saveUserMessage(&common.Message{
		ID: messageIDContactOnly,
		ChatMessage: &protobuf.ChatMessage{
			UnfurledStatusLinks: &protobuf.UnfurledStatusLinks{
				UnfurledStatusLinks: []*protobuf.UnfurledStatusLink{
					unfurledContact,
				},
			},
		},
	})

	s.saveUserMessage(&common.Message{
		ID: messageIDCommunityOnly,
		ChatMessage: &protobuf.ChatMessage{
			UnfurledStatusLinks: &protobuf.UnfurledStatusLinks{
				UnfurledStatusLinks: []*protobuf.UnfurledStatusLink{
					unfurledCommunity,
				},
			},
		},
	})

	s.saveUserMessage(&common.Message{
		ID: messageIDChannelOnly,
		ChatMessage: &protobuf.ChatMessage{
			UnfurledStatusLinks: &protobuf.UnfurledStatusLinks{
				UnfurledStatusLinks: []*protobuf.UnfurledStatusLink{
					unfurledChannel,
				},
			},
		},
	})

	s.saveUserMessage(&common.Message{
		ID: messageIDAllLinks,
		ChatMessage: &protobuf.ChatMessage{
			UnfurledStatusLinks: &protobuf.UnfurledStatusLinks{
				UnfurledStatusLinks: []*protobuf.UnfurledStatusLink{
					unfurledContact,
					unfurledCommunity,
					unfurledChannel,
				},
			},
		},
	})

	s.saveUserMessage(&common.Message{
		ID: messageIDUnsupportedImage,
		ChatMessage: &protobuf.ChatMessage{
			UnfurledStatusLinks: &protobuf.UnfurledStatusLinks{
				UnfurledStatusLinks: []*protobuf.UnfurledStatusLink{
					unfurledContactWithUnsupportedImage,
				},
			},
		},
	})

	testCases := []struct {
		Name                   string
		ExpectedHTTPStatusCode int
		Parameters             url.Values
		CheckFunc              func(s *HandlersSuite, rr *httptest.ResponseRecorder)
	}{
		{
			Name: "Test valid contact icon link",
			Parameters: url.Values{
				"message-id": {messageIDContactOnly},
				"url":        {unfurledContact.Url},
				"image-id":   {string(common.MediaServerContactIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusOK,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.verifyHTTPResponseThumbnail(rr, unfurledContact.GetContact().Icon.Payload)
			},
		},
		{
			Name: "Test invalid request for community icon in a contact link",
			Parameters: url.Values{
				"message-id": {messageIDContactOnly},
				"url":        {unfurledContact.Url},
				"image-id":   {string(common.MediaServerCommunityIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusBadRequest,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.Require().Equal("invalid query parameter 'image-id' value: this is not a community link\n", rr.Body.String())
			},
		},
		{
			Name: "Test invalid request for cahnnel community banner in a contact link",
			Parameters: url.Values{
				"message-id": {messageIDContactOnly},
				"url":        {unfurledContact.Url},
				"image-id":   {string(common.MediaServerChannelCommunityBanner)},
			},
			ExpectedHTTPStatusCode: http.StatusBadRequest,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.Require().Equal("invalid query parameter 'image-id' value: this is not a community channel link\n", rr.Body.String())
			},
		},
		{
			Name: "Test invalid request for channel community banner in a contact link",
			Parameters: url.Values{
				"message-id": {messageIDContactOnly},
				"url":        {unfurledContact.Url},
				"image-id":   {"contact-banner"},
			},
			ExpectedHTTPStatusCode: http.StatusBadRequest,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.Require().Equal("invalid query parameter 'image-id' value: value not supported\n", rr.Body.String())
			},
		},
		{
			Name: "Test valid community icon link",
			Parameters: url.Values{
				"message-id": {messageIDCommunityOnly},
				"url":        {unfurledCommunity.Url},
				"image-id":   {string(common.MediaServerCommunityIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusOK,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.verifyHTTPResponseThumbnail(rr, unfurledCommunity.GetCommunity().Icon.Payload)
			},
		},
		{
			Name: "Test valid community banner link",
			Parameters: url.Values{
				"message-id": {messageIDCommunityOnly},
				"url":        {unfurledCommunity.Url},
				"image-id":   {string(common.MediaServerCommunityBanner)},
			},
			ExpectedHTTPStatusCode: http.StatusOK,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.verifyHTTPResponseThumbnail(rr, unfurledCommunity.GetCommunity().Banner.Payload)
			},
		},
		{
			Name: "Test valid channel community icon link",
			Parameters: url.Values{
				"message-id": {messageIDChannelOnly},
				"url":        {unfurledChannel.Url},
				"image-id":   {string(common.MediaServerChannelCommunityIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusOK,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.verifyHTTPResponseThumbnail(rr, unfurledChannel.GetChannel().GetCommunity().Icon.Payload)
			},
		},
		{
			Name: "Test valid channel community banner link",
			Parameters: url.Values{
				"message-id": {messageIDChannelOnly},
				"url":        {unfurledChannel.Url},
				"image-id":   {string(common.MediaServerChannelCommunityBanner)},
			},
			ExpectedHTTPStatusCode: http.StatusOK,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.verifyHTTPResponseThumbnail(rr, unfurledChannel.GetChannel().GetCommunity().Banner.Payload)
			},
		},
		{
			Name: "Test valid contact icon link in a diverse message",
			Parameters: url.Values{
				"message-id": {messageIDAllLinks},
				"url":        {unfurledContact.Url},
				"image-id":   {string(common.MediaServerContactIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusOK,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.verifyHTTPResponseThumbnail(rr, unfurledContact.GetContact().Icon.Payload)
			},
		},
		{
			Name: "Test valid community icon link in a diverse message",
			Parameters: url.Values{
				"message-id": {messageIDAllLinks},
				"url":        {unfurledCommunity.Url},
				"image-id":   {string(common.MediaServerCommunityIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusOK,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.verifyHTTPResponseThumbnail(rr, unfurledCommunity.GetCommunity().Icon.Payload)
			},
		},
		{
			Name: "Test valid channel community icon link in a diverse message",
			Parameters: url.Values{
				"message-id": {messageIDAllLinks},
				"url":        {unfurledChannel.Url},
				"image-id":   {string(common.MediaServerChannelCommunityIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusOK,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.verifyHTTPResponseThumbnail(rr, unfurledChannel.GetChannel().GetCommunity().Icon.Payload)
			},
		},
		{
			Name: "Test mime type not supported",
			Parameters: url.Values{
				"message-id": {messageIDUnsupportedImage},
				"url":        {unfurledContactWithUnsupportedImage.Url},
				"image-id":   {string(common.MediaServerContactIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusNotImplemented,
		},
		{
			Name: "Test request with missing 'message-id' parameter",
			Parameters: url.Values{
				"url":      {unfurledCommunity.Url},
				"image-id": {string(common.MediaServerCommunityIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusBadRequest,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.Require().Equal("missing query parameter 'message-id'\n", rr.Body.String())
			},
		},
		{
			Name: "Test request with missing 'url' parameter",
			Parameters: url.Values{
				"message-id": {messageIDCommunityOnly},
				"image-id":   {string(common.MediaServerCommunityIcon)},
			},
			ExpectedHTTPStatusCode: http.StatusBadRequest,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.Require().Equal("missing query parameter 'url'\n", rr.Body.String())
			},
		},
		{
			Name: "Test request with missing 'image-id' parameter",
			Parameters: url.Values{
				"message-id": {messageIDCommunityOnly},
				"url":        {unfurledCommunity.Url},
			},
			ExpectedHTTPStatusCode: http.StatusBadRequest,
			CheckFunc: func(s *HandlersSuite, rr *httptest.ResponseRecorder) {
				s.Require().Equal("missing query parameter 'image-id'\n", rr.Body.String())
			},
		},
	}

	handler := handleStatusLinkPreviewThumbnail(s.db, s.logger)

	for _, tc := range testCases {
		s.Run(tc.Name, func() {
			requestURL := "/dummy?" + tc.Parameters.Encode()

			rr := s.httpGetReqRecorder(handler, requestURL)
			s.Require().Equal(tc.ExpectedHTTPStatusCode, rr.Code)

			if tc.CheckFunc != nil {
				tc.CheckFunc(s, rr)
			}
		})
	}
}
