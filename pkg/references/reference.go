package references

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"

	"github.com/jessegeens/go-toolbox/pkg/fs"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

type Reference string

func (r Reference) String() string {
	return string(r)
}

func (r Reference) Resolve(repo *repository.Repository) (string, error) {
	path, err := repo.RepositoryFile(false, r.String())
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		//return "", fmt.Errorf("failed to resolve reference %s: %w", r.String(), err)
		return "", nil
	}
	if bytes.HasPrefix(data, []byte("ref: ")) {
		// We trim the "ref: " and the final "\n"
		newRef := Reference(data[5 : len(data)-1])
		return newRef.Resolve(repo)
	}
	return string(data[:len(data)-1]), nil
}

func List(repo *repository.Repository) (map[Reference]any, error) {
	return list(repo, "refs")
}

func list(repo *repository.Repository, path string) (map[Reference]any, error) {
	path, err := repo.RepositoryDir(false, path)
	if err != nil {
		return nil, err
	}

	mapping := make(map[Reference]any)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, dir := range entries {
		subdir := filepath.Join(path, dir.Name())
		ref := Reference(dir.Name())
		if fs.IsDirectory(subdir) {
			res, err := list(repo, subdir)
			if err != nil {
				return nil, err
			}
			mapping[ref] = res
		} else {
			res, err := ref.Resolve(repo)
			if err != nil {
				return nil, err
			}
			mapping[ref] = res
		}
	}
	return mapping, nil
}
