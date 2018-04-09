package main

import (
	"fmt"
	"time"

	"github.com/status-im/status-go/sdk"
)

func main() {
	conn, err := sdk.Connect("supu", "password")
	if err != nil {
		panic("Couldn't connect to status")
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
