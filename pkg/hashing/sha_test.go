package hashing

import (
	"bytes"
	"testing"
)

func TestNewShaFromHex(t *testing.T) {
	input := bytes.Repeat([]byte{'a', 'c'}, 25)
	hash := NewSHA(input)
	hexHash := hash.AsString()
	secondHash, err := NewShaFromHex(hexHash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(hash.AsBytes(), secondHash.AsBytes()) {
		t.Errorf("expected %v, got %v", hash.AsString(), secondHash.AsString())
	}
}
