package root

import (
	"context"

	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/pkg/services/root/requests"
)

type API struct {
	service *Service
}

func (a *API) ListAccounts() ([]multiaccounts.Account, error) {
	accs, err := a.service.multiaccountsDB.GetAccounts()

	if a.service.mediaProvider != nil {
		for i, acc := range accs {
			for j, images := range acc.Images {
				url := a.service.mediaProvider.MakeAccountImageURL(acc.KeyUID, images.Name, images.Clock)
				accs[i].Images[j].LocalURL = url
			}
		}
	}

	return accs, err
}

func (a *API) CreateAccount(ctx context.Context, request *requests.CreateAccount, keycardData *requests.KeycardData) (*multiaccounts.Account, error) {
	return a.service.CreateAccount(ctx, request, keycardData)
}

//func (a *API) DeleteAccount() error {
//
//}
//
//func (a *API) Login() error {
//
//}
//
//func (a *API) Logout() error {
//
//}
