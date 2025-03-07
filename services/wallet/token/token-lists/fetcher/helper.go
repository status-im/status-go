package fetcher

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/xeipuuv/gojsonschema"
)

func validateJsonAgainstSchema(jsonData string, schemaLoader gojsonschema.JSONLoader) error {
	docLoader := gojsonschema.NewStringLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		return errors.New("token list does not match schema")
	}

	return nil
}

func (t *TokenListsFetcher) fetchContent(ctx context.Context, remoteURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status code: %d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}
