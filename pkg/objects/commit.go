package objects

import "github.com/jessegeens/got/pkg/kvlm"

type Commit struct {
	data *kvlm.Kvlm
}

func (c *Commit) Serialize() ([]byte, error) {
	if c.data == nil {
		return []byte{}, nil
	}
	return []byte(c.data.Serialize()), nil
}

func (c *Commit) Deserialize(data []byte) error {
	if c.data == nil {
		c.data = kvlm.New()
	}
	return kvlm.Parse(data, 0, c.data)
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
