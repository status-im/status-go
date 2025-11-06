package common

import (
	"time"

	"github.com/jinzhu/copier"
	"go.uber.org/zap"

	"github.com/status-im/status-go/messaging/layers/segmentation"
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/messaging/waku/types"
)

// reducedMaxMessageSize returns the max message size reduced to 3/4 to leave room for segment metadata
func (s *MessageSender) reducedMaxMessageSize() uint32 {
	return s.transport.MaxMessageSize() * 3 / 4
}

func (s *MessageSender) segmentMessage(newMessage *wakutypes.NewMessage) ([]*wakutypes.NewMessage, error) {
	return s.segmentMessageWithSize(newMessage, int(s.reducedMaxMessageSize()))
}

func (s *MessageSender) segmentMessageWithSize(newMessage *wakutypes.NewMessage, segmentSize int) ([]*wakutypes.NewMessage, error) {
	segments, err := s.segmenter.Segment(newMessage.Payload, segmentSize)
	if err != nil {
		return nil, err
	}

	replicateMessage := func(payload []byte) (*wakutypes.NewMessage, error) {
		copy := &wakutypes.NewMessage{}
		err := copier.Copy(copy, newMessage)
		if err != nil {
			return nil, err
		}

		copy.Payload = payload
		return copy, nil
	}

	newMessages := make([]*wakutypes.NewMessage, 0, len(segments))
	for _, segment := range segments {
		segmentMessage, err := replicateMessage(segment)
		if err != nil {
			return nil, err
		}
		newMessages = append(newMessages, segmentMessage)
	}

	s.logger.Debug("message segmented", zap.Int("segments", len(newMessages)))

	return newMessages, err
}

// handleSegmentationLayer is capable of reconstructing the message from both complete and partial sets of data segments.
func (s *MessageSender) handleSegmentationLayer(message *types.Message) (segmented, completed bool, err error) {
	var reconstructedPayload []byte
	reconstructedPayload, err = s.segmenter.Reconstruct(message.TransportLayer.Payload, message.TransportLayer.SigPubKey)

	switch err {
	case nil:
		message.TransportLayer.Payload = reconstructedPayload
		segmented = true
		completed = true
	case segmentation.ErrIncomplete:
		segmented = true
		completed = false
		err = nil
	case segmentation.ErrInvalidPayload:
		segmented = false
		completed = false
		err = nil
	}

	return
}

func (s *MessageSender) CleanupSegments() error {
	monthAgo := time.Now().AddDate(0, -1, 0)
	return s.segmenter.CleanupStaleSegments(monthAgo)
}
