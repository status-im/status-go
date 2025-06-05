package sds

type MessageID string

type UnwrappedMessage struct {
	Message     *[]byte      `json:"message"`
	MissingDeps *[]MessageID `json:"missingDeps"`
}
