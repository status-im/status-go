package requests

type ConnectionChange struct {
	Type      string `json:"type"`
	Expensive bool   `json:"expensive"`
}
