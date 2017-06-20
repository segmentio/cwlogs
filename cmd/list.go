package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/segmentio/cwlogs/lib"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list streams for a given service",
	RunE:  list,
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&task, "task", "t", "", "")
	listCmd.Flags().StringVarP(&since, "since", "s", "1h", "Show logs streams with activity since timestamp (e.g. 2013-01-02T13:23:37), relative (e.g. 42m for 42 minutes), or all for all logs")
	listCmd.Flags().StringVarP(&until, "until", "u", "now", "Show log streams until timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes)")
}

func list(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return ErrTooFewArguments
	}
	if len(args) > 1 {
		return ErrTooManyArguments
	}

	start, err := lib.GetTime(since, time.Now())
	if err != nil {
		return fmt.Errorf("Failed to parse time '%s'", since)
	}

	var end time.Time
	if cmd.Flags().Lookup("until").Changed {
		end, err = lib.GetTime(until, time.Now())
		if err != nil {
			return fmt.Errorf("Failed to parse time '%s'", until)
		}
	}

	logReader, err := lib.NewCloudwatchLogsReader(args[0], task, start, end)
	if err != nil {
		return err
	}

	streams, err := logReader.ListStreams()
	if err != nil {
		return err
	}
	sort.Sort(lib.ByLastEvent(streams))

	if len(streams) == 0 {
		return fmt.Errorf("No log streams found since %s.", start.Format(lib.ShortTimeFormat))
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
	fmt.Fprintln(w, "Task\tLast Event\tCreation")

	for _, stream := range streams {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			*stream.LogStreamName,
			lib.ParseAWSTimestamp(stream.LastEventTimestamp).Local().Format(lib.ShortTimeFormat),
			lib.ParseAWSTimestamp(stream.CreationTime).Local().Format(lib.ShortTimeFormat),
		)
	}
	w.Flush()
	return nil
}
