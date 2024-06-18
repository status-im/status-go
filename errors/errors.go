package errors

import (
	"encoding/json"
)

// ErrorCode represents a specific error code.
type ErrorCode string

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

// CreateErrorResponseFromError creates an ErrorResponse from a generic error.
func CreateErrorResponseFromError(err error) error {
	if err == nil {
		return nil
	}
	if errResp, ok := err.(*ErrorResponse); ok {
		return errResp
	}
	return &ErrorResponse{
		Code:    "0",
		Details: err.Error(),
	}
}
