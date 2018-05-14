package personal

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
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
	//
	// 1) account (from)
	// 2) data to sign
	// here, we always ignore (3) because we send a confirmation for the password to UI
	if len(rpcParams) != 2 {
		return nil, ErrSignInvalidNumberOfParameters
	}
	address := rpcParams[0].(string)
	data := rpcParams[1]

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

// Recover is an implementation of `personal_ecRecover` or `web3.eth.personal_ecRecover` API
func (api *PublicAPI) Recover(context context.Context, rpcParams ...interface{}) (interface{}, error) {
	var response interface{}

	err := api.rpcClient.CallContextIgnoringLocalHandlers(
		context, &response, params.PersonalRecoverMethodName, rpcParams...)

	return response, err
}

// Sign is a *MetaMask-compatible* implementation of `personal_sign` or `web3.eth.personal_sign` API
// The main difference between MetaMask's and geth implementations of `personal_sign`
// is the argument ordering.
// The geth version uses (data, address, [password])
// (see https://github.com/ethereum/go-ethereum/wiki/Management-APIs#personal_sign)
// The MetaMask version uses (address, data).
// (see https://medium.com/metamask/the-new-secure-way-to-sign-data-in-your-browser-6af9dd2a1527)
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
	return func(acc *account.SelectedExtKey, password string) (response sign.Response, err error) {
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
