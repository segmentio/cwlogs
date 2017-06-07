package ecslogs

import (
	"encoding/json"
	"fmt"
	"reflect"
	"syscall"
	"time"
)

type EventError struct {
	Type  string      `json:"type,omitempty"`
	Error string      `json:"error,omitempty"`
	Errno int         `json:"errno,omitempty"`
	Stack interface{} `json:"stack,omitempty"`
}

func MakeEventError(err error) EventError {
	e := EventError{
		Type:  reflect.TypeOf(err).String(),
		Error: err.Error(),
	}

	if errno, ok := err.(syscall.Errno); ok {
		e.Errno = int(errno)
	}

	return e
}

type EventInfo struct {
	Host   string       `json:"host,omitempty"`
	Source string       `json:"source,omitempty"`
	ID     string       `json:"id,omitempty"`
	PID    int          `json:"pid,omitempty"`
	UID    int          `json:"uid,omitempty"`
	GID    int          `json:"gid,omitempty"`
	Errors []EventError `json:"errors,omitempty"`
}

func (e EventInfo) Bytes() []byte {
	b, _ := json.Marshal(e)
	return b
}

func (e EventInfo) String() string {
	return string(e.Bytes())
}

type EventData map[string]interface{}

func (e EventData) Bytes() []byte {
	b, _ := json.Marshal(e)
	return b
}

func (e EventData) String() string {
	return string(e.Bytes())
}

type Event struct {
	Level   Level     `json:"level"`
	Time    time.Time `json:"time"`
	Info    EventInfo `json:"info"`
	Data    EventData `json:"data"`
	Message string    `json:"message"`
}

func Eprintf(level Level, format string, args ...interface{}) Event {
	return MakeEvent(level, sprintf(format, args...), args...)
}

func Eprint(level Level, args ...interface{}) Event {
	return MakeEvent(level, sprint(args...), args...)
}

func MakeEvent(level Level, message string, values ...interface{}) Event {
	var errors []EventError

	for _, val := range values {
		switch v := val.(type) {
		case error:
			errors = append(errors, MakeEventError(v))
		}
	}

	return Event{
		Info:    EventInfo{Errors: errors},
		Data:    EventData{},
		Level:   level,
		Message: message,
	}
}

func (e Event) Bytes() []byte {
	b, _ := json.Marshal(e)
	return b
}

func (e Event) String() string {
	return string(e.Bytes())
}

func copyEventData(data ...EventData) EventData {
	copy := EventData{}

	for _, d := range data {
		for k, v := range d {
			copy[k] = v
		}
	}

	return copy
}

func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func sprint(args ...interface{}) string {
	s := fmt.Sprintln(args...)
	return s[:len(s)-1]
}
