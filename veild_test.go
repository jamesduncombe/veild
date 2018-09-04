package veild

import (
	"reflect"
	"testing"
)

func TestVeild_addHostForPort(t *testing.T) {
	if addHostForPort("9.9.9.9:853", 853) != true {
		t.Error("should match a port within the string if it exists")
	}
}

func TestVeild_parseDomainName(t *testing.T) {
	packetBytes := []byte{
		0x0d, 0x6a, 0x61, 0x6d, 0x65, 0x73, 0x64, 0x75,
		0x6e, 0x63, 0x6f, 0x6d, 0x62, 0x65, 0x03, 0x63,
		0x6f, 0x6d, 0x00,
	}
	if !reflect.DeepEqual(parseDomainName(packetBytes), "jamesduncombe.com") {
		t.Fail()
	}
}

func BenchmarkVeild_parseDomainName(b *testing.B) {
	packetBytes := []byte{
		0x0d, 0x6a, 0x61, 0x6d, 0x65, 0x73, 0x64, 0x75,
		0x6e, 0x63, 0x6f, 0x6d, 0x62, 0x65, 0x03, 0x63,
		0x6f, 0x6d, 0x00,
	}
	for n := 0; n < b.N; n++ {
		parseDomainName(packetBytes)
	}
}
