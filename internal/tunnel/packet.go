package tunnel

import (
	"encoding/binary"
	"fmt"
	"net"
)

// IP protocol numbers.
const (
	ProtoICMP   = 1
	ProtoTCP    = 6
	ProtoUDP    = 17
	ProtoICMPv6 = 58
)

// IPv4Header holds the relevant fields of an IPv4 header.
type IPv4Header struct {
	Src      net.IP
	Dst      net.IP
	Protocol uint8
	TotalLen uint16
}

// ParseIPv4Header parses the first 20 bytes of an IPv4 packet.
func ParseIPv4Header(buf []byte) (*IPv4Header, error) {
	if len(buf) < 20 {
		return nil, fmt.Errorf("ipv4: too short %d", len(buf))
	}
	version := buf[0] >> 4
	if version != 4 {
		return nil, fmt.Errorf("ipv4: wrong version %d", version)
	}
	ihl := int(buf[0]&0x0f) * 4
	if ihl < 20 || len(buf) < ihl {
		return nil, fmt.Errorf("ipv4: invalid IHL %d", ihl)
	}
	return &IPv4Header{
		TotalLen: binary.BigEndian.Uint16(buf[2:4]),
		Protocol: buf[9],
		Src:      net.IP(append([]byte{}, buf[12:16]...)),
		Dst:      net.IP(append([]byte{}, buf[16:20]...)),
	}, nil
}

// IPv6Header holds the relevant fields of an IPv6 header.
type IPv6Header struct {
	Src        net.IP
	Dst        net.IP
	NextHeader uint8
	PayloadLen uint16
}

// ParseIPv6Header parses the fixed 40-byte IPv6 header.
func ParseIPv6Header(buf []byte) (*IPv6Header, error) {
	if len(buf) < 40 {
		return nil, fmt.Errorf("ipv6: too short %d", len(buf))
	}
	version := buf[0] >> 4
	if version != 6 {
		return nil, fmt.Errorf("ipv6: wrong version %d", version)
	}
	return &IPv6Header{
		PayloadLen: binary.BigEndian.Uint16(buf[4:6]),
		NextHeader: buf[6],
		Src:        net.IP(append([]byte{}, buf[8:24]...)),
		Dst:        net.IP(append([]byte{}, buf[24:40]...)),
	}, nil
}

// PacketSrcDst extracts (src, dst) IPs from an IP packet (v4 or v6).
func PacketSrcDst(buf []byte) (src, dst net.IP, err error) {
	if len(buf) == 0 {
		return nil, nil, fmt.Errorf("empty packet")
	}
	version := buf[0] >> 4
	switch version {
	case 4:
		h, e := ParseIPv4Header(buf)
		if e != nil {
			return nil, nil, e
		}
		return h.Src, h.Dst, nil
	case 6:
		h, e := ParseIPv6Header(buf)
		if e != nil {
			return nil, nil, e
		}
		return h.Src, h.Dst, nil
	default:
		return nil, nil, fmt.Errorf("unknown IP version %d", version)
	}
}

// IsIPv4 reports whether buf starts with an IPv4 header.
func IsIPv4(buf []byte) bool {
	return len(buf) >= 1 && buf[0]>>4 == 4
}

// IsIPv6 reports whether buf starts with an IPv6 header.
func IsIPv6(buf []byte) bool {
	return len(buf) >= 1 && buf[0]>>4 == 6
}
