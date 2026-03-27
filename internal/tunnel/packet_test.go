package tunnel

import (
	"net"
	"testing"
)

// buildIPv4Packet constructs a minimal 20-byte IPv4 header with optional extra payload.
func buildIPv4Packet(src, dst net.IP, proto uint8, payload []byte) []byte {
	src4 := src.To4()
	dst4 := dst.To4()
	totalLen := 20 + len(payload)
	pkt := make([]byte, totalLen)
	pkt[0] = 0x45 // version=4, IHL=5 (20 bytes)
	pkt[1] = 0x00
	pkt[2] = byte(totalLen >> 8)
	pkt[3] = byte(totalLen)
	// bytes 4-7: identification, flags, fragment offset — zero
	pkt[8] = 64 // TTL
	pkt[9] = proto
	// bytes 10-11: checksum — zero (not validated by parser)
	copy(pkt[12:16], src4)
	copy(pkt[16:20], dst4)
	copy(pkt[20:], payload)
	return pkt
}

// buildIPv6Packet constructs a minimal 40-byte IPv6 header with optional extra payload.
func buildIPv6Packet(src, dst net.IP, nextHeader uint8, payload []byte) []byte {
	src16 := src.To16()
	dst16 := dst.To16()
	payloadLen := len(payload)
	pkt := make([]byte, 40+payloadLen)
	pkt[0] = 0x60 // version=6, traffic class upper bits=0
	// bytes 1-3: traffic class lower + flow label — zero
	pkt[4] = byte(payloadLen >> 8)
	pkt[5] = byte(payloadLen)
	pkt[6] = nextHeader
	pkt[7] = 64 // hop limit
	copy(pkt[8:24], src16)
	copy(pkt[24:40], dst16)
	copy(pkt[40:], payload)
	return pkt
}

// ─── ParseIPv4Header ──────────────────────────────────────────────────────────

func TestParseIPv4Header_Valid(t *testing.T) {
	src := net.IPv4(192, 168, 1, 1)
	dst := net.IPv4(10, 0, 0, 1)
	pkt := buildIPv4Packet(src, dst, ProtoTCP, nil)

	h, err := ParseIPv4Header(pkt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !h.Src.Equal(src.To4()) {
		t.Errorf("src: got %s, want %s", h.Src, src)
	}
	if !h.Dst.Equal(dst.To4()) {
		t.Errorf("dst: got %s, want %s", h.Dst, dst)
	}
	if h.Protocol != ProtoTCP {
		t.Errorf("proto: got %d, want %d", h.Protocol, ProtoTCP)
	}
	if h.TotalLen != 20 {
		t.Errorf("total len: got %d, want 20", h.TotalLen)
	}
}

func TestParseIPv4Header_TooShort(t *testing.T) {
	if _, err := ParseIPv4Header(make([]byte, 10)); err == nil {
		t.Fatal("expected error for too-short buffer")
	}
}

func TestParseIPv4Header_WrongVersion(t *testing.T) {
	pkt := buildIPv4Packet(net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8), ProtoUDP, nil)
	pkt[0] = 0x65 // change version nibble to 6 (IPv6) but keep IHL=5
	if _, err := ParseIPv4Header(pkt); err == nil {
		t.Fatal("expected error for wrong version")
	}
}

func TestParseIPv4Header_InvalidIHL(t *testing.T) {
	pkt := buildIPv4Packet(net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8), ProtoICMP, nil)
	pkt[0] = 0x41 // version=4, IHL=1 (4 bytes — invalid, < 20)
	if _, err := ParseIPv4Header(pkt); err == nil {
		t.Fatal("expected error for invalid IHL")
	}
}

