package objects

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/jessegeens/go-toolbox/pkg/fs"
	"github.com/jessegeens/go-toolbox/pkg/index"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

// Trees look like this when serialized:
// [mode] space [path] 0x00 [sha-1]
// e.g.
// 100644 e69de29bb2d1d6434b8b29ae775ad8c2e48c5391 0x00 file1.txt
// 100644 7601d7f6231db6a45a62a9377c57425eba9623c3 0x00 file2.txt
// 040000 4b825dc642cb6eb9a060e54bf8d69288fbee4904 0x00 subdirectory
type TreeLeaf struct {
	Sha  []byte
	Path []byte
	Mode []byte
}

type Tree struct {
	Items []*TreeLeaf
}

func (t *Tree) Serialize() ([]byte, error) {
	// First we sort the leaves
	sort.Slice(t.Items, func(i, j int) bool {
		return sortingKey(t.Items[i]) < sortingKey(t.Items[j])
	})

	data := []byte{}
	for _, leaf := range t.Items {
		data = append(data, leaf.Mode...)
		data = append(data, ' ')
		data = append(data, leaf.Path...)
		data = append(data, '\x00')
		data = append(data, leaf.Sha...)

	}
	return data, nil
}

func (t *Tree) Deserialize(data []byte) error {
	items, err := parseTree(data)
	t.Items = items
	return err
}

func (t *Tree) Type() GitObjectType {
	return TypeTree
}

func (l *TreeLeaf) PrintSHA() string {
	rawSha := binary.BigEndian.Uint64(l.Sha)

	// Convert to hex, with left padding to 40 chars if necessary
	return fmt.Sprintf("%40x", int64(rawSha))
}

func (l *TreeLeaf) PrintPath() string {
	return string(l.Path)
}

func parseTree(data []byte) ([]*TreeLeaf, error) {
	pos := 0
	max := len(data)
	list := []*TreeLeaf{}

	var err error
	var leaf *TreeLeaf

	for pos < max {
		pos, leaf, err = parseLeaf(data, pos)
		if err != nil {
			return nil, err
		}
		list = append(list, leaf)
	}

	return list, nil
}

// Return the new position, a TreeLeaf and any eventual errors
func parseLeaf(data []byte, start int) (int, *TreeLeaf, error) {
	// Find the space terminator of the mode
	spaceTermLoc := bytes.IndexByte(data[start:], ' ')

	// Mode should be 5 or 6 bytes
	if !(spaceTermLoc-start == 5 || spaceTermLoc-start == 6) {
		return 0, nil, errors.New("invalid mode length")
	}

	// Normalize to six bytes
	mode := data[start:spaceTermLoc]
	if len(mode) == 5 {
		mode = append([]byte{'0'}, mode...)
	}

	// Now we find the NULL terminator of the path
	nullTermLoc := bytes.IndexByte(data[spaceTermLoc:], '\x00')

	// Now we can read the path
	path := data[spaceTermLoc+1 : nullTermLoc]

	// And then we read the SHA, which has length 40 (which is equal to 20 bytes)
	rawSha := data[nullTermLoc+1 : nullTermLoc+21]

	return nullTermLoc + 21, &TreeLeaf{rawSha, path, mode}, nil
}

// Sorting matters for trees, because the order of the tree determines its hash
// Git sorts by file name, with a '/' added to paths of subdirectories
// This function returns the sorting key of a specific leaf
func sortingKey(leaf *TreeLeaf) string {
	if strings.HasPrefix(string(leaf.Mode), "10") {
		return string(leaf.Path)
	}
	return string(append(leaf.Path, '/'))
}

// Given a repository and a reference to a tree object, return the tree
// in the form of a map, where the keys are full paths and tha values are SHAs
func MapFromTree(repo *repository.Repository, treeRef string) (map[string]string, error) {
	return mapFromTree(repo, treeRef, "")
}

func mapFromTree(repo *repository.Repository, treeRef string, pathPrefix string) (map[string]string, error) {
	ret := make(map[string]string)

	treeSha, err := Find(repo, treeRef, TypeNoTypeSpecified, true)
	if err != nil {
		return nil, err
	}

	obj, err := ReadObject(repo, treeSha)
	if err != nil {
		return nil, err
	}
	tree, ok := obj.(*Tree)
	if !ok {
		return nil, errors.New("passed reference " + treeSha + " does not correspond to a tree, but is a " + obj.Type().String())
	}

	for _, leaf := range tree.Items {
		fullPath := path.Join(pathPrefix, string(leaf.Path))

		// If the path is a directory, (i.e. the child is another tree), we recurse
		// Otherwise, we set the SHA
		if fs.IsDirectory(fullPath) {
			res, err := mapFromTree(repo, string(leaf.Sha), fullPath)
			if err != nil {
				return nil, err
			}
			maps.Copy(ret, res)
		} else {
			ret[fullPath] = string(leaf.Sha)
		}
	}

	return ret, nil

}

func TreeFromIndex(repo *repository.Repository, idx *index.Index) (string, error) {
	return treeFromIndex(repo, idx)
}

func treeFromIndex(repo *repository.Repository, idx *index.Index) (string, error) {
	contents := make(map[string][]*index.Entry)
	var err error

	for _, e := range idx.Entries {
		dirname := filepath.Dir(e.Name)
		contents[dirname] = append(contents[dirname], e)
	}

	// We sort reversed by length, so that we always come across an element
	// before we come across its parent (i.e. the longest elements come first in the list)
	paths := slices.Collect(maps.Keys(contents))
	sort.Slice(paths, func(i, j int) bool {
		l1, l2 := len(paths[i]), len(paths[j])

		if l1 != l2 {
			return l1 > l2
		}
		return paths[i] > paths[j]
	})

	currentSha := ""
	enc := binary.BigEndian

	for _, p := range paths {
		tree := Tree{
			Items: []*TreeLeaf{},
		}

		for _, e := range contents[p] {
			var modeBytes []byte
			enc.PutUint16(modeBytes, uint16(index.ModeTypeRegular))
			leaf := TreeLeaf{
				Mode: modeBytes,
				Sha:  []byte(e.SHA),
				Path: []byte(filepath.Base(e.Name)),
			}

			tree.Items = append(tree.Items, &leaf)
		}

		currentSha, err = WriteObject(GitObject(&tree), repo)
		if err != nil {
			return "", err
		}

	}

	return currentSha, nil

}
