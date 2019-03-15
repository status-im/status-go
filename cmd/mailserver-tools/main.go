package main

import (
	"flag"
	"fmt"
	"os"

	database "github.com/status-im/status-go/db"
)

var (
	global = flag.NewFlagSet("", flag.ExitOnError)
	dir    = global.String("dir", "", "dir with the mailserver data")

	countCmd  = flag.NewFlagSet("count", flag.ExitOnError)
	topicName = countCmd.String("topic-name", "", "topic name to count envelopes in ('contact-discovery' for private chats)")
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Usage:")
		fmt.Println("  mailserver-tools COMMAND")
		fmt.Println("")
		fmt.Println("Commands:")
		fmt.Println("  count: returns a number of envelopes")
		os.Exit(1)
	}

	if err := global.Parse(os.Args[1:2]); err != nil {
		exitErr(err)
	}

	if *dir == "" {
		global.PrintDefaults()
		exitErr(fmt.Errorf("invalid value for -dir: %s", *dir))
	}

	switch os.Args[2] {
	case "count":
		if err := countCmd.Parse(os.Args[3:]); err != nil {
			exitErr(err)
		}
	default:
		exitErr(fmt.Errorf("invalid command: %s", os.Args[1]))
	}

	topic, err := nameToTopic(*topicName)
	if err != nil {
		exitErr(err)
	}

	db, err := database.Open(*dir, nil)
	if err != nil {
		exitErr(err)
	}

	counter, err := countLast24HoursInTopic(db, topic)
	if err != nil {
		exitErr(err)
	}

	fmt.Printf("For topic '%s' there are %d envelopes\n", *topicName, counter)
}

func exitErr(err error) {
	fmt.Println(err)
	os.Exit(1)
}
