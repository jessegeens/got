package command

import (
	"flag"
	"fmt"
)

// Command is the representation to create commands.
type Command struct {
	*flag.FlagSet
	Name        string
	Action      func(args []string) error
	Usage       func() string
	Description func() string
	ResetFlags  func()
}

// newCommand creates a new command.
func newCommand(name string) *Command {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	cmd := &Command{
		Name: name,
		Usage: func() string {
			return fmt.Sprintf("Usage: %s", name)
		},
		Action: func(args []string) error {
			return nil
		},
		Description: func() string {
			return "Command description"
		},
		FlagSet:    fs,
		ResetFlags: func() {},
	}
	return cmd
}
