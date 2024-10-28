package objects

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strconv"

	"github.com/jessegeens/go-toolbox/pkg/repository"
)

type GitObject interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
	Type() string
}

// func New() (*GitObject, error) {
// 	return nil, nil
// }

// func (o *GitObject) Serialize() (string, error) {
// 	return "", errors.New("Not implemented")
// }

// func (o *GitObject) Deserialize() {}

func ReadObject(repo *repository.Repository, sha string) (GitObject, error) {
	path, err := repo.RepositoryFile(false, "objects", sha[0:2], sha[2:])
	if err != nil {
		return nil, err
	}

	if !repository.IsFile(path) {
		return nil, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	zlibReader, err := zlib.NewReader(f)
	if err != nil {
		return nil, errors.New("failed to open file: " + err.Error())
	}
	defer zlibReader.Close()
	rawObjectContents, err := io.ReadAll(zlibReader)
	if err != nil {
		return nil, errors.New("failed to read file: " + err.Error())
	}

	// Read object type
	idxType := bytes.IndexByte(rawObjectContents, '\x00')
	objType := string(rawObjectContents[0:idxType])

	// Read and validate obj size
	idxSize := idxType + bytes.IndexByte(rawObjectContents[idxType:], '\x00')
	size, err := strconv.Atoi(string(rawObjectContents[idxType:idxSize]))
	if err != nil {
		return nil, errors.New("invalid object size " + string(rawObjectContents[idxType:idxSize]))
	}

	if size != len(rawObjectContents)-idxSize-1 {
		return nil, errors.New("malformed object " + sha + ", bad length")
	}

	switch objType {
	case "commit":
	case "tree":
	case "tag":
	case "blob":
		return nil, nil
	}
	return nil, errors.New("invalid object type " + objType)
}

// encode serializes the object, including the header
func Encode(o GitObject) ([]byte, error) {
	data, err := o.Serialize()
	if err != nil {
		return nil, err
	}

	header := []byte(o.Type() + " " + strconv.Itoa(len(data)))
	encoded := append(append(header, '\x00'), data...)
	return encoded, nil
}

func CalculateSha(o GitObject) (string, error) {
	encoded, err := Encode(o)
	if err != nil {
		return "", err
	}

	hasher := sha1.New()
	hasher.Write(encoded)
	hash := hasher.Sum(nil)

	return hex.EncodeToString(hash), nil
}

func WriteObject(o GitObject, repo *repository.Repository) (string, error) {
	hash, err := CalculateSha(o)
	if err != nil {
		return "", err
	}

	path, err := repo.RepositoryFile(true, "objects", hash[0:2], hash[2:])
	if err != nil {
		return "", err
	}

	if !repository.PathExists(path) {
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()

		encodedObject, err := Encode(o)
		if err != nil {
			return "", err
		}

		zlibWriter := zlib.NewWriter(f)
		_, err = zlibWriter.Write(encodedObject)
		if err != nil {
			return "", err
		}

	}

	return hash, nil
}

func Find(repo *repository.Repository, name string) (string, error) {
	return name, nil
}
