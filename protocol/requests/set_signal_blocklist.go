package requests

type SetSignalBlocklist struct {
	Blocklist []string `json:"blocklist"`
}
