package veild

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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
	37:  "CERT",
	60:  "CDNSKEY",
	64:  "SVCB",
	65:  "HTTPS",
	255: "ANY",
	257: "CAA",
}

// Errors in the DNS parse phase.
var (
	// ErrInvalidDnsPacket is returned when the packet doesn't look like a DNS packet.
	ErrInvalidDnsPacket = errors.New("invalid dns packet")

	// ErrInvalidRType is returned when a mapping cannot be found between
	// the numeric representation of an RR type and it's string.
	ErrInvalidRType = errors.New("invalid rtype")

	// ErrProblemParsingOffsets is returned when a TTL offset cannot be parsed.
	ErrProblemParsingOffsets = errors.New("problem parsing TTL offsets")
)

// NewRR returns a new RR.
func NewRR(data []byte) (*RR, error) {

	nameType, err := sliceNameType(data)
	if err != nil {
		return nil, fmt.Errorf("error creating rr: %w", err)
	}

	host := parseDomainName(nameType[:len(nameType)-2])
	rtype := binary.BigEndian.Uint16(nameType[len(nameType)-2:])

	rType, ok := ResourceTypes[rtype]
	if !ok {
		return nil, ErrInvalidRType
	}

	return &RR{
		hostname: host,
		rType:    rType,
		cacheKey: nameType,
	}, nil
}

// parseDomainName takes a slice of bytes and returns a parsed domain name.
func parseDomainName(data []byte) string {
	parts := make([]byte, 0)
	i := 0
	for {
		if data[i] == 0x00 {
			// End of label/name.
			break
		}
		if i != 0x00 {
			// Append a `.`.
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
func sliceNameType(packet []byte) ([]byte, error) {
	// Scan for end of name (0x00).
	if i := bytes.IndexByte(packet, 0x00); i != -1 {
		// Return the name and type.
		return packet[:i+3], nil
	}

	return []byte{}, ErrInvalidDnsPacket
}

// ttlOffsets scans a DNS record and returns offsets of all the TTLs within it.
// SEE: https://www.rfc-editor.org/rfc/rfc1035#section-3.2
// SEE: https://cs.opensource.google/go/x/net/+/master:dns/dnsmessage/message.go;l=2105;drc=ea0c1d94f5e0c4b4c18b927e26e188ad8fadb38e
func ttlOffsets(data []byte) ([]int, error) {

	var ttlOffsets []int

	// Get total answers etc.
	answers := binary.BigEndian.Uint16(data[6:8])
	authority := binary.BigEndian.Uint16(data[8:10])

	total := int(answers + authority)

	// Skip first 12 bytes (always the header, no TTLs).
	offset := HeaderLength

	// Attempting to jump over Questions section.

	// Quickly run through the query (single one).
	i := bytes.IndexByte(data[offset:], 0x00)
	i += 5 // jump 1 + 4 more bytes (End of Name, Type and Class).
	offset += i

	// Parsing Answers and Authority RRs.

	for n := 0; n < total; n++ {

		// Handle NAME field.
		// This could be a pointer or a label.

		// Check we're not overrunning the length of the message.
		if len(data) < offset+1 {
			return nil, ErrProblemParsingOffsets
		}

		marker := data[offset : offset+1]
		c := marker[0]

		switch c & 0xc0 {

		case 0xc0: // Pointer ref, only 2 bytes.
			offset += 2

		case 0x00: // End of record.
			offset++

		default:
			return nil, fmt.Errorf("error on marker: 0x%x", marker)

		}

		// Advance past the TYPE and CLASS fields.
		offset += 4

		// TTL field.
		ttlOffsets = append(ttlOffsets, offset)
		offset += 4

		// RDLENGTH field.
		rdLength := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2

		// Advance past the RDATA field using RDLENGTH.
		offset += int(rdLength)
	}

	return ttlOffsets, nil
}
