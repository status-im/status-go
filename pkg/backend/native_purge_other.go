//go:build !android || !cgo

package backend

func purgeNativeHeap() {}
