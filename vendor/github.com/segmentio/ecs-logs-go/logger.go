package ecslogs

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Logger interface {
	Log(Event) error
}

type LoggerFunc func(Event) error

func (f LoggerFunc) Log(e Event) error {
	return f(e)
}

func NewLogger(w io.Writer) Logger {
	if w == nil {
		w = os.Stderr
	}
	enc := json.NewEncoder(w)
	return LoggerFunc(func(event Event) error { return encode(enc, event) })
}

func encode(enc *json.Encoder, event Event) (err error) {
	if err = enc.Encode(event); err == nil {
		return
	}

	// Attempts to recover from invalid data put in the free form Event.Data field.
	switch err.(type) {
	case *json.UnsupportedTypeError, *json.UnsupportedValueError, *json.MarshalerError:
		event.Level = ALERT
		event.Info.Errors = append(event.Info.Errors, MakeEventError(err))
		event.Data = EventData{"unserializable": fmt.Sprintf("%#v", event.Data)}
		err = enc.Encode(event)
	}

	return
}
