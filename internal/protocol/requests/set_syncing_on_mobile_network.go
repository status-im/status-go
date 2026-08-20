package requests

type SetSyncingOnMobileNetwork struct {
	Enabled bool `json:"enabled"`
}

func (r *SetSyncingOnMobileNetwork) Validate() error {
	return nil
}
