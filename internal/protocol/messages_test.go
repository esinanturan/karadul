package protocol

import (
	"testing"
)

func TestHandshakeInitMarshal(t *testing.T) {
	m := &MsgHandshakeInit{
		SenderIndex: 0xDEADBEEF,
	}
	for i := range m.Ephemeral {
		m.Ephemeral[i] = byte(i)
	}
	for i := range m.EncStatic {
		m.EncStatic[i] = byte(i + 32)
	}
	for i := range m.EncPayload {
		m.EncPayload[i] = byte(i + 80)
	}

	b := m.MarshalBinary()
	if len(b) != HandshakeInitSize {
		t.Fatalf("marshal size: got %d, want %d", len(b), HandshakeInitSize)
	}
	if b[0] != TypeHandshakeInit {
		t.Fatalf("wrong type byte: %d", b[0])
	}

	m2, err := UnmarshalMsgHandshakeInit(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != m.SenderIndex {
		t.Fatalf("SenderIndex: got %d, want %d", m2.SenderIndex, m.SenderIndex)
	}
	if m2.Ephemeral != m.Ephemeral {
		t.Fatal("Ephemeral mismatch")
	}
	if m2.EncStatic != m.EncStatic {
		t.Fatal("EncStatic mismatch")
	}
	if m2.EncPayload != m.EncPayload {
		t.Fatal("EncPayload mismatch")
	}
}

func TestHandshakeRespMarshal(t *testing.T) {
	m := &MsgHandshakeResp{
		SenderIndex:   0x11223344,
		ReceiverIndex: 0xAABBCCDD,
	}
	for i := range m.Ephemeral {
		m.Ephemeral[i] = byte(i + 1)
	}

	b := m.MarshalBinary()
	if len(b) != HandshakeRespSize {
		t.Fatalf("marshal size: got %d, want %d", len(b), HandshakeRespSize)
	}

	m2, err := UnmarshalMsgHandshakeResp(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.SenderIndex != m.SenderIndex {
		t.Fatalf("SenderIndex: got %d, want %d", m2.SenderIndex, m.SenderIndex)
	}
	if m2.ReceiverIndex != m.ReceiverIndex {
		t.Fatalf("ReceiverIndex: got %d, want %d", m2.ReceiverIndex, m.ReceiverIndex)
	}
}

func TestDataMsgMarshal(t *testing.T) {
	ct := make([]byte, 100+AuthTagSize)
	for i := range ct {
		ct[i] = byte(i)
	}
	m := &MsgData{
		ReceiverIndex: 0x12345678,
		Counter:       42,
		Ciphertext:    ct,
	}
	b := m.MarshalBinary()

	m2, err := UnmarshalMsgData(b)
	if err != nil {
		t.Fatal(err)
	}
	if m2.ReceiverIndex != m.ReceiverIndex {
		t.Fatalf("ReceiverIndex mismatch")
	}
	if m2.Counter != m.Counter {
		t.Fatalf("Counter mismatch")
	}
	if len(m2.Ciphertext) != len(ct) {
		t.Fatalf("ciphertext length: got %d, want %d", len(m2.Ciphertext), len(ct))
	}
}

func TestKeepaliveMarshal(t *testing.T) {
	m := &MsgKeepalive{ReceiverIndex: 0xCAFEBABE}
	b := m.MarshalBinary()
	if len(b) != 8 {
		t.Fatalf("MsgKeepalive: want 8 bytes, got %d", len(b))
	}
	if b[0] != TypeKeepalive {
		t.Errorf("type byte: want %d, got %d", TypeKeepalive, b[0])
	}
	// ReceiverIndex is at bytes [4:8] little-endian.
	idx := uint32(b[4]) | uint32(b[5])<<8 | uint32(b[6])<<16 | uint32(b[7])<<24
	if idx != 0xCAFEBABE {
		t.Errorf("ReceiverIndex: want 0xCAFEBABE, got 0x%08X", idx)
	}
}

func TestParseType(t *testing.T) {
	for _, tc := range []struct {
		pkt  []byte
		want uint8
	}{
		{[]byte{TypeHandshakeInit, 0, 0, 0}, TypeHandshakeInit},
		{[]byte{TypeHandshakeResp}, TypeHandshakeResp},
		{[]byte{TypeData, 1, 2}, TypeData},
		{[]byte{TypeKeepalive}, TypeKeepalive},
	} {
		got, err := ParseType(tc.pkt)
		if err != nil {
			t.Fatalf("ParseType(%x): %v", tc.pkt, err)
		}
		if got != tc.want {
			t.Fatalf("ParseType(%x): got %d, want %d", tc.pkt, got, tc.want)
		}
	}

	// Unknown type.
	if _, err := ParseType([]byte{0xFF}); err == nil {
		t.Fatal("expected error for unknown type")
	}

	// Empty.
	if _, err := ParseType(nil); err == nil {
		t.Fatal("expected error for empty packet")
	}
}

// TestUnmarshalMsgData_WrongType verifies UnmarshalMsgData returns an error when
// the type byte is not TypeData (covers the b[0] != TypeData branch).
func TestUnmarshalMsgData_WrongType(t *testing.T) {
	// Build a buffer large enough (DataHeaderSize + AuthTagSize) but with wrong type.
	b := make([]byte, DataHeaderSize+AuthTagSize)
	b[0] = TypeHandshakeInit // wrong type
	if _, err := UnmarshalMsgData(b); err == nil {
		t.Fatal("expected error for wrong type byte in UnmarshalMsgData")
	}
}

// TestUnmarshalMsgData_TooShort verifies UnmarshalMsgData returns an error for short buffers.
func TestUnmarshalMsgData_TooShort(t *testing.T) {
	b := make([]byte, DataHeaderSize-1)
	if _, err := UnmarshalMsgData(b); err == nil {
		t.Fatal("expected error for too-short buffer in UnmarshalMsgData")
	}
}
