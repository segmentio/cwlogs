# cwlogs

Simple CLI for viewing cloudwatch logs

## Installing

To install:

```bash
$ go get github.com/segmentio/cwlogs
```

## Authenticating

Using `cwlogs` requires you to be running in an environment with an authenticated AWS user which has read access to your logs.  The easiest way to do so is by using `aws-vault`, like:

```bash
$ aws-okta exec prod -- cwlogs
```

For this reason, it is recommended that you create an alias in your shell of choice to save yourself some typing, for example (from my `.zshrc`):

```
alias logsprod='aws-okta exec production -- cwlogs'
```

## Terminology

`Log Group`: Cloudwatch logs are organized first into log groups.  At Segment, we do this top level grouping by service/worker name.

`Log Stream`: Each log group is then split into log streams.  At Segment, we have each individual task of a service write to it's own stream.  To better correlate those streams with running tasks, we name the streams as the UUID of the ECS task ARN.

## Fetching logs

```bash
$ cwlogs help fetch
fetch logs for a given service

Usage:
  cwlogs fetch [service] [flags]

Flags:
  -f, --follow          Follow log streams
  -o, --format string   Format template for displaying log events (default "[ {{ uniquecolor (print .TaskShort) }} ] {{ .TimeShort }} {{ colorlevel .Level }} - {{ .Message }}")
  -s, --since string    Fetch logs since timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes) (default "all")
  -t, --task string     Task UUID or prefix
  -u, --until string    Fetch logs until timestamp (e.g. 2013-01-02T13:23:37) or relative (e.g. 42m for 42 minutes) (default "now")
  -v, --verbose         Verbose log output (includes log context in data fields)

Global Flags:
  -c, --color   Enable color output (default true)

```

To fetch logs for a specific service, simply do:

```bash
$ cwlogs fetch my-service
```

By default, this will try to fetch logs for your service from the last hour, from the 100 most recently active log streams.  This is potentially a large amount of data, and could take a while for long running or busy services.

To avoid grabbing all of that data, you can specify time ranges to fetch data from.  To set the time range, you will use the `--since` and/or the `--until` flags.  These flags can be specified in a few ways:
* Relative times (`10m`, `1h`, etc)
* Absolute dates (`2017-05-21`)
* Absolute time (`2017-05-21T15:00:00`)

For example, to grab all the logs for a service for the last hour, you can do:

```bash
$ cwlogs fetch my-service --since 1h
```

Fetching logs by service will interpolate the logs from any active streams into one stream of output.  If you would like to grab logs from a specific stream (which is a specific task in our case), you can use the `--task` flag.  This flag can be the exact task UUID or a prefix, like:

```bash
$ cwlogs fetch my-service --task 12345
```

In order see which tasks/streams are available for a given service/log group, you can use the `list` command:

```bash
$ cwlogs list my-service
```

By default, this will list the 100 most recently active streams.  If you want to look for streams within a given time period, you can use the `--since` and `--until` flags, as with fetch.  For example:

```bash
$cwlogs list my-service --since 2017-05-20 --until 2017-05-21
```

will show you the streams active during the day `2017-05-20`.


## Controlling Log Output

If you'd like to change the format for outputting log events, the `fetch` command has an `--format` flag which allows you to modify the output format.  The value of this string is a [go template](https://golang.org/pkg/text/template).  The specified template will be applied to each log event as it is output.

For reference, the following fields are available on the log event:

`.Group` - The log group that the event came from  
`.Stream` - The log stream that the event came from  
`.ID` - The AWS defined event ID  
`.IngestTime` - The time that the event was ingested by cloudwatch  
`.CreationTime` - The time the event was created  
`.Level` - The log level  
`.Time` - The client reported time stamp  
`.Info.Host` - The host which produced the event  
`.Message` - The log message  
`.Data` - Structured log context  
`.DataFlat` - A flattened copy of the structured log context  
`.TaskShort` - A shortened format for task UUID based on stream name  
`.TimeShort` - A shortened time stamp based on client reported time  

In addition to these fields, we've made a few functions available to provide more output options:

`red` - Prints arguments in red  
`green` - Prints arguments in green  
`yellow` - Prints arguments in yellow  
`blue` - Prints arguments in blue  
`magenta` - Prints arguments in magenta  
`cyan` - Prints arguments in cyan  
`white` - Prints arguments in white  
`colorlevel` - Takes a log level argument and colors it based on severity  
`uniquecolor` - Picks a unique color based on the string input.  Will always return the same color for the same string argument.  

For example, if you always only cared about the time and message of a log, and wanted the message printed in blue, you could do:

```bash
$ cwlogs -o "{{ .TimeShort }} - {{ blue .Message }}"
```
