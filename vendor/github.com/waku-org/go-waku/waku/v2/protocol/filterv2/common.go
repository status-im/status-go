package filterv2

import (
	"fmt"
	"time"
)

const DefaultMaxSubscriptions = 1000
const MaxCriteriaPerSubscription = 1000
const MaxContentTopicsPerRequest = 30
const MessagePushTimeout = 20 * time.Second

type FilterError struct {
	Code    int
	Message string
}

func NewFilterError(code int, message string) FilterError {
	return FilterError{
		Code:    code,
		Message: message,
	}
}

func (e *FilterError) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Message)
}
