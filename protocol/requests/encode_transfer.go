package requests

type EncodeTransfer struct {
	To    string `json:"to"`
	Value string `json:"value"`
}
