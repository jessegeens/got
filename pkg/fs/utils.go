// Utils for FS-related operations
package fs

import (
	"errors"
	"io"
	"os"
	"os/user"
	"strings"
)

func Exists(path string) bool {
	_, err := os.Open(path)
	return !errors.Is(err, os.ErrNotExist)
}

func HomeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}

func Parent(path string) (string, bool) {
	parts := strings.Split(path, string(os.PathSeparator))
	if len(parts) < 2 {
		return path, false
	}
	parentParts := parts[:len(parts)-1]
	return strings.Join(parentParts, string(os.PathSeparator)), true
}

func Parents(path string) []string {
	return parents(path, []string{path})
}

func parents(path string, prts []string) []string {
	parent, ok := Parent(path)
	if ok {
		prts = append(prts, parent)
		return parents(parent, prts)
	}
	return prts
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func IsDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return fileInfo.IsDir()
}

func IsFile(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return !fileInfo.IsDir()
}

func IsEmptyDirectory(path string) bool {
	if !IsDirectory(path) {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	return err == io.EOF
}

func WriteStringToFile(path string, contents string) error {
	return os.WriteFile(path, []byte(contents), os.ModePerm)
}

func ReadContents(path string) (string, error) {
	contentsBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	contents := strings.TrimSpace(string(contentsBytes))
	return strings.TrimSuffix(contents, "\n"), nil
}
