package protocol

import "fmt"

// ParseType returns the packet type from the first byte of a packet.
func ParseType(b []byte) (uint8, error) {
	if len(b) == 0 {
		return 0, fmt.Errorf("empty packet")
	}
	switch b[0] {
	case TypeHandshakeInit, TypeHandshakeResp, TypeData, TypeKeepalive:
		return b[0], nil
	default:
		return 0, fmt.Errorf("unknown packet type 0x%02x", b[0])
	}
}
