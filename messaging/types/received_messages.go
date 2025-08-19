package types

type ReceivedMessages struct {
	Filter     ChatFilter
	SHHMessage *ReceivedMessage
	Messages   []*Message
}
