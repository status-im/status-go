package thirdparty

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPClient_GetRequests(t *testing.T) {
	const (
		callDoGetRequest                = "DoGetRequest"
		callDoGetRequestWithCredentials = "DoGetRequestWithCredentials"
		callDoGetRequestWithEtag        = "DoGetRequestWithHeaders"
	)

	testData := []struct {
		name            string
		call            string
		url             string
		statusCode      int
		params          url.Values
		credentials     *BasicCreds
		etag            string
		expectedHeaders map[string]string
		expectedBody    []byte
		expectedEtag    string
	}{
		{
			name:       "simple get request with no params",
			call:       callDoGetRequest,
			url:        "/test-no-params",
			statusCode: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			expectedBody: []byte("test response for no params"),
		},
		{
			name:       "simple get request with params",
			call:       callDoGetRequest,
			url:        "/test-params",
			statusCode: http.StatusOK,
			params: url.Values{
				"param1": []string{"value1"},
				"param2": []string{"value2"},
			},
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			expectedBody: []byte("test response for params"),
		},
		{
			name:       "simple get request with credentials",
			call:       callDoGetRequestWithCredentials,
			url:        "/test-credentials",
			statusCode: http.StatusOK,
			credentials: &BasicCreds{
				User:     "username",
				Password: "password",
			},
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			expectedBody: []byte("test response for credentials"),
		},
		{
			name:       "simple get request with etag not modified content",
			call:       callDoGetRequestWithEtag,
			url:        "/test-etag",
			statusCode: http.StatusNotModified,
			params: url.Values{
				"param1": []string{"value1"},
				"param2": []string{"value2"},
			},
			etag: "oldEtag",
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
				"ETag":         "oldEtag",
			},
			expectedBody: nil,
			expectedEtag: "oldEtag",
		},
		{
			name:       "simple get request with etag modified content",
			call:       callDoGetRequestWithEtag,
			url:        "/test-etag",
			statusCode: http.StatusOK,
			params: url.Values{
				"param1": []string{"value1"},
				"param2": []string{"value2"},
			},
			etag: "oldEtag",
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
				"ETag":         "newEtag",
			},
			expectedBody: []byte("test response for etag"),
			expectedEtag: "newEtag",
		},
	}

	ctx := context.Background()
	client := NewHTTPClient()
	for _, data := range testData {
		t.Run(data.name, func(t *testing.T) {
			server := createMockServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				require.Equal(t, data.url, r.URL.Path)
				for param, value := range data.params {
					require.Equal(t, value[0], r.URL.Query().Get(param))
				}
				if data.credentials != nil {
					authToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", data.credentials.User, data.credentials.Password)))
					require.Equal(t, fmt.Sprintf("Basic %s", authToken), r.Header.Get("Authorization"))
				}

				if data.etag != "" {
					require.Equal(t, data.etag, r.Header.Get("If-None-Match"))
				}

				for key, value := range data.expectedHeaders {
					w.Header().Set(key, value)
				}
				w.WriteHeader(data.statusCode)
				_, _ = w.Write(data.expectedBody)
			}))
			defer server.Close()

			if data.call == callDoGetRequest {
				response, err := client.DoGetRequest(ctx, server.URL+data.url, data.params)
				require.NoError(t, err)
				require.Equal(t, data.expectedBody, response)
			} else if data.call == callDoGetRequestWithCredentials {
				response, err := client.DoGetRequestWithCredentials(ctx, server.URL+data.url, data.params, data.credentials)
				require.NoError(t, err)
				require.Equal(t, data.expectedBody, response)
			} else if data.call == callDoGetRequestWithEtag {
				response, newEtag, err := client.DoGetRequestWithEtag(ctx, server.URL+data.url, data.params, data.etag)
				require.NoError(t, err)
				require.Equal(t, data.expectedBody, response)
				require.Equal(t, data.expectedEtag, newEtag)
			}
		})
	}
}

func createMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}
