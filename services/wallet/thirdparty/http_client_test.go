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

func TestHTTPClient_DoGetRequest(t *testing.T) {
	// Create a new HTTPClient
	client := NewHTTPClient()

	expectedResponse := []byte("test response")

	// Create a mock server
	server := createMockServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "/test", r.URL.Path)
		require.Equal(t, "value1", r.URL.Query().Get("param1"))
		require.Equal(t, "value2", r.URL.Query().Get("param2"))

		authToken := base64.StdEncoding.EncodeToString([]byte("username:password"))
		require.Equal(t, fmt.Sprintf("Basic %s", authToken), r.Header.Get("Authorization"))

		// Set the response headers
		w.Header().Set("Content-Type", "application/json")
		// Set the response body
		_, _ = w.Write(expectedResponse)
	}))
	defer server.Close()

	// Set up test data
	expectedURL := server.URL + "/test"
	expectedParams := url.Values{}
	expectedParams.Set("param1", "value1")
	expectedParams.Set("param2", "value2")
	expectedCreds := &BasicCreds{
		User:     "username",
		Password: "password",
	}

	// Make the GET request
	ctx := context.Background()
	response, err := client.DoGetRequest(ctx, expectedURL, expectedParams, expectedCreds)

	// Verify the request
	require.NoError(t, err)
	require.Equal(t, expectedResponse, response)
}

func createMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}
