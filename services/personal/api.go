package personal

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/sign"
)

var (
	// ErrInvalidPersonalSignAccount is returned when the account passed to
	// personal_sign isn't equal to the currently selected account.
	ErrInvalidPersonalSignAccount = errors.New("invalid account as only the selected one can generate a signature")

	// ErrSignInvalidNumberOfParameters is returned when the number of parameters for personal_sign
	// is not valid.
	ErrSignInvalidNumberOfParameters = errors.New("invalid number of parameters for personal_sign (2 or 3 expected)")
)

type Metadata struct {
	Data     interface{} `json:"data"`
	Address  string      `json:"account"`
	Password string      `json:"password"`
}

// UnmarshalSignRPCParams puts the RPC params for `personal_sign` or `web3.personal.sign`
// into Metadata
func UnmarshalSignRPCParams(rpcParamsJSON string) (Metadata, error) {
	var params Metadata
	err := json.Unmarshal([]byte(rpcParamsJSON), &params)
	return params, err
}

// PublicAPI represents a set of APIs from the `web3.personal` namespace.
type PublicAPI struct {
	rpcClient  *rpc.Client
	rpcTimeout time.Duration
}

// NewAPI creates an instance of the personal API.
func NewAPI() *PublicAPI {
	return &PublicAPI{}
}

// SetRPC sets RPC params (client and timeout) for the API calls.
func (api *PublicAPI) SetRPC(rpcClient *rpc.Client, timeout time.Duration) {
	api.rpcClient = rpcClient
	api.rpcTimeout = timeout
}

// Recover is an implementation of `personal_ecRecover` or `web3.personal.ecRecover` API
func (api *PublicAPI) Recover(context context.Context, rpcParams ...interface{}) (interface{}, error) {
	var response interface{}

	err := api.rpcClient.CallContextIgnoringLocalHandlers(
		context, &response, params.PersonalRecoverMethodName, rpcParams...)

	return response, err
}

// Sign is an implementation of `personal_sign` or `web3.personal.sign` API
func (api *PublicAPI) Sign(context context.Context, rpcParams Metadata) sign.Result {
	response := sign.EmptyResponse
	err := api.rpcClient.CallContextIgnoringLocalHandlers(
		context,
		&response,
		params.PersonalSignMethodName,
		rpcParams.Data, rpcParams.Address, rpcParams.Password)

	result := sign.Result{Error: err}
	if err == nil {
		result.Response = response
	}

	return result
}
