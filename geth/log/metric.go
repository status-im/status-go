// Package log defines a basic structure foundation for handling logs without
// much hassle, allow more different entries to be created.
// Inspired by https://medium.com/@tjholowaychuk/apex-log-e8d9627f4a9a.
package log

import (
	"errors"
	"fmt"
	"strings"
)

// level constants
const (
	RedAlertLvl    Level = iota // Immediately notify everyone by mail level, because this is bad
	YellowAlertLvl              // Immediately notify everyone but we can wait to tomorrow
	ErrorLvl                    // Error occured with some code due to normal opperation or odd behaviour (not critical)
	InfoLvl                     // Information for view about code operation (replaces Debug, Notice, Trace).
)

// Level defines a int type which represent the a giving level of entry for a giving entry.
type Level int

// GetLevel returns Level value for the giving string.
// It returns -1 if it does not know the level string.
func GetLevel(lvl string) Level {
	switch strings.ToLower(lvl) {
	case "redalert", "redalartlvl":
		return RedAlertLvl
	case "yellowalert", "yellowalertlvl":
		return YellowAlertLvl
	case "error", "errorlvl":
		return ErrorLvl
	case "info", "infolvl":
		return InfoLvl
	}

	return -1
}

// String returns the string version of the Level.
func (l Level) String() string {
	switch l {
	case RedAlertLvl:
		return "REDALERT"
	case YellowAlertLvl:
		return "YELLOWALERT"
	case ErrorLvl:
		return "ERROR"
	case InfoLvl:
		return "INFO"
	}

	return "UNKNOWN"
}

// YellowAlert returns an Entry with the level set to YellowAlertLvl.
func YellowAlert(err error, message string, m ...interface{}) Entry {
	return WithMessage(YellowAlertLvl, message, m...).With("error", err)
}

// // Trace returns an Entry with the level set to TraceLvl.
// func Trace(message string, m ...interface{}) Entry {
// 	me := WithMessage(TraceLvl, message, m...).With("error", err)
// 	tr := NewTraceWithCallDepth(2, me.Message)
// 	return me.WithTrace(tr.End())
// }

// RedAlert returns an Entry with the level set to RedAlertLvl.
func RedAlert(err error, message string, m ...interface{}) Entry {
	return WithMessage(RedAlertLvl, message, m...).With("error", err)
}

// Errorf returns a entry where the message is the provided error.Error() value
// produced from the message and its provided values
// and the error is added as a key-value within the Entry fields.
func Errorf(message string, m ...interface{}) Entry {
	err := fmt.Errorf(message, m...)
	return WithMessage(ErrorLvl, err.Error()).With("error", err)
}

// Error returns a entry where the message is the provided error.Error() value
// and the error is added as a key-value within the Entry fields.
func Error(err error) Entry {
	return WithMessage(ErrorLvl, err.Error()).With("error", err)
}

// Info returns an Entry with the level set to Info.
func Info(message string, m ...interface{}) Entry {
	return WithMessage(InfoLvl, message, m...)
}

// Metrics defines an interface with a single method for receiving
// new Entry objects.
type Metrics interface {
	Emit(Entry) error
}

// DoFn defines a function type which takes a giving Entry.
type DoFn func(Entry) error

// FilterFn defines a function type which takes a giving Entry returning a bool to indicate filtering state.
type FilterFn func(Entry) bool

// Augmenter defines a function type which takes a giving Entry returning a new associated entry value.
type Augmenter func(Entry) Entry

// FilterLevel will return a metrics where all Entry will be filtered by their Entry.Level
// if the level giving is greater or equal to the provided, then it will be received by
// the metrics subscribers.
func FilterLevel(l Level, vals ...Metrics) Metrics {
	return Filter(func(en Entry) bool {
		return en.Level >= l
	}, vals...)
}

// Filter returns a Metrics object with the provided Augmenters and  Metrics
// implemement objects for receiving metric Entries, where entries are filtered
// out based on a provided function.
func Filter(filterFn FilterFn, vals ...Metrics) Metrics {
	return filteredMetrics{
		filterFn: filterFn,
		Metrics:  New(vals, nil),
	}
}

// DoWith returns a Metrics object where all entries are applied to the provided function.
func DoWith(do DoFn) Metrics {
	return fnMetrics{
		do: do,
	}
}

// Switch returns a new instance of a SwitchMaster.
func Switch(keyName string, selections map[string]Metrics) Metrics {
	return switchMaster{
		key:        keyName,
		selections: selections,
	}
}

// New returns a Metrics object with the provided Augmenters and  Metrics
// implemement objects for receiving metric Entries.
func New(ms []Metrics, augmenters []Augmenter) Metrics {
	return &metrics{
		metrics:    ms,
		augmenters: augmenters,
	}
}

type metrics struct {
	metrics    []Metrics
	augmenters []Augmenter
}

// Emit implements the Metrics interface and delivers Entry
// to undeline log.
func (m metrics) Emit(en Entry) error {
	// Augment Entry with available augmenters.
	for _, aug := range m.augmenters {
		en = aug(en)
	}

	// Deliver augmented Entry to child Metrics
	for _, met := range m.metrics {
		if err := met.Emit(en); err != nil {
			fmt.Printf("Metric failed to emit %#v\n", err)
			return err
		}
	}

	return nil
}

type fnMetrics struct {
	do DoFn
}

// Emit implements the Metrics interface and delivers Entry
// to undeline log.
func (m fnMetrics) Emit(en Entry) error {
	return m.do(en)
}

type filteredMetrics struct {
	Metrics
	filterFn FilterFn
}

// Emit implements the Metrics interface and delivers Entry
// to undeline log.
func (m filteredMetrics) Emit(en Entry) error {
	if !m.filterFn(en) {
		return nil
	}

	return m.Metrics.Emit(en)
}

// switchMaster defines that mod out Entry objects based on a provided function.
type switchMaster struct {
	key        string
	selections map[string]Metrics
}

// Emit delivers the giving entry to all available metricss.
func (fm switchMaster) Emit(e Entry) error {
	val, ok := e.Field[fm.key].(string)
	if !ok {
		return errors.New("Entry.Field has no such key")
	}

	selector, ok := fm.selections[val]
	if !ok {
		return errors.New("Metrics for key not found")
	}

	return selector.Emit(e)
}
