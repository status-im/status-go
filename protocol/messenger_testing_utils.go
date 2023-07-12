package protocol

import (
	"errors"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/tt"
)

// WaitOnMessengerResponse Wait until the condition is true or the timeout is reached.
func WaitOnMessengerResponse(m *Messenger, condition func(*MessengerResponse) bool, errorMessage string) (*MessengerResponse, error) {
	response := &MessengerResponse{}
	err := tt.RetryWithBackOff(func() error {
		var err error
		r, err := m.RetrieveAll()
		if err := response.Merge(r); err != nil {
			panic(err)
		}

		if err == nil && !condition(response) {
			err = errors.New(errorMessage)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func FindFirstByContentType(messages []*common.Message, contentType protobuf.ChatMessage_ContentType) *common.Message {
	for _, message := range messages {
		if message.ContentType == contentType {
			return message
		}
	}
	return nil
}
