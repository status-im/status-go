package utils

import "github.com/ethereum/go-ethereum/log"

func LogOnPanic() {
	log.Info("<<< panic")
}
