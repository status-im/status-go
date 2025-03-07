package fetcher

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/async"
)

const (
	dialTimeout           = 5 * time.Second
	tlsHandshakeTimeout   = 5 * time.Second
	responseHeaderTimeout = 5 * time.Second
	requestTimeout        = 20 * time.Second
)

type TokenList struct {
	ID        string `json:"id"`
	SourceURL string `json:"sourceUrl"`
	Schema    string `json:"schema"`
}

type FetchedTokenList struct {
	TokenList
	Fetched  time.Time
	JsonData string
}

type TokenListsFetcher struct {
	listOfTokenListsURL string
	walletDb            *sql.DB
	client              *http.Client
}

// NewTokenListsFetcher creates a new instance of TokenListsFetcher.
func NewTokenListsFetcher(walletDb *sql.DB) *TokenListsFetcher {
	return &TokenListsFetcher{
		walletDb: walletDb,
		client: &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: dialTimeout, // Timeout for establishing a connection
				}).DialContext,
				TLSHandshakeTimeout:   tlsHandshakeTimeout,   // Timeout for TLS handshake
				ResponseHeaderTimeout: responseHeaderTimeout, // Timeout for receiving response headers
			},
		},
	}
}

// SetURLOfRemoteListOfTokenLists sets the URL to fetch the list of token lists from.
func (t *TokenListsFetcher) SetURLOfRemoteListOfTokenLists(url string) {
	t.listOfTokenListsURL = url
}

// FetchAndStore fetches token lists from remote sources and stores them in the database.
func (t *TokenListsFetcher) FetchAndStore(ctx context.Context) (int, error) {
	tokenLists, err := t.fetchListOfTokenLists(ctx)
	if err != nil {
		return 0, err
	}

	var group = async.NewAtomicGroup(ctx)
	tokenChannel := make(chan FetchedTokenList, len(tokenLists))

	for _, tokenList := range tokenLists {
		group.Add(func(c context.Context) error {
			err := t.fetchTokenList(ctx, tokenList, tokenChannel)
			if err != nil {
				logutils.ZapLogger().Error("Failed to fetch token list", zap.Error(err), zap.String("list-id", tokenList.ID))
			}
			// Don't return error, continue fetching other token lists
			return nil
		})
	}

	group.Wait()
	close(tokenChannel)

	var successfullyFetchedListsCount int
	for fetchedList := range tokenChannel {
		if err := t.StoreTokenList(fetchedList.ID, fetchedList.JsonData); err != nil {
			logutils.ZapLogger().Error("Failed to store token list", zap.Error(err))
		} else {
			successfullyFetchedListsCount++
		}
	}

	return successfullyFetchedListsCount, nil
}
