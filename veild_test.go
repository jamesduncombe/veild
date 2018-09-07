package veild

import (
	"testing"
)

func TestVeild_addHostForPort(t *testing.T) {
	if addHostForPort("9.9.9.9:853", 853) != true {
		t.Error("should match a port within the string if it exists")
	}
}
