package backend

import (
	"context"

	"github.com/status-im/status-go/multiaccounts"
	requests2 "github.com/status-im/status-go/pkg/backend/requests"
)

type API struct {
	backend *StatusBackend
}

func (a *API) ListAccounts(ctx context.Context) ([]multiaccounts.Account, error) {
	return a.backend.ListAccounts(ctx)
}

func (a *API) CreateAccount(ctx context.Context, request *requests2.CreateAccount, keycardData *requests2.KeycardData) (*multiaccounts.Account, error) {
	return a.backend.CreateAccount(ctx, request, keycardData)
}

//func (a *API) DeleteAccount() error {
//
//}
//

//func (a *API) Login(request requests2.Login) error {
//	if err := request.Validate(); err != nil {
//		return err
//	}
//	return a.service.Login(request)
//}

//func (a *API) Logout() error {
//
//}
