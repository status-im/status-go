package testutils

import "reflect"

func StructExistsInSlice[T any](target T, slice []T) bool {
	for _, item := range slice {
		if reflect.DeepEqual(target, item) {
			return true
		}
	}
	return false
}
