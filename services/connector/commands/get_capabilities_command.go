package commands

import (
	"context"
)

// GetCapabilitiesCommand implements wallet_getCapabilities (EIP-5792) as an empty capabilities map.
type GetCapabilitiesCommand struct{}

func NewGetCapabilitiesCommand() *GetCapabilitiesCommand {
	return &GetCapabilitiesCommand{}
}

func (*GetCapabilitiesCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
	if err := request.Validate(); err != nil {
		return "", err
	}
	return map[string]any{}, nil
}
