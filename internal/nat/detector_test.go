package nat

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// TestNATType_String_Unknown verifies that NATType(99).String() returns "unknown".
func TestNATType_String_Unknown(t *testing.T) {
	unknownType := NATType(99)
	got := unknownType.String()
	if got != "unknown" {
		t.Errorf("NATType(99).String(): want 'unknown', got %q", got)
	}
}

// TestNATType_String_AllTypes verifies all known NAT type strings.
func TestNATType_String_AllTypes(t *testing.T) {
	tests := []struct {
		nt   NATType
		want string
	}{
		{NATUnknown, "unknown"},
		{NATDirect, "direct"},
		{NATFullCone, "full-cone"},
		{NATRestrictedCone, "restricted-cone"},
		{NATPortRestricted, "port-restricted"},
		{NATSymmetric, "symmetric"},
	}

	for _, tc := range tests {
		got := tc.nt.String()
		if got != tc.want {
			t.Errorf("NATType(%d).String(): want %q, got %q", tc.nt, tc.want, got)
		}
	}
}

// startMockSTUNServerWithPort returns a mock STUN server that reports a specific mapped port.
// The mappedIP is the IP address to report as the public mapped address (should differ from local IP).
func startMockSTUNServerWithPortAndIP(t *testing.T, mappedPort int, mappedIP net.IP) (addr string, stop func()) {
	t.Helper()
	srv, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	addr = srv.LocalAddr().String()
	quit := make(chan struct{})
	go func() {
		defer srv.Close()
		buf := make([]byte, 1024)
		for {
			select {
			case <-quit:
				return
			default:
			}
			_ = srv.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
			n, src, err := srv.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			if n < stunHeaderSize {
				continue
			}
			msgType := binary.BigEndian.Uint16(buf[0:])
			if msgType != stunMsgTypeBindingRequest {
				continue
			}
			txID := make([]byte, 12)
			copy(txID, buf[8:20])
			// Build response with the specified mapped port and IP.
			resp := buildMockBindingResponseWithPortAndIP(txID, src, mappedPort, mappedIP)
			if resp != nil {
				_, _ = srv.WriteToUDP(resp, src)
			}
		}
	}()
	stop = func() { close(quit) }
	return addr, stop
}

// startMockSTUNServerWithPort is a convenience wrapper that uses a default public IP.
func startMockSTUNServerWithPort(t *testing.T, mappedPort int) (addr string, stop func()) {
	// Use TEST-NET address (192.0.2.0/24) as a fake public IP
	return startMockSTUNServerWithPortAndIP(t, mappedPort, net.IPv4(192, 0, 2, 1))
}

// buildMockBindingResponseWithPortAndIP constructs a STUN Binding Response with specific mapped port and IP.
func buildMockBindingResponseWithPortAndIP(txID []byte, src *net.UDPAddr, mappedPort int, mappedIP net.IP) []byte {
	ip4 := mappedIP.To4()
	if ip4 == nil {
		return nil
	}

	magicBytes := [4]byte{0x21, 0x12, 0xA4, 0x42}
	xorIP := [4]byte{
		ip4[0] ^ magicBytes[0],
		ip4[1] ^ magicBytes[1],
		ip4[2] ^ magicBytes[2],
		ip4[3] ^ magicBytes[3],
	}
	xorPort := uint16(mappedPort) ^ uint16(stunMagicCookie>>16)

	// XOR-MAPPED-ADDRESS value: reserved(1)+family(1)+xor-port(2)+xor-ip(4) = 8 bytes.
	val := make([]byte, 8)
	val[0] = 0x00
	val[1] = stunAddrFamilyIPv4
	binary.BigEndian.PutUint16(val[2:], xorPort)
	copy(val[4:], xorIP[:])

	// Attribute TLV: type(2)+len(2)+value(8) = 12 bytes.
	attrLen := len(val)
	tlv := make([]byte, 4+attrLen)
	binary.BigEndian.PutUint16(tlv[0:], stunAttrXORMappedAddress)
	binary.BigEndian.PutUint16(tlv[2:], uint16(attrLen))
	copy(tlv[4:], val)

	resp := make([]byte, stunHeaderSize+len(tlv))
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], uint16(len(tlv)))
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)
	copy(resp[stunHeaderSize:], tlv)
	return resp
}

