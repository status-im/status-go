package tokenlists

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/signal"
)

type TokensList struct {
	Name             string `json:"name"`
	Timestamp        string `json:"timestamp"`        // time when the list was last updated
	FetchedTimestamp string `json:"fetchedTimestamp"` // time when the list was fetched
	Source           string `json:"source"`
	Version          struct {
		Major int `json:"major"`
		Minor int `json:"minor"`
		Patch int `json:"patch"`
	} `json:"version"`
	Tags     map[string]interface{} `json:"tags"`
	LogoURI  string                 `json:"logoURI"`
	Keywords []string               `json:"keywords"`
	Tokens   []*tokenTypes.Token    `json:"tokens"`
}

func (t *TokensList) GetVersion() string {
	return fmt.Sprintf("%d.%d.%d", t.Version.Major, t.Version.Minor, t.Version.Patch)
}

type TokenLists struct {
	walletDb *sql.DB
	settings *settings.Database

	notifyCh          chan struct{}
	tokenListsFetcher *fetcher.TokenListsFetcher

	tokensListsMu sync.RWMutex
	tokensLists   map[string]*TokensList // map[list-id][tokens-list]
}

// NewTokenLists creates a new instance of TokenLists.
func NewTokenLists(appDb *sql.DB, walletDb *sql.DB) (*TokenLists, error) {
	settings, err := settings.MakeNewDB(appDb)
	if err != nil {
		return nil, err
	}
	return &TokenLists{
		walletDb: walletDb,
		settings: settings,

		tokenListsFetcher: fetcher.NewTokenListsFetcher(walletDb),

		tokensLists: make(map[string]*TokensList),

		notifyCh: make(chan struct{}),
	}, nil
}

// Start starts the token lists service.
// It fetches the list of token lists from the remote source and starts the auto-refresh loop.
// If the remote list url is not set (empty string provided), the hardcoded default list will be used.
// The auto-refresh interval is used to fetch the list of token lists from the remote source and update the local cache.
// The auto-refresh check interval is used to check if the auto-refresh should be triggered.
func (t *TokenLists) Start(ctx context.Context, remoteListUrl string, autoRefreshInterval time.Duration,
	autoRefreshCheckInterval time.Duration) {
	err := t.initializeTokensLists()
	if err != nil {
		logutils.ZapLogger().Error("Failed to initialize token lists", zap.Error(err))
	}

	t.tokenListsFetcher.SetURLOfRemoteListOfTokenLists(remoteListUrl)

	go func() {
		defer common.LogOnPanic()
		t.listenForNotifications(ctx)
	}()

	t.startAutoRefreshLoop(ctx, autoRefreshInterval, autoRefreshCheckInterval)
}

func (t *TokenLists) Stop() {
}

func (t *TokenLists) initializeTokensLists() error {
	allTokens, err := t.tokenListsFetcher.GetAllTokenLists()
	if err != nil {
		logutils.ZapLogger().Error("Failed to get all token lists", zap.Error(err))
		return err
	}

	if len(allTokens) == 0 {
		return t.buildInitialTokensListsMap()
	}

	return t.rebuildTokensListsMap()
}

func (t *TokenLists) listenForNotifications(ctx context.Context) {
	for {
		select {
		case <-t.notifyCh:
			err := t.rebuildTokensListsMap()
			if err != nil {
				logutils.ZapLogger().Error("Failed to rebuild tokens map", zap.Error(err))
				continue
			}
			signal.SendWalletEvent(signal.TokenListsUpdated, nil)
		case <-ctx.Done():
			return
		}
	}
}

func (t *TokenLists) LastTokensUpdate() (time.Time, error) {
	return t.settings.LastTokensUpdate()
}

func (t *TokenLists) GetTokensLists() []*TokensList {
	t.tokensListsMu.RLock()
	defer t.tokensListsMu.RUnlock()

	var lists []*TokensList
	for _, list := range t.tokensLists {
		lists = append(lists, list)
	}
	return lists
}

func (t *TokenLists) GetTokensList(listID string) *TokensList {
	t.tokensListsMu.RLock()
	defer t.tokensListsMu.RUnlock()
	return t.tokensLists[listID] // we should be safe to return a pointer to the list, as it's not going to be modified, adding new list to the map always creates a new instance
}

func (t *TokenLists) GetUniqueTokens() []*tokenTypes.Token {
	t.tokensListsMu.RLock()
	defer t.tokensListsMu.RUnlock()

	tokenIdentifier := make(map[string]struct{}) // map[chainID+address]struct{}
	var tokens []*tokenTypes.Token
	for _, list := range t.tokensLists {
		for _, token := range list.Tokens {
			tokenID := strconv.FormatUint(token.ChainID, 10) + token.Address.String()
			if _, exists := tokenIdentifier[tokenID]; !exists {
				tokenIdentifier[tokenID] = struct{}{}
				tokens = append(tokens, token)
			}
		}

	}
	return tokens
}
