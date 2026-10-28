package sds

type MessageID string

type HistoryEntry struct {
	MessageID     MessageID `json:"messageId"`
	RetrievalHint []byte    `json:"retrievalHint"`
}

type UnwrappedMessage struct {
	Message     *[]byte         `json:"message"`
	MissingDeps *[]HistoryEntry `json:"missingDeps"`
	ChannelId   *string         `json:"channelId"`
}

type HistoryUpdate struct {
	Append []HistoryMessage `json:"append"`
	Evict  []string         `json:"evict"`
}

type HistoryMessage struct {
	MessageID        string `json:"messageId"`
	LamportTimestamp int64  `json:"lamportTimestamp"`
	Data             []byte `json:"data"`
}

type ChannelData struct {
	Meta           []byte   `json:"meta"`
	MessageHistory [][]byte `json:"messageHistory"`
}

type PersistenceCallbacks struct {
	SaveChannelMeta func(channelID string, meta []byte) error
	UpdateHistory   func(channelID string, update HistoryUpdate) error
	LoadChannel     func(channelID string) (ChannelData, error)
	DropChannel     func(channelID string) error
}
