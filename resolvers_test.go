package veild

import (
	"io/ioutil"
	"testing"
)

func TestResolvers_LoadResolvers(t *testing.T) {
	d, _ := ioutil.ReadFile("fixtures/test_resolvers.yml")
	_, err := LoadResolvers(d)
	if err != nil {
		t.Error("should be parseable")
	}
}

func TestResolvers_LoadResolversErr(t *testing.T) {
	_, err := LoadResolvers([]byte("can't parse"))
	if err == nil {
		t.Error("should fail on non-parseable YAML")
	}
}
