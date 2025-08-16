package hashing

import (
	"bytes"
	"encoding/hex"
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

func TestHexEncodeDecode(t *testing.T) {
	input := bytes.Repeat([]byte{'a', 'c'}, 25)
	hexEncoded := hex.EncodeToString(input)
	decoded, _ := hex.DecodeString(hexEncoded)
	if !bytes.Equal(input, decoded) {
		t.Errorf("expected %v, got %v", input, decoded)
	}
}
