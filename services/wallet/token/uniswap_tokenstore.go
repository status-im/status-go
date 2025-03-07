package token

import tokenTypes "github.com/status-im/status-go/services/wallet/token/types"

type uniswapStore struct {
}

func NewUniswapStore() *uniswapStore {
	return &uniswapStore{}
}

func (s *uniswapStore) GetTokens() []*tokenTypes.Token {
	return uniswapTokens
}

func (s *uniswapStore) GetName() string {
	return TokensSources[UniswapTokenListID].Name

}

func (s *uniswapStore) GetVersion() string {
	return uniswapVersion
}

func (s *uniswapStore) GetUpdatedAt() int64 {
	return uniswapTimestamp
}

func (s *uniswapStore) GetSource() string {
	return TokensSources[UniswapTokenListID].SourceURL
}
