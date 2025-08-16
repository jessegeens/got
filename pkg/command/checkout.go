package command

import (
	"errors"
	"flag"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jessegeens/go-toolbox/pkg/hashing"
	"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func CheckoutCommand() *Command {
	command := newCommand("checkout")
	command.Action = func(args []string) error {
		commit := *flag.String("commit", "", "The commit or tree to checkout")
		path := *flag.String("path", "", "The empty directory to checkout on")
		repo, err := repository.Find(".")
		if err != nil {
			return err
		}

		commitHash, err := hashing.NewShaFromHex(commit)
		if err != nil {
			return err
		}
		object, err := objects.ReadObject(repo, commitHash)
		if err != nil {
			return err
		}

		finfo, err := os.Stat(path)
		if err == nil {
			// exists, check that it is a directory and is empty
			if !finfo.IsDir() {
				return errors.New("Not a directory: " + path)
			}
			if !isEmptyDirectory(path) {
				return errors.New("Not empty: " + path)
			}
		} else if errors.Is(err, fs.ErrNotExist) {
			err = os.MkdirAll(path, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			return err
		}

		if object.Type() != objects.TypeTree {
			return errors.New("ref should point to a tree")
		}
		tree := object.(*objects.Tree)

		return treeCheckout(repo, tree, path)
	}
	command.Description = func() string { return "Checkout a commit inside of a directory" }
	return command
}

func treeCheckout(repo *repository.Repository, tree *objects.Tree, path string) error {
	for _, item := range tree.Items {
		obj, err := objects.ReadObject(repo, item.Sha)
		if err != nil {
			return err
		}

		dest := filepath.Join(path, item.PrintPath())

		if obj.Type() == objects.TypeTree {
			os.Mkdir(dest, os.ModePerm)
			return treeCheckout(repo, tree, dest)
		} else if obj.Type() == objects.TypeBlob {
			data, err := obj.Serialize()
			if err != nil {
				return err
			}

			f, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer f.Close()
			f.Write(data)
			err = f.Sync()
			if err != nil {
				return err
			}

		}
	}
	return nil
}

func isEmptyDirectory(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	return err == io.EOF
}
