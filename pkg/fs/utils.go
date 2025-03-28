// Utils for FS-related operations
package fs

import (
	"errors"
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
