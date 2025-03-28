package command

import (
	"flag"
	"fmt"

	"github.com/jessegeens/go-toolbox/pkg/ignore"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func CheckIgnoreCommand() *Command {
	command := newCommand("check-ignore")
	command.Action = func(args []string) error {
		paths := []string{}
		firstPath := *flag.String("path", "", "Paths to check")
		flag.Parse()
		if firstPath != "" {
			paths = []string{firstPath}
			tail := flag.Args()
			paths = append(paths, tail...)
		}

		repo, err := repository.Find(".")
		if err != nil {
			return err
		}

		ign, err := ignore.Read(repo)
		if err != nil {
			return err
		}

		for _, path := range paths {
			if ign.ShouldBeIgnored(path) {
				fmt.Println(path)
			}
		}

		return nil
	}
	command.Description = func() string { return "Check path(s) against ignore rules" }
	return command
}
