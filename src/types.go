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

type AddWhisperFilterResult struct {
    Id int `json:"id"`
    Error string `json:"error"`
}

type WhisperMessageEvent struct {
    Payload string `json:"payload"`
    To      string `json:"to"`
    From    string `json:"from"`
    Sent    int64  `json:"sent"`
    TTL     int64  `json:"ttl"`
    Hash    string `json:"hash"`
}

type GethEvent struct {
    Type  string `json:"type"`
    Event interface{} `json:"event"`
}