package repository

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jessegeens/go-toolbox/pkg/fs"
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
			return nil, fmt.Errorf("failed to read repository configuration: %s", err.Error())
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
	if fs.PathExists(repo.worktree) {
		if !fs.IsDirectory(repo.worktree) {
			return nil, errors.New("not a directory: " + repo.worktree)
		}

		if fs.IsDirectory(repo.gitdir) && !fs.IsEmptyDirectory(repo.gitdir) {
			return nil, errors.New("gitdir does not seem to be empty: " + repo.gitdir)
		}
	} else {
		err := os.MkdirAll(repo.worktree, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	repositorySubDirectories := [][]string{{"branches"}, {"objects"}, {"refs", "tags"}, {"refs", "heads"}}
	for _, subDirectoryList := range repositorySubDirectories {
		_, err := repo.RepositoryDir(true, subDirectoryList...)
		if err != nil {
			return nil, errors.New("Failed to create repositorySubDirectory: " + err.Error())
		}
	}

	repoFile, _ := repo.RepositoryFile(true, "description")
	err := fs.WriteStringToFile(repoFile, "Unnamed repository; edit this file 'description' to name the repository.\n")
	if err != nil {
		return nil, errors.New("Failed to create repository description: " + err.Error())
	}

	repoFile, _ = repo.RepositoryFile(true, "HEAD")
	err = fs.WriteStringToFile(repoFile, "ref: refs/heads/master\n")
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
		return nil, err
	}

	if fs.IsDirectory(path.Join(realPath, ".git")) {
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
			err = os.MkdirAll(path, os.ModePerm)
			return path, err
		}
		return "", errors.New("path does not exist and create = false")
	}
}

// Returns the branch name if we're on a branch, whether we're on a branch,
// and any eventual errors
func (r *Repository) GetActiveBranch() (string, bool, error) {
	headFile, err := r.RepositoryFile(false, "HEAD")
	if err != nil {
		return "", false, err
	}
	head, err := os.ReadFile(headFile)
	if err != nil {
		return "", false, err
	}

	if strings.HasPrefix(string(head), "ref: refs/heads/") {
		return string(head[16:]), true, nil
	}
	return "", false, nil
}

func (r *Repository) WorkTree() string {
	sep := string(os.PathSeparator)
	if strings.HasSuffix(r.worktree, sep) {
		return r.worktree
	}
	return r.worktree + sep
}

func (r *Repository) GitDir() string {
	return r.gitdir
}

func defaultRepositoryConfig() *ini.File {
	cfg := ini.Empty()
	cfg.NewSection("core")
	cfg.Section("core").NewKey("repositoryformatversion", "0")
	cfg.Section("core").NewKey("filemode", "true")
	cfg.Section("core").NewKey("bare", "false")

	return cfg
}
