package command

import (
	"errors"
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jessegeens/go-toolbox/pkg/index"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func RmCommand() *Command {
	command := newCommand("rm")
	command.Action = func(args []string) error {
		path := *flag.String("path", "", "Specify the path of the file or directory to remove")
		repo, err := repository.Find(".")
		if err != nil {
			return err
		}

		return rm(repo, path, true)
	}
	command.Description = func() string { return "Remove files from the working tree and the index" }
	return command
}

func rm(repo *repository.Repository, rmPath string, delete bool) error {
	idx, err := index.Read(repo)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(rmPath)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(absPath, repo.WorkTree()) {
		return errors.New("cannot remove a path outside the worktree")
	}

	toKeep := []*index.Entry{}
	var toDelete *index.Entry

	for _, e := range idx.Entries {
		fullPath := path.Join(repo.WorkTree(), e.Name)
		if fullPath == absPath {
			toDelete = e
		} else {
			toKeep = append(toKeep, e)
		}
	}

	if toDelete == nil {
		return errors.New("path not found in the worktree")
	}

	if delete {
		err = os.RemoveAll(absPath)
		if err != nil {
			return err
		}
	}

	idx.Entries = toKeep
	return idx.Write(repo)
}
