package main

type AccountInfo struct {
	Address string `json:"address"`
	PubKey  string `json:"pubkey"`
	Error   string `json:"error"`
}

type JSONError struct {
	Error string `json:"error"`
}

type AddPeerResult struct {
    Success bool `json:"success"`
    Error   string `json:"error"`
}
