package rest

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	storepb "github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"google.golang.org/protobuf/proto"
)

type StoreQueryService struct {
	node *node.WakuNode
	mux  *chi.Mux
}

const routeStoreMessagesV1 = "/store/v3/messages"

func NewStoreQueryService(node *node.WakuNode, m *chi.Mux) *StoreQueryService {
	s := &StoreQueryService{
		node: node,
		mux:  m,
	}

	m.Get(routeStoreMessagesV1, s.getV3Messages)

	return s
}

func getStoreParams(r *http.Request) (store.Criteria, []store.RequestOption, error) {
	var options []store.RequestOption
	var err error
	peerAddrStr := r.URL.Query().Get("peerAddr")
	var m multiaddr.Multiaddr
	if peerAddrStr != "" {
		m, err = multiaddr.NewMultiaddr(peerAddrStr)
		if err != nil {
			return nil, nil, err
		}
		options = append(options, store.WithPeerAddr(m))
	}

	includeData := false
	includeDataStr := r.URL.Query().Get("includeData")
	if includeDataStr != "" {
		includeData, err = strconv.ParseBool(includeDataStr)
		if err != nil {
			return nil, nil, errors.New("invalid value for includeData. Use true|false")
		}
	}
	options = append(options, store.IncludeData(includeData))

	pubsubTopic := r.URL.Query().Get("pubsubTopic")

	contentTopics := r.URL.Query().Get("contentTopics")
	var contentTopicsArr []string
	if contentTopics != "" {
		contentTopicsArr = strings.Split(contentTopics, ",")
	}

	hashesStr := r.URL.Query().Get("hashes")
	var hashes []pb.MessageHash
	if hashesStr != "" {
		hashesStrArr := strings.Split(hashesStr, ",")
		for _, hashStr := range hashesStrArr {
			hash, err := base64.URLEncoding.DecodeString(hashStr)
			if err != nil {
				return nil, nil, err
			}
			hashes = append(hashes, pb.ToMessageHash(hash))
		}
	}

	isMsgHashCriteria := false
	if len(hashes) != 0 {
		isMsgHashCriteria = true
		if pubsubTopic != "" || len(contentTopics) != 0 {
			return nil, nil, errors.New("cant use content filters while specifying message hashes")
		}
	} else {
		if pubsubTopic == "" || len(contentTopicsArr) == 0 {
			return nil, nil, errors.New("pubsubTopic and contentTopics are required")
		}
	}

	startTimeStr := r.URL.Query().Get("startTime")
	var startTime *int64
	if startTimeStr != "" {
		startTimeValue, err := strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		startTime = &startTimeValue
	}

	endTimeStr := r.URL.Query().Get("endTime")
	var endTime *int64
	if endTimeStr != "" {
		endTimeValue, err := strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		endTime = &endTimeValue
	}

	var cursor []byte
	cursorStr := r.URL.Query().Get("cursor")
	if cursorStr != "" {
		cursor, err = base64.URLEncoding.DecodeString(cursorStr)
		if err != nil {
			return nil, nil, err
		}
		options = append(options, store.WithCursor(cursor))
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

		options = append(options, store.WithPaging(ascending, pageSize))
	}

	var query store.Criteria
	if isMsgHashCriteria {
		query = store.MessageHashCriteria{
			MessageHashes: hashes,
		}
	} else {
		query = store.FilterCriteria{
			ContentFilter: protocol.NewContentFilter(pubsubTopic, contentTopicsArr...),
			TimeStart:     startTime,
			TimeEnd:       endTime,
		}
	}

	return query, options, nil
}

func writeStoreError(w http.ResponseWriter, code int, err error) {
	writeResponse(w, &storepb.StoreQueryResponse{StatusCode: proto.Uint32(uint32(code)), StatusDesc: proto.String(err.Error())}, code)
}

func (d *StoreQueryService) getV3Messages(w http.ResponseWriter, r *http.Request) {
	query, options, err := getStoreParams(r)
	if err != nil {
		writeStoreError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := d.node.Store().Request(ctx, query, options...)
	if err != nil {
		writeLegacyStoreError(w, http.StatusInternalServerError, err)
		return
	}

	writeErrOrResponse(w, nil, result.Response())
}
