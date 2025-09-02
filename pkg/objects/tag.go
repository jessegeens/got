package objects

import "github.com/jessegeens/got/pkg/kvlm"

type Tag Commit

// Go does not support inheritance, so we have to re-declare all methods...

func (t *Tag) Serialize() ([]byte, error) {
	return []byte(t.data.Serialize()), nil
}

func (t *Tag) Deserialize(data []byte) error {
	if t.data == nil {
		t.data = kvlm.New()
	}
	return kvlm.Parse(data, 0, t.data)
}

func (t *Tag) Type() GitObjectType {
	return TypeTag
}

func (t *Tag) Message() string {
	return string(t.data.Message)
}

func (t *Tag) GetValue(key string) ([]byte, bool) {
	return t.data.Okv.Get(key)
}
