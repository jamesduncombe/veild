package veild

import (
	"os"
	"reflect"
	"testing"
)

func Test_NewRR(t *testing.T) {
	// Query for protonmail.com A record.
	packet := []byte{
		0xa, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x6e, 0x6d,
		0x61, 0x69, 0x6c, 0x3, 0x63, 0x6f, 0x6d, 0x0,
		0x00, 0x01,
	}
	if _, err := NewRR(packet); err != nil {
		t.Error(err)
	}
}

func Test_sliceNameType(t *testing.T) {
	packet := []byte{
		0xa, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x6e, 0x6d,
		0x61, 0x69, 0x6c, 0x3, 0x63, 0x6f, 0x6d, 0x0,
		0x00, 0x01, 0x1, 0x3, 0x5,
	}

	want := []byte{
		0xa, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x6e, 0x6d,
		0x61, 0x69, 0x6c, 0x3, 0x63, 0x6f, 0x6d, 0x0,
		0x00, 0x01,
	}

	got, _ := sliceNameType(packet)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wanted %s, got %s", want, got)
	}

	failureCase := []byte{0x01}

	_, err := sliceNameType(failureCase)
	if err != ErrInvalidDnsPacket {
		t.Errorf("expected error %v", err)
	}

}

func Test_parseDomainName(t *testing.T) {
	packetBytes := []byte{
		0xa, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x6e, 0x6d,
		0x61, 0x69, 0x6c, 0x3, 0x63, 0x6f, 0x6d, 0x0,
	}

	want := "protonmail.com"
	got := parseDomainName(packetBytes)
	if got != want {
		t.Errorf("%s should equal %s", got, want)
	}
}

func Benchmark_parseDomainName(b *testing.B) {
	packetBytes := []byte{
		0xa, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x6e, 0x6d,
		0x61, 0x69, 0x6c, 0x3, 0x63, 0x6f, 0x6d, 0x0,
	}

	for n := 0; n < b.N; n++ {
		parseDomainName(packetBytes)
	}
}

func TestQueryCache_ttlOffsets(t *testing.T) {

	tests := []struct {
		filename string
		want     []int
	}{
		{
			filename: "fixtures/phishing-detection.api.cx.metamask.io_a.pkt",
			want:     []int{61, 128, 186, 213, 251, 307, 321},
		},
	}

	for i, test := range tests {
		data, _ := os.ReadFile(test.filename)
		got, _ := ttlOffsets(data)
		if len(got) != len(test.want) {
			t.Errorf("wanted %d length got %d", len(got), len(test.want))
			break
		}
		if got[i] != test.want[i] {
			t.Errorf("wanted %d offset got %d", got[i], test.want[i])
		}
	}
}
