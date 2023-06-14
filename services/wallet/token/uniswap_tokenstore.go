package token

type uniswapStore struct {
}

func newUniswapStore() *uniswapStore {
	return &uniswapStore{}
}

func (ts *uniswapStore) GetTokens() []*Token {
	return uniswapTokens
}
