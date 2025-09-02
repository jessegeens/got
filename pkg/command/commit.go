package command

import (
	"flag"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/jessegeens/got/pkg/config"
	"github.com/jessegeens/got/pkg/fs"
	"github.com/jessegeens/got/pkg/hashing"
	"github.com/jessegeens/got/pkg/index"
	"github.com/jessegeens/got/pkg/kvlm"
	"github.com/jessegeens/got/pkg/objects"
	"github.com/jessegeens/got/pkg/repository"
)

func CommitCommand() *Command {
	command := newCommand("commit")
	command.Action = func(args []string) error {
		message := flag.String("m", "", "Message to associate with this commit")
		flag.Parse()
		if message == nil || *message == "" {
			message = flag.String("message", "", "Message to associate with this commit")
			flag.Parse()
		}

		repo, err := repository.Find(".")
		if err != nil {
			return err
		}

		_, err = commit(repo, *message)
		return err
	}
	command.Description = func() string { return "Record changes to the repository" }
	return command
}

func commit(repo *repository.Repository, message string) (*hashing.SHA, error) {
	// We ignore errors on purpose, because the user may not have a gitconfig file
	cfg, _ := config.Read()

	idx, err := index.Read(repo)
	if err != nil {
		return nil, err
	}

	tree, err := objects.TreeFromIndex(repo, idx)
	if err != nil {
		return nil, err
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
		file, err := repo.RepositoryFile(true, path.Join("refs/heads", branch))
		if err != nil {
			return commit, err
		}

		err = fs.WriteStringToFile(file, fmt.Sprintf("%s\n", commit.AsString()))

		if err == nil {
			printCommitResult(branch, message, commit)
		}

		return commit, err
	} else {
		// If we are not on a branch, we update HEAD itself
		file, err := repo.RepositoryFile(false, "HEAD")
		if err != nil {
			return commit, err
		}

		err = fs.WriteStringToFile(file, commit.AsString()+"\n")

		return commit, err
	}

}

func createCommit(repo *repository.Repository, tree *hashing.SHA, parent *hashing.SHA, author, message string, timestamp time.Time) (*hashing.SHA, error) {
	data := kvlm.New()

	data.Okv.Set("tree", []byte(tree.AsString()))

	if parent != nil {
		data.Okv.Set("parent", []byte(parent.AsString()))
	}

	message = strings.TrimSpace(message) + "\n"
	data.Message = []byte(message)

	author = fmt.Sprintf("%s %d %s", author, time.Now().Unix(), calculateTimeOffset())

	data.Okv.Set("author", []byte(author))
	data.Okv.Set("committer", []byte(author))

	commit := objects.NewCommit(data)

	return objects.WriteObject(commit, repo)
}

func calculateTimeOffset() string {
	_, offset := time.Now().Zone()
	offsetDuration := time.Duration(float64(offset) * float64(time.Second))
	symbol := "+"
	if offset < 0 {
		symbol = "-"
	}

	hours := int(offsetDuration.Hours())
	minutes := int(offsetDuration.Minutes()) % 60

	timezone := fmt.Sprintf("%s%02d%02d", symbol, hours, minutes)
	return timezone
}

func printCommitResult(branch, message string, commit *hashing.SHA) {
	shortCommit := commit.AsString()[:7]
	fmt.Printf("[%s %s] %s\n", branch, shortCommit, message)
}
