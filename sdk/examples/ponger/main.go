package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/status-im/status-go/sdk"
)

func main() {
	cfg := sdk.DefaultConfig()
	cfg.NodeConfig.DataDir = "/tmp/testnet_rpc"

	conn := sdk.New()
	if err := conn.Start(cfg); err != nil {
		panic("Couldn't connect to status")
	}

	if err := conn.SignupOrLogin("supu", "password"); err != nil {
		panic(err)
	}

	ch, err := conn.Join("supu")
	if err != nil {
		panic("Couldn't connect to status")
	}

	ch.Subscribe(func(m *sdk.Msg) {
		log.Println("Message from ", m.From, " with body: ", m.Text)

		if strings.Contains(m.Text, "PING :") {
			time.Sleep(5 * time.Second)
			message := fmt.Sprintf("PONG : %d", time.Now().Unix())
			ch.Publish(message)
		}

	})

	runtime.Goexit()
}
