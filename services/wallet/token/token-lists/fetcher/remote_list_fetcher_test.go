package fetcher

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchingListOfTokensList(t *testing.T) {
	var tests = []struct {
		name           string
		setURL         bool
		responseStatus int
		response       string
		err            error
		expected       []TokenList
	}{
		{
			name:           "status ok response",
			setURL:         true,
			responseStatus: http.StatusOK,
			response:       listOfTokenListsJsonResponse,
			expected:       listOfTokenLists,
		},
		{
			name:           "status not found response",
			setURL:         true,
			responseStatus: http.StatusNotFound,
			response:       listOfTokenListsJsonResponse,
			expected:       defaultTokensList,
		},
		{
			name:           "content of the response does not match the schema",
			setURL:         true,
			responseStatus: http.StatusOK,
			response: `[
									{
										"id": "uniswap",
									}
								]`,
			expected: defaultTokensList,
		},
		{
			name:     "remote url not set",
			setURL:   false,
			expected: defaultTokensList,
		},
	}

	tokenListsFetcher := NewTokenListsFetcher(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedListCopy := make([]TokenList, len(tt.expected))
			copy(expectedListCopy, tt.expected)
			tt.expected = expectedListCopy

			if tt.setURL {
				serverURL := ""
				var server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.responseStatus)
					resp := strings.ReplaceAll(tt.response, serverURLPlaceholder, serverURL)
					if _, err := w.Write([]byte(resp)); err != nil {
						log.Println(err.Error())
					}
				}))
				defer server.Close()

				serverURL = server.URL
				tokenListsFetcher.SetURLOfRemoteListOfTokenLists(server.URL)

				for i := range tt.expected {
					tokenList := &tt.expected[i]
					tokenList.SourceURL = strings.ReplaceAll(tokenList.SourceURL, serverURLPlaceholder, serverURL)
					tokenList.Schema = strings.ReplaceAll(tokenList.Schema, serverURLPlaceholder, serverURL)
				}
			}

			tokenLists, err := tokenListsFetcher.fetchListOfTokenLists(context.TODO())
			require.NoError(t, err)
			require.Equal(t, tt.expected, tokenLists)
		})
	}
}
