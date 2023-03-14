package token

import (
	"io/ioutil"
	"net/http"
	"time"
)

type uniswapStore struct {
	client        *http.Client
	tokensFetched bool
}

const uniswapTokensURL = "https://gateway.ipfs.io/ipns/tokens.uniswap.org" // nolint:gosec
const tokenListSchemaURL = "https://uniswap.org/tokenlist.schema.json"     // nolint:gosec

func newUniswapStore() *uniswapStore {
	return &uniswapStore{client: &http.Client{Timeout: time.Minute}, tokensFetched: false}
}

func (ts *uniswapStore) doQuery(url string) (*http.Response, error) {
	return ts.client.Get(url)
}

func (ts *uniswapStore) areTokensFetched() bool {
	return ts.tokensFetched
}

func (ts *uniswapStore) GetTokens() ([]*Token, error) {
	resp, err := ts.doQuery(uniswapTokensURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// In an unlikely case when token list is fetched fine,
	// but fails to validate against the schema, we don't want
	// to refetch the tokens on every GetTokens call as it will
	// still fail but will be wasting CPU cycles until restart,
	// so lets keep tokensFetched before validate() call
	ts.tokensFetched = true

	_, err = validateDocument(string(body), tokenListSchemaURL)
	if err != nil {
		return nil, err
	}

	return bytesToTokens(body)
}
