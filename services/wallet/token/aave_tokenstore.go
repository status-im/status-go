package token

type aaveStore struct {
}

func NewAaveStore() *aaveStore {
	return &aaveStore{}
}

func (s *aaveStore) GetTokens() []*Token {
	return aaveTokens
}

func (s *aaveStore) GetName() string {
	return TokensSources[AaveTokenListID].Name

}

func (s *aaveStore) GetVersion() string {
	return aaveVersion
}

func (s *aaveStore) GetUpdatedAt() int64 {
	return aaveTimestamp
}

func (s *aaveStore) GetSource() string {
	return TokensSources[AaveTokenListID].SourceURL
}
