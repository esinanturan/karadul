package nat

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// parseBindingResponse edge cases
// ---------------------------------------------------------------------------

// TestParseBindingResponse_TooShortHeader verifies the early return when the
// response buffer is smaller than the 20-byte STUN header.
func TestParseBindingResponse_TooShortHeader(t *testing.T) {
	txID := make([]byte, 12)
	// 19 bytes — one short of the required header size.
	buf := make([]byte, stunHeaderSize-1)
	binary.BigEndian.PutUint16(buf[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint32(buf[4:], stunMagicCookie)
	copy(buf[8:], txID)

	_, err := parseBindingResponse(buf, txID)
	if err == nil {
		t.Fatal("expected error for response shorter than STUN header")
	}
}

// TestParseBindingResponse_EmptyBuffer verifies that an empty buffer is rejected.
func TestParseBindingResponse_EmptyBuffer(t *testing.T) {
	_, err := parseBindingResponse([]byte{}, make([]byte, 12))
	if err == nil {
		t.Fatal("expected error for empty buffer")
	}
}

// TestParseBindingResponse_AttributeOverflow verifies the break when an attribute
// body extends beyond the message end (pos+attrLen > end).
func TestParseBindingResponse_AttributeOverflow(t *testing.T) {
	txID := make([]byte, 12)

	// Build a response that claims 8 bytes of attributes, but the attribute
	// says its value is 200 bytes long — far beyond the available data.
	val := make([]byte, 200) // large fake value
	tlv := make([]byte, 4+8) // only 8 bytes of value actually present
	binary.BigEndian.PutUint16(tlv[0:], stunAttrXORMappedAddress)
	binary.BigEndian.PutUint16(tlv[2:], uint16(len(val))) // claims 200 bytes
	copy(tlv[4:], []byte{0x00, stunAddrFamilyIPv4, 0x00, 0x50, 192, 168, 1, 1})

	resp := make([]byte, stunHeaderSize+len(tlv))
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], uint16(len(tlv)))
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)
	copy(resp[stunHeaderSize:], tlv)

	_, err := parseBindingResponse(resp, txID)
	if err == nil {
		t.Fatal("expected error: no mapped address should be found when attribute overflows")
	}
}

// TestParseBindingResponse_BindingErrorType verifies that a STUN Binding Error
// response type (0x0111) is rejected.
func TestParseBindingResponse_BindingErrorType(t *testing.T) {
	txID := make([]byte, 12)
	resp := make([]byte, stunHeaderSize)
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingError)
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)

	_, err := parseBindingResponse(resp, txID)
	if err == nil {
		t.Fatal("expected error for binding error message type")
	}
}

// TestParseBindingResponse_UnknownAttributeSkipped verifies that unknown attribute
// types are silently skipped and the parser continues looking for mapped addresses.
func TestParseBindingResponse_UnknownAttributeSkipped(t *testing.T) {
	txID := make([]byte, 12)

	// Build an attribute with an unknown type (0x8888), followed by no valid addr.
	unknownVal := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	tlv := make([]byte, 4+len(unknownVal))
	binary.BigEndian.PutUint16(tlv[0:], 0x8888) // unknown type
	binary.BigEndian.PutUint16(tlv[2:], uint16(len(unknownVal)))
	copy(tlv[4:], unknownVal)

	resp := make([]byte, stunHeaderSize+len(tlv))
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], uint16(len(tlv)))
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)
	copy(resp[stunHeaderSize:], tlv)

	_, err := parseBindingResponse(resp, txID)
	if err == nil {
		t.Fatal("expected error: no mapped address after skipping unknown attribute")
	}
}

