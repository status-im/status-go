package provider_errors

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rpc"
)

// ProviderErrorType defines the type of non-RPC error for JSON serialization.
type ProviderErrorType string

const (
	// Non-RPC Errors
	ProviderErrorTypeNone                    ProviderErrorType = "none"
	ProviderErrorTypeContextCanceled         ProviderErrorType = "context_canceled"
	ProviderErrorTypeContextDeadlineExceeded ProviderErrorType = "context_deadline"
	ProviderErrorTypeConnection              ProviderErrorType = "connection"
	ProviderErrorTypeNotAuthorized           ProviderErrorType = "not_authorized"
	ProviderErrorTypeForbidden               ProviderErrorType = "forbidden"
	ProviderErrorTypeBadRequest              ProviderErrorType = "bad_request"
	ProviderErrorTypeContentTooLarge         ProviderErrorType = "content_too_large"
	ProviderErrorTypeInternalError           ProviderErrorType = "internal"
	ProviderErrorTypeServiceUnavailable      ProviderErrorType = "service_unavailable"
	ProviderErrorTypeRateLimit               ProviderErrorType = "rate_limit"
	ProviderErrorTypeOther                   ProviderErrorType = "other"
)

// IsConnectionError checks if the error is related to network issues.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error (timeout or other network errors)
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Check for network operation errors (e.g., connection refused)
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Check for TLS errors
	var tlsRecordErr *tls.RecordHeaderError
	if errors.As(err, &tlsRecordErr) {
		return true
	}

	// FIXME: Check for TLS ECH Rejection Error (tls.ECHRejectionError is added in go 1.23)

	// Check for TLS Certificate Verification Error
	var certVerifyErr *tls.CertificateVerificationError
	if errors.As(err, &certVerifyErr) {
		return true
	}

	// Check for TLS Alert Error
	var alertErr tls.AlertError
	if errors.As(err, &alertErr) {
		return true
	}

	// Check for specific HTTP server closed error
	if errors.Is(err, http.ErrServerClosed) {
		return true
	}

	// Common connection refused or timeout error messages
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "network is unreachable") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "tls handshake timeout") {
		return true
	}

	return false
}

func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	if ok, statusCode := IsHTTPError(err); ok && statusCode == 429 {
		return true
	}

	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "backoff_seconds") ||
		strings.Contains(errMsg, "has exceeded its throughput limit") ||
		strings.Contains(errMsg, "request rate exceeded") {
		return true
	}
	return false
}

// Don't mark connection as failed if we get one of these errors
var propagateErrors = []error{
	vm.ErrOutOfGas,
	vm.ErrCodeStoreOutOfGas,
	vm.ErrDepth,
	vm.ErrInsufficientBalance,
	vm.ErrContractAddressCollision,
	vm.ErrExecutionReverted,
	vm.ErrMaxCodeSizeExceeded,
	vm.ErrInvalidJump,
	vm.ErrWriteProtection,
	vm.ErrReturnDataOutOfBounds,
	vm.ErrGasUintOverflow,
	vm.ErrInvalidCode,
	vm.ErrNonceUintOverflow,

	// Used by balance history to check state
	bind.ErrNoCode,
}

func IsHTTPError(err error) (bool, int) {
	var httpErrPtr *rpc.HTTPError
	if errors.As(err, &httpErrPtr) {
		return true, httpErrPtr.StatusCode
	}

	var httpErr rpc.HTTPError
	if errors.As(err, &httpErr) {
		return true, httpErr.StatusCode
	}

	return false, 0
}

func IsNotAuthorizedError(err error) bool {
	if ok, statusCode := IsHTTPError(err); ok {
		return statusCode == 401
	}
	return false
}

func IsForbiddenError(err error) bool {
	if ok, statusCode := IsHTTPError(err); ok {
		return statusCode == 403
	}
	return false
}

func IsBadRequestError(err error) bool {
	if ok, statusCode := IsHTTPError(err); ok {
		return statusCode == 400
	}
	return false
}

func IsContentTooLargeError(err error) bool {
	if ok, statusCode := IsHTTPError(err); ok {
		return statusCode == 413
	}
	return false
}

func IsInternalServerError(err error) bool {
	if ok, statusCode := IsHTTPError(err); ok {
		return statusCode == 500
	}
	return false
}

func IsServiceUnavailableError(err error) bool {
	if ok, statusCode := IsHTTPError(err); ok {
		return statusCode == 503
	}
	return false
}

// determineProviderErrorType determines the ProviderErrorType based on the error.
func determineProviderErrorType(err error) ProviderErrorType {
	if err == nil {
		return ProviderErrorTypeNone
	}
	if errors.Is(err, context.Canceled) {
		return ProviderErrorTypeContextCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ProviderErrorTypeContextDeadlineExceeded
	}
	if IsConnectionError(err) {
		return ProviderErrorTypeConnection
	}
	if IsNotAuthorizedError(err) {
		return ProviderErrorTypeNotAuthorized
	}
	if IsForbiddenError(err) {
		return ProviderErrorTypeForbidden
	}
	if IsBadRequestError(err) {
		return ProviderErrorTypeBadRequest
	}
	if IsContentTooLargeError(err) {
		return ProviderErrorTypeContentTooLarge
	}
	if IsInternalServerError(err) {
		return ProviderErrorTypeInternalError
	}
	if IsServiceUnavailableError(err) {
		return ProviderErrorTypeServiceUnavailable
	}
	if IsRateLimitError(err) {
		return ProviderErrorTypeRateLimit
	}
	// Add additional non-RPC checks as necessary
	return ProviderErrorTypeOther
}

// IsNonCriticalProviderError determines if the non-RPC error is not critical.
func IsNonCriticalProviderError(err error) bool {
	errorType := determineProviderErrorType(err)

	switch errorType {
	case ProviderErrorTypeNone, ProviderErrorTypeContextCanceled, ProviderErrorTypeContentTooLarge, ProviderErrorTypeRateLimit:
		return true
	default:
		return false
	}
}
