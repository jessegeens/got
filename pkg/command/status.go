package command

import (
	"fmt"

	"github.com/jessegeens/go-toolbox/pkg/index"
	"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func StatusCommand() *Command {
	command := newCommand("status")
	command.Action = func(args []string) error {
		repo, err := repository.Find(".")
		if err != nil {
			return err
		}

		idx, err := index.Read(repo)
		if err != nil {
			return err
		}

		err = statusBranch(repo)
		if err != nil {
			return err
		}
		err = statusHeadIndex(repo, idx)
		if err != nil {
			return err
		}
		err = statusIndexWorktree(repo, idx)
		return err
	}
	command.Description = func() string { return "Show the working tree status" }
	return command
}

func statusBranch(repo *repository.Repository) error {
	branch, onBranch, err := repo.GetActiveBranch()
	if err != nil {
		return err
	}
	if onBranch {
		fmt.Printf("On branch %s\n", branch)
	} else {
		obj, err := objects.Find(repo, "HEAD", objects.TypeNoTypeSpecified, true)
		if err != nil {
			return err
		}
		fmt.Printf("HEAD detached at %s\n", obj)
	}
	return nil
}

// We compare HEAD to the index
func statusHeadIndex(repo *repository.Repository, idx *index.Index) error {
	head, err := objects.MapFromTree(repo, "HEAD")
	if err != nil {
		return err
	}

	for _, entry := range idx.Entries {
		if sha, ok := head[entry.Name]; ok {
			if sha != entry.SHA {
				fmt.Printf("  modified: %s\n", entry.Name)
			}
			delete(head, entry.Name)
		} else {
			fmt.Printf("  added: %s\n", entry.Name)
		}
	}

	for path := range head {
		fmt.Printf("  deleted: %s\n", path)
	}
	return nil
}

func statusIndexWorktree(repo *repository.Repository, idx *index.Index) error {
	return nil
}
