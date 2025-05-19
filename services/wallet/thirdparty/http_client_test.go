package thirdparty

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/security"
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
				User:     security.NewSensitiveString("username"),
				Password: security.NewSensitiveString("password"),
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
					authToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", data.credentials.User.Reveal(), data.credentials.Password.Reveal())))
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

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		endpoint string
		wantURL  string
	}{
		{
			name:     "no slashes",
			baseURL:  "https://api.example.com",
			endpoint: "v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "proxy URL with trailing slash",
			baseURL:  "https://api.example.com/",
			endpoint: "v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "endpoint with leading slash",
			baseURL:  "https://api.example.com",
			endpoint: "/v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "both with slashes",
			baseURL:  "https://api.example.com/",
			endpoint: "/v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "multiple trailing slashes",
			baseURL:  "https://api.example.com///",
			endpoint: "v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "multiple leading slashes",
			baseURL:  "https://api.example.com",
			endpoint: "///v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
	}

	client := NewHTTPClient()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.BuildURL(tt.baseURL, tt.endpoint)
			require.Equal(t, tt.wantURL, got)
		})
	}
}

func TestFetchDataCompression(t *testing.T) {
	tests := []struct {
		name           string
		useGzip        bool
		enableGzip     bool
		responseBody   string
		expectedStatus int
	}{
		{
			name:           "server sends plain text when gzip not requested",
			useGzip:        false,
			enableGzip:     false,
			responseBody:   `{"data": "test"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "server sends plain text when gzip requested",
			useGzip:        false,
			enableGzip:     true,
			responseBody:   `{"data": "test"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "server sends gzip when requested",
			useGzip:        true,
			enableGzip:     true,
			responseBody:   `{"data": "test"}`,
			expectedStatus: http.StatusOK,
		},
	}

	ctx := context.Background()
	client := NewHTTPClient()
	creds := BasicCreds{User: security.NewSensitiveString("testuser"), Password: security.NewSensitiveString("testpass")}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if client requested gzip
				acceptsGzip := false
				for _, encoding := range r.Header["Accept-Encoding"] {
					if encoding == "gzip" {
						acceptsGzip = true
						break
					}
				}

				// Verify auth
				username, password, ok := r.BasicAuth()
				require.True(t, ok)
				require.Equal(t, "testuser", username)
				require.Equal(t, "testpass", password)

				if tt.useGzip && acceptsGzip {
					// Send gzipped response
					gzippedData, err := gzipEncode([]byte(tt.responseBody))
					require.NoError(t, err)

					w.Header().Set("Content-Encoding", "gzip")
					w.WriteHeader(tt.expectedStatus)
					_, err = w.Write(gzippedData)
					require.NoError(t, err)
				} else {
					// Send plain response
					w.WriteHeader(tt.expectedStatus)
					_, err := w.Write([]byte(tt.responseBody))
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			options := []RequestOption{}
			if tt.enableGzip {
				options = append(options, WithGzip())
			}
			// Make request
			body, err := client.DoGetRequestWithCredentials(ctx, server.URL, nil, &creds, options...)

			// Verify response
			require.NoError(t, err)
			require.Equal(t, tt.responseBody, string(body))
		})
	}
}

func gzipEncode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, err
	}
	err = gzipWriter.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
