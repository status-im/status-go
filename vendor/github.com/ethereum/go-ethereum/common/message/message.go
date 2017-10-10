package message

// consts of all message delivery status.
const (
	PendingStatus = iota
	SentStatus
	QueuedStatus
	FutureStatus
	ExpiredStatus
	ResendStatus
	OversizedMessageStatus
	FailedSendingStatus
)
