package objects

import "github.com/jessegeens/go-toolbox/pkg/kvlm"

type Tag Commit

// A bit sad that Go does not support inheritance here, so we have to re-declare all methods...

func (t *Tag) Serialize() ([]byte, error) {
	return []byte(t.data.Serialize()), nil
}

func (t *Tag) Deserialize(data []byte) error {
	kvlm.Parse(data, 0, t.data)
	return nil
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
