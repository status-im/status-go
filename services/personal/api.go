package personal

import (
	"context"
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

type metadata struct {
	Data    interface{} `json:"data"`
	Address string      `json:"account"`
}

func newMetadata(rpcParams []interface{}) (*metadata, error) {
	// personal_sign can be called with the following parameters
	// 1) data to sign
	// 2) account
	// 3) (optional) password
	// here, we always ignore (3) because we send a confirmation for the password to UI
	if len(rpcParams) < 2 || len(rpcParams) > 3 {
		return nil, ErrSignInvalidNumberOfParameters
	}
	data := rpcParams[0]
	address := rpcParams[1].(string)

	return &metadata{data, address}, nil
}

// PublicAPI represents a set of APIs from the `web3.personal` namespace.
type PublicAPI struct {
	pendingSignRequests *sign.PendingRequests
	rpcClient           *rpc.Client
	rpcTimeout          time.Duration
}

// NewAPI creates an instance of the personal API.
func NewAPI(pendingSignRequests *sign.PendingRequests) *PublicAPI {
	return &PublicAPI{
		pendingSignRequests: pendingSignRequests,
	}
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
func (api *PublicAPI) Sign(context context.Context, rpcParams ...interface{}) (interface{}, error) {
	metadata, err := newMetadata(rpcParams)
	if err != nil {
		return nil, err
	}
	req, err := api.pendingSignRequests.Add(context, params.PersonalSignMethodName, metadata, api.completeFunc(context, *metadata))
	if err != nil {
		return nil, err
	}

	result := api.pendingSignRequests.Wait(req.ID, api.rpcTimeout)
	return result.Response, result.Error
}

func (api *PublicAPI) completeFunc(context context.Context, metadata metadata) sign.CompleteFunc {
	return func(acc *account.SelectedExtKey, password string, signArgs *sign.TxArgs) (response sign.Response, err error) {
		response = sign.EmptyResponse

		err = api.validateAccount(metadata, acc)
		if err != nil {
			return
		}

		err = api.rpcClient.CallContextIgnoringLocalHandlers(
			context,
			&response,
			params.PersonalSignMethodName,
			metadata.Data, metadata.Address, password)

		return
	}
}

// make sure that only account which created the tx can complete it
func (api *PublicAPI) validateAccount(metadata metadata, selectedAccount *account.SelectedExtKey) error {
	if selectedAccount == nil {
		return account.ErrNoAccountSelected
	}

	// case-insensitive string comparison
	if !strings.EqualFold(metadata.Address, selectedAccount.Address.Hex()) {
		return ErrInvalidPersonalSignAccount
	}

	return nil
}
