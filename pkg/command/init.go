package command

import (
	"flag"

	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func InitCommand() *Command {
	command := newCommand("init")
	command.Action = func(args []string) error {
		path := *flag.String("Path", ".", "Path to create repository")
		_, err := repository.Create(path)
		return err
	}
	command.Description = func() string { return "Create a new git repository" }
	return command
}
