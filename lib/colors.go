package lib

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	ecslogs "github.com/segmentio/ecs-logs-go"
)

// These color functions are used by output templates to control text color
var (
	Red     = color.New(color.FgRed).SprintFunc()
	Green   = color.New(color.FgGreen).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
	Blue    = color.New(color.FgBlue).SprintFunc()
	Magenta = color.New(color.FgMagenta).SprintFunc()
	Cyan    = color.New(color.FgCyan).SprintFunc()
	White   = color.New(color.FgWhite).SprintFunc()
)

var colorPool = []*color.Color{
	color.New(color.FgBlue),
	color.New(color.FgCyan),
	color.New(color.FgGreen),
	color.New(color.FgMagenta),
}

var colorIndex = 0

var usedColors = map[string]int{}

// Unique takes a string (or series of strings) and colors that string uniquely
// based on the string contents.  Calling this function with the same string input
// should always return the same color.
func Unique(args ...string) string {
	text := strings.Join(args, "")
	ix, ok := usedColors[text]
	if !ok {
		ix = colorIndex
		usedColors[text] = ix
		colorIndex = (colorIndex + 1) % len(colorPool)
	}

	return colorPool[ix].Sprint(text)
}

// ColorLevel takes a log level and colors it based on severity
func ColorLevel(l ecslogs.Level) string {
	switch l {
	case ecslogs.ERROR, ecslogs.ALERT, ecslogs.CRIT:
		return Red(l)
	case ecslogs.WARN:
		return Yellow(l)
	default:
		return fmt.Sprint(l)
	}
}
