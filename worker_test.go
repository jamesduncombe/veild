package veild

import (
	"testing"
)

func TestWorker_NewWorker(t *testing.T) {

	worker := NewWorker("9.9.9.9:853", "dns.quad9.net")

	if worker.host != "9.9.9.9:853" {
		t.Errorf("Expected host to be 9.9.9.9:853, got %s", worker.host)
	}
	if worker.serverName != "dns.quad9.net" {
		t.Errorf("Expected serverName to be dns.quad9.net, got %s", worker.serverName)
	}
}
