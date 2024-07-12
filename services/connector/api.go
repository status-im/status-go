package connector

import (
	"encoding/json"
	"fmt"
)

type API struct {
	s *Service
	r *CommandRegistry
}

func NewAPI(s *Service) *API {
	r := NewCommandRegistry()

	return &API{
		s: s,
		r: r,
	}
}

type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

func (api *API) CallRPC(inputJSON string) (string, error) {
	var request RPCRequest

	err := json.Unmarshal([]byte(inputJSON), &request)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	if command, exists := api.r.GetCommand(request.Method); exists {
		return command.Execute(inputJSON)
	}

	return api.s.rpcClient.CallRaw(inputJSON), nil
}
