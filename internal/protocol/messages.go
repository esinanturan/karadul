package protocol

import (
	"encoding/binary"
	"fmt"
)

// MsgHandshakeInit is the initiator's first Noise IK handshake message.
//
// Wire layout (104 bytes):
//
//	[0]    type = TypeHandshakeInit
//	[1:4]  reserved (zero)
//	[4:8]  senderIndex (local session ID, little-endian uint32)
//	[8:40] ephemeral public key (32 bytes)
//	[40:88] encrypted static public key + tag (32 + 16 = 48 bytes)
//	[88:104] encrypted empty payload tag (16 bytes)
type MsgHandshakeInit struct {
	SenderIndex uint32
	Ephemeral   [32]byte
	EncStatic   [48]byte // encrypted static + poly1305 tag
	EncPayload  [16]byte // 16-byte auth tag for empty payload
}

// MarshalBinary encodes the message to its 104-byte wire representation.
func (m *MsgHandshakeInit) MarshalBinary() []byte {
	b := make([]byte, HandshakeInitSize)
	b[0] = TypeHandshakeInit
	// b[1:4] = 0 (reserved)
	binary.LittleEndian.PutUint32(b[4:8], m.SenderIndex)
	copy(b[8:40], m.Ephemeral[:])
	copy(b[40:88], m.EncStatic[:])
	copy(b[88:104], m.EncPayload[:])
	return b
}

// UnmarshalMsgHandshakeInit decodes a HandshakeInit from wire bytes.
func UnmarshalMsgHandshakeInit(b []byte) (*MsgHandshakeInit, error) {
	if len(b) < HandshakeInitSize {
		return nil, fmt.Errorf("handshake init: too short %d < %d", len(b), HandshakeInitSize)
	}
	if b[0] != TypeHandshakeInit {
		return nil, fmt.Errorf("handshake init: wrong type 0x%02x", b[0])
	}
	m := &MsgHandshakeInit{}
	m.SenderIndex = binary.LittleEndian.Uint32(b[4:8])
	copy(m.Ephemeral[:], b[8:40])
	copy(m.EncStatic[:], b[40:88])
	copy(m.EncPayload[:], b[88:104])
	return m, nil
}

// MsgHandshakeResp is the responder's reply in the Noise IK handshake.
//
// Wire layout (60 bytes):
//
//	[0]    type = TypeHandshakeResp
//	[1:4]  reserved
//	[4:8]  senderIndex
//	[8:12] receiverIndex (echo of initiator's senderIndex)
//	[12:44] ephemeral public key (32 bytes)
//	[44:60] encrypted empty payload tag (16 bytes)
type MsgHandshakeResp struct {
	SenderIndex   uint32
	ReceiverIndex uint32
	Ephemeral     [32]byte
	EncPayload    [16]byte
}

// MarshalBinary encodes the message to its 60-byte wire representation.
func (m *MsgHandshakeResp) MarshalBinary() []byte {
	b := make([]byte, HandshakeRespSize)
	b[0] = TypeHandshakeResp
	binary.LittleEndian.PutUint32(b[4:8], m.SenderIndex)
	binary.LittleEndian.PutUint32(b[8:12], m.ReceiverIndex)
	copy(b[12:44], m.Ephemeral[:])
	copy(b[44:60], m.EncPayload[:])
	return b
}

// UnmarshalMsgHandshakeResp decodes a HandshakeResp from wire bytes.
func UnmarshalMsgHandshakeResp(b []byte) (*MsgHandshakeResp, error) {
	if len(b) < HandshakeRespSize {
		return nil, fmt.Errorf("handshake resp: too short %d < %d", len(b), HandshakeRespSize)
	}
	if b[0] != TypeHandshakeResp {
		return nil, fmt.Errorf("handshake resp: wrong type 0x%02x", b[0])
	}
	m := &MsgHandshakeResp{}
	m.SenderIndex = binary.LittleEndian.Uint32(b[4:8])
	m.ReceiverIndex = binary.LittleEndian.Uint32(b[8:12])
	copy(m.Ephemeral[:], b[12:44])
	copy(m.EncPayload[:], b[44:60])
	return m, nil
}

// MsgData is a transport data packet.
//
// Wire layout:
//
//	[0]    type = TypeData
//	[1:4]  reserved
//	[4:8]  receiverIndex (little-endian uint32)
//	[8:16] counter (little-endian uint64)
//	[16:]  ciphertext + 16-byte auth tag
type MsgData struct {
	ReceiverIndex uint32
	Counter       uint64
	Ciphertext    []byte // includes auth tag
}

// MarshalBinary encodes the data message.
func (m *MsgData) MarshalBinary() []byte {
	b := make([]byte, DataHeaderSize+len(m.Ciphertext))
	b[0] = TypeData
	binary.LittleEndian.PutUint32(b[4:8], m.ReceiverIndex)
	binary.LittleEndian.PutUint64(b[8:16], m.Counter)
	copy(b[DataHeaderSize:], m.Ciphertext)
	return b
}

// UnmarshalMsgData decodes a data message from wire bytes (header only; ciphertext is a slice).
func UnmarshalMsgData(b []byte) (*MsgData, error) {
	if len(b) < DataHeaderSize+AuthTagSize {
		return nil, fmt.Errorf("data msg: too short %d", len(b))
	}
	if b[0] != TypeData {
		return nil, fmt.Errorf("data msg: wrong type 0x%02x", b[0])
	}
	m := &MsgData{}
	m.ReceiverIndex = binary.LittleEndian.Uint32(b[4:8])
	m.Counter = binary.LittleEndian.Uint64(b[8:16])
	m.Ciphertext = b[DataHeaderSize:]
	return m, nil
}

// MsgKeepalive is a keepalive packet.
//
// Wire layout (8 bytes):
//
//	[0]   type = TypeKeepalive
//	[1:4] reserved
//	[4:8] receiverIndex
type MsgKeepalive struct {
	ReceiverIndex uint32
}

func (m *MsgKeepalive) MarshalBinary() []byte {
	b := make([]byte, 8)
	b[0] = TypeKeepalive
	binary.LittleEndian.PutUint32(b[4:8], m.ReceiverIndex)
	return b
}
