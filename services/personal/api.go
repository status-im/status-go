package personal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/status-im/status-go/account"
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

// SignParams required to sign messages
type SignParams struct {
	Data     interface{} `json:"data"`
	Address  string      `json:"account"`
	Password string      `json:"password"`
}

// RecoverParams are for calling `personal_ecRecover`
type RecoverParams struct {
	Message   string
	Signature string
}

// UnmarshalSignRPCParams puts the RPC params for `personal_sign` or `web3.personal.sign`
// into SignParams
func UnmarshalSignRPCParams(rpcParamsJSON string) (SignParams, error) {
	var params SignParams
	err := json.Unmarshal([]byte(rpcParamsJSON), &params)
	return params, err
}

// UnmarshalRecoverRPCParams
func UnmarshalRecoverRPCParams(rpcParamsJSON string) (RecoverParams, error) {
	var params RecoverParams
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
	return &PublicAPI{
		rpcTimeout: 300 * time.Second,
	}
}

// SetRPC sets RPC params (client and timeout) for the API calls.
func (api *PublicAPI) SetRPC(rpcClient *rpc.Client, timeout time.Duration) {
	api.rpcClient = rpcClient
	api.rpcTimeout = timeout
}

// Recover is an implementation of `personal_ecRecover` or `web3.personal.ecRecover` API
func (api *PublicAPI) Recover(rpcParams RecoverParams) sign.Result {
	var response sign.Response
	ctx, cancel := context.WithTimeout(context.Background(), api.rpcTimeout)
	defer cancel()
	err := api.rpcClient.CallContextIgnoringLocalHandlers(
		ctx,
		&response,
		params.PersonalRecoverMethodName,
		rpcParams.Message, rpcParams.Signature)

	result := sign.Result{Error: err}
	if err == nil {
		result.Response = response
	}
	return result
}

// Sign is an implementation of `personal_sign` or `web3.personal.sign` API
func (api *PublicAPI) Sign(rpcParams SignParams, verifiedAccount *account.SelectedExtKey) sign.Result {
	if !strings.EqualFold(rpcParams.Address, verifiedAccount.Address.Hex()) {
		return sign.NewErrResult(ErrInvalidPersonalSignAccount)
	}
	response := sign.EmptyResponse
	ctx, cancel := context.WithTimeout(context.Background(), api.rpcTimeout)
	defer cancel()
	err := api.rpcClient.CallContextIgnoringLocalHandlers(
		ctx,
		&response,
		params.PersonalSignMethodName,
		rpcParams.Data, rpcParams.Address, rpcParams.Password)

	result := sign.Result{Error: err}
	if err == nil {
		result.Response = response
	}

	return result
}
