package main

import (
	"context"
	"time"
)

func createContextFromTimeout(timeout int) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return context.WithCancel(context.Background())
	}

	return context.WithTimeout(context.Background(), time.Duration(timeout)*time.Minute)
}
