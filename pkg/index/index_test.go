package index

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jessegeens/go-toolbox/pkg/hashing"
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

func TestNewIndex(t *testing.T) {
	sha, err := hashing.NewShaFromHex("0123456789abcdef0123456789abcdef01234567")
	if err != nil {
		t.Errorf("Failed to create SHA hash")
	}
	entries := []*Entry{
		{
			CTime:           time.Now(),
			MTime:           time.Now(),
			Dev:             123,
			Inode:           456,
			ModeType:        ModeTypeRegular,
			ModePerms:       0o644,
			UID:             1000,
			GID:             1000,
			Size:            1024,
			SHA:             sha,
			FlagAssumeValid: false,
			FlagStage:       0,
			Name:            "test.txt",
		},
	}

	idx := New(entries)
	if idx.Version != 2 {
		t.Errorf("Expected version 2, got %d", idx.Version)
	}
	if len(idx.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(idx.Entries))
	}
}

func TestIndexWriteAndRead(t *testing.T) {
	repo := setupTestRepo(t)
	defer cleanupTestRepo(t, repo)

	sha1, _ := hashing.NewShaFromHex("0123456789abcdef01230123456789abcdef0123")
	sha2, _ := hashing.NewShaFromHex("abcdef0123456789abcdabcdef0123456789abcd")

	// Create test entries
	now := time.Now()
	testEntries := []*Entry{
		{
			CTime:           now,
			MTime:           now,
			Dev:             123,
			Inode:           456,
			ModeType:        ModeTypeRegular,
			ModePerms:       0o644,
			UID:             1000,
			GID:             1000,
			Size:            1024,
			SHA:             sha1,
			FlagAssumeValid: false,
			FlagStage:       0,
			Name:            "test.txt",
		},
		{
			CTime:           now,
			MTime:           now,
			Dev:             123,
			Inode:           789,
			ModeType:        ModeTypeSymlink,
			ModePerms:       0o755,
			UID:             1000,
			GID:             1000,
			Size:            0,
			SHA:             sha2,
			FlagAssumeValid: true,
			FlagStage:       0xFF & uint16(12288),
			Name:            "link.txt",
		},
	}

	// Create and write index
	idx := New(testEntries)
	err := idx.Write(repo)
	if err != nil {
		t.Fatalf("Failed to write index: %v", err)
	}

	// Read index back
	readIdx, err := Read(repo)
	if err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	// Verify index contents
	if readIdx.Version != 2 {
		t.Errorf("Expected version 2, got %d", readIdx.Version)
	}
	if len(readIdx.Entries) != len(testEntries) {
		t.Errorf("Expected %d entries, got %d", len(testEntries), len(readIdx.Entries))
	}

	for i, e := range testEntries {
		entry := readIdx.Entries[i]
		if entry.Name != e.Name {
			t.Errorf("Expected name '%s', got '%s'", e.Name, entry.Name)
		}
		if entry.ModeType != e.ModeType {
			t.Errorf("Expected %v, got %v", e.ModeType, entry.ModeType)
		}
		if entry.ModePerms != e.ModePerms {
			t.Errorf("Expected mode %o, got %o", e.ModePerms, entry.ModePerms)
		}
		if entry.SHA != e.SHA {
			t.Errorf("Expected SHA '%s', got '%s'", e.SHA.AsString(), entry.SHA.AsString())
		}
		if entry.FlagAssumeValid != e.FlagAssumeValid {
			t.Errorf("Expected FlagAssumeValid to be %t", e.FlagAssumeValid)
		}
		if entry.FlagStage != e.FlagStage {
			t.Errorf("Expected FlagStage %d, got %d", e.FlagStage, entry.FlagStage)
		}

	}

}

func TestModeTypeString(t *testing.T) {
	tests := []struct {
		name     string
		modeType ModeType
		want     string
	}{
		{"regular", ModeTypeRegular, "regular file"},
		{"symlink", ModeTypeSymlink, "symlink"},
		{"gitlink", ModeTypeGitlink, "git link"},
		{"invalid", ModeType(999), "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.modeType.String(); got != tt.want {
				t.Errorf("ModeType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidModeType(t *testing.T) {
	tests := []struct {
		name     string
		modeType uint16
		want     bool
	}{
		{"regular", uint16(ModeTypeRegular), true},
		{"symlink", uint16(ModeTypeSymlink), true},
		{"gitlink", uint16(ModeTypeGitlink), true},
		{"invalid", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidModeType(tt.modeType); got != tt.want {
				t.Errorf("isValidModeType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIndexWithEmptyEntries(t *testing.T) {
	repo := setupTestRepo(t)
	defer cleanupTestRepo(t, repo)

	// Create empty index
	idx := New([]*Entry{})
	err := idx.Write(repo)
	if err != nil {
		t.Fatalf("Failed to write empty index: %v", err)
	}

	// Read index back
	readIdx, err := Read(repo)
	if err != nil {
		t.Fatalf("Failed to read empty index: %v", err)
	}

	if readIdx.Version != 2 {
		t.Errorf("Expected version 2, got %d", readIdx.Version)
	}
	if len(readIdx.Entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(readIdx.Entries))
	}
}

func TestIndexWithLongFilename(t *testing.T) {
	repo := setupTestRepo(t)
	defer cleanupTestRepo(t, repo)
	sha, _ := hashing.NewShaFromHex("0123456789abcdef01230123456789abcdef0123")

	// Create entry with long filename
	longName := strings.Repeat("a", 0xFF+1)
	entries := []*Entry{
		{
			CTime:           time.Now(),
			MTime:           time.Now(),
			Dev:             123,
			Inode:           456,
			ModeType:        ModeTypeRegular,
			ModePerms:       0o644,
			UID:             1000,
			GID:             1000,
			Size:            1024,
			SHA:             sha,
			FlagAssumeValid: false,
			FlagStage:       0,
			Name:            longName,
		},
	}

	// Create and write index
	idx := New(entries)
	err := idx.Write(repo)
	if err != nil {
		t.Fatalf("Failed to write index with long filename: %v", err)
	}

	// Read index back
	readIdx, err := Read(repo)
	if err != nil {
		t.Fatalf("Failed to read index with long filename: %v", err)
	}

	if len(readIdx.Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(readIdx.Entries))
	}
	if readIdx.Entries[0].Name != longName {
		t.Errorf("Expected long filename to be preserved, got '%s'", readIdx.Entries[0].Name)
	}
}
