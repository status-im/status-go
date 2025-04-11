package leaderboard

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name     string
		proxyURL string
		endpoint string
		wantURL  string
	}{
		{
			name:     "no slashes",
			proxyURL: "https://api.example.com",
			endpoint: "v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "proxy URL with trailing slash",
			proxyURL: "https://api.example.com/",
			endpoint: "v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "endpoint with leading slash",
			proxyURL: "https://api.example.com",
			endpoint: "/v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "both with slashes",
			proxyURL: "https://api.example.com/",
			endpoint: "/v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "multiple trailing slashes",
			proxyURL: "https://api.example.com///",
			endpoint: "v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
		{
			name:     "multiple leading slashes",
			proxyURL: "https://api.example.com",
			endpoint: "///v1/data",
			wantURL:  "https://api.example.com/v1/data",
		},
	}

	handler := &RequestHandler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.buildURL(tt.proxyURL, tt.endpoint)
			require.Equal(t, tt.wantURL, got)
		})
	}
}

func TestCreateRequest(t *testing.T) {
	handler := &RequestHandler{
		config: ServiceConfig{
			ProxyURL:  "https://api.example.com",
			Login:     "testuser",
			Password:  "testpass",
			AllowGzip: true,
			AllowETag: true,
		},
	}

	etag := "test-etag"
	req, err := handler.createRequest(context.Background(), "v1/data", &etag)
	require.NoError(t, err)

	// Check URL
	require.Equal(t, "https://api.example.com/v1/data", req.URL.String())

	// Check auth
	username, password, ok := req.BasicAuth()
	require.True(t, ok)
	require.Equal(t, "testuser", username)
	require.Equal(t, "testpass", password)

	// Check headers
	require.Equal(t, "gzip", req.Header.Get("Accept-Encoding"))
	require.Equal(t, "test-etag", req.Header.Get("If-None-Match"))
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

			// Create handler with test configuration
			handler := NewRequestHandler(ServiceConfig{
				ProxyURL:  server.URL,
				Login:     "testuser",
				Password:  "testpass",
				AllowGzip: tt.enableGzip,
			}, server.Client())

			// Make request
			body, ok := handler.FetchData(context.Background(), "/test", new(string))

			// Verify response
			require.True(t, ok)
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