// TestParseBindingResponse_MultipleAttributesWithXORPreferred verifies that when
// both MAPPED-ADDRESS and XOR-MAPPED-ADDRESS are present, the XOR variant is
// preferred.
func TestParseBindingResponse_MultipleAttributesWithXORPreferred(t *testing.T) {
	txID := make([]byte, 12)

	// MAPPED-ADDRESS attribute: port 11111, IP 10.0.0.1
	maVal := []byte{
		0x00, stunAddrFamilyIPv4,
		0x2B, 0x67, // port 11111
		10, 0, 0, 1,
	}
	maTLV := make([]byte, 4+len(maVal))
	binary.BigEndian.PutUint16(maTLV[0:], stunAttrMappedAddress)
	binary.BigEndian.PutUint16(maTLV[2:], uint16(len(maVal)))
	copy(maTLV[4:], maVal)

	// XOR-MAPPED-ADDRESS attribute: port 22222, IP 192.168.1.1
	magicBytes := [4]byte{0x21, 0x12, 0xA4, 0x42}
	xorPort := uint16(22222) ^ uint16(stunMagicCookie>>16)
	xorIP := [4]byte{
		192 ^ magicBytes[0],
		168 ^ magicBytes[1],
		1 ^ magicBytes[2],
		1 ^ magicBytes[3],
	}
	xmaVal := make([]byte, 8)
	xmaVal[1] = stunAddrFamilyIPv4
	binary.BigEndian.PutUint16(xmaVal[2:], xorPort)
	copy(xmaVal[4:], xorIP[:])

	xmaTLV := make([]byte, 4+len(xmaVal))
	binary.BigEndian.PutUint16(xmaTLV[0:], stunAttrXORMappedAddress)
	binary.BigEndian.PutUint16(xmaTLV[2:], uint16(len(xmaVal)))
	copy(xmaTLV[4:], xmaVal)

	attrPayload := append(maTLV, xmaTLV...)

	resp := make([]byte, stunHeaderSize+len(attrPayload))
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], uint16(len(attrPayload)))
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)
	copy(resp[stunHeaderSize:], attrPayload)

	addr, err := parseBindingResponse(resp, txID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// XOR-MAPPED-ADDRESS should be preferred over MAPPED-ADDRESS.
	if addr.Port != 22222 {
		t.Errorf("expected XOR-mapped port 22222, got %d", addr.Port)
	}
	if !addr.IP.Equal(net.IPv4(192, 168, 1, 1)) {
		t.Errorf("expected XOR-mapped IP 192.168.1.1, got %s", addr.IP)
	}
}

// TestParseBindingResponse_PaddedAttribute verifies that attributes with padding
// (non-4-byte-aligned length) are parsed correctly — the parser advances by the
// padded length and correctly finds the subsequent attribute.
func TestParseBindingResponse_PaddedAttribute(t *testing.T) {
	txID := make([]byte, 12)

	// Use an unknown attribute with a 5-byte value (padded to 8 for alignment),
	// followed by a valid MAPPED-ADDRESS attribute.
	unknownVal := []byte{0x01, 0x02, 0x03, 0x04, 0x05} // 5 bytes, padded to 8
	unknownTLV := make([]byte, 4+len(unknownVal))
	binary.BigEndian.PutUint16(unknownTLV[0:], 0x9999)
	binary.BigEndian.PutUint16(unknownTLV[2:], uint16(len(unknownVal)))
	copy(unknownTLV[4:], unknownVal)

	maVal := []byte{
		0x00, stunAddrFamilyIPv4,
		0x1F, 0x90, // port 8080
		172, 16, 0, 1,
	}
	maTLV := make([]byte, 4+len(maVal))
	binary.BigEndian.PutUint16(maTLV[0:], stunAttrMappedAddress)
	binary.BigEndian.PutUint16(maTLV[2:], uint16(len(maVal)))
	copy(maTLV[4:], maVal)

	// Padding: 5 bytes → padded to 8. Total attr space:
	//   4 (unknown TLV header) + 8 (padded value) + 12 (MAPPED-ADDRESS TLV) = 24
	paddedValLen := (len(unknownVal) + 3) &^ 3 // 8
	totalAttrLen := 4 + paddedValLen + len(maTLV) // 4 + 8 + 12 = 24

	// Build the full payload: unknown TLV header + padded value + mapped address
	payload := make([]byte, totalAttrLen)
	// Copy the unknown TLV (header + value, 9 bytes)
	copy(payload, unknownTLV)
	// Copy the MAPPED-ADDRESS TLV at offset 4 + paddedValLen = 12
	copy(payload[4+paddedValLen:], maTLV)

	resp := make([]byte, stunHeaderSize+len(payload))
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], uint16(len(payload)))
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)
	copy(resp[stunHeaderSize:], payload)

	addr, err := parseBindingResponse(resp, txID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr.Port != 8080 {
		t.Errorf("expected port 8080, got %d", addr.Port)
	}
	if !addr.IP.Equal(net.IPv4(172, 16, 0, 1)) {
		t.Errorf("expected IP 172.16.0.1, got %s", addr.IP)
	}
}

