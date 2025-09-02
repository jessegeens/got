package objects

import (
	"bytes"
	"errors"
	"maps"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/jessegeens/got/pkg/fs"
	"github.com/jessegeens/got/pkg/hashing"
	"github.com/jessegeens/got/pkg/index"
	"github.com/jessegeens/got/pkg/repository"
)

// Trees look like this when serialized:
// [mode] space [path] 0x00 [sha-1]
// e.g.
// 100644 file1.txt 	0x00 e69de29bb2d1d6434b8b29ae775ad8c2e48c5391
// 100644 file2.txt 	0x00 7601d7f6231db6a45a62a9377c57425eba9623c3
// 040000 subdirectory 	0x00 4b825dc642cb6eb9a060e54bf8d69288fbee4904

// [mode] is up to six bytes and is an octal representation of a file mode, stored in ASCII.
// It’s followed by 0x20, an ASCII space;
// Followed by the null-terminated (0x00) path;
// Followed by the object’s SHA-1 in binary encoding, on 20 bytes.

type TreeLeaf struct {
	Sha  *hashing.SHA
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
		data = append(data, 0x00)
		data = append(data, leaf.Sha.AsBytes()...)
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
	return l.Sha.AsString()
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
func parseLeaf(databuffer []byte, start int) (int, *TreeLeaf, error) {
	data := databuffer[start:]

	// Find the space terminator of the mode
	spaceTermLoc := bytes.IndexByte(data, ' ')

	// Mode should be 5 or 6 bytes
	if !(spaceTermLoc == 5 || spaceTermLoc == 6) {
		return 0, nil, errors.New("invalid mode length: " + strconv.Itoa(spaceTermLoc))
	}

	// Normalize to six bytes
	mode := data[:spaceTermLoc]
	if len(mode) == 5 {
		mode = append([]byte{'0'}, mode...)
	}

	// Now we find the NULL terminator of the path
	nullTermLoc := bytes.IndexByte(data, 0x00)

	// Now we can read the path
	path := data[spaceTermLoc+1 : nullTermLoc]

	// And then we read the SHA, which in bytes hash length 20
	rawSha := data[nullTermLoc+1 : nullTermLoc+21]
	sha := hashing.NewShaFromBytes(rawSha)

	nextLeafLocation := start + nullTermLoc + 21
	return nextLeafLocation, &TreeLeaf{sha, path, mode}, nil
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
func MapFromTree(repo *repository.Repository, treeRef string) (map[string]*hashing.SHA, error) {
	return mapFromTree(repo, treeRef, "")
}

func mapFromTree(repo *repository.Repository, treeRef string, pathPrefix string) (map[string]*hashing.SHA, error) {
	ret := make(map[string]*hashing.SHA)

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
		return nil, errors.New("passed reference " + treeSha.AsString() + " does not correspond to a tree, but is a " + obj.Type().String())
	}

	for _, leaf := range tree.Items {
		fullPath := path.Join(pathPrefix, string(leaf.Path))

		// If the path is a directory, (i.e. the child is another tree), we recurse
		// Otherwise, we set the SHA
		if fs.IsDirectory(fullPath) {
			res, err := mapFromTree(repo, leaf.Sha.AsString(), fullPath)
			if err != nil {
				return nil, err
			}
			maps.Copy(ret, res)
		} else {
			ret[fullPath] = leaf.Sha
		}
	}

	return ret, nil

}

func TreeFromIndex(repo *repository.Repository, idx *index.Index) (*hashing.SHA, error) {
	return treeFromIndex(repo, idx)
}

func treeFromIndex(repo *repository.Repository, idx *index.Index) (*hashing.SHA, error) {
	contents := make(map[string][]*index.Entry)

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

	// This will contain the current tree's SHA
	// In our iteration, the root will come last,
	// so we will end up with the root tree's SHA

	// We initialize to the hash of an empty tree
	// $ git hash-object -t tree /dev/null
	// 4b825dc642cb6eb9a060e54bf8d69288fbee4904
	// See https://floatingoctothorpe.uk/2017/empty-trees-in-git.html
	currentSha, _ := hashing.NewShaFromHex("4b825dc642cb6eb9a060e54bf8d69288fbee4904")

	for _, p := range paths {
		tree := Tree{
			Items: []*TreeLeaf{},
		}

		for _, e := range contents[p] {
			leaf := TreeLeaf{
				Mode: index.ModeTypeRegular.Octal(),
				Sha:  e.SHA,
				Path: []byte(filepath.Base(e.Name)),
			}

			tree.Items = append(tree.Items, &leaf)
		}

		sha, err := WriteObject(GitObject(&tree), repo)
		if err != nil {
			return nil, err
		}
		currentSha = sha
	}

	return currentSha, nil

}
