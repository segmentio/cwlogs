package cmd

import (
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/segmentio/cwlogs/lib"
	"github.com/spf13/cobra"
)

const (
	verboseFormatString = `[ {{ uniquecolor (print .TaskShort) }} ] {{ .TimeShort }} {{ colorlevel .Level }} - {{ range $key, $value := .DataFlat }} {{ printf "%v=%v" $key $value }} {{end}} {{ .Message }}`
	defaultFormatString = `[ {{ uniquecolor (print .TaskShort) }} ] {{ .TimeShort }} {{ colorlevel .Level }} - {{ .Message }}`
)

var templateFuncMap = template.FuncMap{
	"red":         lib.Red,
	"green":       lib.Green,
	"yellow":      lib.Yellow,
	"blue":        lib.Blue,
	"magenta":     lib.Magenta,
	"cyan":        lib.Cyan,
	"white":       lib.White,
	"uniquecolor": lib.Unique,
	"colorlevel":  lib.ColorLevel,
}

var (
	follow        bool
	task          string
	eventTemplate string
	since         string
	until         string
	verbose       bool
)

// Error messages
var (
	ErrTooFewArguments  = errors.New("Too few arguments")
	ErrTooManyArguments = errors.New("Too many arguments")
	ErrNoEventsFound    = errors.New("No log events found")
)

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch [service]",
	Short: "fetch logs for a given service",
	RunE:  fetch,
}

func init() {
	RootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringVarP(&task, "task", "t", "", "Task UUID or prefix")
	fetchCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log streams")
	fetchCmd.Flags().StringVarP(&eventTemplate, "format", "o", defaultFormatString, "Format template for displaying log events")
	fetchCmd.Flags().StringVarP(&since, "since", "s", "all", "Fetch logs since timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes)")
	fetchCmd.Flags().StringVarP(&until, "until", "u", "now", "Fetch logs until timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes)")
	fetchCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose log output (includes log context in data fields)")
}

func fetch(cmd *cobra.Command, args []string) error {
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
		if cmd.Flags().Lookup("follow").Changed {
			return fmt.Errorf("Can't set both --until and --follow")
		}
		end, err = lib.GetTime(until, time.Now())
		if err != nil {
			return fmt.Errorf("Failed to parse time '%s'", until)
		}
	}

	logReader, err := lib.NewCloudwatchLogsReader(args[0], task, start, end)
	if err != nil {
		return err
	}

	if verbose {
		eventTemplate = verboseFormatString
	}

	output, err := template.New("event").Funcs(templateFuncMap).Parse(eventTemplate)
	if err != nil {
		return err
	}

	eventChan := logReader.StreamEvents(follow)

	ticker := time.After(7 * time.Second)

ReadLoop:
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				break ReadLoop
			}
			err = output.Execute(os.Stdout, event)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "\n")
			// reset slow log warning timer
			ticker = time.After(7 * time.Second)
		case <-ticker:
			if !follow {
				fmt.Fprintf(os.Stdout, "logs are taking a while to load... possibly try a smaller time window")
			}
		}
	}

	return logReader.Error()
}
