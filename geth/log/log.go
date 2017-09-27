package log

import (
	"fmt"
	"time"
)

// Field represents a giving map of values associated with a giving field value.
type Field map[string]interface{}

// Timelapse defines a message attached with a giving time value.
type Timelapse struct {
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Field   Field     `json:"fields"`
}

// WithTimelapse returns a Timelapse with associated field and message.
func WithTimelapse(message string, f Field) Timelapse {
	return Timelapse{
		Field:   f,
		Message: message,
		Time:    time.Now(),
	}
}

// Level defines a int type which represent the a giving level of entry for a giving entry.
type Level int

// level constants
const (
	RedAlertLvl    Level = iota // Immediately notify everyone by mail level, because this is bad
	YellowAlertLvl              // Immediately notify everyone but we can wait to tomorrow
	ErrorLvl                    // Error occured with some code due to normal opperation or odd behaviour (not critical)
	InfoLvl                     // Information for view about code operation (replaces Debug, Notice, Trace).
)

// WithMessage returns a new Entry with the provided Level and message used.
func WithMessage(level Level, message string, m ...interface{}) Entry {
	var e Entry
	e.Level = level
	e.Field = make(Field)
	e.Message = fmt.Sprintf(message, m...)

	return e
}

// WithTrace returns itself after setting the giving trace value
// has the method trace for the giving Entry.
func WithTrace(t *Trace) Entry {
	var e Entry
	e.Field = make(Field)
	e.Trace = t
	return e
}

// With returns a Entry set to the LogLevel of the previous and
// adds the giving key-value pair to the entry.
func With(key string, value interface{}) Entry {
	var e Entry
	e.Field = make(Field)
	e.Field[key] = value
	return e
}

// WithFields adds all field key-value pair into associated Entry
// returning the Entry.
func WithFields(f Field) Entry {
	var e Entry
	e.Field = make(Field)

	for k, v := range f {
		e.Field[k] = v
	}

	return e
}

// Entry represent a giving record of data at a giving period of time.
type Entry struct {
	ID        string      `json:"id"`
	Level     Level       `json:"level"`
	Field     Field       `json:"fields"`
	Time      time.Time   `json:"time"`
	Message   string      `json:"message"`
	Trace     *Trace      `json:"trace"`
	Timelapse []Timelapse `json:"timelapse"`
}

// WithID sets the Entry ID value.
func (e Entry) WithID(id string) Entry {
	e.ID = id
	return e
}

// WithLevel sets the Entry level.
func (e Entry) WithLevel(l Level) Entry {
	e.Level = l
	return e
}

// WithTimelapse adds provided Timelapse into Entry.Timelapse slice.
func (e Entry) WithTimelapse(t Timelapse) Entry {
	e.Timelapse = append(e.Timelapse, t)
	return e
}

// WithTrace returns itself after setting the giving trace value
// has the method trace for the giving Entry.
func (e Entry) WithTrace(t *Trace) Entry {
	e.Trace = t
	return e
}

// With returns a Entry set to the LogLevel of the previous and
// adds the giving key-value pair to the entry.
func (e Entry) With(key string, value interface{}) Entry {
	e.Field[key] = value
	return e
}

// WithFields adds all field key-value pair into associated Entry
// returning the Entry.
func (e Entry) WithFields(f Field) Entry {
	for k, v := range f {
		e.Field[k] = v
	}

	return e
}

// Metric defines an interface with a single method for receiving
// new Entry objects.
type Metric interface {
	Emit(Entry) error
}

// DoFn defines a function type which takes a giving Entry.
type DoFn func(Entry) error

// FilterFn defines a function type which takes a giving Entry returning a bool to indicate filtering state.
type FilterFn func(Entry) bool

// Augmenter defines a function type which takes a giving Entry returning a new associated entry value.
type Augmenter func(Entry) Entry

// Filter returns a Metric object with the provided Augmenters and  Metrics
// implemement objects for receiving metric Entries, where entries are filtered
// out based on a provided function.
func Filter(filterFn FilterFn, vals ...interface{}) Metric {
	return filteredMetrics{
		filterFn: filterFn,
		Metric:   New(vals...),
	}
}

// FilterLevelWith returns a Metric will will only emit Entrys that matches provided level.
func FilterLevelWith(lvl Level, fn DoFn) Metric {
	return Filter(func(en Entry) bool {
		return en.Level == lvl
	}, DoWith(fn))
}

// FilterLevel returns a Metric will will only emit Entrys that matches provided level.
func FilterLevel(lvl Level, m ...interface{}) Metric {
	return Filter(func(en Entry) bool {
		return en.Level == lvl
	}, m...)
}

// DoWith returns a Metric object where all entries are applied to the provided function.
func DoWith(do DoFn) Metric {
	return fnMetrics{
		do: do,
	}
}

// New returns a Metric object with the provided Augmenters and  Metrics
// implemement objects for receiving metric Entries.
func New(vals ...interface{}) Metric {
	var augmenters []Augmenter
	var childmetrics []Metric

	for _, val := range vals {
		switch item := val.(type) {
		case Augmenter:
			augmenters = append(augmenters, item)
		case Metric:
			childmetrics = append(childmetrics, item)
		}
	}

	return &metrics{
		augmenters: augmenters,
		metrics:    childmetrics,
	}
}

type metrics struct {
	augmenters []Augmenter
	metrics    []Metric
}

// Emit implements the Metric interface and delivers Entry
// to undeline metrics.
func (m metrics) Emit(en Entry) error {

	// Augment Entry with available augmenters.
	for _, aug := range m.augmenters {
		en = aug(en)
	}

	// Deliver augmented Entry to child Metrics
	for _, met := range m.metrics {
		if err := met.Emit(en); err != nil {
			return err
		}
	}

	return nil
}

type fnMetrics struct {
	do DoFn
}

// Emit implements the Metric interface and delivers Entry
// to undeline metrics.
func (m fnMetrics) Emit(en Entry) error {
	return m.do(en)
}

type filteredMetrics struct {
	Metric
	filterFn FilterFn
}

// Emit implements the Metric interface and delivers Entry
// to undeline metrics.
func (m filteredMetrics) Emit(en Entry) error {
	if !m.filterFn(en) {
		return nil
	}

	return m.Metric.Emit(en)
}