// ---------------------------------------------------------------------------
// parseMappedAddress edge cases
// ---------------------------------------------------------------------------

// TestParseMappedAddress_IPv4_TooShortAtBoundary verifies that exactly 8 bytes
// with family IPv4 succeeds, but 7 bytes fails (the inner IPv4 length check).
func TestParseMappedAddress_IPv4_TooShortAtBoundary(t *testing.T) {
	// 7 bytes: passes the outer len(b) < 8 check but fails the IPv4 inner check.
	b := make([]byte, 7)
	b[1] = stunAddrFamilyIPv4
	binary.BigEndian.PutUint16(b[2:], 80)

	_, err := parseMappedAddress(b)
	if err == nil {
		t.Fatal("expected error for IPv4 mapped address with 7 bytes")
	}
}

// TestParseMappedAddress_Exactly8BytesIPv4 verifies that exactly 8 bytes with
// family IPv4 succeeds.
func TestParseMappedAddress_Exactly8BytesIPv4(t *testing.T) {
	b := make([]byte, 8)
	b[1] = stunAddrFamilyIPv4
	binary.BigEndian.PutUint16(b[2:], 443)
	b[4] = 8
	b[5] = 8
	b[6] = 8
	b[7] = 8

	addr, err := parseMappedAddress(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr.Port != 443 {
		t.Errorf("port: want 443, got %d", addr.Port)
	}
	if !addr.IP.Equal(net.IPv4(8, 8, 8, 8)) {
		t.Errorf("IP: want 8.8.8.8, got %s", addr.IP)
	}
}

// TestParseMappedAddress_IPv6_Exactly20Bytes verifies that exactly 20 bytes with
// family IPv6 succeeds.
func TestParseMappedAddress_IPv6_Exactly20Bytes(t *testing.T) {
	b := make([]byte, 20)
	b[1] = stunAddrFamilyIPv6
	binary.BigEndian.PutUint16(b[2:], 54321)
	ip := net.ParseIP("fe80::1").To16()
	copy(b[4:], ip)

	addr, err := parseMappedAddress(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr.Port != 54321 {
		t.Errorf("port: want 54321, got %d", addr.Port)
	}
	if !addr.IP.Equal(ip) {
		t.Errorf("IP: want %s, got %s", ip, addr.IP)
	}
}

// TestParseMappedAddress_IPv6_19BytesTooShort verifies that 19 bytes with family
// IPv6 fails the inner length check.
func TestParseMappedAddress_IPv6_19BytesTooShort(t *testing.T) {
	b := make([]byte, 19)
	b[1] = stunAddrFamilyIPv6
	binary.BigEndian.PutUint16(b[2:], 9999)

	_, err := parseMappedAddress(b)
	if err == nil {
		t.Fatal("expected error for IPv6 with 19 bytes")
	}
}

// ---------------------------------------------------------------------------
// parseXORMappedAddress edge cases
// ---------------------------------------------------------------------------

// TestParseXORMappedAddress_IPv4_Exactly8Bytes verifies that exactly 8 bytes
// with IPv4 family decodes correctly.
func TestParseXORMappedAddress_IPv4_Exactly8Bytes(t *testing.T) {
	expectedPort := uint16(12345)
	expectedIP := []byte{203, 0, 113, 50} // 4-byte IPv4

	magicBytes := [4]byte{0x21, 0x12, 0xA4, 0x42}
	xorPort := expectedPort ^ uint16(stunMagicCookie>>16)
	xorIP := [4]byte{
		expectedIP[0] ^ magicBytes[0],
		expectedIP[1] ^ magicBytes[1],
		expectedIP[2] ^ magicBytes[2],
		expectedIP[3] ^ magicBytes[3],
	}

	b := make([]byte, 8)
	b[1] = stunAddrFamilyIPv4
	binary.BigEndian.PutUint16(b[2:], xorPort)
	copy(b[4:], xorIP[:])

	addr, err := parseXORMappedAddress(b, make([]byte, 12))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr.Port != int(expectedPort) {
		t.Errorf("port: want %d, got %d", expectedPort, addr.Port)
	}
	if !addr.IP.Equal(net.IP(expectedIP)) {
		t.Errorf("IP: want %s, got %s", net.IP(expectedIP), addr.IP)
	}
}

// TestParseXORMappedAddress_IPv4_5BytesTooShort verifies that 5 bytes with IPv4
// family fails the inner length check.
func TestParseXORMappedAddress_IPv4_5BytesTooShort(t *testing.T) {
	b := make([]byte, 5)
	b[1] = stunAddrFamilyIPv4
	binary.BigEndian.PutUint16(b[2:], 0x1234)

	_, err := parseXORMappedAddress(b, make([]byte, 12))
	if err == nil {
		t.Fatal("expected error for IPv4 XOR mapped address with only 5 bytes")
	}
}

// TestParseXORMappedAddress_IPv4_7BytesTooShort verifies that 7 bytes with IPv4
// family fails the inner length check.
func TestParseXORMappedAddress_IPv4_7BytesTooShort(t *testing.T) {
	b := make([]byte, 7)
	b[1] = stunAddrFamilyIPv4
	binary.BigEndian.PutUint16(b[2:], 0x5678)

	_, err := parseXORMappedAddress(b, make([]byte, 12))
	if err == nil {
		t.Fatal("expected error for IPv4 XOR mapped address with only 7 bytes")
	}
}

// TestParseXORMappedAddress_IPv6_19BytesTooShort verifies that 19 bytes with IPv6
// family fails the inner length check.
func TestParseXORMappedAddress_IPv6_19BytesTooShort(t *testing.T) {
	b := make([]byte, 19)
	b[1] = stunAddrFamilyIPv6
	binary.BigEndian.PutUint16(b[2:], 0x9999)

	_, err := parseXORMappedAddress(b, make([]byte, 12))
	if err == nil {
		t.Fatal("expected error for IPv6 XOR mapped address with only 19 bytes")
	}
}

// TestParseXORMappedAddress_IPv6_ShortTxID verifies that IPv6 decoding fails when
// the txID is fewer than 12 bytes.
func TestParseXORMappedAddress_IPv6_ShortTxID(t *testing.T) {
	b := make([]byte, 20)
	b[1] = stunAddrFamilyIPv6
	binary.BigEndian.PutUint16(b[2:], 0x1234)

	_, err := parseXORMappedAddress(b, make([]byte, 5)) // only 5-byte txID
	if err == nil {
		t.Fatal("expected error for IPv6 XOR with short txID")
	}
}

// TestParseXORMappedAddress_IPv6_NilTxID verifies that IPv6 decoding fails when
// txID is nil.
func TestParseXORMappedAddress_IPv6_NilTxID(t *testing.T) {
	b := make([]byte, 20)
	b[1] = stunAddrFamilyIPv6
	binary.BigEndian.PutUint16(b[2:], 0x1234)

	_, err := parseXORMappedAddress(b, nil)
	if err == nil {
		t.Fatal("expected error for IPv6 XOR with nil txID")
	}
}

// TestParseXORMappedAddress_UnknownFamily verifies that an unknown address family
// is rejected.
func TestParseXORMappedAddress_UnknownFamily(t *testing.T) {
	b := make([]byte, 8)
	b[1] = 0xFF // unknown family
	binary.BigEndian.PutUint16(b[2:], 0x1234)

	_, err := parseXORMappedAddress(b, make([]byte, 12))
	if err == nil {
		t.Fatal("expected error for unknown address family in XOR mapped address")
	}
}

// TestParseXORMappedAddress_3BytesTooShort verifies the minimum length check
// (< 4 bytes).
func TestParseXORMappedAddress_3BytesTooShort(t *testing.T) {
	_, err := parseXORMappedAddress([]byte{0x00, 0x01, 0x02}, nil)
	if err == nil {
		t.Fatal("expected error for XOR mapped address with only 3 bytes")
	}
}

// ---------------------------------------------------------------------------
// parseBindingResponse: MAPPED-ADDRESS parse error is swallowed
// ---------------------------------------------------------------------------

// TestParseBindingResponse_MalformedMappedAddressFallsThrough verifies that if
// MAPPED-ADDRESS attribute data is malformed (e.g. only 3 bytes), the error is
// silently swallowed and parsing continues. With no other valid address, the
// function should return "no mapped address" error.
func TestParseBindingResponse_MalformedMappedAddressFallsThrough(t *testing.T) {
	txID := make([]byte, 12)

	// MAPPED-ADDRESS with only 3 bytes of value — too short to parse.
	maVal := []byte{0x00, stunAddrFamilyIPv4, 0x10} // only 3 bytes
	maTLV := make([]byte, 4+len(maVal))
	binary.BigEndian.PutUint16(maTLV[0:], stunAttrMappedAddress)
	binary.BigEndian.PutUint16(maTLV[2:], uint16(len(maVal)))
	copy(maTLV[4:], maVal)

	resp := make([]byte, stunHeaderSize+len(maTLV))
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], uint16(len(maTLV)))
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)
	copy(resp[stunHeaderSize:], maTLV)

	_, err := parseBindingResponse(resp, txID)
	if err == nil {
		t.Fatal("expected error when mapped address attribute is malformed")
	}
}

