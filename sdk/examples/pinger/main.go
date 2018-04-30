package main

import (
	"fmt"
	"time"

	"github.com/status-im/status-go/sdk"
)

func main() {
	conn := sdk.New("rpc.server.addr:12345")
	if err := conn.Signup("111222333"); err != nil {
		panic("Couldn't create an account")
	}

	ch, err := conn.Join("supu")
	if err != nil {
		panic("Couldn't connect to status")
	}

	for range time.Tick(10 * time.Second) {
		message := fmt.Sprintf("PING : %d", time.Now().Unix())
		ch.Publish(message)
	}
}
