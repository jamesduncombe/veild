package main

import "testing"

func TestMain_NewConfig(t *testing.T) {
	config := NewConfig(":853", true, "blacklist.txt", "resolvers.txt")

	if config.ListenAddr != ":853" {
		t.Errorf("Expected ListenAddr to be :853, got %s", config.ListenAddr)
	}
}
