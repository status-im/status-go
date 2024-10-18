package requests

type RouterBuildTransactionsParams struct {
	Uuid               string  `json:"uuid"`
	SlippagePercentage float32 `json:"slippagePercentage"`
}
