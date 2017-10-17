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
	RejectedStatus
	DeliveredStatus
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
	case RejectedStatus:
		return "Rejected"
	case DeliveredStatus:
		return "Delivered"
	}

	return "unknown"
}
