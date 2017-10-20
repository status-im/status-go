package node

import (
	"fmt"

	"github.com/status-im/status-go/geth/log"
)

type globalLog struct {
	file  string
	level string
}

type logger interface {
	Init(file, level string)
}

func newLog() *globalLog {
	return &globalLog{level: "ERROR"}
}

func (l globalLog) Init(file, level string) {
	l.level = level
	l.file = file

	log.SetLevel(l.level)

	err := log.SetLogFile(l.file)
	if err != nil {
		fmt.Println("Failed to open log file, using stdout")
	}
}
