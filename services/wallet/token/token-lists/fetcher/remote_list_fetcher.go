package fetcher

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"

	"github.com/xeipuuv/gojsonschema"
)

func (t *TokenListsFetcher) fetchRemoteListOfTokenLists(ctx context.Context) ([]TokenList, error) {
	body, err := t.fetchContent(ctx, t.listOfTokenListsURL)
	if err != nil {
		return nil, err
	}

	err = validateJsonAgainstSchema(string(body), gojsonschema.NewStringLoader(listOfTokenListsSchema))
	if err != nil {
		return nil, err
	}

	var tokenLists []TokenList
	if err = json.Unmarshal(body, &tokenLists); err != nil {
		return nil, err
	}

	return tokenLists, nil
}

func (t *TokenListsFetcher) fetchListOfTokenLists(ctx context.Context) (tokenLists []TokenList, err error) {
	if t.listOfTokenListsURL != "" {
		tokenLists, err = t.fetchRemoteListOfTokenLists(ctx)
		if err == nil {
			// If we successfully fetched the remote list, return it. Otherwise, use the hardcoded list.
			return
		}
		logutils.ZapLogger().Error("Failed to fetch remote list of token lists", zap.Error(err))
	}

	err = json.Unmarshal([]byte(defaultListOfTokenLists), &tokenLists)
	return
}
