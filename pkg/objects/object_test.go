package objects

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jessegeens/go-toolbox/pkg/index"
	"github.com/jessegeens/go-toolbox/pkg/kvlm"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func setupTestRepo(t *testing.T) *repository.Repository {
	// Create a temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "got-test-repo-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create a new repository
	repo, err := repository.Create(tempDir)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	return repo
}

func cleanupTestRepo(t *testing.T, repo *repository.Repository) {
	if err := os.RemoveAll(repo.WorkTree()); err != nil {
		t.Logf("Warning: failed to clean up test repository: %v", err)
	}
}

func TestParseType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    GitObjectType
		wantErr bool
	}{
		{"commit", "commit", TypeCommit, false},
		{"tree", "tree", TypeTree, false},
		{"tag", "tag", TypeTag, false},
		{"blob", "blob", TypeBlob, false},
		{"invalid", "invalid", TypeNoTypeSpecified, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseType(tt.input)
			if (err != nil) && !tt.wantErr {
				t.Errorf("ParseType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncode(t *testing.T) {
	// Create a simple blob for testing
	blob := &Blob{data: []byte("test content")}

	encoded, err := Encode(blob)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Verify header format: "blob 12\x00test content"
	expectedHeader := []byte("blob 12\x00")
	if !bytes.HasPrefix(encoded, expectedHeader) {
		t.Errorf("Encode() header = %q, want prefix %q", encoded, expectedHeader)
	}

	// Verify content
	expectedContent := []byte("test content")
	if !bytes.Contains(encoded, expectedContent) {
		t.Errorf("Encode() content = %q, want %q", encoded, expectedContent)
	}
}

func TestObjectHash(t *testing.T) {
	repo := setupTestRepo(t)
	defer cleanupTestRepo(t, repo)

	content := []byte("test content")

	hash, err := ObjectHash(content, TypeBlob, repo)
	if err != nil {
		t.Fatalf("ObjectHash() error = %v", err)
	}

	// Verify hex hash is 40 characters long (SHA-1 hex)
	if len(hash.AsString()) != 40 {
		t.Errorf("ObjectHash() string length = %d, want 40", len(hash.AsString()))
	}

	// Verify hash is valid hex
	if _, err := hex.DecodeString(hash.AsString()); err != nil {
		t.Errorf("ObjectHash() = %v, want valid hex", hash)
	}

	// Verify object was written to the correct location
	objPath := filepath.Join(repo.GitDir(), "objects", hash.AsString()[:2], hash.AsString()[2:])
	if _, err := os.Stat(objPath); os.IsNotExist(err) {
		t.Errorf("Object file not created at %s", objPath)
	}
}

func TestGitObjectType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  GitObjectType
		want string
	}{
		{"commit", TypeCommit, "commit"},
		{"tree", TypeTree, "tree"},
		{"tag", TypeTag, "tag"},
		{"blob", TypeBlob, "blob"},
		{"none", TypeNoTypeSpecified, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("GitObjectType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	repo := setupTestRepo(t)
	defer cleanupTestRepo(t, repo)

	// Create and write a test blob
	blob := &Blob{data: []byte("test content")}
	hash, err := WriteObject(blob, repo)
	if err != nil {
		t.Fatalf("Failed to write test blob: %v", err)
	}

	// Create and write a test commit that points to our blob
	data := kvlm.New()

	idx, err := index.Read(repo)
	if err != nil {
		t.FailNow()
	}
	tree, err := TreeFromIndex(repo, idx)
	if err != nil {
		t.FailNow()
	}
	data.Okv.Set("tree", []byte(tree))
	data.Message = []byte("my commit message")
	data.Okv.Set("author", []byte("jesse"))
	data.Okv.Set("committer", []byte("jesse"))
	commit := NewCommit(data)

	commitHash, err := WriteObject(commit, repo)
	if err != nil {
		t.Fatalf("Failed to write test commit: %v", err)
	}

	// Update HEAD to point to our commit
	headPath := filepath.Join(repo.GitDir(), "HEAD")
	if err := os.WriteFile(headPath, []byte(commitHash.AsString()+"\n"), 0644); err != nil {
		t.Fatalf("Failed to update HEAD: %v", err)
	}

	// Create a branch reference
	branchPath := filepath.Join(repo.GitDir(), "refs", "heads", "master")
	if err := os.MkdirAll(filepath.Dir(branchPath), 0755); err != nil {
		t.Fatalf("Failed to create branch directory: %v", err)
	}
	if err := os.WriteFile(branchPath, []byte(commitHash.AsString()+"\n"), 0644); err != nil {
		t.Fatalf("Failed to create branch reference: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty name", "", true},
		{"HEAD", "HEAD", false},
		{"master", "master", false},
		{"short hash", hash.AsString()[:4], false},
		{"full hash", hash.AsString(), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shas, err := Resolve(repo, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(shas) == 0 {
				t.Error("Resolve() returned empty shas when it shouldn't")
			}
		})
	}
}

func TestFind(t *testing.T) {
	repo := setupTestRepo(t)
	defer cleanupTestRepo(t, repo)

	// Create and write a test blob
	blob := &Blob{data: []byte("test content")}
	blobHash, err := WriteObject(blob, repo)
	if err != nil {
		t.Fatalf("Failed to write test blob: %v", err)
	}

	//Create and write a test tree that points to our blob
	tree := &Tree{
		Items: []*TreeLeaf{
			{
				Mode: []byte("100644"),
				Path: []byte("test.txt"),
				Sha:  blobHash,
			},
		},
	}
	treeHash, err := WriteObject(tree, repo)
	if err != nil {
		t.Fatalf("Failed to write test tree: %v", err)
	}

	// Now we write this to the index (a la `git add`)
	idx, err := index.Read(repo)
	if err != nil {
		t.FailNow()
	}

	entry := &index.Entry{
		CTime:           time.Now(),
		MTime:           time.Now(),
		Dev:             uint32(123),
		Inode:           uint32(456),
		SHA:             blobHash,
		ModeType:        index.ModeTypeRegular,
		ModePerms:       0o644,
		UID:             0,
		GID:             0,
		Size:            uint32(0),
		FlagAssumeValid: false,
		FlagStage:       0,
		Name:            "test.txt",
	}
	idx.Entries = append(idx.Entries, entry)
	err = idx.Write(repo)
	if err != nil {
		t.Fatalf("Failed to write index: %v", err)
	}

	// Create and write a test commit that points to our tree
	data := kvlm.New()

	idx, err = index.Read(repo)
	if err != nil {
		t.FailNow()
	}
	treeFromIdx, err := TreeFromIndex(repo, idx)
	if err != nil {
		t.FailNow()
	}

	data.Okv.Set("tree", []byte(treeFromIdx))
	data.Message = []byte("my commit message")
	data.Okv.Set("author", []byte("jesse"))
	data.Okv.Set("committer", []byte("jesse"))
	commit := NewCommit(data)

	commitHash, err := WriteObject(commit, repo)
	if err != nil {
		t.Fatalf("Failed to write test commit: %v", err)
	}

	// Verify commit was written correctly
	commitPath := filepath.Join(repo.GitDir(), "objects", commitHash.AsString()[:2], commitHash.AsString()[2:])
	if _, err := os.Stat(commitPath); os.IsNotExist(err) {
		t.Fatalf("Commit not written to %s", commitPath)
	}

	// Update HEAD to point to our commit
	headPath := filepath.Join(repo.GitDir(), "HEAD")
	if err := os.WriteFile(headPath, []byte("ref: refs/heads/master\n"), 0644); err != nil {
		t.Fatalf("Failed to update HEAD: %v", err)
	}

	// Create a branch reference
	branchPath := filepath.Join(repo.GitDir(), "refs", "heads", "master")
	if err := os.MkdirAll(filepath.Dir(branchPath), 0755); err != nil {
		t.Fatalf("Failed to create branch directory: %v", err)
	}
	if err := os.WriteFile(branchPath, []byte(commitHash.AsString()+"\n"), 0644); err != nil {
		t.Fatalf("Failed to create branch reference: %v", err)
	}

	// Verify branch reference was written correctly
	if _, err := os.Stat(branchPath); os.IsNotExist(err) {
		t.Fatalf("Branch reference not written to %s", branchPath)
	}

	tests := []struct {
		name    string
		input   string
		format  GitObjectType
		follow  bool
		wantErr bool
	}{
		{"find blob by hash", blobHash.AsString(), TypeBlob, true, false},
		{"find tree by hash", treeHash.AsString(), TypeTree, true, false},
		{"find commit by hash", commitHash.AsString(), TypeCommit, true, false},
		{"find blob by ref", "master", TypeBlob, true, true},  // Should fail as it's a commit
		{"find tree by ref", "master", TypeTree, true, false}, // Should succeed as commit points to tree
		{"find commit by ref", "master", TypeCommit, true, false},
		{"find any by ref", "master", TypeNoTypeSpecified, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Find(repo, tt.input, tt.format, tt.follow)
			if (err != nil) && !tt.wantErr {
				t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteObject(t *testing.T) {
	repo := setupTestRepo(t)
	defer cleanupTestRepo(t, repo)

	// Test writing a blob
	t.Run("write blob", func(t *testing.T) {
		blob := &Blob{data: []byte("test content")}
		hash, err := WriteObject(blob, repo)
		if err != nil {
			t.Fatalf("WriteObject() error = %v", err)
		}

		// Verify object file exists
		objPath := filepath.Join(repo.GitDir(), "objects", hash.AsString()[:2], hash.AsString()[2:])
		if _, err := os.Stat(objPath); os.IsNotExist(err) {
			t.Errorf("Object file not created at %s", objPath)
		}

		// Verify object can be read back
		obj, err := ReadObject(repo, hash)
		if err != nil {
			t.Errorf("Failed to read back object: %v", err)
		}
		if obj.Type() != TypeBlob {
			t.Errorf("Object type = %v, want %v", obj.Type(), TypeBlob)
		}
		blob = obj.(*Blob)
		if !bytes.Equal(blob.data, []byte("test content")) {
			t.Errorf("Blob content = %q, want %q", blob.data, "test content")
		}
	})

	// Test writing a tree
	t.Run("write tree", func(t *testing.T) {
		// First create a blob that our tree will point to
		blob := &Blob{data: []byte("test content")}
		blobHash, err := WriteObject(blob, repo)
		if err != nil {
			t.Fatalf("Failed to write test blob: %v", err)
		}

		tree := &Tree{
			Items: []*TreeLeaf{
				{
					Mode: []byte("100644"),
					Path: []byte("test.txt"),
					Sha:  blobHash,
				},
			},
		}
		hash, err := WriteObject(tree, repo)
		if err != nil {
			t.Fatalf("WriteObject() error = %v", err)
		}

		// Verify object file exists
		objPath := filepath.Join(repo.GitDir(), "objects", hash.AsString()[:2], hash.AsString()[2:])
		if _, err := os.Stat(objPath); os.IsNotExist(err) {
			t.Errorf("Object file not created at %s", objPath)
		}

		// Verify object can be read back
		obj, err := ReadObject(repo, hash)
		if err != nil {
			t.Errorf("Failed to read back object: %v", err)
		}
		if obj.Type() != TypeTree {
			t.Errorf("Object type = %v, want %v", obj.Type(), TypeTree)
		}
		tree = obj.(*Tree)
		if len(tree.Items) != 1 {
			t.Errorf("Tree items length = %d, want 1", len(tree.Items))
		}
		if !bytes.Equal(tree.Items[0].Path, []byte("test.txt")) {
			t.Errorf("Tree item path = %q, want %q", tree.Items[0].Path, "test.txt")
		}
		if tree.Items[0].Sha.AsString() != blobHash.AsString() {
			t.Errorf("Tree item sha = %q, want %q", tree.Items[0].Sha.AsString(), blobHash.AsString())
		}
	})

	// Test writing a commit
	t.Run("write commit", func(t *testing.T) {
		// First create a tree that our commit will point to
		blob := &Blob{data: []byte("test content")}
		blobHash, err := WriteObject(blob, repo)
		if err != nil {
			t.Fatalf("Failed to write test blob: %v", err)
		}

		tree := &Tree{
			Items: []*TreeLeaf{
				{
					Mode: []byte("100644"),
					Path: []byte("test.txt"),
					Sha:  blobHash,
				},
			},
		}
		treeHash, err := WriteObject(tree, repo)
		if err != nil {
			t.Fatalf("Failed to write test tree: %v", err)
		}

		// Create the commit
		commitData := kvlm.New()
		commitData.Okv.Set("tree", []byte(treeHash.AsString()))
		commitData.Okv.Set("author", []byte("Test Author <test@example.com>"))
		commitData.Okv.Set("committer", []byte("Test Committer <test@example.com>"))
		commitData.Message = []byte("Test commit message")
		commit := NewCommit(commitData)

		hash, err := WriteObject(commit, repo)
		if err != nil {
			t.Fatalf("WriteObject() error = %v", err)
		}

		// Verify object file exists
		objPath := filepath.Join(repo.GitDir(), "objects", hash.AsString()[:2], hash.AsString()[2:])
		if _, err := os.Stat(objPath); os.IsNotExist(err) {
			t.Errorf("Object file not created at %s", objPath)
		}

		// Verify object can be read back
		obj, err := ReadObject(repo, hash)
		if err != nil {
			t.Errorf("Failed to read back object: %v", err)
		}
		if obj.Type() != TypeCommit {
			t.Errorf("Object type = %v, want %v", obj.Type(), TypeCommit)
		}
		commit = obj.(*Commit)
		if commit.Message() != "Test commit message" {
			t.Errorf("Commit message = %q, want %q", commit.Message(), "Test commit message")
		}
		treeVal, ok := commit.GetValue("tree")
		if !ok {
			t.Error("Commit missing tree value")
		}
		if !bytes.Equal(treeVal, []byte(treeHash.AsString())) {
			t.Errorf("Commit tree = %q, want %q", treeVal, treeHash)
		}
	})

	// Test writing a tag
	t.Run("write tag", func(t *testing.T) {
		// First create a commit that our tag will point to
		blob := &Blob{data: []byte("test content")}
		blobHash, err := WriteObject(blob, repo)
		if err != nil {
			t.Fatalf("Failed to write test blob: %v", err)
		}

		tree := &Tree{
			Items: []*TreeLeaf{
				{
					Mode: []byte("100644"),
					Path: []byte("test.txt"),
					Sha:  blobHash,
				},
			},
		}
		treeHash, err := WriteObject(tree, repo)
		if err != nil {
			t.Fatalf("Failed to write test tree: %v", err)
		}

		// Create a commit
		commitData := kvlm.New()
		commitData.Okv.Set("tree", []byte(treeHash.AsString()))
		commitData.Okv.Set("author", []byte("Test Author <test@example.com>"))
		commitData.Okv.Set("committer", []byte("Test Committer <test@example.com>"))
		commitData.Message = []byte("Test commit message")
		commit := NewCommit(commitData)
		commitHash, err := WriteObject(commit, repo)
		if err != nil {
			t.Fatalf("Failed to write test commit: %v", err)
		}

		// Create the tag
		tagData := kvlm.New()
		tagData.Okv.Set("object", []byte(commitHash.AsString()))
		tagData.Okv.Set("type", []byte("commit"))
		tagData.Okv.Set("tagger", []byte("Test Tagger <test@example.com>"))
		tagData.Message = []byte("Test tag message")
		tag := &Tag{data: tagData}

		hash, err := WriteObject(tag, repo)
		if err != nil {
			t.Fatalf("WriteObject() error = %v", err)
		}

		// Verify object file exists
		objPath := filepath.Join(repo.GitDir(), "objects", hash.AsString()[:2], hash.AsString()[2:])
		if _, err := os.Stat(objPath); os.IsNotExist(err) {
			t.Errorf("Object file not created at %s", objPath)
		}

		// Verify object can be read back
		obj, err := ReadObject(repo, hash)
		if err != nil {
			t.Errorf("Failed to read back object: %v", err)
		}
		if obj.Type() != TypeTag {
			t.Errorf("Object type = %v, want %v", obj.Type(), TypeTag)
		}
		tag = obj.(*Tag)
		if tag.Message() != "Test tag message" {
			t.Errorf("Tag message = %q, want %q", tag.Message(), "Test tag message")
		}
		objVal, ok := tag.GetValue("object")
		if !ok {
			t.Error("Tag missing object value")
		}
		if !bytes.Equal(objVal, []byte(commitHash.AsString())) {
			t.Errorf("Tag object = %q, want %q", objVal, commitHash)
		}
	})
}
