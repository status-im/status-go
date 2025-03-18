package fetcher

import (
	"context"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

func (t *TokenListsFetcher) fetchTokenList(ctx context.Context, tokenList TokenList, etag string, ch chan<- FetchedTokenList) error {
	body, newEtag, err := t.httpClient.DoGetRequestWithEtag(ctx, tokenList.SourceURL, nil, etag)
	if err != nil {
		return err
	}

	if newEtag == etag {
		return nil
	}

	if tokenList.Schema != "" {
		err = validateJsonAgainstSchema(string(body), gojsonschema.NewReferenceLoader(tokenList.Schema))
		if err != nil {
			return err
		}
	}

	ch <- FetchedTokenList{
		TokenList: TokenList{
			ID:        tokenList.ID,
			SourceURL: tokenList.SourceURL,
			Schema:    tokenList.Schema,
		},
		Etag:     newEtag,
		Fetched:  time.Now(),
		JsonData: string(body),
	}

	return nil
}
