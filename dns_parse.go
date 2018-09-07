package veild

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// RR represents a domain name and resource type.
type RR struct {
	hostname string
	rType    string
	cacheKey []byte
}

// ResourceTypes maps resource record (RR) types to string representations.
var ResourceTypes = map[uint16]string{
	1:   "A",
	2:   "NS",
	5:   "CNAME",
	6:   "SOA",
	12:  "PTR",
	15:  "MX",
	16:  "TXT",
	28:  "AAAA",
	33:  "SRV",
	257: "CAA",
}

// Errors in the DNS parse phase.
var (
	// ErrInvalidRType is returned when a mapping cannot be found between
	// the numeric representation of an RR type and it's string.
	ErrInvalidRType = errors.New("invalid rtype")
)

// NewRR returns a new RR.
func NewRR(data []byte) (*RR, error) {

	nameType := sliceNameType(data)
	host := parseDomainName(nameType[:len(nameType)-2])
	rtype := binary.BigEndian.Uint16(nameType[len(nameType)-2:])

	if rType, ok := ResourceTypes[rtype]; ok {
		return &RR{
			hostname: host,
			rType:    rType,
			cacheKey: nameType,
		}, nil
	}

	return nil, ErrInvalidRType
}

// parseDomainName takes a slice of bytes and returns a parsed domain name.
func parseDomainName(data []byte) string {
	parts := make([]byte, 0)
	i := 0
	for {
		if data[i] == 0x00 {
			break
		}
		if i != 0x00 {
			parts = append(parts, 0x2e)
		}
		l := int(data[i])
		parts = append(parts, data[i+1:i+l+1]...)
		// Increment to next label offset.
		i += l + 1
	}
	return string(parts)
}

// sliceNameType takes a DNS request and slices out the name + type of the request.
// This is mainly used for the cache key when storing a request.
func sliceNameType(packet []byte) []byte {
	// Scan for end of name (0x00).
	if i := bytes.IndexByte(packet, 0x00); i != -1 {
		// Return the name and type.
		return packet[:i+3]
	}
	return []byte{}
}
