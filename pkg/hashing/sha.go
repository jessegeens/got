package hashing

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
)

type SHA struct {
	hash []byte
}

// `data` must always be of length 20
// TODO: add a check and error if this fails
func NewSHA(data []byte) *SHA {
	hasher := sha1.New()
	hasher.Write(data)
	hash := hasher.Sum(nil)
	return &SHA{
		hash: hash,
	}
}

// Given a string representing a hex-encoded sha\
// of length 40, return a SHA object
func NewShaFromHex(hash string) (*SHA, error) {
	bytes, err := hex.DecodeString(hash)
	if err != nil {
		return nil, errors.New("failed to decode hex-encoded hash")
	}
	if len(hash) != 40 {
		return nil, errors.New("SHAs in hex-encoded string format must always be of length 40")
	}
	return NewSHA(bytes), nil
}

func NewShaFromBytes(hash []byte) *SHA {
	return &SHA{
		hash: hash,
	}
}

func (s *SHA) AsBytes() []byte {
	return s.hash
}

func (s *SHA) AsString() string {
	return hex.EncodeToString(s.hash)
}
