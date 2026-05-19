package command

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/jessegeens/got/pkg/fs"
	"github.com/jessegeens/got/pkg/hashing"
	"github.com/jessegeens/got/pkg/objects"
	"github.com/jessegeens/got/pkg/repository"
)

func FetchCommand() *Command {
	command := newCommand("fetch")
	command.Action = func(args []string) error {
		remoteURL := *flag.String("url", "", "HTTPS remote URL, e.g. https://example.com/repo.git")
		remoteName := *flag.String("name", "origin", "Remote name for tracking refs under refs/remotes/<name>/...")
		flag.Parse()

		if remoteURL == "" {
			if len(args) > 0 {
				remoteURL = args[0]
			} else {
				return errors.New("remote URL is required: --url <https-url> or as first positional arg")
			}
		}

		repo, err := repository.Find(".")
		if err != nil {
			return err
		}

		u, err := url.Parse(remoteURL)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}
		if u.Scheme != "https" {
			return errors.New("got only supports https remotes")
		}

		refs, err := fetchRemoteRefs(remoteURL)
		if err != nil {
			return err
		}

		// Download all objects reachable from advertised refs
		for refName, shaHex := range refs {
			sha, err := hashing.NewShaFromHex(shaHex)
			if err != nil {
				return err
			}
			if err := fetchObjectRecursively(repo, remoteURL, sha, make(map[string]bool)); err != nil {
				return err
			}

			// Update refs: branches under refs/remotes/<remoteName>/..., tags under refs/tags/...
			if strings.HasPrefix(refName, "refs/heads/") {
				branch := strings.TrimPrefix(refName, "refs/heads/")
				refPath, err := repo.RepositoryFile(true, path.Join("refs", "remotes", remoteName, branch))
				if err != nil {
					return err
				}
				if err := fs.WriteStringToFile(refPath, shaHex+"\n"); err != nil {
					return err
				}
			} else if strings.HasPrefix(refName, "refs/tags/") {
				// Write tag reference as-is
				tag := strings.TrimPrefix(refName, "refs/tags/")
				refPath, err := repo.RepositoryFile(true, path.Join("refs", "tags", tag))
				if err != nil {
					return err
				}
				if err := fs.WriteStringToFile(refPath, shaHex+"\n"); err != nil {
					return err
				}
			}
		}

		return nil
	}
	command.Description = func() string {
		return "Download objects and refs from another repository (HTTPS dumb protocol, no pack)"
	}
	return command
}

// fetchRemoteRefs retrieves refs via the dumb HTTP protocol's info/refs (text) endpoint.
func fetchRemoteRefs(remote string) (map[string]string, error) {
	infoURL := strings.TrimSuffix(remote, "/") + "/info/refs"
	resp, err := http.Get(infoURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get %s: %s", infoURL, resp.Status)
	}

	refs := map[string]string{}
	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" { // skip
			continue
		}
		// Lines are of the form: <hex>\t<refname>
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}
		sha := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		// Skip peeled annotations like "refs/tags/v1.0^{}"
		if strings.HasSuffix(name, "^{ }") || strings.HasSuffix(name, "^{}") {
			continue
		}
		refs[name] = sha
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return refs, nil
}

// fetchObjectRecursively downloads the object if missing, then parses it and downloads its children.
func fetchObjectRecursively(repo *repository.Repository, remote string, sha *hashing.SHA, seen map[string]bool) error {
	hex := sha.AsString()
	if seen[hex] {
		return nil
	}
	seen[hex] = true

	objPath, err := repo.RepositoryFile(false, "objects", hex[0:2], hex[2:])
	if err == nil && fs.IsFile(objPath) {
		// already present
		return traverseChildren(repo, remote, sha, seen)
	}
	// ensure directory exists
	_, err = repo.RepositoryDir(true, "objects", hex[0:2])
	if err != nil {
		return err
	}

	objURL := strings.TrimSuffix(remote, "/") + "/objects/" + hex[0:2] + "/" + hex[2:]
	resp, err := http.Get(objURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download object %s: %s", hex, resp.Status)
	}

	// Save as-is; server serves zlib-compressed object content
	filePath := repo.RepositoryPath("objects", hex[0:2], hex[2:])
	f, err := osCreateAppend(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	return traverseChildren(repo, remote, sha, seen)
}

func traverseChildren(repo *repository.Repository, remote string, sha *hashing.SHA, seen map[string]bool) error {
	obj, err := objects.ReadObject(repo, sha)
	if err != nil {
		return err
	}
	switch obj.Type() {
	case objects.TypeCommit:
		commit := obj.(*objects.Commit)
		if treeShaBytes, ok := commit.GetValue("tree"); ok {
			child, _ := hashing.NewShaFromHex(string(treeShaBytes))
			if err := fetchObjectRecursively(repo, remote, child, seen); err != nil {
				return err
			}
		}
		if parentsBytes, ok := commit.GetValue("parent"); ok {
			parents := strings.Split(string(parentsBytes), ",")
			for _, p := range parents {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				child, _ := hashing.NewShaFromHex(p)
				if err := fetchObjectRecursively(repo, remote, child, seen); err != nil {
					return err
				}
			}
		}
	case objects.TypeTree:
		tree := obj.(*objects.Tree)
		for _, item := range tree.Items {
			if err := fetchObjectRecursively(repo, remote, item.Sha, seen); err != nil {
				return err
			}
		}
	case objects.TypeTag:
		tag := obj.(*objects.Tag)
		if objShaBytes, ok := tag.GetValue("object"); ok {
			child, _ := hashing.NewShaFromHex(string(objShaBytes))
			if err := fetchObjectRecursively(repo, remote, child, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// Minimal helper to create or truncate file for writing append stream
func osCreateAppend(p string) (io.WriteCloser, error) {
	// Ensure parent dirs exist (callers already do for objects/<aa>)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return f, nil
}
