package lib

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/segmentio/bellows"
	"github.com/segmentio/ecs-logs-go"
)

const (
	// ShortTimeFormat is a short format for printing timestamps
	ShortTimeFormat = "01-02 15:04:05"
)

// TaskUUIDPattern is used to match task UUIDs
var TaskUUIDPattern = regexp.MustCompile(`^[[:alnum:]]{8}-[[:alnum:]]{4}-[[:alnum:]]{4}-[[:alnum:]]{4}-[[:alnum:]]{12}$`)

// Event represents a log event
type Event struct {
	ecslogs.Event
	Stream       string
	Group        string
	ID           string
	IngestTime   time.Time
	CreationTime time.Time
}

// NewEvent takes a cloudwatch log event and returns an Event
func NewEvent(cwEvent cloudwatchlogs.FilteredLogEvent, group string) Event {
	var ecsLogsEvent ecslogs.Event
	if err := json.Unmarshal([]byte(*cwEvent.Message), &ecsLogsEvent); err != nil {
		ecsLogsEvent = ecslogs.MakeEvent(ecslogs.INFO, *cwEvent.Message)
	}

	// If time was not found use AWS Timestamp
	if ecsLogsEvent.Time.IsZero() {
		ecsLogsEvent.Time = ParseAWSTimestamp(cwEvent.Timestamp)
	} else {
		ecsLogsEvent.Message = *cwEvent.Message
	}

	return Event{
		Event:        ecsLogsEvent,
		Stream:       *cwEvent.LogStreamName,
		Group:        group,
		ID:           *cwEvent.EventId,
		IngestTime:   ParseAWSTimestamp(cwEvent.IngestionTime),
		CreationTime: ParseAWSTimestamp(cwEvent.Timestamp),
	}

}

// ParseAWSTimestamp takes the time stamp format given by AWS and returns an equivalent time.Time value
func ParseAWSTimestamp(i *int64) time.Time {
	if i == nil {
		return time.Unix(0, 0)
	}
	return time.Unix(*i/1e3, (*i%1e3)*1e6)
}

// TaskShort attempts to shorten a stream name if it is a task UUID, leaving the stream
// name intact if it is not a UUID
func (e Event) TaskShort() string {
	if TaskUUIDPattern.MatchString(e.Stream) {
		uuidParts := strings.Split(e.Stream, "-")
		return uuidParts[0]
	}
	return e.Stream
}

// TimeShort gives the timestamp of an event in a readable format
func (e Event) TimeShort() string {
	return e.Time.Local().Format(ShortTimeFormat)
}

func (e Event) DataFlat() map[string]interface{} {
	return bellows.Flatten(e.Data)
}

func (e Event) PrettyPrint() string {
	pretty, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", e)
	}
	return string(pretty)
}

// ByCreationTime is used to sort events by their creation time
type ByCreationTime []Event

func (b ByCreationTime) Len() int           { return len(b) }
func (b ByCreationTime) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByCreationTime) Less(i, j int) bool { return b[i].CreationTime.Before(b[j].CreationTime) }
