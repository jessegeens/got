package command

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func LsTreeCommand() *Command {
	command := newCommand("ls-tree")
	command.Action = func(args []string) error {
		recursive := *flag.Bool("r", false, "Recurse into sub-trees")
		tree := *flag.String("tree", "", "A tree-ish object")
		repo, err := repository.Find(".")
		if err != nil {
			return err
		}
		return lsTree(repo, tree, "", recursive)
	}
	command.Description = func() string { return "Compute object ID and optionally creates a blob from a file" }
	return command
}

func lsTree(repo *repository.Repository, ref, prefix string, recursive bool) error {
	sha, err := objects.Find(repo, ref, objects.TypeNoTypeSpecified, true)
	if err != nil {
		return err
	}

	object, err := objects.ReadObject(repo, sha)
	if err != nil {
		return err
	}

	if object.Type() != objects.TypeTree {
		return errors.New("ref should point to a tree")
	}
	tree := object.(*objects.Tree)

	for _, item := range tree.Items {
		var rawobjtype []byte
		var objtype objects.GitObjectType
		if len(item.Mode) == 5 {
			rawobjtype = item.Mode[0:1]
		} else {
			rawobjtype = item.Mode[0:2]
		}

		switch string(rawobjtype) {
		case "4":
			objtype = objects.TypeTree
		case "10":
			objtype = objects.TypeBlob // A regular file
		case "12":
			objtype = objects.TypeBlob // A symlink
		case "16":
			objtype = objects.TypeCommit // A submodule
		}

		if !(recursive && objtype == objects.TypeTree) {
			fmt.Printf("")
		} else {
			err = lsTree(repo, item.PrintSHA(), filepath.Join(prefix, item.PrintPath()), recursive)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
