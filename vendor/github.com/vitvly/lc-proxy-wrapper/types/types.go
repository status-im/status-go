package types

const (
	OptimisticHeader int = iota
	FinalizedHeader
	Stopped
	Error
)

type ProxyEvent struct {
	EventType int
	Msg       string
}
