package objects

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jessegeens/got/pkg/hashing"
	"github.com/jessegeens/got/pkg/index"
	"github.com/jessegeens/got/pkg/repository"
)

func generateFakeHashFromChar(char byte) *hashing.SHA {
	return hashing.NewSHA(bytes.Repeat([]byte{char}, 20))
}

func setupTreeTestRepo(t *testing.T) *repository.Repository {
	tempDir, err := os.MkdirTemp("", "got-treefromindex-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	repo, err := repository.Create(tempDir)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	return repo
}

func cleanupTreeTestRepo(t *testing.T, repo *repository.Repository) {
	if err := os.RemoveAll(repo.WorkTree()); err != nil {
		t.Logf("Warning: failed to clean up test repository: %v", err)
	}
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
	sha1 := hashing.NewShaFromBytes(bytes.Repeat([]byte{'a'}, 20))
	sha2 := hashing.NewShaFromBytes(bytes.Repeat([]byte{'b'}, 20))
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
			input:      append([]byte("100644 test.txt\x00"), sha1.AsBytes()...),
			wantMode:   []byte("100644"),
			wantPath:   []byte("test.txt"),
			wantSHAHex: sha1.AsString(),
		},
		{
			name:       "directory",
			input:      append([]byte("040000 testdir\x00"), sha2.AsBytes()...),
			wantMode:   []byte("040000"),
			wantPath:   []byte("testdir"),
			wantSHAHex: sha2.AsString(),
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
			t.Errorf("Item %d SHA = %x, want %x", i, got.Sha.AsString(), original.Sha.AsString())
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

func TestTreeFromIndex_SingleFile(t *testing.T) {
	repo := setupTreeTestRepo(t)
	defer cleanupTreeTestRepo(t, repo)

	// Create a blob object for the file content
	blob := &Blob{data: []byte("hello from got")}
	blobSha, err := WriteObject(blob, repo)
	if err != nil {
		t.Fatalf("Failed to write blob: %v", err)
	}

	// Build an index with a single entry at repository root
	idx := index.New([]*index.Entry{})
	idx.Entries = append(idx.Entries, &index.Entry{
		CTime:           time.Now(),
		MTime:           time.Now(),
		Dev:             0,
		Inode:           0,
		ModeType:        index.ModeTypeRegular,
		ModePerms:       0o644,
		UID:             0,
		GID:             0,
		Size:            0,
		SHA:             blobSha,
		FlagAssumeValid: false,
		FlagStage:       0,
		Name:            "test.txt",
	})

	// Generate a tree from the index
	treeSha, err := TreeFromIndex(repo, idx)
	if err != nil {
		t.Fatalf("TreeFromIndex() error = %v", err)
	}
	if treeSha == nil {
		t.Fatal("TreeFromIndex() returned nil SHA")
	}

	// Read the tree object back and validate its contents
	obj, err := ReadObject(repo, treeSha)
	if err != nil {
		t.Fatalf("Failed to read tree object: %v", err)
	}
	tree, ok := obj.(*Tree)
	if !ok {
		t.Fatalf("Expected object type Tree, got %T", obj)
	}

	if len(tree.Items) != 1 {
		t.Fatalf("Expected 1 tree item, got %d", len(tree.Items))
	}
	item := tree.Items[0]
	if string(item.Path) != filepath.Base("test.txt") {
		t.Errorf("Tree item path = %q, want %q", string(item.Path), "test.txt")
	}
	if item.Sha.AsString() != blobSha.AsString() {
		t.Errorf("Tree item sha = %s, want %s", item.Sha.AsString(), blobSha.AsString())
	}
}

func TestTreeFromIndex_MultipleFiles_SingleDirectory(t *testing.T) {
	repo := setupTreeTestRepo(t)
	defer cleanupTreeTestRepo(t, repo)

	// Create three blobs for three files
	blobA := &Blob{data: []byte("alpha")}
	shaA, err := WriteObject(blobA, repo)
	if err != nil {
		t.Fatalf("Failed to write blob A: %v", err)
	}
	blobB := &Blob{data: []byte("bravo")}
	shaB, err := WriteObject(blobB, repo)
	if err != nil {
		t.Fatalf("Failed to write blob B: %v", err)
	}
	blobC := &Blob{data: []byte("charlie")}
	shaC, err := WriteObject(blobC, repo)
	if err != nil {
		t.Fatalf("Failed to write blob C: %v", err)
	}

	// Build an index with three entries in the repository root
	idx := index.New([]*index.Entry{})
	add := func(name string, sha interface{}) {
		var s = shaA
		switch name {
		case "a.txt":
			s = shaA
		case "b.txt":
			s = shaB
		case "c.txt":
			s = shaC
		}
		idx.Entries = append(idx.Entries, &index.Entry{
			CTime:           time.Now(),
			MTime:           time.Now(),
			ModeType:        index.ModeTypeRegular,
			ModePerms:       0o644,
			UID:             0,
			GID:             0,
			Size:            0,
			SHA:             s,
			FlagAssumeValid: false,
			FlagStage:       0,
			Name:            name,
		})
	}
	add("a.txt", shaA)
	add("b.txt", shaB)
	add("c.txt", shaC)

	// Generate a tree from the index
	treeSha, err := TreeFromIndex(repo, idx)
	if err != nil {
		t.Fatalf("TreeFromIndex() error = %v", err)
	}
	if treeSha == nil {
		t.Fatal("TreeFromIndex() returned nil SHA")
	}

	// Read and validate the tree
	obj, err := ReadObject(repo, treeSha)
	if err != nil {
		t.Fatalf("Failed to read tree object: %v", err)
	}
	tree, ok := obj.(*Tree)
	if !ok {
		t.Fatalf("Expected object type Tree, got %T", obj)
	}

	if len(tree.Items) != 3 {
		t.Fatalf("Expected 3 tree items, got %d", len(tree.Items))
	}

	// Build a map for easy lookup and verify all files exist with correct SHAs
	found := map[string]string{}
	for _, it := range tree.Items {
		found[string(it.Path)] = it.Sha.AsString()
	}

	if found["a.txt"] != shaA.AsString() {
		t.Errorf("a.txt sha = %s, want %s", found["a.txt"], shaA.AsString())
	}
	if found["b.txt"] != shaB.AsString() {
		t.Errorf("b.txt sha = %s, want %s", found["b.txt"], shaB.AsString())
	}
	if found["c.txt"] != shaC.AsString() {
		t.Errorf("c.txt sha = %s, want %s", found["c.txt"], shaC.AsString())
	}
}
