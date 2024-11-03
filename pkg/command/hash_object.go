package command

import (
	"flag"
	"fmt"
	"os"

	"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func HashObjectCommand() *Command {
	command := newCommand("hash-object")
	command.Action = func(args []string) error {
		write := *flag.Bool("w", true, "Actually write the object into the database")
		path := *flag.String("path", "", "Read object from <file>")
		objType := *flag.String("type", "", "Object type. Possible values are blob, commit, tag, tree")

		parsedObjType, err := objects.ParseType(objType)
		if err != nil {
			return err
		}

		var repo *repository.Repository = nil

		if write {
			repo, err = repository.Find(".")
			if err != nil {
				return err
			}
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sha, err := objects.ObjectHash(b, parsedObjType, repo)
		if err != nil {
			return err
		}

		fmt.Println(sha)

		return nil
	}
	command.Description = func() string { return "Compute object ID and optionally creates a blob from a file" }
	return command
}
