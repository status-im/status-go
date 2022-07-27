package discord

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

const (
  MessageTypeDefault string = "Default"
  MessageTypeReply string = "Reply"
)

type Channel struct {
	ID           string `json:"id"`
	CategoryName string `json:"category"`
	CategoryID   string `json:"categoryId"`
	Name         string `json:"name"`
	Description  string `json:"topic"`
	FilePath     string `json:"filePath"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ExportedData struct {
	Channel  Channel                    `json:"channel"`
	Messages []*protobuf.DiscordMessage `json:"messages"`
}
