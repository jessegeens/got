// Utils for FS-related operations
package fs

import (
	"errors"
	"os"
	"os/user"
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
