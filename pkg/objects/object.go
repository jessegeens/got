package objects

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jessegeens/go-toolbox/pkg/fs"
	"github.com/jessegeens/go-toolbox/pkg/references"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

type GitObject interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
	Type() GitObjectType
}

// Enum for Git object types
type GitObjectType string

const (
	TypeCommit          GitObjectType = "commit"
	TypeTree            GitObjectType = "tree"
	TypeTag             GitObjectType = "tag"
	TypeBlob            GitObjectType = "blob"
	TypeNoTypeSpecified GitObjectType = ""
)

func (g GitObjectType) String() string {
	return string(g)
}

func ParseType(objectType string) (GitObjectType, error) {
	switch objectType {
	case string(TypeCommit):
		return TypeCommit, nil
	case string(TypeTree):
		return TypeTree, nil
	case string(TypeTag):
		return TypeTag, nil
	case string(TypeBlob):
		return TypeBlob, nil
	}
	return "", errors.New("Not a valid object type: " + objectType)
}

func ReadObject(repo *repository.Repository, sha string) (GitObject, error) {
	path, err := repo.RepositoryFile(false, "objects", sha[0:2], sha[2:])
	if err != nil {
		return nil, err
	}

	if !fs.IsFile(path) {
		return nil, errors.New("not a file: " + path)
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

	// TODO(jgeens)
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

	header := []byte(string(o.Type()) + " " + strconv.Itoa(len(data)))
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

	if !fs.PathExists(path) {
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

// Find finds an object called `name` in a repository `repo`.
//
//   - Name: can be short or long hash, HEAD, a branch name or a tag name
//   - Follow: determines if we follow tags or not. Recommended default is `true`
//   - Format: determines what type of object we want to locate. Use TypeNoTypeSpecified if you do not want a specific object type
func Find(repo *repository.Repository, name string, format GitObjectType, follow bool) (string, error) {
	shas, err := Resolve(repo, name)
	if err != nil {
		return "", err
	}

	if len(shas) > 1 {
		return "", errors.New("cannot find object ambiguous name: found " + strconv.Itoa(len(shas)) + " possible objects!")
	}

	if len(shas) == 0 {
		return "", errors.New("did not find any match for object named " + name)
	}

	sha := shas[0]

	for {
		// Not really efficient: we read the whole object just to determine its type (in a loop!)
		obj, err := ReadObject(repo, sha)
		if err != nil {
			return "", err
		}

		if obj.Type() == format {
			return sha, nil
		}

		if !follow {
			return "", errors.New("did not find any match for object named " + name + " matching the specified format")
		}

		if obj.Type() == TypeTag {
			tag := obj.(*Tag)
			objSha, ok := tag.GetValue("object")
			if !ok {
				return "", errors.New("failed to parse tag")
			}
			sha = string(objSha)
		} else if obj.Type() == TypeCommit && format == TypeTree {
			tag := obj.(*Commit)
			objSha, ok := tag.GetValue("tree")
			if !ok {
				return "", errors.New("failed to parse commit")
			}
			sha = string(objSha)
		} else {
			return "", errors.New("did not find any match for object named " + name + " matching the specified format")
		}
	}

	//return name, nil
}

func ObjectHash(fileContents []byte, objectType GitObjectType, repo *repository.Repository) (string, error) {
	var obj GitObject = nil
	switch objectType {
	case TypeBlob:
		obj = &Blob{data: fileContents}
	}
	return WriteObject(obj, repo)
}

// Resolve name to an object hash in repo.
//
// This function is aware of:
//
//   - the HEAD literal
//   - short and long hashes
//   - tags
//   - branches
//   - remote branches
func Resolve(repo *repository.Repository, name string) ([]string, error) {
	candidates := []string{}
	hashRegex, err := regexp.Compile("^[0-9A-Fa-f]{4,40}$")
	if err != nil {
		return nil, err
	}

	if name == "" {
		return nil, errors.New("no name given to objects.Resolve")
	}

	// HEAD is non-ambiguous, so we can return directly
	// instead of also trying for hashes, branches etc
	if name == "HEAD" {
		res, err := references.Reference(name).Resolve(repo)
		return []string{res}, err
	}

	// Next we try for hashes
	if hashRegex.Match([]byte(name)) {
		name = strings.ToLower(name)
		prefix := name[0:2]
		path, err := repo.RepositoryDir(false, "objects", prefix)
		if err != nil {
			return nil, err
		}
		if path != "" {
			remainder := name[2:]
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), remainder) {
					candidates = append(candidates, prefix+entry.Name())
				}
			}
		}
	}

	// Next we try for tags
	tag, err := references.Reference("refs/tags/" + name).Resolve(repo)
	if err == nil && tag != "" {
		candidates = append(candidates, tag)
	}

	// Finally we try for branches
	branch, err := references.Reference("refs/heads/" + name).Resolve(repo)
	if err == nil && branch != "" {
		candidates = append(candidates, branch)
	}

	return candidates, nil
}