// TestParseBindingResponse_MalformedXORMappedAddressFallsThrough verifies that if
// XOR-MAPPED-ADDRESS attribute data is malformed, the error is silently swallowed
// and the function returns "no mapped address".
func TestParseBindingResponse_MalformedXORMappedAddressFallsThrough(t *testing.T) {
	txID := make([]byte, 12)

	// XOR-MAPPED-ADDRESS with only 3 bytes of value — too short.
	xmaVal := []byte{0x00, stunAddrFamilyIPv4, 0x10} // only 3 bytes
	xmaTLV := make([]byte, 4+len(xmaVal))
	binary.BigEndian.PutUint16(xmaTLV[0:], stunAttrXORMappedAddress)
	binary.BigEndian.PutUint16(xmaTLV[2:], uint16(len(xmaVal)))
	copy(xmaTLV[4:], xmaVal)

	resp := make([]byte, stunHeaderSize+len(xmaTLV))
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], uint16(len(xmaTLV)))
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)
	copy(resp[stunHeaderSize:], xmaTLV)

	_, err := parseBindingResponse(resp, txID)
	if err == nil {
		t.Fatal("expected error when XOR mapped address attribute is malformed")
	}
}

// ---------------------------------------------------------------------------
// parseBindingResponse: attribute iteration boundary
// ---------------------------------------------------------------------------

