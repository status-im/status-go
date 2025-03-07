package fetcher

import (
	"context"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

func (t *TokenListsFetcher) fetchTokenList(ctx context.Context, tokenList TokenList, ch chan<- FetchedTokenList) error {
	body, err := t.fetchContent(ctx, tokenList.SourceURL)
	if err != nil {
		return err
	}

	if tokenList.Schema != "" {
		err = validateJsonAgainstSchema(string(body), gojsonschema.NewReferenceLoader(tokenList.Schema))
		if err != nil {
			return err
		}
	}

	ch <- FetchedTokenList{
		TokenList: tokenList,
		Fetched:   time.Now(),
		JsonData:  string(body),
	}

	return nil
}
