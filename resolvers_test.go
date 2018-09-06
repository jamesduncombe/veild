package veild

import (
	"testing"
)

func TestResolvers_NewResolvers(t *testing.T) {
	_, err := NewResolvers("fixtures/test_resolvers.yml")
	if err != nil {
		t.Error("should be parseable")
	}
}

func TestResolvers_NewResolversErr(t *testing.T) {
	_, err := NewResolvers("non-existent file")
	if err == nil {
		t.Error("should fail on non-parseable YAML")
	}
}