// TestParseBindingResponse_AttributeExactly4BytesFromEnd verifies that an
// attribute starting exactly at end-4 is parsed (pos+4 <= end is true).
func TestParseBindingResponse_AttributeExactly4BytesFromEnd(t *testing.T) {
	txID := make([]byte, 12)

	maVal := []byte{
		0x00, stunAddrFamilyIPv4,
		0x00, 0x50, // port 80
		10, 0, 0, 1,
	}
	maTLV := make([]byte, 4+len(maVal))
	binary.BigEndian.PutUint16(maTLV[0:], stunAttrMappedAddress)
	binary.BigEndian.PutUint16(maTLV[2:], uint16(len(maVal)))
	copy(maTLV[4:], maVal)

	resp := make([]byte, stunHeaderSize+len(maTLV))
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], uint16(len(maTLV)))
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)
	copy(resp[stunHeaderSize:], maTLV)

	addr, err := parseBindingResponse(resp, txID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr.Port != 80 {
		t.Errorf("port: want 80, got %d", addr.Port)
	}
	if !addr.IP.Equal(net.IPv4(10, 0, 0, 1)) {
		t.Errorf("IP: want 10.0.0.1, got %s", addr.IP)
	}
}

// TestParseBindingResponse_AttributePastEnd verifies that when the message length
// header claims 0 attributes, the loop body is never entered and "no mapped
// address" error is returned.
func TestParseBindingResponse_ZeroAttributes(t *testing.T) {
	txID := make([]byte, 12)
	resp := make([]byte, stunHeaderSize)
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], 0) // 0 bytes of attributes
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)

	_, err := parseBindingResponse(resp, txID)
	if err == nil {
		t.Fatal("expected error for response with zero attributes")
	}
}