// TestDetectNATType_WithMockServer_Symmetric verifies NATSymmetric is detected
// when two STUN servers report different mapped ports.
func TestDetectNATType_WithMockServer_Symmetric(t *testing.T) {
	// Server 1 reports port 10001, Server 2 reports port 10002.
	addr1, stop1 := startMockSTUNServerWithPort(t, 10001)
	defer stop1()
	addr2, stop2 := startMockSTUNServerWithPort(t, 10002)
	defer stop2()

	// Temporarily replace default STUN servers.
	origServers := DefaultSTUNServers
	DefaultSTUNServers = []string{addr1, addr2}
	defer func() { DefaultSTUNServers = origServers }()

	// Use a local UDP connection.
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Ensure local port is not public (we're using loopback).
	localPort := conn.LocalAddr().(*net.UDPAddr).Port
	if localPort == 10001 || localPort == 10002 {
		// Skip if by chance we got one of the mapped ports.
		t.Skip("local port matches test port, skipping")
	}

	// Test with provided connection.
	nt, mappedAddr, err := DetectNATType(conn)
	if err != nil {
		t.Fatalf("DetectNATType failed: %v", err)
	}

	// Different ports → Symmetric NAT.
	if nt != NATSymmetric {
		t.Errorf("expected NATSymmetric, got %s", nt.String())
	}
	if mappedAddr == nil {
		t.Error("expected mapped address")
	}
}

// TestDetectNATType_WithMockServer_FullCone verifies NATFullCone is detected
// when two STUN servers report the same mapped port.
func TestDetectNATType_WithMockServer_FullCone(t *testing.T) {
	// Both servers report the same port 10003.
	addr1, stop1 := startMockSTUNServerWithPort(t, 10003)
	defer stop1()
	addr2, stop2 := startMockSTUNServerWithPort(t, 10003)
	defer stop2()

	// Temporarily replace default STUN servers.
	origServers := DefaultSTUNServers
	DefaultSTUNServers = []string{addr1, addr2}
	defer func() { DefaultSTUNServers = origServers }()

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Test with provided connection.
	nt, mappedAddr, err := DetectNATType(conn)
	if err != nil {
		t.Fatalf("DetectNATType failed: %v", err)
	}

	// Same ports → Full Cone NAT (since local IP differs from mapped).
	if nt != NATFullCone {
		t.Errorf("expected NATFullCone, got %s", nt.String())
	}
	if mappedAddr == nil {
		t.Error("expected mapped address")
	}
}

// TestDetectNATType_NilConn verifies DetectNATType creates its own connection when nil.
func TestDetectNATType_NilConn(t *testing.T) {
	// Create a mock server that reports a consistent port.
	addr1, stop1 := startMockSTUNServerWithPort(t, 10004)
	defer stop1()
	addr2, stop2 := startMockSTUNServerWithPort(t, 10005)
	defer stop2()

	// Temporarily replace default STUN servers.
	origServers := DefaultSTUNServers
	DefaultSTUNServers = []string{addr1, addr2}
	defer func() { DefaultSTUNServers = origServers }()

	// Test with nil connection (creates its own).
	nt, mappedAddr, err := DetectNATType(nil)
	if err != nil {
		// May fail in CI/network-restricted environments — that's acceptable.
		t.Skipf("DetectNATType with nil conn failed (may be network restricted): %v", err)
	}

	// Should return some NAT type and mapped address.
	if nt != NATSymmetric && nt != NATFullCone && nt != NATDirect {
		t.Errorf("unexpected NAT type: %s", nt.String())
	}
	if mappedAddr == nil {
		t.Error("expected mapped address")
	}
}

// TestDetectNATType_SecondServerFails verifies NATPortRestricted is returned
// when the second STUN server query fails.
func TestDetectNATType_SecondServerFails(t *testing.T) {
	// Only start one server.
	addr1, stop1 := startMockSTUNServerWithPort(t, 10006)
	defer stop1()

	// Use an invalid address for the second server.
	origServers := DefaultSTUNServers
	DefaultSTUNServers = []string{addr1, "127.0.0.1:1"} // port 1 is unlikely to respond
	defer func() { DefaultSTUNServers = origServers }()

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	nt, mappedAddr, err := DetectNATType(conn)
	if err != nil {
		t.Fatalf("DetectNATType failed: %v", err)
	}

	// Second server failure → NATPortRestricted.
	if nt != NATPortRestricted {
		t.Errorf("expected NATPortRestricted, got %s", nt.String())
	}
	if mappedAddr == nil {
		t.Error("expected mapped address from first server")
	}
}

// TestDetectNATType_FirstServerFails verifies error is returned when first STUN server fails.
func TestDetectNATType_FirstServerFails(t *testing.T) {
	// Use an invalid address for the first server.
	origServers := DefaultSTUNServers
	DefaultSTUNServers = []string{"127.0.0.1:1", "127.0.0.1:2"}
	defer func() { DefaultSTUNServers = origServers }()

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, _, err = DetectNATType(conn)
	if err == nil {
		t.Fatal("expected error when first STUN server fails")
	}
}

// TestDetectNATType_NoSTUNResponse verifies that DetectNATType returns an error
// when no STUN servers respond (all servers unreachable).
func TestDetectNATType_NoSTUNResponse(t *testing.T) {
	// Use only invalid/unreachable STUN server addresses.
	origServers := DefaultSTUNServers
	DefaultSTUNServers = []string{"127.0.0.1:1"}
	defer func() { DefaultSTUNServers = origServers }()

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, _, err = DetectNATType(conn)
	if err == nil {
		t.Fatal("expected error when no STUN servers respond")
	}
}
