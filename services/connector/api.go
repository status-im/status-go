package connector

import (
	"encoding/json"
	"fmt"

	"github.com/status-im/status-go/services/connector/commands"
	persistence "github.com/status-im/status-go/services/connector/database"
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

func (api *API) CallRPC(inputJSON string) (string, error) {
	var request commands.RPCRequest

	err := json.Unmarshal([]byte(inputJSON), &request)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	if command, exists := api.r.GetCommand(request.Method); exists {
		return command.Execute(request)
	}

	return api.s.rpcClient.CallRaw(inputJSON), nil
}

func (api *API) RecallDAppPermission(origin string) error {
	return persistence.DeleteDApp(api.s.db, origin)
}
