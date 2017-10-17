package log

import (
	"fmt"
	"time"
)

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

// WithMessage returns a new Entry with the provided Level and message used.
func WithMessage(level Level, message string, m ...interface{}) Entry {
	var e Entry
	e.Level = level
	e.Field = make(Field)
	e.Time = time.Now()
	e.Function, e.File, e.Line = getFunctionName(4)

	if len(m) == 0 {
		e.Message = message
		return e
	}

	e.Message = fmt.Sprintf(message, m...)
	return e
}

// WithTrace returns itself after setting the giving trace value
// has the method trace for the giving Entry.
func WithTrace(t *Trace) Entry {
	var e Entry
	e.Field = make(Field)
	e.Time = time.Now()
	e.Trace = t
	e.Function, e.File, e.Line = getFunctionName(4)
	return e
}

// WithID returns a Entry and set the ID to the provided value.
func WithID(id string) Entry {
	var e Entry
	e.ID = id
	e.Time = time.Now()
	e.Field = make(Field)
	e.Function, e.File, e.Line = getFunctionName(4)
	return e
}

// With returns a Entry set to the LogLevel of the previous and
// adds the giving key-value pair to the entry.
func With(key string, value interface{}) Entry {
	var e Entry
	e.Time = time.Now()
	e.Field = make(Field)
	e.Field[key] = value
	e.Function, e.File, e.Line = getFunctionName(4)
	return e
}

// WithFields adds all field key-value pair into associated Entry
// returning the Entry.
func WithFields(f Field) Entry {
	var e Entry
	e.Field = make(Field)
	e.Time = time.Now()

	e.Function, e.File, e.Line = getFunctionName(4)

	for k, v := range f {
		e.Field[k] = v
	}

	return e
}

// Entry represent a giving record of data at a giving period of time.
// TODO(influx6): Currently all Entry methods are on value and return themselves
// to safe uses with concurrency, but i do need to decide if its necessary to guard
// Entry at all with mutex, and are their cases of race condition in their use.
// Currenty most usage are a once-off set and send type of situation, but if for
// example, we wish to store Timelapse and this will span multiple points, then we
// either ensure people are aware just like with slices to set the new value to the returned
// variable, else use pointers, but will these not cause issues with concurreny later?
//
type Entry struct {
	ID        string      `json:"id"`
	Function  string      `json:"function"`
	File      string      `json:"file"`
	Line      int         `json:"line"`
	Level     Level       `json:"level"`
	Field     Field       `json:"fields"`
	Time      time.Time   `json:"time"`
	Message   string      `json:"message"`
	Trace     *Trace      `json:"trace"`
	Timelapse []Timelapse `json:"timelapse"`
}

// WithMessage sets the Entry Message value.
func (e Entry) WithMessage(message string, m ...interface{}) Entry {
	if len(m) == 0 {
		e.Message = message
		return e
	}

	e.Message = fmt.Sprintf(message, m...)
	return e
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
