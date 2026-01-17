package veild

import (
	"testing"
)

func TestBlocklist_NewBlocklist(t *testing.T) {
	logger := newLogger()
	_, err := NewBlocklist("nonexistantfile.txt", logger)
	if err == nil {
		t.Error("non-existence blocklist, should error")
	}
}

func TestBlocklist_Exists(t *testing.T) {
	logger := newLogger()
	blocklist, _ := NewBlocklist("fixtures/blocklist_test.txt", logger)
	blocklist.Exists("0-edge-chat.facebook.com")
	if blocklist.Exists("protonmail.com") {
		t.Error("exists when it shouldn't")
	}
}
