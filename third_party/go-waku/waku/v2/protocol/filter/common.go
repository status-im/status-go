package filter

import (
	"fmt"
	"time"
)

const DefaultMaxSubscribers = 20
const MaxCriteriaPerSubscription = 1000
const MaxContentTopicsPerRequest = 100
const MessagePushTimeout = 20 * time.Second
const DefaultIdleSubscriptionTimeout = 5 * time.Minute

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

const errorStringFmt = "%d - %s"

func (e *FilterError) Error() string {
	return fmt.Sprintf(errorStringFmt, e.Code, e.Message)
}

func ExtractCodeFromFilterError(fErr string) int {
	code := 0
	var message string
	_, err := fmt.Sscanf(fErr, errorStringFmt, &code, &message)
	if err != nil {
		return -1
	}
	return code
}
