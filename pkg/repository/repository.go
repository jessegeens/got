package repository

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/ini.v1"
)

type Repository struct {
	worktree string
	gitdir   string
}

// Constructor
func New(repositoryPath string, disableChecks bool) (*Repository, error) {
	worktree := repositoryPath
	gitdir := path.Join(repositoryPath, ".git")

	if !disableChecks {
		if _, err := os.Stat(gitdir); os.IsNotExist(err) {
			return nil, errors.New("not a git repository " + repositoryPath)
		}

		cfg, err := ini.Load(path.Join(gitdir, "config"))
		if err != nil {
			return nil, errors.New("failed to read repository configuration")
		}
		if cfg.Section("core").Key("repositoryformatversion").MustInt(0) != 0 {
			return nil, errors.New("wrong repositoryformatversion")
		}
	}

	return &Repository{
		worktree: worktree,
		gitdir:   gitdir,
	}, nil
}

// Create repository on filesystem
func Create(repositoryPath string) (*Repository, error) {
	repo, _ := New(repositoryPath, true)

	// Make sure path doesn't exist or that it is an empty dir
	if PathExists(repo.worktree) {
		if !IsDirectory(repo.worktree) {
			return nil, errors.New("not a directory: " + repo.worktree)
		}

		if IsDirectory(repo.gitdir) && !IsEmptyDirectory(repo.gitdir) {
			return nil, errors.New("gitdir does not seem to be empty: " + repo.gitdir)
		}
	} else {
		os.MkdirAll(repo.worktree, os.ModePerm)
	}

	repositorySubDirectories := [][]string{{"branches"}, {"objects"}, {"refs", "tags"}, {"refs", "heads"}}
	for _, subDirectoryList := range repositorySubDirectories {
		_, err := repo.RepositoryDir(true, subDirectoryList...)
		if err != nil {
			return nil, errors.New("Failed to create repositorySubDirectory: " + err.Error())
		}
	}

	repoFile, _ := repo.RepositoryFile(true, "description")
	err := WriteStringToFile(repoFile, "Unnamed repository; edit this file 'description' to name the repository.\n")
	if err != nil {
		return nil, errors.New("Failed to create repository description: " + err.Error())
	}

	repoFile, _ = repo.RepositoryFile(true, "HEAD")
	err = WriteStringToFile(repoFile, "ref: refs/heads/master\n")
	if err != nil {
		return nil, errors.New("Failed to create repository HEAD: " + err.Error())
	}

	repoFile, _ = repo.RepositoryFile(true, "config")
	config := defaultRepositoryConfig()
	config.SaveTo(repoFile)

	fmt.Println("Initialized new empty git repository")

	return repo, nil
}

// Locate the root of a git repo among the parent directories
func Find(childPath string) (*Repository, error) {
	realPath, err := filepath.Abs(childPath)
	if err != nil {
		// handle err
	}

	if IsDirectory(path.Join(realPath, ".git")) {
		return New(realPath, false)
	}

	parent := path.Join(realPath, "..")
	// base case, if parent == child then we are in /
	if parent == realPath {
		return nil, errors.New("not a git directory")
	}

	return Find(parent)
}

// Compute path under repo's gitdir
func (r *Repository) RepositoryPath(paths ...string) string {
	return path.Join(append([]string{r.gitdir}, paths...)...)
}

// Same as RepositoryPath, but create directory / file if absent
func (r *Repository) RepositoryFile(create bool, paths ...string) (string, error) {
	_, err := r.RepositoryDir(create, paths[:len(paths)-1]...)
	if err == nil {
		return r.RepositoryPath(paths...), nil
	}
	return "", err
}

// Same as RepositoryPath, but create directory if absent
func (r *Repository) RepositoryDir(create bool, paths ...string) (string, error) {
	path := r.RepositoryPath(paths...)
	fileInfo, err := os.Stat(path)
	// path exists
	if !errors.Is(err, os.ErrNotExist) {
		if fileInfo.IsDir() {
			return path, nil
		} else {
			return "", errors.New("not a directory " + path)
		}
	} else { // path does not exist
		if create {
			os.MkdirAll(path, os.ModePerm)
			return path, nil
		}
		return "", errors.New("path does not exist and create = false")
	}
}

func defaultRepositoryConfig() *ini.File {
	cfg := ini.Empty()
	cfg.NewSection("core")
	cfg.Section("core").NewKey("repositoryformatversion", "0")
	cfg.Section("core").NewBooleanKey("filemode")
	cfg.Section("core").Key("filemode").SetValue("false")
	cfg.Section("core").NewBooleanKey("bare")
	cfg.Section("core").Key("bare").SetValue("false")

	return cfg
}
