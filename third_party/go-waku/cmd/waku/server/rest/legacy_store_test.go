package rest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/tests"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

func TestGetMessages(t *testing.T) {

	db := MemoryDB(t)

	node1, err := node.New(node.WithWakuStore(), node.WithMessageProvider(db))
	require.NoError(t, err)
	err = node1.Start(context.Background())
	require.NoError(t, err)
	defer node1.Stop()

	topic1 := "1"
	pubsubTopic1 := "topic1"

	now := *utils.GetUnixEpoch()
	msg1 := tests.CreateWakuMessage(topic1, proto.Int64(now+1))
	msg2 := tests.CreateWakuMessage(topic1, proto.Int64(now+2))
	msg3 := tests.CreateWakuMessage(topic1, proto.Int64(now+3))

	node1.Broadcaster().Submit(protocol.NewEnvelope(msg1, *utils.GetUnixEpoch(), pubsubTopic1))
	node1.Broadcaster().Submit(protocol.NewEnvelope(msg2, *utils.GetUnixEpoch(), pubsubTopic1))
	node1.Broadcaster().Submit(protocol.NewEnvelope(msg3, *utils.GetUnixEpoch(), pubsubTopic1))

	n1HostInfo, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", node1.Host().ID().String()))
	n1Addr := node1.ListenAddresses()[0].Encapsulate(n1HostInfo)

	node2, err := node.New()
	require.NoError(t, err)
	err = node2.Start(context.Background())
	require.NoError(t, err)
	defer node2.Stop()
	router := chi.NewRouter()

	_ = NewLegacyStoreService(node2, router)

	// TEST: get cursor
	// TEST: get no messages

	// First page
	rr := httptest.NewRecorder()
	queryParams := url.Values{
		"peerAddr":    {n1Addr.String()},
		"pubsubTopic": {pubsubTopic1},
		"pageSize":    {"2"},
	}
	path := routeLegacyStoreMessagesV1 + "?" + queryParams.Encode()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	response := LegacyStoreResponse{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Len(t, response.Messages, 2)

	// Second page
	rr = httptest.NewRecorder()
	queryParams = url.Values{
		"peerAddr":    {n1Addr.String()},
		"pubsubTopic": {pubsubTopic1},
		"senderTime":  {response.Cursor.SenderTime},
		"storeTime":   {response.Cursor.StoreTime},
		"digest":      {base64.URLEncoding.EncodeToString(response.Cursor.Digest)},
		"pageSize":    {"2"},
	}
	path = routeLegacyStoreMessagesV1 + "?" + queryParams.Encode()
	req, _ = http.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	response = LegacyStoreResponse{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	require.Len(t, response.Messages, 1)
	require.Nil(t, response.Cursor)
}
