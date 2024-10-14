package requests

type SwitchFleet struct {
	Fleet      string `json:"fleet"`
	ConfigJSON string `json:"configJSON"`
}
