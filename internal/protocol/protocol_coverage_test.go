package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// ---------------------------------------------------------------------------
// Constants verification
// ---------------------------------------------------------------------------

func TestConstants(t *testing.T) {
	if HandshakeInitSize != 104 {
		t.Errorf("HandshakeInitSize: want 104, got %d", HandshakeInitSize)
	}
	if HandshakeRespSize != 60 {
		t.Errorf("HandshakeRespSize: want 60, got %d", HandshakeRespSize)
	}
	if DataHeaderSize != 16 {
		t.Errorf("DataHeaderSize: want 16, got %d", DataHeaderSize)
	}
	if AuthTagSize != 16 {
		t.Errorf("AuthTagSize: want 16, got %d", AuthTagSize)
	}
	if MaxMTU != 1420 {
		t.Errorf("MaxMTU: want 1420, got %d", MaxMTU)
	}
	if MaxPacketSize != DataHeaderSize+MaxMTU+AuthTagSize+64 {
		t.Errorf("MaxPacketSize mismatch")
	}
}

func TestTypeConstants(t *testing.T) {
	if TypeHandshakeInit != 0x01 {
		t.Errorf("TypeHandshakeInit: want 0x01, got 0x%02x", TypeHandshakeInit)
	}
	if TypeHandshakeResp != 0x02 {
		t.Errorf("TypeHandshakeResp: want 0x02, got 0x%02x", TypeHandshakeResp)
	}
	if TypeData != 0x03 {
		t.Errorf("TypeData: want 0x03, got 0x%02x", TypeData)
	}
	if TypeKeepalive != 0x04 {
		t.Errorf("TypeKeepalive: want 0x04, got 0x%02x", TypeKeepalive)
	}
}

func TestMagic(t *testing.T) {
	want := [4]byte{'K', 'R', 'D', 'L'}
	if Magic != want {
		t.Errorf("Magic: want %v, got %v", want, Magic)
	}
}

// ---------------------------------------------------------------------------
// HandshakeInit — edge cases
// ---------------------------------------------------------------------------

func TestHandshakeInit_ZeroValues(t *testing.T) {
	m := &MsgHandshakeInit{}
	b := m.MarshalBinary()
	if len(b) != HandshakeInitSize {
		t.Fatalf("size: got %d, want %d", len(b), HandshakeInitSize)
	}
	if b[0] != TypeHandshakeInit {
		t.Fatalf("type byte: got 0x%02x", b[0])
	}
	// Reserved bytes should be zero.
	for i := 1; i < 4; i++ {
		if b[i] != 0 {
			t.Errorf("reserved byte %d: want 0, got %d", i, b[i])
		}
	}
	// SenderIndex should be 0.
	if binary.LittleEndian.Uint32(b[4:8]) != 0 {
		t.Error("expected zero SenderIndex")
	}
}

