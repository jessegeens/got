package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func LogCommand() *Command {
	command := newCommand("log")
	command.Action = func(args []string) error {
		commit := *flag.String("commit", "HEAD", "Commit to start at") //args[0]
		return handleLogCommand(commit)
	}
	command.Description = func() string { return "Display history of a given commit" }
	return command
}

func handleLogCommand(commit string) error {
	repo, err := repository.Find(".")
	if err != nil {
		return err
	}
	obj, err := objects.Find(repo, commit, objects.TypeNoTypeSpecified, true)
	if err != nil {
		return err
	}

	fmt.Println("digraph gitlog{")
	fmt.Println("  node[shape=rect]")
	logGraphviz(repo, obj, make(map[string]bool))
	fmt.Println("}")
	return nil
}

func logGraphviz(repo *repository.Repository, objSha string, seen map[string]bool) error {
	// We already handled this commit
	if _, in := seen[objSha]; in {
		return nil
	}

	seen[objSha] = true

	// Get commit data
	gitobj, err := objects.ReadObject(repo, objSha)
	if err != nil {
		return err
	}
	commit := gitobj.(*objects.Commit)
	shortHash := objSha[0:7]
	message := commit.Message()

	// Only display first line of commit message
	if strings.Contains(message, "\n") {
		message = strings.Split(message, "\n")[0]
	}

	// Print line
	fmt.Printf("  c_%s [label=\"%s: %s\"]\n", objSha, shortHash, message)

	// Now, we go on to the recursion
	parents, hasParent := commit.GetValue("parent")
	// Base case: initial commit (i.e. no parents)
	if !hasParent {
		return nil
	}

	// Recursive case
	parentsList := strings.Split(string(parents), ",")
	for _, parent := range parentsList {
		fmt.Printf("  c_%s -> c_%s;\n", objSha, parent)
		err = logGraphviz(repo, parent, seen)
		if err != nil {
			return err
		}
	}
	return nil
}
