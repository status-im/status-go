package rest

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store/pb"
)

type LegacyStoreService struct {
	node *node.WakuNode
	mux  *chi.Mux
}

type LegacyStoreResponse struct {
	Messages     []LegacyStoreWakuMessage `json:"messages"`
	Cursor       *LegacyHistoryCursor     `json:"cursor,omitempty"`
	ErrorMessage string                   `json:"error_message,omitempty"`
}

type LegacyHistoryCursor struct {
	PubsubTopic string `json:"pubsubTopic"`
	SenderTime  string `json:"senderTime"`
	StoreTime   string `json:"storeTime"`
	Digest      []byte `json:"digest"`
}

type LegacyStoreWakuMessage struct {
	Payload      []byte  `json:"payload"`
	ContentTopic string  `json:"contentTopic"`
	Version      *uint32 `json:"version,omitempty"`
	Timestamp    *int64  `json:"timestamp,omitempty"`
	Meta         []byte  `json:"meta,omitempty"`
}

const routeLegacyStoreMessagesV1 = "/store/v1/messages"

func NewLegacyStoreService(node *node.WakuNode, m *chi.Mux) *LegacyStoreService {
	s := &LegacyStoreService{
		node: node,
		mux:  m,
	}

	m.Get(routeLegacyStoreMessagesV1, s.getV1Messages)

	return s
}

func getLegacyStoreParams(r *http.Request) (*legacy_store.Query, []legacy_store.HistoryRequestOption, error) {
	query := &legacy_store.Query{}
	var options []legacy_store.HistoryRequestOption
	var err error
	peerAddrStr := r.URL.Query().Get("peerAddr")
	var m multiaddr.Multiaddr
	if peerAddrStr != "" {
		m, err = multiaddr.NewMultiaddr(peerAddrStr)
		if err != nil {
			return nil, nil, err
		}
		options = append(options, legacy_store.WithPeerAddr(m))
	} else {
		// The user didn't specify a peer address and self-node is configured as a store node.
		// In this case we assume that the user is willing to retrieve the messages stored by
		// the local/self store node.
		options = append(options, legacy_store.WithLocalQuery())
	}

	query.PubsubTopic = r.URL.Query().Get("pubsubTopic")

	contentTopics := r.URL.Query().Get("contentTopics")
	if contentTopics != "" {
		query.ContentTopics = strings.Split(contentTopics, ",")
	}

	startTimeStr := r.URL.Query().Get("startTime")
	if startTimeStr != "" {
		startTime, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		query.StartTime = &startTime
	}

	endTimeStr := r.URL.Query().Get("endTime")
	if endTimeStr != "" {
		endTime, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		query.EndTime = &endTime
	}

	var cursor *pb.Index

	senderTimeStr := r.URL.Query().Get("senderTime")
	storeTimeStr := r.URL.Query().Get("storeTime")
	digestStr := r.URL.Query().Get("digest")

	if senderTimeStr != "" || storeTimeStr != "" || digestStr != "" {
		cursor = &pb.Index{}

		if senderTimeStr != "" {
			cursor.SenderTime, err = strconv.ParseInt(senderTimeStr, 10, 64)
			if err != nil {
				return nil, nil, err
			}
		}

		if storeTimeStr != "" {
			cursor.ReceiverTime, err = strconv.ParseInt(storeTimeStr, 10, 64)
			if err != nil {
				return nil, nil, err
			}
		}

		if digestStr != "" {
			cursor.Digest, err = base64.URLEncoding.DecodeString(digestStr)
			if err != nil {
				return nil, nil, err
			}
		}

		cursor.PubsubTopic = query.PubsubTopic

		options = append(options, legacy_store.WithCursor(cursor))
	}

	pageSizeStr := r.URL.Query().Get("pageSize")
	ascendingStr := r.URL.Query().Get("ascending")
	if ascendingStr != "" || pageSizeStr != "" {
		ascending := true
		pageSize := uint64(legacy_store.DefaultPageSize)
		if ascendingStr != "" {
			ascending, err = strconv.ParseBool(ascendingStr)
			if err != nil {
				return nil, nil, err
			}
		}

		if pageSizeStr != "" {
			pageSize, err = strconv.ParseUint(pageSizeStr, 10, 64)
			if err != nil {
				return nil, nil, err
			}
			if pageSize > legacy_store.MaxPageSize {
				pageSize = legacy_store.MaxPageSize
			}
		}

		options = append(options, legacy_store.WithPaging(ascending, pageSize))
	}

	return query, options, nil
}

func writeLegacyStoreError(w http.ResponseWriter, code int, err error) {
	writeResponse(w, LegacyStoreResponse{ErrorMessage: err.Error()}, code)
}

func toLegacyStoreResponse(result *legacy_store.Result) LegacyStoreResponse {
	response := LegacyStoreResponse{}

	cursor := result.Cursor()
	if cursor != nil {
		response.Cursor = &LegacyHistoryCursor{
			PubsubTopic: cursor.PubsubTopic,
			SenderTime:  fmt.Sprintf("%d", cursor.SenderTime),
			StoreTime:   fmt.Sprintf("%d", cursor.ReceiverTime),
			Digest:      cursor.Digest,
		}
	}

	for _, m := range result.Messages {
		response.Messages = append(response.Messages, LegacyStoreWakuMessage{
			Payload:      m.Payload,
			ContentTopic: m.ContentTopic,
			Version:      m.Version,
			Timestamp:    m.Timestamp,
			Meta:         m.Meta,
		})
	}

	return response
}

func (d *LegacyStoreService) getV1Messages(w http.ResponseWriter, r *http.Request) {
	query, options, err := getLegacyStoreParams(r)
	if err != nil {
		writeLegacyStoreError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := d.node.LegacyStore().Query(ctx, *query, options...)
	if err != nil {
		writeLegacyStoreError(w, http.StatusInternalServerError, err)
		return
	}

	writeErrOrResponse(w, nil, toLegacyStoreResponse(result))
}
