package object

type Blob struct {
	data []byte
}

func (b *Blob) Serialize() ([]byte, error) {
	return b.data, nil
}

func (b *Blob) Deserialize(data []byte) error {
	b.data = data
	return nil
}

func (b *Blob) Type() string {
	return "blob"
}
