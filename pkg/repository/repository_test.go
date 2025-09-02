package repository

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jessegeens/got/pkg/fs"
	"gopkg.in/ini.v1"
)

func setupTestDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "got-test-repo-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	if tempDir[len(tempDir)-1] != filepath.Separator {
		tempDir = tempDir + string(filepath.Separator)
	}
	return tempDir
}

func cleanupTestDir(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Logf("Warning: failed to clean up test directory: %v", err)
	}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		wantErr     bool
		description string
	}{
		{
			name: "create in empty directory",
			setup: func(t *testing.T) string {
				return setupTestDir(t)
			},
			wantErr:     false,
			description: "should create repository in empty directory",
		},
		{
			name: "create in non-empty directory",
			setup: func(t *testing.T) string {
				dir := setupTestDir(t)
				// Create a file in the directory
				if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return dir
			},
			wantErr:     false,
			description: "should create repository in non-empty directory",
		},
		{
			name: "create in existing repository",
			setup: func(t *testing.T) string {
				dir := setupTestDir(t)
				// Create a .git directory
				if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
					t.Fatalf("Failed to create .git directory: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("[core]\nrepositoryformatversion = 0\n"), 0644); err != nil {
					t.Fatalf("Failed to create config file: %v", err)
				}
				return dir
			},
			wantErr:     true,
			description: "should fail when .git directory exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			defer cleanupTestDir(t, dir)

			_, err := Create(dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify repository structure
				gitDir := filepath.Join(dir, ".git")
				if !fs.IsDirectory(gitDir) {
					t.Error("Expected .git directory to exist")
				}

				// Check required files
				requiredFiles := []string{"config", "description", "HEAD"}
				for _, file := range requiredFiles {
					filePath := filepath.Join(gitDir, file)
					if !fs.IsFile(filePath) {
						t.Errorf("Expected %s to exist", file)
					}
				}

				// Check required directories
				requiredDirs := []string{"objects", "refs/heads", "refs/tags", "branches"}
				for _, dir := range requiredDirs {
					dirPath := filepath.Join(gitDir, dir)
					if !fs.IsDirectory(dirPath) {
						t.Errorf("Expected %s directory to exist", dir)
					}
				}

				// Verify config file contents
				cfg, err := ini.Load(filepath.Join(gitDir, "config"))
				if err != nil {
					t.Fatalf("Failed to read config file: %v", err)
				}

				if cfg.Section("core").Key("repositoryformatversion").MustInt(0) != 0 {
					t.Error("Expected repositoryformatversion to be 0")
				}
				if cfg.Section("core").Key("filemode").MustBool(true) != true {
					t.Error("Expected filemode to be true")
				}
				if cfg.Section("core").Key("bare").MustBool(false) != false {
					t.Error("Expected bare to be false")
				}
			}
		})
	}
}

func TestExistingRepository(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) string
		disableChecks bool
		wantErr       bool
		description   string
	}{
		{
			name: "valid repository",
			setup: func(t *testing.T) string {
				dir := setupTestDir(t)
				repo, err := Create(dir)
				if err != nil {
					t.Fatalf("Failed to create test repository: %v", err)
				}
				return repo.WorkTree()
			},
			disableChecks: false,
			wantErr:       false,
			description:   "should open valid repository",
		},
		{
			name: "non-existent repository",
			setup: func(t *testing.T) string {
				return setupTestDir(t)
			},
			disableChecks: false,
			wantErr:       true,
			description:   "should fail for non-existent repository",
		},
		{
			name: "non-existent repository with checks disabled",
			setup: func(t *testing.T) string {
				return setupTestDir(t)
			},
			disableChecks: true,
			wantErr:       false,
			description:   "should succeed for non-existent repository with checks disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			defer cleanupTestDir(t, dir)

			repo, err := New(dir, tt.disableChecks)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if repo.WorkTree() != dir {
					t.Errorf("Expected worktree to be %s, got %s", dir, repo.WorkTree())
				}
				if repo.GitDir() != filepath.Join(dir, ".git") {
					t.Errorf("Expected gitdir to be %s, got %s", filepath.Join(dir, ".git"), repo.GitDir())
				}
			}
		})
	}
}

