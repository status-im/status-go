package errors

import (
	"encoding/json"
)

// ErrorCode represents a specific error code.
type ErrorCode string

const GenericErrorCode ErrorCode = "0"

// ErrorResponse represents an error response structure.
type ErrorResponse struct {
	Code    ErrorCode `json:"code"`
	Details string    `json:"details,omitempty"`
}

// Error implements the error interface for ErrorResponse.
func (e *ErrorResponse) Error() string {
	errorJSON, _ := json.Marshal(e)
	return string(errorJSON)
}

// IsErrorResponse determines if an error is an ErrorResponse.
func IsErrorResponse(err error) bool {
	_, ok := err.(*ErrorResponse)
	return ok
}

// ErrorCodeFromError returns the ErrorCode from an error.
func ErrorCodeFromError(err error) ErrorCode {
	if err == nil {
		return GenericErrorCode
	}
	if errResp, ok := err.(*ErrorResponse); ok {
		return errResp.Code
	}
	return GenericErrorCode
}

// DetailsFromError returns the details from an error.
func DetailsFromError(err error) string {
	if err == nil {
		return ""
	}
	if errResp, ok := err.(*ErrorResponse); ok {
		return errResp.Details
	}
	return err.Error()
}

// CreateErrorResponseFromError creates an ErrorResponse from a generic error.
func CreateErrorResponseFromError(err error) error {
	if err == nil {
		return nil
	}
	if errResp, ok := err.(*ErrorResponse); ok {
		return errResp
	}
	return &ErrorResponse{
		Code:    GenericErrorCode,
		Details: err.Error(),
	}
}
