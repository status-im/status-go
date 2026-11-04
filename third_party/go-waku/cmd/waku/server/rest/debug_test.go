package rest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/node"
)

func TestGetV1Info(t *testing.T) {
	wakuNode1, err := node.New()
	require.NoError(t, err)
	defer wakuNode1.Stop()
	err = wakuNode1.Start(context.Background())
	require.NoError(t, err)

	d := &DebugService{
		node: wakuNode1,
	}

	request, err := http.NewRequest(http.MethodPost, routeDebugInfoV1, bytes.NewReader([]byte("")))
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	d.getV1Info(rr, request)

	require.Equal(t, http.StatusOK, rr.Code)
}
