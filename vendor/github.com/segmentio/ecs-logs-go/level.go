package ecslogs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Level int

const (
	NONE Level = iota
	EMERG
	ALERT
	CRIT
	ERROR
	WARN
	NOTICE
	INFO
	DEBUG
	TRACE
)

type ParseLevelError struct {
	Level string
}

func (e ParseLevelError) Error() string {
	return fmt.Sprintf("invalid message level %#v", e.Level)
}

func MakeLevel(p int) Level {
	return Level(p + 1)
}

func ParseLevel(s string) (lvl Level, err error) {
	switch strings.ToUpper(s) {
	case "EMERG":
		lvl = EMERG
	case "ALERT":
		lvl = ALERT
	case "CRIT":
		lvl = CRIT
	case "ERROR":
		lvl = ERROR
	case "WARN":
		lvl = WARN
	case "NOTICE":
		lvl = NOTICE
	case "INFO":
		lvl = INFO
	case "DEBUG":
		lvl = DEBUG
	case "TRACE":
		lvl = TRACE
	default:
		err = ParseLevelError{s}
	}
	return
}

func (lvl Level) String() string {
	switch lvl {
	case EMERG:
		return "EMERG"
	case ALERT:
		return "ALERT"
	case CRIT:
		return "CRIT"
	case ERROR:
		return "ERROR"
	case WARN:
		return "WARN"
	case NOTICE:
		return "NOTICE"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	case TRACE:
		return "TRACE"
	default:
		return lvl.GoString()
	}
}

func (lvl Level) Priority() int {
	return int(lvl - 1)
}

func (lvl Level) GoString() string {
	return "Level(" + strconv.Itoa(lvl.Priority()) + ")"
}

func (lvl Level) MarshalText() (b []byte, err error) {
	b = []byte(lvl.String())
	return
}

func (lvl *Level) UnmarshalText(b []byte) (err error) {
	*lvl, err = ParseLevel(string(b))
	return
}

func (lvl *Level) MarshalJSON() (b []byte, err error) {
	b = make([]byte, 0, 20)
	b = append(b, '"')
	b = append(b, lvl.String()...)
	b = append(b, '"')
	return
}

func (lvl *Level) UnmarshalJSON(b []byte) (err error) {
	if !startsWith(b, '"') {
		return &json.UnsupportedValueError{Str: string(b)}
	}

	if !endsWith(b[1:], '"') {
		return &json.UnsupportedValueError{Str: string(b)}
	}

	if *lvl, err = ParseLevel(string(b[1 : len(b)-1])); err != nil {
		return &json.UnsupportedValueError{Str: string(b)}
	}

	return
}

func (lvl Level) MarshalYAML() (b []byte, err error) {
	b = []byte(lvl.String())
	return
}

func (lvl *Level) UnmarshalYAML(f func(interface{}) error) (err error) {
	var s string

	if err = f(&s); err != nil {
		return
	}

	return lvl.Set(s)
}

func (lvl Level) Get() interface{} {
	return lvl
}

func (lvl *Level) Set(s string) (err error) {
	*lvl, err = ParseLevel(s)
	return
}

func startsWith(b []byte, c byte) bool {
	return len(b) != 0 && b[0] == c
}

func endsWith(b []byte, c byte) bool {
	return len(b) != 0 && b[len(b)-1] == c
}
