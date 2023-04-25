package scaffold

type DataInfo struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type EntriesInfo struct {
	TotalCount int `json:"total_count"`
	TotalPages int `json:"total_pages"`
	Page       int `json:"page"`
}

type Exchange struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Logo   string `json:"logo"`
}

type ExchangesEntries struct {
	EntriesInfo
	List []*Exchange `json:"list"`
}

type ExchangesData struct {
	DataInfo
	Entries *ExchangesEntries `json:"data"`
}

type ExchangeAddress struct {
	Address string `json:"address"`
	Flag    string `json:"flag"`
}

type ExchangesAddressesEntries struct {
	EntriesInfo
	List []*ExchangeAddress `json:"list"`
}

type ExchangeAddressesData struct {
	DataInfo
	Entries *ExchangesAddressesEntries `json:"data"`
}
