package message

// Status defines a int type to indicate different status value of a
// message state.
type Status int

// consts of all message delivery status.
const (
	PendingStatus Status = iota
	QueuedStatus
	CachedStatus
	SentStatus
	ExpiredStatus
	ResendStatus
	FutureStatus
	RejectedStatus
	DeliveredStatus
	LowPowStatus
	InvalidAESStatus
	OversizedMessageStatus
	OversizedVersionStatus
)

// String returns the representation of giving state.
func (s Status) String() string {
	switch s {
	case PendingStatus:
		return "Pending"
	case QueuedStatus:
		return "Queued"
	case CachedStatus:
		return "Cached"
	case SentStatus:
		return "Sent"
	case ExpiredStatus:
		return "ExpiredTTL"
	case ResendStatus:
		return "Resend"
	case FutureStatus:
		return "FutureDelivery"
	case RejectedStatus:
		return "Rejected"
	case DeliveredStatus:
		return "Delivered"
	case LowPowStatus:
		return "LowPOWValue"
	case InvalidAESStatus:
		return "Invalid AES-GCM-Nonce"
	case OversizedMessageStatus:
		return "OversizedMessage"
	case OversizedVersionStatus:
		return "HigherWhisperVersion"
	}

	return "unknown"
}
