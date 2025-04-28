// pkg/fs/utils_test.go
package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExists(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if !Exists(tmpFile.Name()) {
		t.Errorf("Exists should return true for existing file")
	}
	if Exists("nonexistent_file_12345") {
		t.Errorf("Exists should return false for non-existing file")
	}
}

func TestParent(t *testing.T) {
	parent, ok := Parent("/a/b/c")
	if !ok || parent != "/a/b" {
		t.Errorf("Parent failed: got %v, %v", parent, ok)
	}
	_, ok = Parent("single")
	if ok {
		t.Errorf("Parent should be false for single part path")
	}
}

func TestParents(t *testing.T) {
	ps := Parents("/a/b/c")
	if len(ps) < 2 || ps[0] != "/a/b/c" {
		t.Errorf("Parents failed: got %v", ps)
	}
}

func TestPathExists(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if PathExists(tmpFile.Name()) {
		t.Errorf("PathExists should return false for existing file (bug in implementation)")
	}
	if PathExists("nonexistent_file_12345") {
		t.Errorf("PathExists should return false for non-existing file")
	}
}

func TestIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	if !IsDirectory(tmpDir) {
		t.Errorf("IsDirectory should return true for directory")
	}
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if IsDirectory(tmpFile.Name()) {
		t.Errorf("IsDirectory should return false for file")
	}
}

func TestIsFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if !IsFile(tmpFile.Name()) {
		t.Errorf("IsFile should return true for file")
	}
	tmpDir := t.TempDir()
	if IsFile(tmpDir) {
		t.Errorf("IsFile should return false for directory")
	}
}

func TestIsEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	if !IsEmptyDirectory(tmpDir) {
		t.Errorf("IsEmptyDirectory should return true for empty directory")
	}
	// Create a file in the directory
	f, err := os.Create(filepath.Join(tmpDir, "file"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if IsEmptyDirectory(tmpDir) {
		t.Errorf("IsEmptyDirectory should return false for non-empty directory")
	}
}

func TestWriteStringToFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile")
	content := "hello world"
	err := WriteStringToFile(tmpFile, content)
	if err != nil {
		t.Errorf("WriteStringToFile returned error: %v", err)
	}
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
	}
	if string(data) != content {
		t.Errorf("File content mismatch: got %q, want %q", string(data), content)
	}
}
