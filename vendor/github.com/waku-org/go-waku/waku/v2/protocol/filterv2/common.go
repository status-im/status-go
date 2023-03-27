package filterv2

import "fmt"

const DefaultMaxSubscriptions = 1000
const MaxCriteriaPerSubscription = 1000

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
