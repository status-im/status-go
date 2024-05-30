package pb

import (
	"errors"
)

// MaxContentTopics is the maximum number of allowed contenttopics in a query
const MaxContentTopics = 10

var (
	errMissingRequestID       = errors.New("missing RequestId field")
	errMessageHashOtherFields = errors.New("cannot use MessageHashes with ContentTopics/PubsubTopic")
	errRequestIDMismatch      = errors.New("requestID in response does not match request")
	errMaxContentTopics       = errors.New("exceeds the maximum number of ContentTopics allowed")
	errEmptyContentTopic      = errors.New("one or more content topics specified is empty")
	errMissingPubsubTopic     = errors.New("missing PubsubTopic field")
	errMissingStatusCode      = errors.New("missing StatusCode field")
	errInvalidTimeRange       = errors.New("invalid time range")
	errInvalidMessageHash     = errors.New("invalid message hash")
)

func (x *StoreQueryRequest) Validate() error {
	if x.RequestId == "" {
		return errMissingRequestID
	}

	if len(x.MessageHashes) != 0 {
		if len(x.ContentTopics) != 0 || x.GetPubsubTopic() != "" {
			return errMessageHashOtherFields
		}

		for _, x := range x.MessageHashes {
			if len(x) != 32 {
				return errInvalidMessageHash
			}
		}
	} else {
		if x.GetPubsubTopic() == "" {
			return errMissingPubsubTopic
		}

		if len(x.ContentTopics) > MaxContentTopics {
			return errMaxContentTopics
		} else {
			for _, m := range x.ContentTopics {
				if m == "" {
					return errEmptyContentTopic
				}
			}
		}

		if x.GetTimeStart() > 0 && x.GetTimeEnd() > 0 && x.GetTimeStart() > x.GetTimeEnd() {
			return errInvalidTimeRange
		}
	}
	return nil
}

func (x *StoreQueryResponse) Validate(requestID string) error {
	if x.RequestId != "" && x.RequestId != requestID {
		return errRequestIDMismatch
	}

	if x.StatusCode == nil {
		return errMissingStatusCode
	}

	for _, m := range x.Messages {
		if err := m.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (x *WakuMessageKeyValue) Validate() error {
	if len(x.MessageHash) != 32 {
		return errInvalidMessageHash
	}

	if x.Message != nil {
		if x.GetPubsubTopic() == "" {
			return errMissingPubsubTopic
		}

		return x.Message.Validate()
	}

	return nil
}
