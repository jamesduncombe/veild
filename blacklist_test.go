package veild

import (
	"testing"
)

func TestBlacklist_NewBlacklist(t *testing.T) {
	_, err := NewBlacklist("nonexistantfile.txt")
	if err == nil {
		t.Error("non-existence blacklist, should error")
	}
}

func TestBlacklist_Exists(t *testing.T) {
	blacklist, _ := NewBlacklist("fixtures/blacklist_test.txt")
	blacklist.Exists("0-edge-chat.facebook.com")
	if blacklist.Exists("jamesduncombe.com") {
		t.Error("exists when it shouldn't")
	}
}
