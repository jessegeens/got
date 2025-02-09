package command

import (
	"fmt"

	"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func CatFileCommand() *Command {
	command := newCommand("cat-file")
	command.Action = func(args []string) error {
		objHash := args[0]
		repo, err := repository.Find(".")
		if err != nil {
			return err
		}
		obj, err := objects.Find(repo, objHash)
		if err != nil {
			return err
		}
		object, err := objects.ReadObject(repo, obj)
		if err != nil {
			return err
		}
		fmt.Println(object.Serialize())

		return nil
	}
	command.Description = func() string { return "Provide content of repository objects" }
	return command
}