// ---------------------------------------------------------------------------
// IPv6 XOR-MAPPED-ADDRESS: full round-trip through parseBindingResponse
// ---------------------------------------------------------------------------

// TestParseBindingResponse_XORMappedAddress_IPv6 verifies end-to-end parsing of
// an IPv6 XOR-MAPPED-ADDRESS within a full STUN binding response.
func TestParseBindingResponse_XORMappedAddress_IPv6(t *testing.T) {
	txID := make([]byte, 12)
	for i := range txID {
		txID[i] = byte(i + 1)
	}

	expectedIP := net.ParseIP("2001:db8:abcd::1").To16()
	expectedPort := 55555

	xorKey := make([]byte, 16)
	binary.BigEndian.PutUint32(xorKey[0:], stunMagicCookie)
	copy(xorKey[4:], txID)

	xorPort := uint16(expectedPort) ^ uint16(stunMagicCookie>>16)

	xmaVal := make([]byte, 20)
	xmaVal[1] = stunAddrFamilyIPv6
	binary.BigEndian.PutUint16(xmaVal[2:], xorPort)
	for i := 0; i < 16; i++ {
		xmaVal[4+i] = expectedIP[i] ^ xorKey[i]
	}

	xmaTLV := make([]byte, 4+len(xmaVal))
	binary.BigEndian.PutUint16(xmaTLV[0:], stunAttrXORMappedAddress)
	binary.BigEndian.PutUint16(xmaTLV[2:], uint16(len(xmaVal)))
	copy(xmaTLV[4:], xmaVal)

	resp := make([]byte, stunHeaderSize+len(xmaTLV))
	binary.BigEndian.PutUint16(resp[0:], stunMsgTypeBindingResponse)
	binary.BigEndian.PutUint16(resp[2:], uint16(len(xmaTLV)))
	binary.BigEndian.PutUint32(resp[4:], stunMagicCookie)
	copy(resp[8:20], txID)
	copy(resp[stunHeaderSize:], xmaTLV)

	addr, err := parseBindingResponse(resp, txID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr.Port != expectedPort {
		t.Errorf("port: want %d, got %d", expectedPort, addr.Port)
	}
	if !addr.IP.Equal(expectedIP) {
		t.Errorf("IP: want %s, got %s", expectedIP, addr.IP)
	}
}

// ---------------------------------------------------------------------------
// buildBindingRequest verification
// ---------------------------------------------------------------------------

// TestBuildBindingRequest_VerifyFields verifies that buildBindingRequest produces
// a correctly formatted STUN Binding Request header.
func TestBuildBindingRequest_VerifyFields(t *testing.T) {
	txID := make([]byte, 12)
	for i := range txID {
		txID[i] = byte(i)
	}

	req := buildBindingRequest(txID)

	if len(req) != stunHeaderSize {
		t.Fatalf("request length: want %d, got %d", stunHeaderSize, len(req))
	}

	msgType := binary.BigEndian.Uint16(req[0:])
	if msgType != stunMsgTypeBindingRequest {
		t.Errorf("message type: want 0x%04x, got 0x%04x", stunMsgTypeBindingRequest, msgType)
	}

	msgLen := binary.BigEndian.Uint16(req[2:])
	if msgLen != 0 {
		t.Errorf("message length: want 0, got %d", msgLen)
	}

	cookie := binary.BigEndian.Uint32(req[4:])
	if cookie != stunMagicCookie {
		t.Errorf("magic cookie: want 0x%08x, got 0x%08x", stunMagicCookie, cookie)
	}

	for i := 0; i < 12; i++ {
		if req[8+i] != txID[i] {
			t.Errorf("txID[%d]: want %02x, got %02x", i, txID[i], req[8+i])
		}
	}
}

// ---------------------------------------------------------------------------
// HolePunch: context cancellation
// ---------------------------------------------------------------------------

// TestHolePunch_ContextCancelled verifies that HolePunch returns the context
// error when the context is already cancelled at call time.
func TestHolePunch_ContextCancelled(t *testing.T) {
	local, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer local.Close()

	remote, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer remote.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = HolePunch(ctx, local, remote.LocalAddr().(*net.UDPAddr))
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
	if err != context.Canceled {
		t.Errorf("want context.Canceled, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DetectNATType: Direct (local IP == mapped IP)
// ---------------------------------------------------------------------------

// TestDetectNATType_DirectConnection verifies that when the local IP matches
// the mapped IP from STUN, NATDirect is returned.
func TestDetectNATType_DirectConnection(t *testing.T) {
	localIP := net.IPv4(127, 0, 0, 1)
	addr1, stop1 := startMockSTUNServerWithPortAndIP(t, 10050, localIP)
	defer stop1()
	addr2, stop2 := startMockSTUNServerWithPortAndIP(t, 10050, localIP)
	defer stop2()

	origServers := DefaultSTUNServers
	DefaultSTUNServers = []string{addr1, addr2}
	defer func() { DefaultSTUNServers = origServers }()

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: localIP, Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Ensure local port matches the mapped port to trigger Direct detection.
	// Since the mock server reports localIP as the mapped IP, and the local
	// address matches, this should detect NATDirect.
	nt, addr, err := DetectNATType(conn)
	if err != nil {
		t.Fatalf("DetectNATType: %v", err)
	}
	if nt != NATDirect {
		t.Errorf("expected NATDirect, got %s", nt.String())
	}
	if addr == nil {
		t.Error("expected non-nil mapped address")
	}
}

// ---------------------------------------------------------------------------
// DetectNATType: nil conn creates its own socket
// ---------------------------------------------------------------------------

// TestDetectNATType_NilConn_WithMock verifies the nil conn path with mock servers
// so it succeeds deterministically.
func TestDetectNATType_NilConn_WithMock(t *testing.T) {
	addr1, stop1 := startMockSTUNServerWithPort(t, 10070)
	defer stop1()
	addr2, stop2 := startMockSTUNServerWithPort(t, 10071)
	defer stop2()

	origServers := DefaultSTUNServers
	DefaultSTUNServers = []string{addr1, addr2}
	defer func() { DefaultSTUNServers = origServers }()

	nt, addr, err := DetectNATType(nil)
	if err != nil {
		t.Fatalf("DetectNATType with nil conn: %v", err)
	}
	if addr == nil {
		t.Error("expected non-nil mapped address")
	}
	// With different ports and local IP != mapped IP, should be symmetric.
	if nt != NATSymmetric {
		t.Errorf("expected NATSymmetric, got %s", nt.String())
	}
}

// ---------------------------------------------------------------------------
// HolePunch: write error path (remote closed)
// ---------------------------------------------------------------------------

// TestHolePunch_WriteError verifies that HolePunch handles write errors gracefully
// and eventually returns a failure result.
func TestHolePunch_WriteError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping write error test in short mode")
	}

	local, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer local.Close()

	// Use a closed remote address to trigger write errors.
	remote, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	remoteAddr := remote.LocalAddr().(*net.UDPAddr)
	remote.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := HolePunch(ctx, local, remoteAddr)
	if err != nil {
		// Context error is acceptable.
		if err == context.DeadlineExceeded {
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("HolePunch should not succeed when writes fail")
	}
}
