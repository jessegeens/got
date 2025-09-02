package command

import (
	"errors"
	"fmt"

	"github.com/jessegeens/got/pkg/objects"
	"github.com/jessegeens/got/pkg/repository"
)

func CatFileCommand() *Command {
	command := newCommand("cat-file")
	command.Action = func(args []string) error {
		if len(args) < 1 {
			return errors.New("must provide object hash as an argument")
		}
		objHash := args[0]
		repo, err := repository.Find(".")
		if err != nil {
			return err
		}
		sha, err := objects.Find(repo, objHash, objects.TypeNoTypeSpecified, true)
		if err != nil {
			return err
		}
		object, err := objects.ReadObject(repo, sha)
		if err != nil {
			return err
		}
		fmt.Println(object.Serialize())

		return nil
	}
	command.Description = func() string { return "Provide content of repository objects" }
	return command
}
