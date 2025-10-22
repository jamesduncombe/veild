package veild

import (
	"testing"
)

func TestBlacklist_NewBlacklist(t *testing.T) {
	logger := newLogger()
	_, err := NewBlacklist("nonexistantfile.txt", logger)
	if err == nil {
		t.Error("non-existence blacklist, should error")
	}
}

func TestBlacklist_Exists(t *testing.T) {
	logger := newLogger()
	blacklist, _ := NewBlacklist("fixtures/blacklist_test.txt", logger)
	blacklist.Exists("0-edge-chat.facebook.com")
	if blacklist.Exists("protonmail.com") {
		t.Error("exists when it shouldn't")
	}
}
