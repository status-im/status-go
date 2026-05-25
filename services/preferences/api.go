package preferences

import (
	"context"
)

type GetResult struct {
	Value string `json:"value"`
	Found bool   `json:"found"`
}

type CountResult struct {
	Removed int `json:"removed"`
}

type API struct {
	store PreferenceStore
}

func NewAPI(store PreferenceStore) *API {
	return &API{store: store}
}

func (api *API) Get(_ context.Context, category, key string) (GetResult, error) {
	value, found, err := api.store.Get(category, key)
	if err != nil {
		return GetResult{}, err
	}
	return GetResult{Value: value, Found: found}, nil
}

func (api *API) GetAll(_ context.Context, category string) (map[string]string, error) {
	return api.store.GetAll(category)
}

func (api *API) ListKeys(_ context.Context, category string) ([]string, error) {
	return api.store.ListKeys(category)
}

func (api *API) ListCategories(_ context.Context) ([]string, error) {
	return api.store.ListCategories()
}

func (api *API) Set(_ context.Context, category, key, value string) error {
	return api.store.Set(category, key, value)
}

func (api *API) SetMany(_ context.Context, category string, kvs map[string]string) error {
	return api.store.SetMany(category, kvs)
}

func (api *API) Delete(_ context.Context, category, key string) error {
	return api.store.Delete(category, key)
}

func (api *API) DeleteCategory(_ context.Context, category string) (CountResult, error) {
	removed, err := api.store.DeleteCategory(category)
	if err != nil {
		return CountResult{}, err
	}
	return CountResult{Removed: removed}, nil
}

func (api *API) PurgeUnknown(_ context.Context, category string, validKeys []string) (CountResult, error) {
	removed, err := api.store.PurgeUnknown(category, validKeys)
	if err != nil {
		return CountResult{}, err
	}
	return CountResult{Removed: removed}, nil
}

func (api *API) LoadAndPurgeUnknown(_ context.Context, category string, validKeys []string) (map[string]string, error) {
	return api.store.LoadAndPurgeUnknown(category, validKeys)
}
