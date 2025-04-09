package config

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/ini.v1"
)

type GitConfig struct {
	data *ini.File
}

func Read() (GitConfig, error) {
	// TODO(jgeens): expand ~
	var cfg *ini.File
	var err error
	if val, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		cfg, err = ini.Load("~/.gitconfig", path.Join(val, "/git/config"))
	} else {
		cfg, err = ini.Load("~/.gitconfig")
	}
	if err != nil {
		return GitConfig{}, err
	}
	return GitConfig{data: cfg}, err
}

func (c *GitConfig) GetUser() (string, bool) {
	userSection := c.data.Section("user")
	if userSection == nil {
		return "", false
	}
	name := userSection.Key("name")
	if name == nil {
		return "", false
	}
	email := userSection.Key("email")
	if email == nil {
		return "", false
	}

	return fmt.Sprintf("%s <%s>", name.String(), email.String()), true
}
