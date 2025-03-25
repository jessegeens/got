package ignore

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/jessegeens/go-toolbox/pkg/fs"
	"github.com/jessegeens/go-toolbox/pkg/index"
	"github.com/jessegeens/go-toolbox/pkg/objects"
	"github.com/jessegeens/go-toolbox/pkg/repository"
)

type Pattern string

type Ignore struct {
	// Rules from global ignore files (usually in ~/.config/git/ignore)
	// and the repository-specific .git/info/exclude
	Absolute []*Rule
	// Scoped rules live in the index, in gitignore files
	// They are scoped to the directory they are located in
	Scoped map[string][]*Rule
}
type Rule struct {
	// True if the file should be ignored,
	// false if the file should not be ignored
	Exclude bool
	Pattern Pattern
}

func Read(repo *repository.Repository) (*Ignore, error) {
	absolute := []*Rule{}
	scoped := map[string][]*Rule{}

	// Read rules defined in .git/info/exclude
	excludeFile := repo.RepositoryPath("info/exclude")
	if fs.Exists(excludeFile) {
		rules, err := parseFile(excludeFile)
		if err != nil {
			return nil, err
		}
		absolute = append(absolute, rules...)
	}

	// Read global configuration
	var configHome string
	if val, set := os.LookupEnv("XDG_CONFIG_HOME"); set {
		configHome = val
	} else {
		home, err := fs.HomeDir()
		if err == nil {
			configHome = path.Join(home, ".config/git/ignore")
		}
	}
	if configHome != "" && fs.Exists(configHome) {
		rules, err := parseFile(configHome)
		if err != nil {
			return nil, err
		}

		absolute = append(absolute, rules...)
	}

	// Read .gitignore files in index
	idx, err := index.Read(repo)
	if err != nil {
		return nil, err
	}

	for _, e := range idx.Entries {
		if e.Name == ".gitignore" || strings.HasSuffix(e.Name, "/.gitignore") {
			directory := path.Dir(e.Name)
			contents, err := objects.ReadObject(repo, e.SHA)
			if err != nil {
				return nil, err
			}
			lines, err := contents.(*objects.Blob).Serialize()
			if err != nil {
				return nil, err
			}
			rules, err := parse(lines)
			if err != nil {
				return nil, err
			}
			scoped[directory] = rules
		}
	}
	return &Ignore{
		Absolute: absolute,
		Scoped:   scoped,
	}, nil
}

// TODO(jgeens): implement
// Returns true if the given path is to be ignored according to the gitignore rules
func (i *Ignore) Check(path string) bool {
	// for _, r := range i.Rules {
	// 	if r.Matches(path) {
	// 		return true
	// 	}
	// }
	return false
}

func parseFile(path string) ([]*Rule, error) {
	fileContents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	rules, err := parse(fileContents)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return rules, nil
}

func parse(raw []byte) ([]*Rule, error) {
	lines := strings.Split(string(raw), "\n")
	rules := []*Rule{}
	for _, line := range lines {
		rule, err := parseLine(line)
		if err != nil {
			return nil, err
		}
		if rule != nil {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func parseLine(line string) (*Rule, error) {
	line = strings.TrimSpace(line)

	if line == "" || line[0] == '#' {
		return nil, nil
	}

	if line[0] == '!' {
		return &Rule{
			Exclude: false,
			Pattern: Pattern(line[1:]),
		}, nil
	}

	return &Rule{
		Exclude: true,
		Pattern: Pattern(line[1:]),
	}, nil
}

func (r *Rule) Matches(path string) bool {
	return false
}
