package requests

type ToggleCentralizedMetrics struct {
	Enabled bool `json:"enabled"`
}

func (a *ToggleCentralizedMetrics) Validate() error {
	return nil
}
