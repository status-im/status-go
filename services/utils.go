package services

import "github.com/ethereum/go-ethereum/rpc"

// APIByNamespace retrieve an api by its namespace or returns nil.
func APIByNamespace(apis []rpc.API, namespace string) interface{} {
	for _, api := range apis {
		if api.Namespace == namespace {
			return api.Service
		}
	}
	return nil
}
