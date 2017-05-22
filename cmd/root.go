package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/fatih/color"
)

var useColor bool

var ErrInvalidCommand = errors.New("Invalid command")

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "cwlogs",
	Short: "Simple CLI for viewing cloudwatch logs",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Handle global flags
		color.NoColor = !useColor
	},
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ArbitraryArgs,
	RunE:          root,
}

func root(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		cmd.Usage()
		return nil
	}

	if len(args) != 1 {
		return ErrInvalidCommand
	}
	return fmt.Errorf("Didn't understand '%s', did you mean 'fetch %s'?", args[0], args[0])
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		switch err {
		case ErrTooFewArguments, ErrInvalidCommand, ErrTooManyArguments:
			RootCmd.Usage()
		}
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&useColor, "color", "c", true, "Enable color output")
}
