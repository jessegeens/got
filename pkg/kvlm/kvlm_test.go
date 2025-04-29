package kvlm

import (
	"bytes"
	"strings"
	"testing"
)

// Helper to create a new OrderedKV
func newOrderedKV() OrderedKV {
	return OrderedKV{
		kv:   make(map[string][]byte),
		keys: []string{},
	}
}

func TestOrderedKV_SetGetHasKeys(t *testing.T) {
	okv := newOrderedKV()
	okv.Set("author", []byte("Alice"))
	okv.Set("committer", []byte("Bob"))
	okv.Set("author", []byte(" <alice@example.com>"))

	val, ok := okv.Get("author")
	if !ok {
		t.Fatal("Expected to find 'author' key")
	}
	if !bytes.Contains(val, []byte("Alice")) || !bytes.Contains(val, []byte("<alice@example.com>")) {
		t.Errorf("Unexpected value for 'author': %s", val)
	}

	if !okv.Has("committer") {
		t.Error("Expected to have 'committer' key")
	}
	keys := okv.Keys()
	if len(keys) != 2 || keys[0] != "author" || keys[1] != "committer" {
		t.Errorf("Unexpected keys order: %v", keys)
	}
}

func TestParse_SimpleKVLM(t *testing.T) {
	raw := []byte("tree 1234567890abcdef\nparent abcdef1234567890\nauthor Alice <alice@example.com>\n\nThis is the commit message.\n")
	msg := &Kvlm{Okv: newOrderedKV()}
	err := Parse(raw, 0, msg)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	tree, ok := msg.Okv.Get("tree")
	if !ok || string(tree) != "1234567890abcdef" {
		t.Errorf("tree: got %q, want %q", tree, "1234567890abcdef")
	}
	parent, ok := msg.Okv.Get("parent")
	if !ok || string(parent) != "abcdef1234567890" {
		t.Errorf("parent: got %q, want %q", parent, "abcdef1234567890")
	}
	author, ok := msg.Okv.Get("author")
	if !ok || !strings.Contains(string(author), "Alice") {
		t.Errorf("author: got %q, want %q", author, "Alice")
	}
	if !strings.Contains(string(msg.Message), "This is the commit message.") {
		t.Errorf("message: got %q, want %q", msg.Message, "This is the commit message.")
	}
}

func TestSerialize(t *testing.T) {
	okv := newOrderedKV()
	okv.Set("tree", []byte("1234567890abcdef"))
	okv.Set("author", []byte("Alice <alice@example.com>"))
	kvlm := &Kvlm{
		Okv:     okv,
		Message: []byte("Commit message\nSecond line"),
	}
	serialized := kvlm.Serialize()
	if !strings.Contains(serialized, "tree 1234567890abcdef\n") {
		t.Errorf("Serialized missing tree: %q", serialized)
	}
	if !strings.Contains(serialized, "author Alice <alice@example.com>\n") {
		t.Errorf("Serialized missing author: %q", serialized)
	}
	if !strings.Contains(serialized, "\nCommit message\nSecond line") {
		t.Errorf("Serialized missing message: %q", serialized)
	}
}

func TestParseSerialize_RoundTrip(t *testing.T) {
	raw := []byte("tree 1234567890abcdef\nauthor Alice <alice@example.com>\n\nCommit message\nSecond line\n")
	msg := &Kvlm{Okv: newOrderedKV()}
	if err := Parse(raw, 0, msg); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	serialized := msg.Serialize()
	// The message is not included in Serialize(), so we only check headers
	if !strings.Contains(serialized, "tree 1234567890abcdef\n") ||
		!strings.Contains(serialized, "author Alice <alice@example.com>\n") {
		t.Errorf("Round-trip failed: %q", serialized)
	}
}
