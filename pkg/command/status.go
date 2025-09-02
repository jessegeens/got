package command

import (
	"fmt"
	iofs "io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jessegeens/got/pkg/fs"
	"github.com/jessegeens/got/pkg/ignore"
	"github.com/jessegeens/got/pkg/index"
	"github.com/jessegeens/got/pkg/objects"
	"github.com/jessegeens/got/pkg/repository"
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
		return statusIndexWorktree(repo, idx)
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
		} else {
			fmt.Printf("HEAD detached at %s\n", obj)
		}
	}
	return nil
}

// We compare HEAD to the index
func statusHeadIndex(repo *repository.Repository, idx *index.Index) error {
	head, err := objects.MapFromTree(repo, "HEAD")
	if err != nil {
		fmt.Printf("No commits yet\n")
		//return err
	}

	for _, entry := range idx.Entries {
		if sha, ok := head[entry.Name]; ok {
			if sha.AsString() != entry.SHA.AsString() {
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
	ignore, err := ignore.Read(repo)
	if err != nil {
		return err
	}

	// We begin by walking the filesystem
	gitDirPrefix := repo.GitDir() + string(os.PathSeparator)
	allFiles := []string{}
	err = filepath.WalkDir(repo.WorkTree(), func(path string, d iofs.DirEntry, e error) error {
		// Skip whatever is in .git
		if strings.HasPrefix(path, gitDirPrefix) {
			return nil
		}

		relativePath, err := filepath.Rel(repo.WorkTree(), path)
		if err != nil {
			return err
		}
		allFiles = append(allFiles, relativePath)
		return nil
	})
	if err != nil {
		return err
	}

	printed := false

	// Now we traverse the index and compare real files with the cached versions
	for _, entry := range idx.Entries {
		fullPath := path.Join(repo.WorkTree(), entry.Name)
		if !fs.Exists(fullPath) {
			if !printed {
				fmt.Println("\nChanges not staged for commit:")
				printed = true
			}
			fmt.Printf("  deleted: %s\n", entry.Name)
		} else {
			finfo, err := os.Stat(fullPath)
			if err != nil {
				return err
			}

			if finfo.ModTime() != entry.MTime {
				// Let's do a deep compare
				content, err := os.ReadFile(fullPath)
				if err != nil {
					return err
				}
				newSha, err := objects.ObjectHash(content, objects.TypeBlob, repo)
				if err != nil {
					return err
				}

				if newSha != entry.SHA {
					if !printed {
						fmt.Println("\nChanges not staged for commit:")
						printed = true
					}
					fmt.Printf("  modified: %s\n", entry.Name)
				}
			}
		}
		allFiles, _ = deleteFromSlice(allFiles, entry.Name)
	}

	// Everything that's left in allFiles was not found in the index,
	// so those files are not tracked
	fmt.Println("\nUntracked files:")
	for _, file := range allFiles {
		if !ignore.ShouldBeIgnored(file) {
			fmt.Printf("  %s\n", file)
		}
	}

	return nil
}

// Delete first occurence of entry from slice, if it exists
func deleteFromSlice[K comparable](slice []K, elem K) ([]K, bool) {
	for idx, arrElem := range slice {
		if arrElem == elem {
			return append(slice[:idx], slice[idx+1:]...), true
		}
	}
	return slice, false
}