func TestHandshakeInit_MaxSenderIndex(t *testing.T) {
	m := &MsgHandshakeInit{SenderIndex: 0xFFFFFFFF}
	b := m.MarshalBinary()
	m2, err := UnmarshalMsgHandshakeInit(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != 0xFFFFFFFF {
		t.Errorf("SenderIndex: got %d, want %d", m2.SenderIndex, uint32(0xFFFFFFFF))
	}
}

func TestHandshakeInit_AllFieldsSet(t *testing.T) {
	m := &MsgHandshakeInit{
		SenderIndex: 0x12345678,
	}
	for i := range m.Ephemeral {
		m.Ephemeral[i] = 0xFF
	}
	for i := range m.EncStatic {
		m.EncStatic[i] = 0xAA
	}
	for i := range m.EncPayload {
		m.EncPayload[i] = 0xBB
	}

	b := m.MarshalBinary()
	m2, err := UnmarshalMsgHandshakeInit(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != m.SenderIndex {
		t.Error("SenderIndex mismatch")
	}
	if m2.Ephemeral != m.Ephemeral {
		t.Error("Ephemeral mismatch")
	}
	if m2.EncStatic != m.EncStatic {
		t.Error("EncStatic mismatch")
	}
	if m2.EncPayload != m.EncPayload {
		t.Error("EncPayload mismatch")
	}
}

func TestHandshakeInit_TooShort(t *testing.T) {
	b := make([]byte, HandshakeInitSize-1)
	b[0] = TypeHandshakeInit
	_, err := UnmarshalMsgHandshakeInit(b)
	if err == nil {
		t.Fatal("expected error for short buffer")
	}
}

func TestHandshakeInit_Empty(t *testing.T) {
	_, err := UnmarshalMsgHandshakeInit(nil)
	if err == nil {
		t.Fatal("expected error for nil buffer")
	}
}

func TestHandshakeInit_WrongType(t *testing.T) {
	b := make([]byte, HandshakeInitSize)
	b[0] = TypeData // wrong type
	_, err := UnmarshalMsgHandshakeInit(b)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestHandshakeInit_ExtraTrailingData(t *testing.T) {
	m := &MsgHandshakeInit{SenderIndex: 42}
	b := m.MarshalBinary()
	// Append extra bytes — should still parse correctly.
	extra := append(b, make([]byte, 100)...)
	m2, err := UnmarshalMsgHandshakeInit(extra)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != 42 {
		t.Error("SenderIndex mismatch with trailing data")
	}
}

func TestHandshakeInit_ReservedBytesIgnored(t *testing.T) {
	m := &MsgHandshakeInit{SenderIndex: 1}
	b := m.MarshalBinary()
	// Set reserved bytes to non-zero.
	b[1] = 0xFF
	b[2] = 0xFF
	b[3] = 0xFF
	m2, err := UnmarshalMsgHandshakeInit(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != 1 {
		t.Error("SenderIndex should still be 1")
	}
}

// ---------------------------------------------------------------------------
// HandshakeResp — edge cases
// ---------------------------------------------------------------------------

func TestHandshakeResp_ZeroValues(t *testing.T) {
	m := &MsgHandshakeResp{}
	b := m.MarshalBinary()
	if len(b) != HandshakeRespSize {
		t.Fatalf("size: got %d, want %d", len(b), HandshakeRespSize)
	}
	m2, err := UnmarshalMsgHandshakeResp(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != 0 || m2.ReceiverIndex != 0 {
		t.Error("expected zero indices")
	}
}

func TestHandshakeResp_MaxIndices(t *testing.T) {
	m := &MsgHandshakeResp{
		SenderIndex:   0xFFFFFFFF,
		ReceiverIndex: 0xFFFFFFFF,
	}
	b := m.MarshalBinary()
	m2, err := UnmarshalMsgHandshakeResp(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != 0xFFFFFFFF {
		t.Error("SenderIndex overflow")
	}
	if m2.ReceiverIndex != 0xFFFFFFFF {
		t.Error("ReceiverIndex overflow")
	}
}

func TestHandshakeResp_TooShort(t *testing.T) {
	b := make([]byte, HandshakeRespSize-1)
	b[0] = TypeHandshakeResp
	_, err := UnmarshalMsgHandshakeResp(b)
	if err == nil {
		t.Fatal("expected error for short buffer")
	}
}

func TestHandshakeResp_Empty(t *testing.T) {
	_, err := UnmarshalMsgHandshakeResp(nil)
	if err == nil {
		t.Fatal("expected error for nil buffer")
	}
}

func TestHandshakeResp_WrongType(t *testing.T) {
	b := make([]byte, HandshakeRespSize)
	b[0] = TypeHandshakeInit
	_, err := UnmarshalMsgHandshakeResp(b)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestHandshakeResp_AllFieldsSet(t *testing.T) {
	m := &MsgHandshakeResp{
		SenderIndex:   0xAABBCCDD,
		ReceiverIndex: 0x11223344,
	}
	for i := range m.Ephemeral {
		m.Ephemeral[i] = byte(i)
	}
	for i := range m.EncPayload {
		m.EncPayload[i] = 0xCC
	}
	b := m.MarshalBinary()
	m2, err := UnmarshalMsgHandshakeResp(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != m.SenderIndex {
		t.Error("SenderIndex mismatch")
	}
	if m2.ReceiverIndex != m.ReceiverIndex {
		t.Error("ReceiverIndex mismatch")
	}
	if m2.Ephemeral != m.Ephemeral {
		t.Error("Ephemeral mismatch")
	}
	if m2.EncPayload != m.EncPayload {
		t.Error("EncPayload mismatch")
	}
}

func TestHandshakeResp_ExtraTrailingData(t *testing.T) {
	m := &MsgHandshakeResp{SenderIndex: 99}
	b := m.MarshalBinary()
	extra := append(b, 0xDE, 0xAD, 0xBE, 0xEF)
	m2, err := UnmarshalMsgHandshakeResp(extra)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != 99 {
		t.Error("SenderIndex mismatch with trailing data")
	}
}

// ---------------------------------------------------------------------------
// DataMsg — edge cases
// ---------------------------------------------------------------------------

func TestDataMsg_EmptyCiphertext(t *testing.T) {
	m := &MsgData{
		ReceiverIndex: 1,
		Counter:       0,
		Ciphertext:    []byte{},
	}
	b := m.MarshalBinary()
	if len(b) != DataHeaderSize {
		t.Fatalf("expected %d bytes for empty ciphertext, got %d", DataHeaderSize, len(b))
	}
	// Empty ciphertext means no auth tag, so UnmarshalMsgData should fail.
	_, err := UnmarshalMsgData(b)
	if err == nil {
		t.Fatal("expected error: no auth tag")
	}
}

func TestDataMsg_MinimalValid(t *testing.T) {
	ct := make([]byte, AuthTagSize) // minimum: just auth tag
	m := &MsgData{
		ReceiverIndex: 0x12345678,
		Counter:       0,
		Ciphertext:    ct,
	}
	b := m.MarshalBinary()
	if len(b) != DataHeaderSize+AuthTagSize {
		t.Fatalf("size: got %d, want %d", len(b), DataHeaderSize+AuthTagSize)
	}
	m2, err := UnmarshalMsgData(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.ReceiverIndex != m.ReceiverIndex {
		t.Error("ReceiverIndex mismatch")
	}
	if m2.Counter != 0 {
		t.Error("Counter should be 0")
	}
	if len(m2.Ciphertext) != AuthTagSize {
		t.Errorf("ciphertext len: got %d, want %d", len(m2.Ciphertext), AuthTagSize)
	}
}

func TestDataMsg_MaxCounter(t *testing.T) {
	m := &MsgData{
		ReceiverIndex: 0,
		Counter:       ^uint64(0),
		Ciphertext:    make([]byte, AuthTagSize),
	}
	b := m.MarshalBinary()
	m2, err := UnmarshalMsgData(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.Counter != ^uint64(0) {
		t.Errorf("Counter: got %d, want max uint64", m2.Counter)
	}
}

func TestDataMsg_LargePayload(t *testing.T) {
	ct := make([]byte, MaxMTU)
	for i := range ct {
		ct[i] = byte(i % 256)
	}
	m := &MsgData{
		ReceiverIndex: 42,
		Counter:       100,
		Ciphertext:    ct,
	}
	b := m.MarshalBinary()
	m2, err := UnmarshalMsgData(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(m2.Ciphertext) != MaxMTU {
		t.Errorf("ciphertext len: got %d, want %d", len(m2.Ciphertext), MaxMTU)
	}
}

func TestDataMsg_ExactlyMinSize(t *testing.T) {
	b := make([]byte, DataHeaderSize+AuthTagSize)
	b[0] = TypeData
	binary.LittleEndian.PutUint32(b[4:8], 0xABCD)
	binary.LittleEndian.PutUint64(b[8:16], 999)
	m, err := UnmarshalMsgData(b)
	if err != nil {
		t.Fatal(err)
	}
	if m.ReceiverIndex != 0xABCD {
		t.Errorf("ReceiverIndex: got 0x%x", m.ReceiverIndex)
	}
	if m.Counter != 999 {
		t.Errorf("Counter: got %d", m.Counter)
	}
}

func TestDataMsg_OneByteShort(t *testing.T) {
	b := make([]byte, DataHeaderSize+AuthTagSize-1)
	b[0] = TypeData
	_, err := UnmarshalMsgData(b)
	if err == nil {
		t.Fatal("expected error for one byte short")
	}
}

// ---------------------------------------------------------------------------
// Keepalive — edge cases
// ---------------------------------------------------------------------------

func TestKeepalive_ZeroReceiverIndex(t *testing.T) {
	m := &MsgKeepalive{ReceiverIndex: 0}
	b := m.MarshalBinary()
	if len(b) != 8 {
		t.Fatalf("size: got %d, want 8", len(b))
	}
	// All bytes after type should be zero (reserved + 0 receiver index).
	for i := 1; i < 8; i++ {
		if b[i] != 0 {
			t.Errorf("byte %d: want 0, got %d", i, b[i])
		}
	}
}

func TestKeepalive_MaxReceiverIndex(t *testing.T) {
	m := &MsgKeepalive{ReceiverIndex: 0xFFFFFFFF}
	b := m.MarshalBinary()
	idx := binary.LittleEndian.Uint32(b[4:8])
	if idx != 0xFFFFFFFF {
		t.Errorf("ReceiverIndex: got 0x%x, want 0xFFFFFFFF", idx)
	}
}

func TestKeepalive_TypeByte(t *testing.T) {
	m := &MsgKeepalive{ReceiverIndex: 1}
	b := m.MarshalBinary()
	if b[0] != TypeKeepalive {
		t.Errorf("type byte: want 0x%02x, got 0x%02x", TypeKeepalive, b[0])
	}
}

// ---------------------------------------------------------------------------
// ParseType — exhaustive boundary tests
// ---------------------------------------------------------------------------

func TestParseType_AllKnownTypes(t *testing.T) {
	types := map[uint8]struct{}{
		TypeHandshakeInit: {},
		TypeHandshakeResp: {},
		TypeData:          {},
		TypeKeepalive:     {},
	}
	for typ := range types {
		got, err := ParseType([]byte{typ})
		if err != nil {
			t.Fatalf("ParseType(0x%02x): %v", typ, err)
		}
		if got != typ {
			t.Errorf("ParseType(0x%02x): got 0x%02x", typ, got)
		}
	}
}

func TestParseType_AllUnknownBytes(t *testing.T) {
	known := map[uint8]bool{
		TypeHandshakeInit: true,
		TypeHandshakeResp: true,
		TypeData:          true,
		TypeKeepalive:     true,
	}
	for i := 0; i < 256; i++ {
		b := byte(i)
		if known[b] {
			continue
		}
		_, err := ParseType([]byte{b})
		if err == nil {
			t.Errorf("ParseType(0x%02x): expected error for unknown type", b)
		}
	}
}

func TestParseType_SingleBytePacket(t *testing.T) {
	got, err := ParseType([]byte{TypeData})
	if err != nil {
		t.Fatal(err)
	}
	if got != TypeData {
		t.Errorf("got 0x%02x, want 0x%02x", got, TypeData)
	}
}

// ---------------------------------------------------------------------------
// Round-trip integrity tests
// ---------------------------------------------------------------------------

func TestRoundTrip_HandshakeInit(t *testing.T) {
	original := &MsgHandshakeInit{
		SenderIndex: 0xDEADBEEF,
	}
	for i := range original.Ephemeral {
		original.Ephemeral[i] = byte(i)
	}
	for i := range original.EncStatic {
		original.EncStatic[i] = byte(255 - i)
	}
	for i := range original.EncPayload {
		original.EncPayload[i] = byte(i * 2)
	}

	wire := original.MarshalBinary()
	decoded, err := UnmarshalMsgHandshakeInit(wire)
	if err != nil {
		t.Fatal(err)
	}

	// Re-encode and compare.
	wire2 := decoded.MarshalBinary()
	if !bytes.Equal(wire, wire2) {
		t.Fatal("round-trip wire mismatch")
	}
}

func TestRoundTrip_HandshakeResp(t *testing.T) {
	original := &MsgHandshakeResp{
		SenderIndex:   0x11223344,
		ReceiverIndex: 0x55667788,
	}
	for i := range original.Ephemeral {
		original.Ephemeral[i] = byte(100 + i)
	}
	for i := range original.EncPayload {
		original.EncPayload[i] = byte(i + 50)
	}

	wire := original.MarshalBinary()
	decoded, err := UnmarshalMsgHandshakeResp(wire)
	if err != nil {
		t.Fatal(err)
	}
	wire2 := decoded.MarshalBinary()
	if !bytes.Equal(wire, wire2) {
		t.Fatal("round-trip wire mismatch")
	}
}

func TestRoundTrip_DataMsg(t *testing.T) {
	ct := make([]byte, 200+AuthTagSize)
	for i := range ct {
		ct[i] = byte(i % 256)
	}
	original := &MsgData{
		ReceiverIndex: 0x87654321,
		Counter:       123456789,
		Ciphertext:    ct,
	}

	wire := original.MarshalBinary()
	decoded, err := UnmarshalMsgData(wire)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.ReceiverIndex != original.ReceiverIndex {
		t.Error("ReceiverIndex mismatch")
	}
	if decoded.Counter != original.Counter {
		t.Error("Counter mismatch")
	}
	if !bytes.Equal(decoded.Ciphertext, original.Ciphertext) {
		t.Error("Ciphertext mismatch")
	}
}

func TestRoundTrip_Keepalive(t *testing.T) {
	original := &MsgKeepalive{ReceiverIndex: 0xCAFEBABE}
	wire := original.MarshalBinary()
	// Keepalive has no Unmarshal, so verify the wire format directly.
	idx := binary.LittleEndian.Uint32(wire[4:8])
	if idx != 0xCAFEBABE {
		t.Errorf("ReceiverIndex: got 0x%x, want 0xCAFEBABE", idx)
	}
}

// ---------------------------------------------------------------------------
// Wire format layout verification
// ---------------------------------------------------------------------------

func TestHandshakeInit_WireLayout(t *testing.T) {
	m := &MsgHandshakeInit{SenderIndex: 0x01020304}
	m.Ephemeral = [32]byte{0xEE}
	m.EncStatic[0] = 0xDD
	m.EncPayload[0] = 0xCC

	b := m.MarshalBinary()

	// [0] type
	if b[0] != TypeHandshakeInit {
		t.Fatalf("type byte: got 0x%02x", b[0])
	}
	// [1:4] reserved
	if !bytes.Equal(b[1:4], []byte{0, 0, 0}) {
		t.Error("reserved bytes not zero")
	}
	// [4:8] senderIndex (little-endian)
	if binary.LittleEndian.Uint32(b[4:8]) != 0x01020304 {
		t.Error("senderIndex encoding wrong")
	}
	// [8:40] ephemeral
	if b[8] != 0xEE {
		t.Error("ephemeral first byte wrong")
	}
	// [40:88] encrypted static
	if b[40] != 0xDD {
		t.Error("encstatic first byte wrong")
	}
	// [88:104] encrypted payload tag
	if b[88] != 0xCC {
		t.Error("encpayload first byte wrong")
	}
}

func TestHandshakeResp_WireLayout(t *testing.T) {
	m := &MsgHandshakeResp{
		SenderIndex:   0xAABBCCDD,
		ReceiverIndex: 0x11223344,
	}
	m.Ephemeral = [32]byte{0xFF}
	m.EncPayload[0] = 0xBB

	b := m.MarshalBinary()

	// [0] type
	if b[0] != TypeHandshakeResp {
		t.Fatalf("type byte: got 0x%02x", b[0])
	}
	// [4:8] senderIndex
	if binary.LittleEndian.Uint32(b[4:8]) != 0xAABBCCDD {
		t.Error("senderIndex encoding wrong")
	}
	// [8:12] receiverIndex
	if binary.LittleEndian.Uint32(b[8:12]) != 0x11223344 {
		t.Error("receiverIndex encoding wrong")
	}
	// [12:44] ephemeral
	if b[12] != 0xFF {
		t.Error("ephemeral first byte wrong")
	}
	// [44:60] encrypted payload tag
	if b[44] != 0xBB {
		t.Error("encpayload first byte wrong")
	}
}

func TestDataMsg_WireLayout(t *testing.T) {
	ct := make([]byte, AuthTagSize)
	ct[0] = 0x99
	m := &MsgData{
		ReceiverIndex: 0x12345678,
		Counter:       0x0102030405060708,
		Ciphertext:    ct,
	}

	b := m.MarshalBinary()

	// [0] type
	if b[0] != TypeData {
		t.Fatalf("type byte: got 0x%02x", b[0])
	}
	// [4:8] receiverIndex
	if binary.LittleEndian.Uint32(b[4:8]) != 0x12345678 {
		t.Error("receiverIndex encoding wrong")
	}
	// [8:16] counter
	if binary.LittleEndian.Uint64(b[8:16]) != 0x0102030405060708 {
		t.Error("counter encoding wrong")
	}
	// [16:] ciphertext
	if b[16] != 0x99 {
		t.Error("ciphertext first byte wrong")
	}
}
