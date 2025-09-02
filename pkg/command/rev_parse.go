package command

import (
	"errors"
	"flag"
	"fmt"

	"github.com/jessegeens/got/pkg/objects"
	"github.com/jessegeens/got/pkg/repository"
)

func RevParseCommand() *Command {
	command := newCommand("rev-parse")
	command.Action = func(args []string) error {
		revType := *flag.String("type", "", "Specify the expected type: one of blob, commit, tag, tree")
		name := *flag.String("name", "", "The name to parse")
		flag.Parse()

		return revParse(revType, name)
	}
	command.Description = func() string { return "Parse revision (or other objects) identifiers" }
	return command
}

func revParse(revType, name string) error {
	format, err := objects.ParseType(revType)
	if err != nil {
		return errors.New("invalid type: " + revType)
	}
	if name == "" {
		return errors.New("no name given")
	}

	repo, err := repository.Find(".")
	if err != nil {
		return err
	}

	output, err := objects.Find(repo, name, format, true)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", output)
	return nil
}
