package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"
)

func setupTest(t *testing.T) (*sql.DB, *zap.Logger) {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	err = sqlite.Migrate(db)
	require.NoError(t, err)

	logger := logutils.ZapLogger()
	return db, logger
}

func createUserMessage(t *testing.T, db *sql.DB, msg *common.Message) {
	whisperTimestamp := 0
	source := ""
	text := ""
	contentType := 0
	timestamp := 0
	chatID := "1"
	localChatID := "1"
	responseTo := ""
	clockValue := 0

	stmt, err := db.Prepare(`
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
	require.NoError(t, err)

	links, err := json.Marshal(msg.UnfurledLinks)
	require.NoError(t, err)

	statusLinks, err := json.Marshal(msg.UnfurledStatusLinks)
	require.NoError(t, err)

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
	require.NoError(t, err)
}

func httpGetReqRecorder(t *testing.T, handler http.HandlerFunc, reqURL string) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", reqURL, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

func TestHandleLinkPreviewThumbnail(t *testing.T) {
	db, logger := setupTest(t)
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
	createUserMessage(t, db, &msg)

	// Test happy path.
	reqURL := "/dummy?" + url.Values{"message-id": {msg.ID}, "url": {previewURL}}.Encode()
	rr := httpGetReqRecorder(t, handleLinkPreviewThumbnail(db, logger), reqURL)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, msg.UnfurledLinks[0].ThumbnailPayload, rr.Body.Bytes())
	require.Equal(t, "image/jpeg", rr.HeaderMap.Get("Content-Type"))
	require.Equal(t, "no-store", rr.HeaderMap.Get("Cache-Control"))

	// Test bad requests.
	reqURL = "/dummy?" + url.Values{"message-id": {msg.ID}}.Encode()
	rr = httpGetReqRecorder(t, handleLinkPreviewThumbnail(db, logger), reqURL)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "missing query parameter 'url'\n", rr.Body.String())

	reqURL = "/dummy?" + url.Values{"url": {previewURL}}.Encode()
	rr = httpGetReqRecorder(t, handleLinkPreviewThumbnail(db, logger), reqURL)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Equal(t, "missing query parameter 'message-id'\n", rr.Body.String())

	// Test mime type not supported.
	msg.UnfurledLinks[0].ThumbnailPayload = []byte("unsupported image")
	createUserMessage(t, db, &msg)
	reqURL = "/dummy?" + url.Values{"message-id": {msg.ID}, "url": {previewURL}}.Encode()
	rr = httpGetReqRecorder(t, handleLinkPreviewThumbnail(db, logger), reqURL)
	require.Equal(t, http.StatusNotImplemented, rr.Code)
}
