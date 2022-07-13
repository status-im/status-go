package discord

import (
	"fmt"

	"github.com/status-im/status-go/protocol/protobuf"
)

type ErrorType uint

const (
	NoError ErrorType = iota
	Warning
	Error
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

type ExtractedData struct {
	Categories             map[string]*Category
	ExportedData           []*ExportedData
	OldestMessageTimestamp int
}

type ImportError struct {
	// This code is used to distinguish between errors
	// that are considered "criticial" and those that are not.
	//
	// Critical errors are the ones that prevent the imported community
	// from functioning properly. For example, if the creation of the community
	// or its categories and channels fails, this is a critical error.
	//
	// Non-critical errors are the ones that would not prevent the imported
	// community from functioning. For example, if the channel data to be imported
	// has no messages, or is not parsable.
	Code    ErrorType `json:"code"`
	Message string    `json:"message"`
}

func (d ImportError) Error() string {
	return fmt.Sprintf("%d: %s", d.Code, d.Message)
}
