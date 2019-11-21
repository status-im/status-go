package protocol

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Content contains the chat ID and the actual text of a message.
type Content struct {
	ChatID     string   `json:"chat_id"`
	Text       string   `json:"text"`
	ResponseTo string   `json:"response-to"`
	Name       string   `json:"name"` // the ENS name of the sender
	ParsedText ast.Node `json:"parsedText"`
	LineCount  int      `json:"lineCount"`
	RTL        bool     `json:"rtl"`
}

// Check if the first character is Hebrew or Arabic or the RTL character
func isRTL(s string) bool {
	first, _ := utf8.DecodeRuneInString(s)
	return unicode.Is(unicode.Hebrew, first) ||
		unicode.Is(unicode.Arabic, first) ||
		// RTL character
		first == '\u200f'
}

// PrepareContent return the parsed content of the message, the line-count and whether
// is a right-to-left message
func PrepareContent(content Content) Content {
	content.ParsedText = markdown.Parse([]byte(content.Text), nil)
	content.LineCount = strings.Count(content.Text, "\n")
	content.RTL = isRTL(content.Text)
	return content
}
