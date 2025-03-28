package repository

import (
	"io"
	"os"
)

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsExist(err)
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
	if !os.IsExist(err) {
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
	return nil
}
