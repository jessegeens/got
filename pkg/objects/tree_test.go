package objects

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/jessegeens/go-toolbox/pkg/hashing"
)

func generateFakeHashFromChar(char byte) *hashing.SHA {
	return hashing.NewSHA(bytes.Repeat([]byte{char}, 20))
}

func TestTreeLeaf_PrintSHA(t *testing.T) {
	// Create a sample SHA (20 bytes)
	sha, _ := hashing.NewShaFromHex("e69de29bb2d1d6434b8b29ae775ad8c2e48c5391")
	leaf := &TreeLeaf{
		Sha:  sha,
		Path: []byte("test.txt"),
		Mode: []byte("100644"),
	}

	printed := leaf.PrintSHA()
	if len(printed) != 40 {
		t.Errorf("PrintSHA(): expected length 40, got %d", len(printed))
	}
}

func TestTreeSorting(t *testing.T) {
	// Test cases for tree sorting
	tests := []struct {
		name     string
		input    []*TreeLeaf
		expected []string // expected paths in order
	}{
		{
			name: "mixed files and directories",
			input: []*TreeLeaf{
				{Path: []byte("file.txt"), Mode: []byte("100644"), Sha: generateFakeHashFromChar('a')},
				{Path: []byte("dir"), Mode: []byte("040000"), Sha: generateFakeHashFromChar('b')},
				{Path: []byte("afile.txt"), Mode: []byte("100644"), Sha: generateFakeHashFromChar('c')},
			},
			expected: []string{"afile.txt", "dir", "file.txt"},
		},
		{
			name: "only files",
			input: []*TreeLeaf{
				{Path: []byte("z.txt"), Mode: []byte("100644"), Sha: generateFakeHashFromChar('a')},
				{Path: []byte("a.txt"), Mode: []byte("100644"), Sha: generateFakeHashFromChar('b')},
				{Path: []byte("m.txt"), Mode: []byte("100644"), Sha: generateFakeHashFromChar('c')},
			},
			expected: []string{"a.txt", "m.txt", "z.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := &Tree{Items: tt.input}
			data, err := tree.Serialize()
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			// Deserialize and check order
			newTree := &Tree{}
			err = newTree.Deserialize(data)
			if err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			// Check if paths are in correct order
			var paths []string
			for _, item := range newTree.Items {
				paths = append(paths, string(item.Path))
			}

			if !reflect.DeepEqual(paths, tt.expected) {
				t.Errorf("Got paths %v, want %v", paths, tt.expected)
			}
		})
	}
}

func TestParseLeaf(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		wantMode    []byte
		wantPath    []byte
		wantSHAHex  string
		shouldError bool
	}{
		{
			name:       "regular file",
			input:      []byte("100644 test.txt\x00" + string(bytes.Repeat([]byte{0x01}, 20))),
			wantMode:   []byte("100644"),
			wantPath:   []byte("test.txt"),
			wantSHAHex: "0101010101010101010101010101010101010101",
		},
		{
			name:       "directory",
			input:      []byte("040000 testdir\x00" + string(bytes.Repeat([]byte{0x02}, 20))),
			wantMode:   []byte("040000"),
			wantPath:   []byte("testdir"),
			wantSHAHex: "0202020202020202020202020202020202020202",
		},
		{
			name:        "invalid mode length",
			input:       []byte("1006444 test.txt\x00" + string(bytes.Repeat([]byte{0x01}, 20))),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, leaf, err := parseLeaf(tt.input, 0)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !bytes.Equal(leaf.Mode, tt.wantMode) {
				t.Errorf("Mode = %s, want %s", leaf.Mode, tt.wantMode)
			}
			if !bytes.Equal(leaf.Path, tt.wantPath) {
				t.Errorf("Path = %s, want %s", leaf.Path, tt.wantPath)
			}
			gotSHAHex := leaf.Sha.AsString()
			if gotSHAHex != tt.wantSHAHex {
				t.Errorf("SHA = %s, want %s", gotSHAHex, tt.wantSHAHex)
			}

			expectedPos := len(tt.input)
			if pos != expectedPos {
				t.Errorf("Position = %d, want %d", pos, expectedPos)
			}
		})
	}
}

func TestTree_SerializeDeserialize(t *testing.T) {
	// Create a sample tree
	sha1, _ := hashing.NewShaFromHex("e69de29bb2d1d6434b8b29ae775ad8c2e48c5391")
	sha2, _ := hashing.NewShaFromHex("7601d7f6231db6a45a62a9377c57425eba9623c3")
	originalTree := &Tree{
		Items: []*TreeLeaf{
			{
				Mode: []byte("100644"),
				Path: []byte("file1.txt"),
				Sha:  sha1,
			},
			{
				Mode: []byte("100644"),
				Path: []byte("file2.txt"),
				Sha:  sha2,
			},
		},
	}

	// Serialize
	data, err := originalTree.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Deserialize
	newTree := &Tree{}
	err = newTree.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Compare trees
	if len(originalTree.Items) != len(newTree.Items) {
		t.Fatalf("Tree items length mismatch: got %d, want %d",
			len(newTree.Items), len(originalTree.Items))
	}

	for i, original := range originalTree.Items {
		got := newTree.Items[i]
		if !bytes.Equal(original.Mode, got.Mode) {
			t.Errorf("Item %d Mode = %s, want %s", i, got.Mode, original.Mode)
		}
		if !bytes.Equal(original.Path, got.Path) {
			t.Errorf("Item %d Path = %s, want %s", i, got.Path, original.Path)
		}
		if original.Sha.AsString() != got.Sha.AsString() {
			t.Errorf("Item %d SHA = %x, want %x", i, got.Sha, original.Sha)
		}
	}
}

func TestTree_Type(t *testing.T) {
	tree := &Tree{}
	if tree.Type() != TypeTree {
		t.Errorf("Type() = %v, want %v", tree.Type(), TypeTree)
	}
}

func TestSortingKey(t *testing.T) {
	tests := []struct {
		name     string
		leaf     *TreeLeaf
		expected string
	}{
		{
			name: "regular file",
			leaf: &TreeLeaf{
				Mode: []byte("100644"),
				Path: []byte("file.txt"),
			},
			expected: "file.txt",
		},
		{
			name: "directory",
			leaf: &TreeLeaf{
				Mode: []byte("040000"),
				Path: []byte("dir"),
			},
			expected: "dir/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortingKey(tt.leaf)
			if got != tt.expected {
				t.Errorf("sortingKey() = %v, want %v", got, tt.expected)
			}
		})
	}
}