func TestParseIPv4Header_WithPayload(t *testing.T) {
	payload := []byte("hello world")
	pkt := buildIPv4Packet(net.IPv4(172, 16, 0, 1), net.IPv4(8, 8, 8, 8), ProtoUDP, payload)

	h, err := ParseIPv4Header(pkt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.TotalLen != uint16(20+len(payload)) {
		t.Errorf("total len: got %d, want %d", h.TotalLen, 20+len(payload))
	}
}

// ─── ParseIPv6Header ──────────────────────────────────────────────────────────

func TestParseIPv6Header_Valid(t *testing.T) {
	src := net.ParseIP("2001:db8::1")
	dst := net.ParseIP("2001:db8::2")
	pkt := buildIPv6Packet(src, dst, ProtoTCP, nil)

	h, err := ParseIPv6Header(pkt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !h.Src.Equal(src) {
		t.Errorf("src: got %s, want %s", h.Src, src)
	}
	if !h.Dst.Equal(dst) {
		t.Errorf("dst: got %s, want %s", h.Dst, dst)
	}
	if h.NextHeader != ProtoTCP {
		t.Errorf("next header: got %d, want %d", h.NextHeader, ProtoTCP)
	}
}

func TestParseIPv6Header_TooShort(t *testing.T) {
	if _, err := ParseIPv6Header(make([]byte, 20)); err == nil {
		t.Fatal("expected error for too-short buffer")
	}
}

func TestParseIPv6Header_WrongVersion(t *testing.T) {
	pkt := buildIPv6Packet(net.ParseIP("::1"), net.ParseIP("::2"), ProtoUDP, nil)
	pkt[0] = 0x45 // change version nibble to 4 (IPv4)
	if _, err := ParseIPv6Header(pkt); err == nil {
		t.Fatal("expected error for wrong version")
	}
}

func TestParseIPv6Header_WithPayload(t *testing.T) {
	payload := []byte("test payload")
	pkt := buildIPv6Packet(net.ParseIP("fe80::1"), net.ParseIP("fe80::2"), ProtoUDP, payload)

	h, err := ParseIPv6Header(pkt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.PayloadLen != uint16(len(payload)) {
		t.Errorf("payload len: got %d, want %d", h.PayloadLen, len(payload))
	}
}

// ─── PacketSrcDst ─────────────────────────────────────────────────────────────

func TestPacketSrcDst_IPv4(t *testing.T) {
	src := net.IPv4(192, 168, 1, 10)
	dst := net.IPv4(10, 0, 0, 5)
	pkt := buildIPv4Packet(src, dst, ProtoTCP, nil)

	gotSrc, gotDst, err := PacketSrcDst(pkt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotSrc.Equal(src.To4()) {
		t.Errorf("src: got %s, want %s", gotSrc, src)
	}
	if !gotDst.Equal(dst.To4()) {
		t.Errorf("dst: got %s, want %s", gotDst, dst)
	}
}

func TestPacketSrcDst_IPv6(t *testing.T) {
	src := net.ParseIP("2001:db8::10")
	dst := net.ParseIP("2001:db8::20")
	pkt := buildIPv6Packet(src, dst, ProtoUDP, nil)

	gotSrc, gotDst, err := PacketSrcDst(pkt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotSrc.Equal(src) {
		t.Errorf("src: got %s, want %s", gotSrc, src)
	}
	if !gotDst.Equal(dst) {
		t.Errorf("dst: got %s, want %s", gotDst, dst)
	}
}

func TestPacketSrcDst_Empty(t *testing.T) {
	_, _, err := PacketSrcDst(nil)
	if err == nil {
		t.Fatal("expected error for empty packet")
	}
}

func TestPacketSrcDst_UnknownVersion(t *testing.T) {
	pkt := []byte{0x30, 0x00, 0x00} // version=3
	_, _, err := PacketSrcDst(pkt)
	if err == nil {
		t.Fatal("expected error for unknown IP version")
	}
}

func TestPacketSrcDst_IPv4_TooShort(t *testing.T) {
	pkt := []byte{0x45, 0, 0, 0, 0, 0, 0, 0, 64, 6} // version=4 but only 10 bytes
	_, _, err := PacketSrcDst(pkt)
	if err == nil {
		t.Fatal("expected error for truncated IPv4 packet")
	}
}

func TestPacketSrcDst_IPv6_TooShort(t *testing.T) {
	pkt := make([]byte, 20) // version=6 but only 20 bytes (< 40)
	pkt[0] = 0x60
	_, _, err := PacketSrcDst(pkt)
	if err == nil {
		t.Fatal("expected error for truncated IPv6 packet")
	}
}

// ─── IsIPv4 / IsIPv6 ─────────────────────────────────────────────────────────

func TestIsIPv4(t *testing.T) {
	ipv4Pkt := buildIPv4Packet(net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8), ProtoUDP, nil)
	if !IsIPv4(ipv4Pkt) {
		t.Error("should be IPv4")
	}

	ipv6Pkt := buildIPv6Packet(net.ParseIP("::1"), net.ParseIP("::2"), ProtoUDP, nil)
	if IsIPv4(ipv6Pkt) {
		t.Error("IPv6 packet should not be IPv4")
	}

	if IsIPv4(nil) {
		t.Error("nil should not be IPv4")
	}
	if IsIPv4([]byte{}) {
		t.Error("empty should not be IPv4")
	}
}

func TestIsIPv6(t *testing.T) {
	ipv6Pkt := buildIPv6Packet(net.ParseIP("2001:db8::1"), net.ParseIP("2001:db8::2"), ProtoTCP, nil)
	if !IsIPv6(ipv6Pkt) {
		t.Error("should be IPv6")
	}

	ipv4Pkt := buildIPv4Packet(net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8), ProtoTCP, nil)
	if IsIPv6(ipv4Pkt) {
		t.Error("IPv4 packet should not be IPv6")
	}

	if IsIPv6(nil) {
		t.Error("nil should not be IPv6")
	}
	if IsIPv6([]byte{}) {
		t.Error("empty should not be IPv6")
	}
}

// ─── Protocol constants coverage ─────────────────────────────────────────────

func TestProtoConstants(t *testing.T) {
	if ProtoICMP != 1 {
		t.Errorf("ProtoICMP: want 1, got %d", ProtoICMP)
	}
	if ProtoTCP != 6 {
		t.Errorf("ProtoTCP: want 6, got %d", ProtoTCP)
	}
	if ProtoUDP != 17 {
		t.Errorf("ProtoUDP: want 17, got %d", ProtoUDP)
	}
	if ProtoICMPv6 != 58 {
		t.Errorf("ProtoICMPv6: want 58, got %d", ProtoICMPv6)
	}
}
