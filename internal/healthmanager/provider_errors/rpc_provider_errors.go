package provider_errors

import (
	"errors"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rpc"
)

type RpcProviderErrorType string

const (
	// RPC Errors
	RpcErrorTypeNone           RpcProviderErrorType = "none"
	RpcErrorTypeMethodNotFound RpcProviderErrorType = "rpc_method_not_found"
	RpcErrorTypeRPSLimit       RpcProviderErrorType = "rpc_rps_limit"
	RpcErrorTypeVMError        RpcProviderErrorType = "rpc_vm_error"
	RpcErrorTypeRPCOther       RpcProviderErrorType = "rpc_other"
)

// safeRPCError safely extracts an RPC error and its error code.
// Returns nil, 0, false if:
// - err is nil
// - err is not an RPC error
// - err is a nil RPC error pointer
func safeRPCError(err error) (rpc.Error, int, bool) {
	var rpcErr rpc.Error
	if !errors.As(err, &rpcErr) {
		return nil, 0, false
	}

	// If it's a nil pointer but wrapped, we still want to handle it as an RPC error
	if v := reflect.ValueOf(rpcErr); v.Kind() == reflect.Ptr && v.IsNil() {
		return nil, 0, true
	}

	return rpcErr, rpcErr.ErrorCode(), true
}

// Not found should not be cancelling the requests, as that's returned
// when we are hitting a non archival node for example, it should continue the
// chain as the next provider might have archival support.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), ethereum.NotFound.Error())
}

func IsRPCError(err error) (rpc.Error, bool) {
	rpcErr, _, ok := safeRPCError(err)
	return rpcErr, ok
}

func IsMethodNotFoundError(err error) bool {
	_, code, ok := safeRPCError(err)
	return ok && code == -32601
}

func IsVMError(err error) bool {
	_, code, ok := safeRPCError(err)
	if ok && code == -32015 { // VM execution error code
		return true
	}
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), core.ErrInsufficientFunds.Error()) {
		return true
	}
	for _, vmError := range propagateErrors {
		if strings.Contains(err.Error(), vmError.Error()) {
			return true
		}
	}
	return false
}

// determineRpcErrorType determines the RpcProviderErrorType based on the error.
func determineRpcErrorType(err error) RpcProviderErrorType {
	if err == nil {
		return RpcErrorTypeNone
	}

	if IsMethodNotFoundError(err) || IsNotFoundError(err) {
		return RpcErrorTypeMethodNotFound
	}
	if IsVMError(err) {
		return RpcErrorTypeVMError
	}
	if _, ok := IsRPCError(err); ok {
		return RpcErrorTypeRPCOther
	}
	return RpcErrorTypeRPCOther
}

// IsNonCriticalRpcError determines if the RPC error is critical.
func IsNonCriticalRpcError(err error) bool {
	errorType := determineRpcErrorType(err)

	switch errorType {
	case RpcErrorTypeNone, RpcErrorTypeMethodNotFound, RpcErrorTypeRPSLimit, RpcErrorTypeVMError:
		return true
	default:
		return false
	}
}
