package provider_errors

import (
	"errors"
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

// Not found should not be cancelling the requests, as that's returned
// when we are hitting a non archival node for example, it should continue the
// chain as the next provider might have archival support.
func IsNotFoundError(err error) bool {
	return strings.Contains(err.Error(), ethereum.NotFound.Error())
}

func IsRPCError(err error) (rpc.Error, bool) {
	var rpcErr rpc.Error
	if errors.As(err, &rpcErr) {
		return rpcErr, true
	}
	return nil, false
}

func IsMethodNotFoundError(err error) bool {
	if rpcErr, ok := IsRPCError(err); ok {
		return rpcErr.ErrorCode() == -32601
	}
	return false
}

func IsVMError(err error) bool {
	if rpcErr, ok := IsRPCError(err); ok {
		return rpcErr.ErrorCode() == -32015 // Код ошибки VM execution error
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
	return RpcErrorTypeRPCOther
}

// IsCriticalRpcError determines if the RPC error is critical.
func IsNonCriticalRpcError(err error) bool {
	errorType := determineRpcErrorType(err)

	switch errorType {
	case RpcErrorTypeNone, RpcErrorTypeMethodNotFound, RpcErrorTypeRPSLimit, RpcErrorTypeVMError:
		return true
	default:
		return false
	}
}
