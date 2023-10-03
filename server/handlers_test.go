package server

import (
	"database/sql"
	"encoding/json"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"
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
	dbPath, err := ioutil.TempFile("", "status-go-test-db-")
	s.Require().NoError(err)

	db, err := sqlite.Open(dbPath.Name(), "", sqlite.ReducedKDFIterationsNumber)
	s.Require().NoError(err)

	s.logger = tt.MustCreateTestLogger()
	s.db = db
}

func (s *HandlersSuite) createUserMessage(msg *common.Message) {
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

	statusLinks, err := json.Marshal(msg.UnfurledStatusLinks)
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

func (s *HandlersSuite) TestHandleLinkPreviewThumbnail() {
	previewURL := "https://github.com"
	msg := common.Message{
		ID: "1",
		ChatMessage: &protobuf.ChatMessage{
			UnfurledLinks: []*protobuf.UnfurledLink{
				{
					Type:             protobuf.UnfurledLink_LINK,
					Url:              previewURL,
					ThumbnailWidth:   100,
					ThumbnailHeight:  200,
					ThumbnailPayload: []byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x50},
				},
			},
		},
	}
	s.createUserMessage(&msg)

	// Test happy path.
	reqURL := "/dummy?" + url.Values{"message-id": {msg.ID}, "url": {previewURL}}.Encode()
	rr := s.httpGetReqRecorder(handleLinkPreviewThumbnail(s.db, s.logger), reqURL)
	s.Require().Equal(http.StatusOK, rr.Code)
	s.Require().Equal(msg.UnfurledLinks[0].ThumbnailPayload, rr.Body.Bytes())
	s.Require().Equal("image/jpeg", rr.HeaderMap.Get("Content-Type"))
	s.Require().Equal("no-store", rr.HeaderMap.Get("Cache-Control"))

	// Test bad requests.
	reqURL = "/dummy?" + url.Values{"message-id": {msg.ID}}.Encode()
	rr = s.httpGetReqRecorder(handleLinkPreviewThumbnail(s.db, s.logger), reqURL)
	s.Require().Equal(http.StatusBadRequest, rr.Code)
	s.Require().Equal("missing query parameter 'url'\n", rr.Body.String())

	reqURL = "/dummy?" + url.Values{"url": {previewURL}}.Encode()
	rr = s.httpGetReqRecorder(handleLinkPreviewThumbnail(s.db, s.logger), reqURL)
	s.Require().Equal(http.StatusBadRequest, rr.Code)
	s.Require().Equal("missing query parameter 'message-id'\n", rr.Body.String())

	// Test mime type not supported.
	msg.UnfurledLinks[0].ThumbnailPayload = []byte("unsupported image")
	s.createUserMessage(&msg)
	reqURL = "/dummy?" + url.Values{"message-id": {msg.ID}, "url": {previewURL}}.Encode()
	rr = s.httpGetReqRecorder(handleLinkPreviewThumbnail(s.db, s.logger), reqURL)
	s.Require().Equal(http.StatusNotImplemented, rr.Code)
}

func (s *HandlersSuite) TestHandleStatusLinkPreviewThumbnail() {
	thumbnailPayload := []byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x50}

	contact := &protobuf.UnfurledStatusContactLink{
		PublicKey: "PublicKey_1",
		Icon: &protobuf.UnfurledLinkThumbnail{
			Width:   10,
			Height:  20,
			Payload: thumbnailPayload,
		},
	}

	unfurledContact := &protobuf.UnfurledStatusLink{
		Url: "https://status.app/u/",
		Payload: &protobuf.UnfurledStatusLink_Contact{
			Contact: contact,
		},
	}

	community := &protobuf.UnfurledStatusCommunityLink{
		CommunityId: "CommunityId_1",
		Icon: &protobuf.UnfurledLinkThumbnail{
			Width:   30,
			Height:  40,
			Payload: thumbnailPayload,
		},
		Banner: &protobuf.UnfurledLinkThumbnail{
			Width:   50,
			Height:  60,
			Payload: thumbnailPayload,
		},
	}

	unfurledCommunity := &protobuf.UnfurledStatusLink{
		Url: "https://status.app/c/",
		Payload: &protobuf.UnfurledStatusLink_Community{
			Community: community,
		},
	}

	channel := &protobuf.UnfurledStatusChannelLink{
		ChannelUuid: "ChannelUuid_1",
		Community: &protobuf.UnfurledStatusCommunityLink{
			CommunityId: "CommunityId_2",
			Icon: &protobuf.UnfurledLinkThumbnail{
				Width:   70,
				Height:  80,
				Payload: thumbnailPayload,
			},
			Banner: &protobuf.UnfurledLinkThumbnail{
				Width:   90,
				Height:  100,
				Payload: thumbnailPayload,
			},
		},
	}

	unfurledChannel := &protobuf.UnfurledStatusLink{
		Url: "https://status.app/cc/",
		Payload: &protobuf.UnfurledStatusLink_Channel{
			Channel: channel,
		},
	}

	msg := common.Message{
		ID: "1",
		ChatMessage: &protobuf.ChatMessage{
			UnfurledStatusLinks: &protobuf.UnfurledStatusLinks{
				UnfurledStatusLinks: []*protobuf.UnfurledStatusLink{
					unfurledContact,
					unfurledCommunity,
					unfurledChannel,
				},
			},
		},
	}

	s.Require().NotNil(msg)
}
