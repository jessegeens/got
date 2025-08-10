package command

import (
	"flag"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/jessegeens/go-toolbox/pkg/config"
	"github.com/jessegeens/go-toolbox/pkg/fs"
	"github.com/jessegeens/go-toolbox/pkg/index"
	"github.com/jessegeens/go-toolbox/pkg/kvlm"
	"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func CommitCommand() *Command {
	command := newCommand("commit")
	command.Action = func(args []string) error {
		message := *flag.String("message", "", "Message to associate with this commit")

		repo, err := repository.Find(".")
		if err != nil {
			return err
		}

		_, err = commit(repo, message)
		return err
	}
	command.Description = func() string { return "Record changes to the repository" }
	return command
}

func commit(repo *repository.Repository, message string) (string, error) {
	cfg, err := config.Read()
	if err != nil {
		return "", err
	}

	idx, err := index.Read(repo)
	if err != nil {
		return "", err
	}

	tree, err := objects.TreeFromIndex(repo, idx)
	if err != nil {
		return "", err
	}

	user, ok := cfg.GetUser()
	if !ok {
		user = "DefaultUser"
	}

	// We don't have to find the parent, so we can ignore the error
	parent, _ := objects.Find(repo, "HEAD", objects.TypeNoTypeSpecified, true)

	commit, err := createCommit(repo, tree, parent, user, message, time.Now())
	if err != nil {
		return commit, err
	}

	// Update head so our commit is now the tip of the active branch
	branch, onBranch, err := repo.GetActiveBranch()
	if err != nil {
		return commit, err
	}

	// If we are on a branch, we update refs/heads/branch
	if onBranch {
		file, err := repo.RepositoryFile(false, path.Join("refs/heads", branch))
		if err != nil {
			return commit, err
		}

		err = fs.WriteStringToFile(file, fmt.Sprintf("%s\n", commit))
		return commit, err
	}

	// If we are not on a branch, we update HEAD itself
	file, err := repo.RepositoryFile(false, "HEAD")
	if err != nil {
		return commit, err
	}

	err = fs.WriteStringToFile(file, commit+"\n")
	return commit, err

}

func createCommit(repo *repository.Repository, tree, parent, author, message string, timestamp time.Time) (string, error) {
	data := kvlm.New()

	data.Okv.Set("tree", []byte(tree))

	if parent != "" {
		data.Okv.Set("parent", []byte(parent))
	}

	message = strings.TrimSpace(message) + "\n"
	data.Message = []byte(message)

	// TODO: format time
	// author = author + timestamp

	data.Okv.Set("author", []byte(author))
	data.Okv.Set("committer", []byte(author))

	commit := objects.NewCommit(data)

	return objects.WriteObject(commit, repo)
}
