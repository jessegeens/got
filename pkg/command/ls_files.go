package command

import (
	"flag"
	"fmt"
	"os/user"
	"strconv"

	"github.com/jessegeens/go-toolbox/pkg/index"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func LsFilesCommand() *Command {
	command := newCommand("ls-files")
	command.Action = func(args []string) error {
		verbose := *flag.Bool("verbose", true, "Show everything")
		flag.Parse()
		repo, err := repository.Find(".")
		if err != nil {
			return err
		}
		idx, err := index.Read(repo)
		if err != nil {
			return err
		}
		return lsFiles(idx, verbose)
	}
	command.Description = func() string { return "List all the stage files" }
	return command
}

func lsFiles(idx *index.Index, verbose bool) error {
	if verbose {
		fmt.Printf("Index file format v%d containing %d entries\n", idx.Version, len(idx.Entries))
	}

	for _, e := range idx.Entries {
		fmt.Println(e.Name)
		if verbose {
			var username, group string
			usr, err := user.LookupId(strconv.Itoa(int(e.UID)))
			if err == nil {
				username = usr.Username
			} else {
				username = "nobody"
			}
			grp, err := user.LookupGroupId(strconv.Itoa(int(e.GID)))
			if err == nil {
				group = grp.Name
			} else {
				group = "nobody"
			}
			fmt.Printf("  %s with perms: %o\n", e.ModeType.String(), e.ModePerms)
			fmt.Printf("  on blob: %s\n", e.SHA)
			fmt.Printf("  created: %s, modified: %s\n", e.CTime.String(), e.MTime.String())
			fmt.Printf("  device: %d, inode: %d\n", e.Dev, e.Inode)
			fmt.Printf("  user: %s (%d)  group: %s (%d)\n", username, e.UID, group, e.GID)
			fmt.Printf("  flags: stage=%d assume_valid=%t\n", e.FlagStage, e.FlagAssumeValid)
		}
	}
	return nil
}
