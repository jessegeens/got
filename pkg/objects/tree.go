package objects

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"
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
