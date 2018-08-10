package veild_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/jamesduncombe/veild"
)

func TestExists(t *testing.T) {
	blacklist, _ := veild.NewBlacklist("fixtures/blacklist_test.txt")
	blacklist.Exists("0-edge-chat.facebook.com")
}

func TestExistsNot(t *testing.T) {
	blacklist, _ := veild.NewBlacklist("fixtures/blacklist_test.txt")
	if blacklist.Exists("jamesduncombe.com") {
		t.Fail()
	}
}

func TestParseBlacklist(t *testing.T) {
	file, _ := ioutil.ReadFile("fixtures/blacklist_test.txt")
	strReader := bytes.NewReader(file)
	veild.ParseBlacklist(strReader)
}
