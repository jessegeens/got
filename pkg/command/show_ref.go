package command

import (
	"fmt"

	"github.com/jessegeens/got/pkg/references"
	"github.com/jessegeens/got/pkg/repository"
)

func ShowRefCommand() *Command {
	command := newCommand("show-ref")
	command.Action = func(args []string) error {
		repo, err := repository.Find(".")
		if err != nil {
			return err
		}

		refs, err := references.List(repo)
		if err != nil {
			return err
		}

		showRefs(repo, true, refs, "")

		return nil
	}
	command.Description = func() string { return "List references" }
	return command
}

func showRefs(repo *repository.Repository, withHash bool, refs map[references.Reference]any, prefix string) {
	for k, v := range refs {
		switch v := v.(type) {
		case string:
			withHashPrefix := ""
			if withHash {
				withHashPrefix = " "
			}
			if prefix != "" {
				fmt.Printf("%s/%s", withHashPrefix, k)
			} else {
				fmt.Printf("%s%s", withHashPrefix, k)
			}
		case map[references.Reference]any:
			if prefix != "" {
				prefix = fmt.Sprintf("%s/%s", prefix, k)
			} else {
				prefix = k.String()
			}
			showRefs(repo, withHash, v, prefix)
		}
	}
}
