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
	homedir, err := os.UserHomeDir()
	if err != nil {
		return GitConfig{}, fmt.Errorf("failed to parse ~/.gitconfig: not able to read home directory: %s", err.Error())
	}
	gitConfigFileLocation := path.Join(homedir, ".gitconfig")
	var cfg *ini.File
	if val, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		cfg, err = ini.Load(gitConfigFileLocation, path.Join(val, "/git/config"))
	} else {
		cfg, err = ini.Load(gitConfigFileLocation)
	}
	if err != nil {
		return GitConfig{}, err
	}
	return GitConfig{data: cfg}, err
}

func (c *GitConfig) GetUser() (string, bool) {
	if c.data == nil {
		return "", false
	}
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
