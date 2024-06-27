package commands

type RPCRequest struct {
	JSONRPC     string        `json:"jsonrpc"`
	ID          int           `json:"id"`
	Method      string        `json:"method"`
	Params      []interface{} `json:"params"`
	Origin      string        `json:"origin"`
	DAppName    string        `json:"dAppName"`
	DAppIconUrl string        `json:"dAppIconUrl"`
}

type RPCCommand interface {
	Execute(request RPCRequest) (string, error)
}

type RPCClientInterface interface {
	CallRaw(body string) string
}
