package veild

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestNextBlacklist(t *testing.T) {
	_, err := NewBlacklist("nonexistantfile.txt")
	if err == nil {
		t.Error("non-existance blacklist, should error")
	}
}

func TestExists(t *testing.T) {
	blacklist, _ := NewBlacklist("fixtures/blacklist_test.txt")
	blacklist.Exists("0-edge-chat.facebook.com")
	if blacklist.Exists("jamesduncombe.com") {
		t.Error("exists when it shouldn't")
	}
}

func TestParseBlacklist(t *testing.T) {
	file, _ := ioutil.ReadFile("fixtures/blacklist_test.txt")
	strReader := bytes.NewReader(file)
	ParseBlacklist(strReader)
}
