package lib

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/hashicorp/golang-lru"
)

const (
	// MaxEventsPerCall is the maximum number events from a filter call
	MaxEventsPerCall = 10000
	// MaxStreams is the maximum number of streams you can give to a filter call
	MaxStreams = 100
)

// CloudwatchLogsReader is responsible for fetching logs for a particular log
// group
type CloudwatchLogsReader struct {
	logGroupName string
	logStreams   []*cloudwatchlogs.LogStream
	svc          *cloudwatchlogs.CloudWatchLogs
	nextToken    *string
	eventCache   *lru.Cache
	start        time.Time
	end          time.Time
}

// NewCloudwatchLogsReader takes a group and optionally a stream prefix, start and
// end time, and returns a reader for any logs that match those parameters.
func NewCloudwatchLogsReader(group string, streamPrefix string, start time.Time, end time.Time) (*CloudwatchLogsReader, error) {
	session := session.New()
	svc := cloudwatchlogs.New(session)

	lg, err := getLogGroup(svc, group)
	if err != nil {
		return nil, err
	}

	streams, err := getLogStreams(svc, lg, streamPrefix, start, end)
	if err != nil {
		return nil, err
	}

	cache, err := lru.New(MaxEventsPerCall)
	if err != nil {
		return nil, err
	}

	reader := &CloudwatchLogsReader{
		logGroupName: group,
		logStreams:   streams,
		svc:          svc,
		eventCache:   cache,
		start:        start,
		end:          end,
	}

	return reader, nil
}

// ListStreams returns any log streams that match the params given in the
// reader's constructor.  Will return at most `MaxStreams` streams
func (c *CloudwatchLogsReader) ListStreams() []*cloudwatchlogs.LogStream {
	return c.logStreams
}

// FetchEvents attempts to read all events matching the params given in the
// reader's constructor.  Subsequent calls to FetchEvents will return new
// events if any have been created since the last call to FetchEvents.
func (c *CloudwatchLogsReader) FetchEvents() ([]Event, error) {
	startTime := c.start.Unix() * 1e3
	params := &cloudwatchlogs.FilterLogEventsInput{
		Interleaved:  aws.Bool(true),
		LogGroupName: aws.String(c.logGroupName),
		NextToken:    c.nextToken,
		StartTime:    aws.Int64(startTime),
	}

	// Only set end time if it is set, otherwise always fetch up
	// to current time.
	if !c.end.IsZero() {
		endTime := c.end.Unix() * 1e3
		params.EndTime = aws.Int64(endTime)
	}

	if len(c.logStreams) > 0 {
		params.LogStreamNames = streamsToNames(c.logStreams)
	}

	// TODO: possibly we should only return a single page at a time, as
	// large queries take a long time to page through all results
	events := []Event{}
	if err := c.svc.FilterLogEventsPages(params, func(o *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
		for _, event := range o.Events {
			if _, ok := c.eventCache.Peek(*event.EventId); !ok {
				events = append(events, NewEvent(*event, c.logGroupName))
				c.eventCache.Add(*event.EventId, nil)
			}
		}
		c.nextToken = o.NextToken
		return !lastPage
	}); err != nil {
		return events, err
	}
	sort.Sort(ByCreationTime(events))
	return events, nil
}

func getLogGroup(svc *cloudwatchlogs.CloudWatchLogs, name string) (*cloudwatchlogs.LogGroup, error) {
	describeLogGroupsInput := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(name),
	}

	resp, err := svc.DescribeLogGroups(describeLogGroupsInput)
	if err != nil {
		return nil, err
	}

	if len(resp.LogGroups) == 0 {
		return nil, fmt.Errorf("Could not find log group '%s'", name)
	}

	if *resp.LogGroups[0].LogGroupName != name {
		// Didn't find exact match, offer some alternatives based on prefix
		errMsg := fmt.Sprintf("Could not find log group '%s'.\n\nDid you mean:\n\n", name)
		for ix, group := range resp.LogGroups {
			if ix > 4 {
				break
			}
			errMsg += fmt.Sprintf("%s\n", *group.LogGroupName)
		}
		return nil, errors.New(errMsg)
	}

	return resp.LogGroups[0], nil
}

func getLogStreams(svc *cloudwatchlogs.CloudWatchLogs, group *cloudwatchlogs.LogGroup, streamPrefix string, start time.Time, end time.Time) ([]*cloudwatchlogs.LogStream, error) {
	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: group.LogGroupName,
	}
	if streamPrefix != "" {
		// If we are looking for a specific stream, search by prefix
		params.LogStreamNamePrefix = aws.String(streamPrefix)
	} else {
		// If not, just give us the most recently active
		params.OrderBy = aws.String("LastEventTime")
		params.Descending = aws.Bool(true)
	}

	startTimestamp := start.Unix() * 1e3
	endTimestamp := end.Unix() * 1e3

	streams := []*cloudwatchlogs.LogStream{}
	if err := svc.DescribeLogStreamsPages(params, func(o *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
		emptyPage := true
		for _, s := range o.LogStreams {
			if len(streams) >= MaxStreams {
				return false
			}
			if !end.IsZero() && s.CreationTime != nil && *s.CreationTime > endTimestamp {
				continue
			}
			if s.LastEventTimestamp != nil && *s.LastEventTimestamp < startTimestamp {
				continue
			}
			streams = append(streams, s)
			emptyPage = false
		}

		// If we've reached a page with no results in our time window (and
		// have already matched at least one stream), then we don't need
		// to look at the rest of the pages
		if emptyPage && len(streams) > 0 {
			return false
		}

		return !lastPage
	}); err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(ByLastEvent(streams)))
	if len(streams) == 0 {
		return nil, fmt.Errorf("No logs found matching task prefix '%s'.  You can get the list of available streams using the `list` command.", streamPrefix)
	}
	return streams, nil
}

func streamsToNames(streams []*cloudwatchlogs.LogStream) []*string {
	names := make([]*string, 0, len(streams))
	for _, s := range streams {
		names = append(names, s.LogStreamName)
	}
	return names
}

// ByLastEvent is used to sort log streams by last event timestamp
type ByLastEvent []*cloudwatchlogs.LogStream

func (b ByLastEvent) Len() int      { return len(b) }
func (b ByLastEvent) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByLastEvent) Less(i, j int) bool {
	if b[i].LastEventTimestamp != nil && b[j].LastEventTimestamp != nil {
		return *b[i].LastEventTimestamp < *b[j].LastEventTimestamp
	}
	return true
}
