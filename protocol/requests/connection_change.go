package requests

type ConnectionChange struct {
	Type      string `json:"type"`
	Expensive int    `json:"expensive"`
}
