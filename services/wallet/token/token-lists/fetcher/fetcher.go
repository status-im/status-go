package fetcher

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const (
	dialTimeout           = 5 * time.Second
	tlsHandshakeTimeout   = 5 * time.Second
	responseHeaderTimeout = 5 * time.Second
	requestTimeout        = 20 * time.Second
	retries               = 3
)

type TokenList struct {
	ID        string `json:"id"`
	SourceURL string `json:"sourceUrl"`
	Schema    string `json:"schema"`
}

type FetchedTokenList struct {
	TokenList
	Etag     string
	Fetched  time.Time
	JsonData string
}

type TokenListsFetcher struct {
	listOfTokenListsURL string
	walletDb            *sql.DB
	httpClient          *thirdparty.HTTPClient
}

// NewTokenListsFetcher creates a new instance of TokenListsFetcher.
func NewTokenListsFetcher(walletDb *sql.DB) *TokenListsFetcher {
	return &TokenListsFetcher{
		walletDb: walletDb,
		httpClient: thirdparty.NewHTTPClient(
			thirdparty.WithDetailedTimeouts(
				dialTimeout,
				tlsHandshakeTimeout,
				responseHeaderTimeout,
				requestTimeout,
			),
			thirdparty.WithMaxRetries(retries),
		),
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
			dbEtag, err := t.GetEtagForTokenList(tokenList.ID)
			if err != nil {
				logutils.ZapLogger().Error("Failed to get etag for token list", zap.Error(err), zap.String("list-id", tokenList.ID))
				return nil
			}
			err = t.fetchTokenList(ctx, tokenList, dbEtag, tokenChannel)
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
		if err := t.StoreTokenList(fetchedList.ID, fetchedList.SourceURL, fetchedList.Etag, fetchedList.JsonData); err != nil {
			logutils.ZapLogger().Error("Failed to store token list", zap.Error(err))
		} else {
			successfullyFetchedListsCount++
		}
	}

	return successfullyFetchedListsCount, nil
}
