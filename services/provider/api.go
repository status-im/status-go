package provider

import "context"

func NewAPI() *API {
	return &API{}
}

// API is class with methods available over RPC.
type API struct {
}

func (a *API) HelloWorld(ctx context.Context) string {
	return "Hello World"
}
