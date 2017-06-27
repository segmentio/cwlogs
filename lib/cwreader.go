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
	svc          *cloudwatchlogs.CloudWatchLogs
	nextToken    *string
	eventCache   *lru.Cache
	start        time.Time
	end          time.Time
	error        error
	streamPrefix string
}

// NewCloudwatchLogsReader takes a group and optionally a stream prefix, start and
// end time, and returns a reader for any logs that match those parameters.
func NewCloudwatchLogsReader(group string, streamPrefix string, start time.Time, end time.Time) (*CloudwatchLogsReader, error) {
	session := session.New()
	svc := cloudwatchlogs.New(session)

	if _, err := getLogGroup(svc, group); err != nil {
		return nil, err
	}

	cache, err := lru.New(MaxEventsPerCall)
	if err != nil {
		return nil, err
	}

	reader := &CloudwatchLogsReader{
		logGroupName: group,
		svc:          svc,
		eventCache:   cache,
		start:        start,
		end:          end,
		streamPrefix: streamPrefix,
	}

	return reader, nil
}

// ListStreams returns any log streams that match the params given in the
// reader's constructor.  Will return at most `MaxStreams` streams
func (c *CloudwatchLogsReader) ListStreams() ([]*cloudwatchlogs.LogStream, error) {
	return c.getLogStreams()
}

// StreamEvents returns a channel where you can read events matching the params
// given in the readers constructor.  The channel will be closed once
// all events are read or an error occurs.  You can check for errors
// after the channel is closed by calling Error()
func (c *CloudwatchLogsReader) StreamEvents(follow bool) <-chan Event {
	eventChan := make(chan Event)
	go c.pumpEvents(eventChan, follow)

	return eventChan
}

func (c *CloudwatchLogsReader) pumpEvents(eventChan chan<- Event, follow bool) {
	startTime := c.start.Unix() * 1e3
	params := &cloudwatchlogs.FilterLogEventsInput{
		Interleaved:  aws.Bool(true),
		LogGroupName: aws.String(c.logGroupName),
		NextToken:    c.nextToken,
		StartTime:    aws.Int64(startTime),
	}

	if !follow && c.end.IsZero() {
		c.end = time.Now()
	}

	if !c.end.IsZero() {
		endTime := c.end.Unix() * 1e3
		params.EndTime = aws.Int64(endTime)
	}

	if c.streamPrefix != "" {
		streams, err := c.getLogStreams()
		if err != nil {
			c.error = err
			return
		}
		params.LogStreamNames = streamsToNames(streams)
	}

	for {
		if err := c.svc.FilterLogEventsPages(params, func(o *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
			for _, event := range o.Events {
				if _, ok := c.eventCache.Peek(*event.EventId); !ok {
					eventChan <- NewEvent(*event, c.logGroupName)
					c.eventCache.Add(*event.EventId, nil)
				}
			}
			c.nextToken = o.NextToken
			return !lastPage
		}); err != nil {
			c.error = err
			close(eventChan)
			return
		}

		if !follow {
			close(eventChan)
			return
		}

		time.Sleep(5 * time.Second)
	}
}

// Error returns an error if one occured while streaming events.
func (c *CloudwatchLogsReader) Error() error {
	return c.error
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

func (c *CloudwatchLogsReader) getLogStreams() ([]*cloudwatchlogs.LogStream, error) {
	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(c.logGroupName),
	}

	sortByTime := false
	if c.streamPrefix != "" {
		// If we are looking for a specific stream, search by prefix
		params.LogStreamNamePrefix = aws.String(c.streamPrefix)
	} else {
		// If not, just give us the most recently active
		params.OrderBy = aws.String("LastEventTime")
		params.Descending = aws.Bool(true)
		sortByTime = true
	}

	startTimestamp := c.start.Unix() * 1e3
	endTimestamp := time.Now().Unix() * 1e3
	if !c.end.IsZero() {
		endTimestamp = c.end.Unix() * 1e3
	}

	streams := []*cloudwatchlogs.LogStream{}
	if err := c.svc.DescribeLogStreamsPages(params, func(o *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
		pastWindow := false
		for _, s := range o.LogStreams {
			if len(streams) >= MaxStreams {
				return false
			}
			if s.LastEventTimestamp == nil {
				// treat nil timestamps as 0
				s.LastEventTimestamp = aws.Int64(0)
			}

			// if we are sorting by time, we can do some shortcuts to end
			// paging early if we are no longer in our time window
			if sortByTime {

				if s.CreationTime != nil && *s.CreationTime > endTimestamp {
					continue
				}
				if *s.LastEventTimestamp < startTimestamp {
					pastWindow = true
					break
				}
				streams = append(streams, s)

			} else {
				// otherwise we have to check all pages, but there are fewer because
				// we are prefix matching
				if s.CreationTime != nil && *s.CreationTime < endTimestamp &&
					*s.LastEventTimestamp > startTimestamp {
					streams = append(streams, s)
				}
			}
		}

		// If we've iterated past our time window and are sorting by time, stop paging
		if pastWindow && sortByTime {
			return false
		}

		return !lastPage
	}); err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(ByLastEvent(streams)))
	if len(streams) == 0 {
		if c.streamPrefix != "" {
			return nil, fmt.Errorf("No log streams found matching task prefix '%s' in your time window.  Consider adjusting your time window with --since and/or --until", c.streamPrefix)
		} else {
			return nil, errors.New("No log streams found in your time window.  Consider adjusting your time window with --since and/or --until")
		}
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
	first := b[i].LastEventTimestamp
	second := b[j].LastEventTimestamp

	if first == nil {
		first = aws.Int64(0)
	}
	if second == nil {
		second = aws.Int64(0)
	}
	return *first < *second
}