func TestFind(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		childPath   string
		wantErr     bool
		description string
	}{
		{
			name: "find in current directory",
			setup: func(t *testing.T) string {
				dir := setupTestDir(t)
				repo, err := Create(dir)
				if err != nil {
					t.Fatalf("Failed to create test repository: %v", err)
				}
				return repo.WorkTree()
			},
			childPath:   "",
			wantErr:     false,
			description: "should find repository in current directory",
		},
		{
			name: "find in parent directory",
			setup: func(t *testing.T) string {
				dir := setupTestDir(t)
				_, err := Create(dir)
				if err != nil {
					t.Fatalf("Failed to create test repository: %v", err)
				}
				// Create a subdirectory
				subDir := filepath.Join(dir, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatalf("Failed to create subdirectory: %v", err)
				}
				return subDir
			},
			childPath:   "",
			wantErr:     false,
			description: "should find repository in parent directory",
		},
		{
			name: "no repository found",
			setup: func(t *testing.T) string {
				return setupTestDir(t)
			},
			childPath:   "",
			wantErr:     true,
			description: "should fail when no repository is found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			defer cleanupTestDir(t, dir)

			childPath := filepath.Join(dir, tt.childPath)
			repo, err := Find(childPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if repo == nil {
					t.Error("Expected repository to be non-nil")
				}
			}
		})
	}
}

func TestRepositoryPath(t *testing.T) {
	dir := setupTestDir(t)
	defer cleanupTestDir(t, dir)

	repo, err := Create(dir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "single path",
			paths:    []string{"config"},
			expected: filepath.Join(repo.GitDir(), "config"),
		},
		{
			name:     "multiple paths",
			paths:    []string{"refs", "heads", "master"},
			expected: filepath.Join(repo.GitDir(), "refs", "heads", "master"),
		},
		{
			name:     "no paths",
			paths:    []string{},
			expected: repo.GitDir(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repo.RepositoryPath(tt.paths...)
			if got != tt.expected {
				t.Errorf("RepositoryPath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRepositoryFile(t *testing.T) {
	dir := setupTestDir(t)
	defer cleanupTestDir(t, dir)

	repo, err := Create(dir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	tests := []struct {
		name    string
		create  bool
		paths   []string
		wantErr bool
	}{
		{
			name:    "create new file",
			create:  true,
			paths:   []string{"test.txt"},
			wantErr: false,
		},
		{
			name:    "create nested file",
			create:  true,
			paths:   []string{"dir", "test.txt"},
			wantErr: false,
		},
		{
			name:    "don't create file",
			create:  false,
			paths:   []string{"nonexistent.txt"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := repo.RepositoryFile(tt.create, tt.paths...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RepositoryFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !fs.IsFile(path) {
					t.Errorf("Expected file to exist at %s", path)
				}
			}
		})
	}
}

func TestRepositoryDir(t *testing.T) {
	dir := setupTestDir(t)
	defer cleanupTestDir(t, dir)

	repo, err := Create(dir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	tests := []struct {
		name    string
		create  bool
		paths   []string
		wantErr bool
	}{
		{
			name:    "create new directory",
			create:  true,
			paths:   []string{"testdir"},
			wantErr: false,
		},
		{
			name:    "create nested directory",
			create:  true,
			paths:   []string{"dir", "subdir"},
			wantErr: false,
		},
		{
			name:    "don't create directory",
			create:  false,
			paths:   []string{"nonexistent"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := repo.RepositoryDir(tt.create, tt.paths...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RepositoryDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !fs.IsDirectory(path) {
					t.Errorf("Expected directory to exist at %s", path)
				}
			}
		})
	}
}

func TestGetActiveBranch(t *testing.T) {
	dir := setupTestDir(t)
	defer cleanupTestDir(t, dir)

	repo, err := Create(dir)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	tests := []struct {
		name         string
		setup        func(t *testing.T)
		wantBranch   string
		wantOnBranch bool
		wantErr      bool
	}{
		{
			name: "on master branch",
			setup: func(t *testing.T) {
				headPath := filepath.Join(repo.GitDir(), "HEAD")
				if err := os.WriteFile(headPath, []byte("ref: refs/heads/master\n"), 0644); err != nil {
					t.Fatalf("Failed to write HEAD file: %v", err)
				}
			},
			wantBranch:   "master",
			wantOnBranch: true,
			wantErr:      false,
		},
		{
			name: "detached HEAD",
			setup: func(t *testing.T) {
				headPath := filepath.Join(repo.GitDir(), "HEAD")
				if err := os.WriteFile(headPath, []byte("0123456789abcdef0123456789abcdef01234567\n"), 0644); err != nil {
					t.Fatalf("Failed to write HEAD file: %v", err)
				}
			},
			wantBranch:   "",
			wantOnBranch: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			branch, onBranch, err := repo.GetActiveBranch()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetActiveBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if branch != tt.wantBranch {
				t.Errorf("GetActiveBranch() branch = %v, want %v", branch, tt.wantBranch)
			}
			if onBranch != tt.wantOnBranch {
				t.Errorf("GetActiveBranch() onBranch = %v, want %v", onBranch, tt.wantOnBranch)
			}
		})
	}
}
