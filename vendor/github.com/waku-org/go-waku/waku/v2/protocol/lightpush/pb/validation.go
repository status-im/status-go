package pb

import "errors"

var (
	errMissingRequestID   = errors.New("missing RequestId field")
	errMissingQuery       = errors.New("missing Query field")
	errMissingMessage     = errors.New("missing Message field")
	errMissingPubsubTopic = errors.New("missing PubsubTopic field")
	errRequestIDMismatch  = errors.New("requestID in response does not match request")
	errMissingResponse    = errors.New("missing Response field")
)

func (x *PushRPC) ValidateRequest() error {
	if x.RequestId == "" {
		return errMissingRequestID
	}

	if x.Query == nil {
		return errMissingQuery
	}

	if x.Query.PubsubTopic == "" {
		return errMissingPubsubTopic
	}

	if x.Query.Message == nil {
		return errMissingMessage
	}

	return x.Query.Message.Validate()
}

func (x *PushRPC) ValidateResponse(requestID string) error {
	if x.RequestId == "" {
		return errMissingRequestID
	}

	if x.RequestId != requestID {
		return errRequestIDMismatch
	}

	if x.Response == nil {
		return errMissingResponse
	}

	return nil
}
