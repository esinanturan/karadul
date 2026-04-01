//go:build linux

package tunnel

import (
	"encoding/binary"
	"net"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

// parseIPv4 is a test helper that parses an IPv4 address string.
func parseIPv4(s string) net.IP {
	return net.ParseIP(s).To4()
}

// parseIPv6 is a test helper that parses an IPv6 address string.
func parseIPv6(s string) net.IP {
	return net.ParseIP(s)
}

// parseCIDR is a test helper that parses a CIDR string.
func parseCIDR(s string) (*net.IPNet, *net.IPNet) {
	_, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return ipNet, ipNet
}

// ─── rtaAlign ─────────────────────────────────────────────────────────────────

func TestRtaAlign(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{1, 4},
		{4, 4},
		{5, 8},
		{12, 12},
		{13, 16},
		{16, 16},
		{17, 20},
		{100, 100},
		{101, 104},
	}
	for _, tc := range tests {
		got := rtaAlign(tc.input)
		if got != tc.want {
			t.Errorf("rtaAlign(%d) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// ─── parseNetlinkMessage ─────────────────────────────────────────────────────

func TestParseNetlinkMessage_EmptyBuffer(t *testing.T) {
	msgs, err := parseNetlinkMessage(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}

	msgs, err = parseNetlinkMessage([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestParseNetlinkMessage_BufferTooSmall(t *testing.T) {
	// A buffer smaller than SizeofNlMsghdr should return empty with no error.
	buf := make([]byte, unix.SizeofNlMsghdr-1)
	msgs, err := parseNetlinkMessage(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages for undersized buffer, got %d", len(msgs))
	}
}

func TestParseNetlinkMessage_HeaderLenTooSmall(t *testing.T) {
	// Build a buffer where the header's Len field is less than SizeofNlMsghdr.
	buf := make([]byte, unix.SizeofNlMsghdr)
	// Set Len to 0 (less than SizeofNlMsghdr).
	binary.LittleEndian.PutUint32(buf[0:4], 0)
	_, err := parseNetlinkMessage(buf)
	if err == nil {
		t.Fatal("expected error for header Len < SizeofNlMsghdr")
	}
}

func TestParseNetlinkMessage_HeaderLenExceedsBuffer(t *testing.T) {
	// Build a buffer where the header's Len field exceeds the buffer size.
	buf := make([]byte, unix.SizeofNlMsghdr)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(unix.SizeofNlMsghdr+100))
	_, err := parseNetlinkMessage(buf)
	if err == nil {
		t.Fatal("expected error for header Len > buffer length")
	}
}

func TestParseNetlinkMessage_SingleValidMessage(t *testing.T) {
	// Construct a single netlink message with header + 4 bytes of payload.
	dataLen := 4
	totalLen := uint32(unix.SizeofNlMsghdr + dataLen)
	buf := make([]byte, totalLen)

	// NlMsghdr fields
	binary.LittleEndian.PutUint32(buf[0:4], totalLen) // Len
	binary.LittleEndian.PutUint16(buf[4:6], 1)        // Type (arbitrary)
	binary.LittleEndian.PutUint16(buf[6:8], 0)        // Flags
	binary.LittleEndian.PutUint32(buf[8:12], 1)       // Seq
	binary.LittleEndian.PutUint32(buf[12:16], 0)      // Pid
	// Payload
	buf[unix.SizeofNlMsghdr] = 0xAA

	msgs, err := parseNetlinkMessage(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Header.Len != totalLen {
		t.Errorf("header Len: got %d, want %d", msgs[0].Header.Len, totalLen)
	}
	if len(msgs[0].Data) != dataLen {
		t.Errorf("data length: got %d, want %d", len(msgs[0].Data), dataLen)
	}
	if msgs[0].Data[0] != 0xAA {
		t.Errorf("data[0]: got 0x%02X, want 0xAA", msgs[0].Data[0])
	}
}

func TestParseNetlinkMessage_TwoMessages(t *testing.T) {
	// Two back-to-back netlink messages, each with minimal payload.
	dataLen := 2
	msgLen := uint32(unix.SizeofNlMsghdr + dataLen)
	totalLen := msgLen * 2
	buf := make([]byte, totalLen)

	for i := 0; i < 2; i++ {
		offset := i * int(msgLen)
		binary.LittleEndian.PutUint32(buf[offset+0:], msgLen)
		binary.LittleEndian.PutUint16(buf[offset+4:], uint16(i+1)) // Type
		binary.LittleEndian.PutUint16(buf[offset+6:], 0)
		binary.LittleEndian.PutUint32(buf[offset+8:], uint32(i+1))  // Seq
		binary.LittleEndian.PutUint32(buf[offset+12:], 0)
		buf[offset+unix.SizeofNlMsghdr] = byte(i)
	}

	msgs, err := parseNetlinkMessage(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	for i, m := range msgs {
		if m.Header.Len != msgLen {
			t.Errorf("msg[%d] header Len: got %d, want %d", i, m.Header.Len, msgLen)
		}
		if m.Header.Type != uint16(i+1) {
			t.Errorf("msg[%d] type: got %d, want %d", i, m.Header.Type, i+1)
		}
		if len(m.Data) != dataLen {
			t.Errorf("msg[%d] data length: got %d, want %d", i, len(m.Data), dataLen)
		}
	}
}

// ─── buildNlAddrMsg ───────────────────────────────────────────────────────────

func TestBuildNlAddrMsg_IPv4(t *testing.T) {
	ip := parseIPv4("10.0.0.1")
	msg := buildNlAddrMsg(1, ip, 24, unix.AF_INET)

	if len(msg) == 0 {
		t.Fatal("expected non-empty message")
	}

	// The first SizeofNlMsghdr bytes are the header.
	if len(msg) < unix.SizeofNlMsghdr {
		t.Fatalf("message too short: %d bytes", len(msg))
	}
	nlhLen := binary.LittleEndian.Uint32(msg[0:4])
	if int(nlhLen) != len(msg) {
		t.Errorf("header Len %d != message length %d", nlhLen, len(msg))
	}

	// Type should be RTM_NEWADDR.
	msgType := binary.LittleEndian.Uint16(msg[4:6])
	if msgType != unix.RTM_NEWADDR {
		t.Errorf("message type: got %d, want RTM_NEWADDR (%d)", msgType, unix.RTM_NEWADDR)
	}
}

func TestBuildNlAddrMsg_IPv6(t *testing.T) {
	ip := parseIPv6("fd00::1")
	msg := buildNlAddrMsg(2, ip, 64, unix.AF_INET6)

	if len(msg) == 0 {
		t.Fatal("expected non-empty message")
	}
	nlhLen := binary.LittleEndian.Uint32(msg[0:4])
	if int(nlhLen) != len(msg) {
		t.Errorf("header Len %d != message length %d", nlhLen, len(msg))
	}
}

func TestBuildNlAddrMsg_IPv4ContainsCorrectIP(t *testing.T) {
	ip := parseIPv4("192.168.1.100")
	msg := buildNlAddrMsg(5, ip, 24, unix.AF_INET)

	// Skip past nlmsghdr + ifaddrmsg to find RTA attributes.
	offset := unix.SizeofNlMsghdr + unix.SizeofIfAddrmsg

	// First attribute should be IFA_LOCAL with the IP address.
	if offset+4 > len(msg) {
		t.Fatalf("message too short to contain RTA at offset %d", offset)
	}
	rtaLen := binary.LittleEndian.Uint16(msg[offset : offset+2])
	rtaType := binary.LittleEndian.Uint16(msg[offset+2 : offset+4])

	if rtaType != unix.IFA_LOCAL {
		t.Errorf("RTA type: got %d, want IFA_LOCAL (%d)", rtaType, unix.IFA_LOCAL)
	}
	if int(rtaLen) < unix.SizeofRtAttr+4 {
		t.Errorf("RTA len too small: %d", rtaLen)
	}

	// The IP should be at offset+4 (after RtAttr header).
	gotIP := msg[offset+4 : offset+8]
	wantIP := ip.To4()
	for i := range wantIP {
		if gotIP[i] != wantIP[i] {
			t.Errorf("IP byte %d: got %d, want %d", i, gotIP[i], wantIP[i])
		}
	}
}

// ─── buildNlRouteMsg ─────────────────────────────────────────────────────────

func TestBuildNlRouteMsg_IPv4(t *testing.T) {
	_, dst := parseCIDR("10.0.0.0/24")
	msg := buildNlRouteMsg(1, dst)

	if len(msg) == 0 {
		t.Fatal("expected non-empty message")
	}
	nlhLen := binary.LittleEndian.Uint32(msg[0:4])
	if int(nlhLen) != len(msg) {
		t.Errorf("header Len %d != message length %d", nlhLen, len(msg))
	}

	msgType := binary.LittleEndian.Uint16(msg[4:6])
	if msgType != unix.RTM_NEWROUTE {
		t.Errorf("message type: got %d, want RTM_NEWROUTE (%d)", msgType, unix.RTM_NEWROUTE)
	}
}

func TestBuildNlRouteMsg_IPv6(t *testing.T) {
	_, dst := parseCIDR("fd00::/64")
	msg := buildNlRouteMsg(3, dst)

	if len(msg) == 0 {
		t.Fatal("expected non-empty message")
	}
	nlhLen := binary.LittleEndian.Uint32(msg[0:4])
	if int(nlhLen) != len(msg) {
		t.Errorf("header Len %d != message length %d", nlhLen, len(msg))
	}
}

func TestBuildNlRouteMsg_IPv4ContainsDstAndOIF(t *testing.T) {
	_, dst := parseCIDR("172.16.0.0/16")
	ifIndex := 7
	msg := buildNlRouteMsg(ifIndex, dst)

	offset := unix.SizeofNlMsghdr + unix.SizeofRtMsg

	// First attribute: RTA_DST
	rtaLen := binary.LittleEndian.Uint16(msg[offset : offset+2])
	rtaType := binary.LittleEndian.Uint16(msg[offset+2 : offset+4])

	if rtaType != unix.RTA_DST {
		t.Errorf("first RTA type: got %d, want RTA_DST (%d)", rtaType, unix.RTA_DST)
	}

	// Verify the destination IP bytes.
	wantIP := dst.IP.To4()
	gotIP := msg[offset+4 : offset+4+len(wantIP)]
	for i := range wantIP {
		if gotIP[i] != wantIP[i] {
			t.Errorf("dst IP byte %d: got %d, want %d", i, gotIP[i], wantIP[i])
		}
	}

	// Advance to next attribute (aligned).
	nextOffset := offset + rtaAlign(int(rtaLen))

	// Second attribute: RTA_OIF
	rtaLen2 := binary.LittleEndian.Uint16(msg[nextOffset : nextOffset+2])
	rtaType2 := binary.LittleEndian.Uint16(msg[nextOffset+2 : nextOffset+4])

	if rtaType2 != unix.RTA_OIF {
		t.Errorf("second RTA type: got %d, want RTA_OIF (%d)", rtaType2, unix.RTA_OIF)
	}
	gotOIF := binary.LittleEndian.Uint32(msg[nextOffset+4 : nextOffset+8])
	if gotOIF != uint32(ifIndex) {
		t.Errorf("OIF: got %d, want %d", gotOIF, ifIndex)
	}

	// Verify the rtaLen2 is correctly SizeofRtAttr + 4.
	if int(rtaLen2) != unix.SizeofRtAttr+4 {
		t.Errorf("RTA_OIF len: got %d, want %d", rtaLen2, unix.SizeofRtAttr+4)
	}
}

func TestBuildNlRouteMsg_RtMsgFields(t *testing.T) {
	_, dst := parseCIDR("192.168.0.0/24")
	ifIndex := 42
	msg := buildNlRouteMsg(ifIndex, dst)

	// RtMsg starts after the nlmsghdr.
	rtmOffset := unix.SizeofNlMsghdr

	// Read the RtMsg struct bytes.
	rtm := (*unix.RtMsg)(unsafe.Pointer(&msg[rtmOffset]))

	if rtm.Family != unix.AF_INET {
		t.Errorf("family: got %d, want AF_INET (%d)", rtm.Family, unix.AF_INET)
	}
	if rtm.Dst_len != 24 {
		t.Errorf("Dst_len: got %d, want 24", rtm.Dst_len)
	}
	if rtm.Protocol != unix.RTPROT_STATIC {
		t.Errorf("protocol: got %d, want RTPROT_STATIC (%d)", rtm.Protocol, unix.RTPROT_STATIC)
	}
	if rtm.Scope != unix.RT_SCOPE_UNIVERSE {
		t.Errorf("scope: got %d, want RT_SCOPE_UNIVERSE (%d)", rtm.Scope, unix.RT_SCOPE_UNIVERSE)
	}
	if rtm.Type != unix.RTN_UNICAST {
		t.Errorf("type: got %d, want RTN_UNICAST (%d)", rtm.Type, unix.RTN_UNICAST)
	}
}
