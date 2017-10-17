package custom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/status-im/status-go/geth/log"
)

// FlatDisplay writes giving Entries as seperated blocks of contents where the each content is
// converted within a block like below:
//
//  Message: We must create new standard behaviour 	Function: BuildPack  |  display: red,  words: 20,
//
//  Message: We must create new standard behaviour 	Function: BuildPack  |  display: red,  words: 20,
//
func FlatDisplay(w io.Writer) log.Metrics {
	return FlatDisplayWith(w, "Message:", nil)
}

// FlatDisplayWith writes giving Entries as seperated blocks of contents where the each content is
// converted within a block like below:
//
//  [Header]: We must create new standard behaviour 	Function: BuildPack  |  display: red,  words: 20,
//
//  [Header]: We must create new standard behaviour 	Function: BuildPack  |  display: red,  words: 20,
//
func FlatDisplayWith(w io.Writer, header string, filterFn func(log.Entry) bool) log.Metrics {
	green := color.New(color.FgGreen)

	return NewEmitter(w, func(en log.Entry) []byte {
		if filterFn != nil && !filterFn(en) {
			return nil
		}

		var bu bytes.Buffer
		bu.WriteString("\n")

		if header != "" {
			fmt.Fprintf(&bu, "%s %+s", green.Sprint(header), en.Message)
		} else {
			fmt.Fprintf(&bu, "%+s", en.Message)
		}

		fmt.Fprint(&bu, printSpaceLine(2))

		if en.Function != "" {
			fmt.Fprintf(&bu, "%s: %+s", green.Sprint("Function"), en.Function)
			fmt.Fprint(&bu, printSpaceLine(2))
			fmt.Fprintf(&bu, "%s: %+s:%d", green.Sprint("File"), en.File, en.Line)
			fmt.Fprint(&bu, printSpaceLine(2))
		}

		fmt.Fprint(&bu, "|")
		fmt.Fprint(&bu, printSpaceLine(2))

		for key, value := range en.Field {
			fmt.Fprintf(&bu, "%+s: %+s", green.Sprint(key), printValue(value))
			fmt.Fprint(&bu, printSpaceLine(2))
		}

		bu.WriteString("\n")
		return bu.Bytes()
	})
}

//=====================================================================================

// SwitchEmitter returns a emitter that converts the behaviour of the output based on giving key and value from
// each Entry.
func SwitchEmitter(keyName string, w io.Writer, transformers map[string]func(log.Entry) []byte) log.Metrics {
	emitters := make(map[string]log.Metrics)

	for id, tm := range transformers {
		emitters[id] = NewEmitter(w, tm)
	}

	return log.Switch(keyName, emitters)
}

//=====================================================================================

// Emitter emits all entries into the entries into a sink io.writer after
// transformation from giving transformer function..
type Emitter struct {
	Sink      io.Writer
	Transform func(log.Entry) []byte
}

// NewEmitter returns a new instance of Emitter.
func NewEmitter(w io.Writer, transform func(log.Entry) []byte) *Emitter {
	return &Emitter{
		Sink:      w,
		Transform: transform,
	}
}

// Emit implements the log.metrics interface.
func (ce *Emitter) Emit(e log.Entry) error {
	_, err := ce.Sink.Write(ce.Transform(e))
	return err
}

//=====================================================================================

func printSpaceLine(length int) string {
	var lines []string

	for i := 0; i < length; i++ {
		lines = append(lines, " ")
	}

	return strings.Join(lines, "")
}

func printBlockLine(length int) string {
	var lines []string

	for i := 0; i < length; i++ {
		lines = append(lines, "-")
	}

	return strings.Join(lines, "")
}

type stringer interface {
	String() string
}

func printValue(item interface{}) string {
	switch bo := item.(type) {
	case stringer:
		return bo.String()
	case string:
		return `"` + bo + `"`
	case error:
		return bo.Error()
	case int:
		return strconv.Itoa(bo)
	case int8:
		return strconv.Itoa(int(bo))
	case int16:
		return strconv.Itoa(int(bo))
	case int64:
		return strconv.Itoa(int(bo))
	case time.Time:
		return bo.UTC().String()
	case rune:
		return strconv.QuoteRune(bo)
	case bool:
		return strconv.FormatBool(bo)
	case byte:
		return strconv.QuoteRune(rune(bo))
	case float64:
		return strconv.FormatFloat(bo, 'f', 4, 64)
	case float32:
		return strconv.FormatFloat(float64(bo), 'f', 4, 64)
	}

	data, err := json.Marshal(item)
	if err != nil {
		return fmt.Sprintf("%#v", item)
	}

	return string(data)
}
