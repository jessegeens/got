package command

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jessegeens/got/pkg/fs"
	"github.com/jessegeens/got/pkg/index"
	"github.com/jessegeens/got/pkg/objects"
	"github.com/jessegeens/got/pkg/repository"
)

func AddCommand() *Command {
	command := newCommand("add")
	command.Action = func(args []string) error {
		if len(args) < 1 {
			return errors.New("must specify a path to add")
		}
		//paths := args[0]
		repo, err := repository.Find(".")
		if err != nil {
			return err
		}

		for _, path := range extractPathsFromArgs(args) {
			err = add(repo, path, true)
			if err != nil {
				return err
			}
		}

		return nil
	}
	command.Description = func() string { return "Add files contents to the index" }
	return command
}

func add(repo *repository.Repository, addPath string, delete bool) error {
	idx, err := index.Read(repo)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(addPath)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(absPath, repo.WorkTree()) {
		return errors.New("cannot add a path outside the worktree")
	}

	// First remove all paths from the index, if they exist
	rm(repo, addPath, false)

	paths := []string{}
	if fs.IsDirectory(absPath) {
		filepath.WalkDir(absPath, func(path string, d iofs.DirEntry, err error) error {
			// Skip whatever is in .git
			if strings.HasPrefix(path, repo.GitDir()) {
				return nil
			}

			if !d.IsDir() {
				paths = append(paths, path)
			}
			return nil
		})
	} else {
		paths = append(paths, absPath)
	}

	for _, p := range paths {
		relPath, err := filepath.Rel(repo.WorkTree(), absPath)
		if err != nil {
			return err
		}

		fileContents, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("failed to read %s: %s", p, err.Error())
		}
		sha, err := objects.ObjectHash(fileContents, objects.TypeBlob, repo)
		if err != nil {
			return fmt.Errorf("failed to hash object: %s", err.Error())
		}

		var stat syscall.Stat_t
		err = syscall.Stat(addPath, &stat)
		if err != nil {
			return err
		}

		ctime := time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec)

		mtime := time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec)

		entry := &index.Entry{
			CTime:           ctime,
			MTime:           mtime,
			Dev:             uint32(stat.Dev),
			Inode:           uint32(stat.Ino),
			SHA:             sha,
			ModeType:        index.ModeTypeRegular,
			ModePerms:       0o644,
			UID:             stat.Uid,
			GID:             stat.Gid,
			Size:            uint32(stat.Size),
			FlagAssumeValid: false,
			FlagStage:       0,
			Name:            relPath,
		}

		idx.Entries = appendIfUnique(idx.Entries, entry)
	}

	return idx.Write(repo)
}

func extractPathsFromArgs(args []string) []string {
	if args[0] == "." {
		paths := make([]string, 0)
		entries, _ := os.ReadDir("./")
		for _, entry := range entries {
			paths = append(paths, entry.Name())
		}
		return paths
	}

	return args
}

// Only append if the value is not there yet
func appendIfUnique(entries []*index.Entry, entry *index.Entry) []*index.Entry {
	for i, e := range entries {
		// Same file with same contents
		// No need to change anything
		if e.SHA.AsString() == entry.SHA.AsString() {
			return entries
		}
		// Same file with different entries
		// => the old version should not be written to the index
		// so we remove it
		if e.Inode == entry.Inode && e.Dev == entry.Dev {
			entries = append(entries[:i], entries[i+1:]...)
			return append(entries, entry)
		}
	}
	// Actually new file, we just append
	return append(entries, entry)
}
