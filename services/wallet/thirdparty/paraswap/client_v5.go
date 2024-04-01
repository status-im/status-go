package paraswap

type ClientV5 struct {
	httpClient *HTTPClient
	chainID    uint64
}

func NewClientV5(chainID uint64) *ClientV5 {
	return &ClientV5{
		httpClient: NewHTTPClient(),
		chainID:    chainID,
	}
}

func (c *ClientV5) SetChainID(chainID uint64) {
	c.chainID = chainID
}
