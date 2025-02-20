package objects

import "github.com/jessegeens/go-toolbox/pkg/kvlm"

type Commit struct {
	data *kvlm.Kvlm
}

func (c *Commit) Serialize() ([]byte, error) {
	return []byte(c.data.Serialize()), nil
}

func (c *Commit) Deserialize(data []byte) error {
	kvlm.Parse(data, 0, c.data)
	return nil
}

func (c *Commit) Type() GitObjectType {
	return TypeCommit
}

func (c *Commit) Message() string {
	return string(c.data.Message)
}

func (c *Commit) GetValue(key string) ([]byte, bool) {
	return c.data.Okv.Get(key)
}

func NewCommit(data *kvlm.Kvlm) *Commit {
	return &Commit{data: data}
}
