//go:build integration

package got

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jessegeens/go-toolbox/pkg/command"
	"github.com/jessegeens/go-toolbox/pkg/fs"
	"github.com/jessegeens/go-toolbox/pkg/index"
	"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

func setupTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "got-integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tempDir
}

func cleanupTestDir(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Logf("Warning: failed to clean up test directory: %v", err)
	}
}

func TestGitWorkflow(t *testing.T) {
	// Setup test directory
	testDir := setupTestDir(t)
	defer cleanupTestDir(t, testDir)

	// Change to test directory for the duration of the test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	t.Run("initialize repository", func(t *testing.T) {
		// Initialize a new repository
		initCmd := command.InitCommand()
		err := initCmd.Action([]string{})
		if err != nil {
			t.Fatalf("Failed to initialize repository: %v", err)
		}

		// Verify repository was created
		_, err = repository.Find(".")
		if err != nil {
			t.Fatalf("Failed to find repository: %v", err)
		}

		// Check that .git directory exists
		gitDir := filepath.Join(testDir, ".git")
		if !fs.IsDirectory(gitDir) {
			t.Error("Expected .git directory to exist")
		}

		// Check that required files exist
		requiredFiles := []string{"config", "description", "HEAD"}
		for _, file := range requiredFiles {
			filePath := filepath.Join(gitDir, file)
			if !fs.IsFile(filePath) {
				t.Errorf("Expected %s to exist", file)
			}
		}

		// Check that required directories exist
		requiredDirs := []string{"objects", "refs/heads", "refs/tags", "branches"}
		for _, dir := range requiredDirs {
			dirPath := filepath.Join(gitDir, dir)
			if !fs.IsDirectory(dirPath) {
				t.Errorf("Expected %s directory to exist", dir)
			}
		}
	})

	t.Run("create and add file", func(t *testing.T) {
		// Create a test file
		testFile := "test.txt"
		testContent := "Hello, World!\nThis is a test file for got."

		err := os.WriteFile(testFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Add the file to the index
		addCmd := command.AddCommand()
		err = addCmd.Action([]string{testFile})
		if err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}

		// Verify file was added to index
		repo, err := repository.Find(".")
		if err != nil {
			t.Fatalf("Failed to find repository: %v", err)
		}

		idx, err := index.Read(repo)
		if err != nil {
			t.Fatalf("Failed to read index: %v", err)
		}

		if len(idx.Entries) != 1 {
			t.Errorf("Expected 1 entry in index, got %d", len(idx.Entries))
		}

		entry := idx.Entries[0]
		if entry.Name != testFile {
			t.Errorf("Expected entry name to be %s, got %s", testFile, entry.Name)
		}

		// Verify the file content was hashed and stored
		if entry.SHA == "" {
			t.Error("Expected SHA to be set")
		}

		// Verify the blob object was created
		objPath := filepath.Join(repo.GitDir(), "objects", entry.SHA[:2], entry.SHA[2:])
		if !fs.IsFile(objPath) {
			t.Errorf("Expected object file to exist at %s", objPath)
		}

		// Verify we can read the object back
		obj, err := objects.ReadObject(repo, entry.SHA)
		if err != nil {
			t.Fatalf("Failed to read object: %v", err)
		}

		blob, ok := obj.(*objects.Blob)
		if !ok {
			t.Fatal("Expected object to be a blob")
		}

		// Get the blob data using Serialize
		blobData, err := blob.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize blob: %v", err)
		}

		if string(blobData) != testContent {
			t.Errorf("Expected blob content to be %q, got %q", testContent, string(blobData))
		}
	})

	t.Run("commit the file", func(t *testing.T) {
		// Commit the changes
		commitCmd := command.CommitCommand()
		err := commitCmd.Action([]string{})
		if err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Verify commit was created
		repo, err := repository.Find(".")
		if err != nil {
			t.Fatalf("Failed to find repository: %v", err)
		}

		// Check that HEAD points to a commit
		headPath := filepath.Join(repo.GitDir(), "HEAD")
		_, err = os.ReadFile(headPath)
		if err != nil {
			t.Fatalf("Failed to read HEAD: %v", err)
		}

		// HEAD should point to a branch (not a direct commit hash)
		if !fs.IsFile(filepath.Join(repo.GitDir(), "refs", "heads", "master")) {
			t.Error("Expected master branch to exist")
		}

		// Read the master branch reference
		masterContent, err := repo.GetBranchCommit("master")
		if err != nil {
			t.Fatalf("Failed to read master branch: %v", err)
		}

		commitHash := string(masterContent)
		if len(commitHash) != 40 {
			t.Errorf("Expected commit hash to be 40 characters, got %d", len(commitHash))
		}

		// Verify the commit object exists
		commitObjPath := filepath.Join(repo.GitDir(), "objects", commitHash[:2], commitHash[2:])
		if !fs.IsFile(commitObjPath) {
			t.Errorf("Expected commit object to exist at %s", commitObjPath)
		}

		// Verify we can read the commit object
		commitObj, err := objects.ReadObject(repo, commitHash)
		if err != nil {
			t.Fatalf("Failed to read commit object: %v", err)
		}

		commit, ok := commitObj.(*objects.Commit)
		if !ok {
			t.Fatal("Expected object to be a commit")
		}

		// Verify commit has a tree
		treeHash, ok := commit.GetValue("tree")
		if !ok {
			t.Error("Expected commit to have a tree")
		}

		// Verify tree object exists
		treeObjPath := filepath.Join(repo.GitDir(), "objects", string(treeHash)[:2], string(treeHash)[2:])
		if !fs.IsFile(treeObjPath) {
			t.Errorf("Expected tree object to exist at %s", treeObjPath)
		}

		// Verify we can read the tree object
		treeObj, err := objects.ReadObject(repo, string(treeHash))
		if err != nil {
			t.Fatalf("Failed to read tree object: %v", err)
		}

		tree, ok := treeObj.(*objects.Tree)
		if !ok {
			t.Fatal("Expected object to be a tree")
		}

		// Verify tree contains our file
		if len(tree.Items) != 1 {
			t.Errorf("Expected tree to have 1 item, got %d", len(tree.Items))
		}

		treeItem := tree.Items[0]
		if string(treeItem.Path) != "test.txt" {
			t.Errorf("Expected tree item to be test.txt, got %s", string(treeItem.Path))
		}

		// Verify the tree item points to our blob
		idx, err := index.Read(repo)
		if err != nil {
			t.Fatalf("Failed to read index: %v", err)
		}

		if len(idx.Entries) != 1 {
			t.Errorf("Expected 1 entry in index, got %d", len(idx.Entries))
		}

		expectedBlobHash := idx.Entries[0].SHA
		actualBlobHash := string(treeItem.Sha)
		if actualBlobHash != expectedBlobHash {
			t.Errorf("Expected tree item to point to blob %s, got %s", expectedBlobHash, actualBlobHash)
		}
	})
}
